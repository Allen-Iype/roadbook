# Phase 2 design brief — one adventure on a map

Status: draft for review (Gate 1). No application code exists for this phase.

Goal, from `docs/PLAN.md`: open a confirmed adventure and see the route, with gaps
visibly marked as unknown. Straight lines only; routing is phase 3.

Inputs to this brief: the phase 2 section of `docs/PLAN.md`; "Carried forward" in
`docs/phase-1/LOG.md`; `docs/feature-comparison-dawarich-roadbook.md` §1.6, §1.9,
§1.10 (adopted algorithms — §2 rejections are final); the committed golden fixture
`testdata/journey-27jul2026.anon.json` / `.expected.json`; and Dawarich's
implementation summary as a mechanics reference, scoped below.

**How Dawarich is used in this phase.** It answers map *mechanics*: GeoJSON feature
layout for segmented routes, property-driven styling, updating a source in place,
layer ordering, bounds fitting, stop markers, flight arcs (phase 3). It does not
answer two things, deliberately. (a) MapLibre-in-React — Dawarich is a Rails app
with no React; the lifecycle pattern below comes from MapLibre's and React's own
documentation. (b) The observed-versus-inferred visual channel — Dawarich renders
every route segment with equal confidence, which is exactly what CLAUDE.md
invariants 5 and 8 forbid; that design is original work here, and must not be
absorbed along with the mechanics. Its scale machinery (viewport fetching, slim
arrays) stays rejected.

---

## 1. Concepts this phase introduces

### 1.1 Wrapping imperative MapLibre inside React

This is the hardest React in the project, so it gets the longest explanation.

**The conflict.** React is declarative: a component is a function from props to a
description of the UI, and React re-runs it whenever inputs change, reconciling the
result against the DOM it owns. MapLibre is the opposite: an imperative library that
wants one `<div>` handed over once, into which it installs a WebGL canvas and event
handlers, and which you then *mutate* through method calls (`addSource`, `addLayer`,
`fitBounds`). If React re-rendered "the map" the way it re-renders a table row, it
would destroy and recreate a WebGL context on every render. The whole pattern below
exists to give MapLibre a stable patch of DOM that React promises never to touch.

**The pattern, against the code we will write** (`web/app/adventure/[id]/route-map.tsx`,
a client component like phase 1's `decide-cell.tsx`):

- **A ref to the container.** `useRef<HTMLDivElement>(null)` attached to a `<div>`
  via `ref={...}`. A ref is a mutable box whose `.current` survives re-renders and
  whose mutation does *not* trigger a re-render — precisely the two properties state
  does not have. After React mounts the div, `ref.current` is the real DOM node.
- **Map creation in `useEffect`.** Effects run *after* the DOM exists, only on the
  client, and declare a cleanup. The shape:

  ```tsx
  useEffect(() => {
    const map = new maplibregl.Map({ container: containerRef.current!, ... });
    map.on("load", () => { /* addSource / addLayer / fitBounds here */ });
    return () => map.remove();
  }, []);
  ```

  Three things are load-bearing. First, sources and layers are added inside the
  `load` handler — the style must finish loading before the map accepts them; adding
  them synchronously after the constructor is the classic first bug. Second, the
  cleanup (`map.remove()`) releases the WebGL context; browsers cap live contexts,
  and a leaked map is invisible until the fourth or fifth navigation kills the page.
  Third, React 19 in development StrictMode deliberately runs every effect twice
  (mount → cleanup → mount) to prove cleanups are correct — the map will visibly
  construct twice in dev. That is the framework auditing us, not a bug to suppress.
- **The map instance lives in a ref, never in state.** `useState` is for data that
  renders. The map object is not renderable data; putting it in state schedules a
  pointless re-render and invites effect loops.
- **Server/client boundary.** The journey JSON is fetched in the server component
  page (generated client, same as the list page) and passed as a prop to the island.
  A `"use client"` component still renders its initial HTML on the server, but
  effects never run there — so the div arrives empty and MapLibre attaches only in
  the browser. No `next/dynamic` with `ssr: false` is needed.
- **Static data, minimal machinery.** In this phase the journey is fetched once per
  page view and never changes while the map is mounted, so create-once/destroy-once
  is the whole lifecycle. The idiomatic path for *changing* data —
  `map.getSource("route").setData(nextGeoJSON)`, one of the Dawarich mechanics —
  becomes relevant in phase 3 (re-render after routing). We do not build prop-diffing
  machinery for changes that cannot happen yet.

**Rejected: `react-map-gl` (or any React wrapper).** It hides exactly the lifecycle
this phase exists to establish, adds a dependency whose release cycle must track
MapLibre's majors, and its value is felt with many synchronized layers and
viewport-as-state — a surface we do not have (one map, one source, a handful of
layers). Dawarich's own migration lesson applies: one map library, pinned, forever;
a wrapper is a second library. Rejected likewise: rendering a static route image
server-side (loses zoom/pan, and honesty markings must survive interaction).

**Version pinning.** `maplibre-gl` gets an exact version in `package.json` (no `^`),
per the Dawarich lesson about unauditable map-library drift. Its bundled docs and
TypeScript types get read at install time, before any map code — same rule as Next.

### 1.2 GeoJSON, sources, layers, paint

**GeoJSON** is the interchange format MapLibre consumes: a `FeatureCollection` of
`Feature`s, each a geometry (`LineString`, `Point`, …) plus free-form `properties`.
Two traps. Coordinates are `[longitude, latitude]` — the reverse of both Google's
`"lat°, lon°"` strings and our API's `{lat, lon}` objects; the conversion lives in
exactly one function in the map island, because this is the single most common
mapping bug in existence. And properties are where our leg metadata rides: each leg
becomes one `LineString` feature with `{ kind: "observed" | "gap", gap_kind: ... }`.

**Sources vs layers.** A source is data (`type: "geojson"`, our FeatureCollection).
A layer is one visual treatment of a source — `type: "line"` or `"circle"`, a
`paint` object (`line-color`, `line-width`, `line-dasharray`), and optionally a
`filter` expression selecting which features it draws. Several layers can draw one
source. The plan:

- one `geojson` source, `route`, holding every leg feature plus stop points;
- a `gap` line layer, `filter: ["==", ["get", "kind"], "gap"]`, added first;
- an `observed` line layer filtered the same way, added second (later layers draw
  on top — layer order is z-order, a Dawarich mechanic we adopt);
- a stop `circle` layer above both.

Layer-per-class is chosen over one layer with data-driven `["match", ...]` paint
expressions for two reasons: z-ordering between classes comes free, and dash
patterns (`line-dasharray`) are not data-drivable per feature in MapLibre — a dashed
gap next to a solid observed line *requires* separate layers. (To be re-verified
against the bundled docs at install time, like every MapLibre claim in this brief.)

**Bounds fitting** (mechanic): compute the bounding box of all journey points once,
`map.fitBounds(bbox, { padding })` inside the `load` handler, so the adventure frames
itself. No viewport state management.

**The basemap.** MapLibre renders our route *over* a base style of tiles that must
come from somewhere. This is the one runtime network dependency a map cannot avoid,
and tile requests leak the viewport to the provider — which is why "configurable
tile provider" is already a phase 5 roadmap item. Phase 2 does the cheap part of
that seam now: the style URL is a config value read from the environment, not a
literal in the component. Choice of default provider is a real decision — §3D.

### 1.3 The leg model and journey assembly (Go)

**The model.** A journey is an ordered list of legs; a leg is observed or a gap;
never one undifferentiated line (invariant 5). New domain types:

```go
type LegKind string   // "observed" | "gap"
type GapKind string   // "unknown" | "road" | "air"

type Leg struct {
    Kind     LegKind
    GapKind  GapKind      // meaningful only when Kind == gap; always "unknown" in phase 2
    Points   []TimedPoint // observed: the actual trace; gap: exactly its two endpoints
    Start    time.Time
    End      time.Time
    DistanceKm float64
}

type Journey struct {
    Legs   []Leg
    Stops  []Stop
    // totals, provenance fractions, params echo
}
```

`GapKind` exists from the first commit even though phase 2 only ever writes
`unknown`: phase 3 classifies gaps (`road` routed, `air` for implied speeds above a
named threshold, rendered as great-circle arcs) and consumes this field. Retrofitting
it after the renderer exists means reworking assembly *and* renderer — the exact
trap the plan names. The kind travels on the leg through the API to the renderer, so
honesty survives serialization (invariant 8) instead of being a styling choice.

**Assembly is a pure function** (invariant 1): `journey.Assemble(points, activities,
params) → Journey`, in a new `internal/journey` package. No I/O, no clock. Inputs are
the merged, time-sorted observation points inside the candidate's span; every
threshold is a named parameter (invariant 3), seeded from the golden fixture's
recorded values: `GapThresholdMinutes: 20` (a silence longer than this closes the
observed leg and emits a gap leg), `MinGeometryPoints: 4`, `JoinSlackSeconds: 120`
(the trace/rawSignals merge tolerance). Stop detection uses the adopted Dawarich
stay-point sweep (§1.6): a single O(n) pass holding an open stay with a running-mean
anchor; a point joins iff it is within `radius` of the anchor *and* within
`radius × 1.5` of the stay's first point (the drift cap that stops a slow walk from
smearing the anchor); break on a time gap, emit on minimum dwell and point count,
merge stays separated by brief absences. All five parameters named. DBSCAN rejected
with it — the sweep is O(n) and explainable, which is the project's detection style.

**The golden fixture pins assembly the way the prototype pins detection.**
`testdata/journey-27jul2026.anon.json` (longitudes shifted by a constant — every
distance, duration, and leg boundary identical to the real journey) must reproduce
`.expected.json` exactly: 69 trace points merging with rawSignals to 121; 672.6 km
chord sum; 7 observed legs, 6 gap legs; 33.0% of distance inferred; the six gap
durations; the rest halt 21:29–21:57 with 0.2 km movement; 2.4% disagreement with
Google's own activity sum. Because the fixture is committed and anonymised, this
regression runs everywhere `make test` runs — unlike the `data/`-gated detection
suites. These are also the only public numbers, reproducible by the test (invariant 13).

### 1.4 PostGIS and the countries table

PostGIS enters the stack here, for point-in-polygon country attribution *only*. Leg
segmentation stays in the pure Go core — that decision is logged (PLAN, Dawarich
audit) and closed; the Dawarich lesson (§1.9) about its JS and SQL segmentation
paths silently disagreeing is the argument: exactly one segmentation implementation,
exercised by every caller, and ours is the Go function the golden fixture pins.

What PostGIS provides: a `geometry`/`geography` column type, a GiST spatial index,
and `ST_Contains(polygon, point)` — so "which countries did this journey cross"
becomes one indexed SQL query over a bundled table of country MultiPolygons, no
network call ever (the same offline property demanded of routing). A new goose
migration enables the extension and creates `countries(iso_code, name, geom)`;
polygon data comes from Natural Earth (public domain). Which resolution, and how it
enters the repo without violating the 1 MB rule, is a real choice — §3E.

Rejected: reverse-geocoding points (runtime service dependency, comparison §2, final);
point-in-polygon in Go against an embedded shapefile (viable, but re-implements the
GiST index PostGIS already has, and PLAN already commits PostGIS to this phase).

### 1.5 rawSignals ingestion

`rawSignals` position fixes are the export's only high-accuracy data: per-fix
accuracy in meters and a source label (WIFI / CELL / GPS). Each export carries only
~30 days of them, which is why exports are banked monthly. The phase 1 parser
recognises and skips them token-by-token; this phase extends it to emit a domain
type (invariant 4 — no parser type escapes) and stores them in a new `raw_positions`
table via a new migration: content-hash idempotent, immutable, never mutated by
derived computation — exactly like visits, activities, and path points.

The anomaly-filter extensions from the plan land here, because this is where
accuracy fields first exist: exact (0,0) and accuracy-worse-than-threshold, as named
parameters, flagging never deleting.

Assembly merges raw positions with trace points by timestamp (`JoinSlackSeconds`
dedup). The golden fixture's own verdict is worth stating honestly: for that
journey, rawSignals added 52 points and filled zero gaps, and only 1 of its 120
in-window fixes was true GPS. The merge is in the pinned pipeline (the golden
numbers are the merged ones), but the measured benefit today is densification, not
gap-filling — which is why ingestion is one checkpoint, not the phase's center. See
§3C for the interpretation this brief assumes.

### 1.6 Confidence scoring with stored breakdown

Adapted from Dawarich's visit scoring (§1.7): a candidate scores 0–100 from named
weighted components — proposed: distance-from-home, destination dwell, observation
density, span duration; exact weights are implementation-time parameters recorded
per run (invariant 3). A component that cannot be computed has its weight
*redistributed* across the rest, not scored zero — otherwise missing data reads as
low confidence, which is a lie of the same family invariant 8 forbids on the map.
The per-component breakdown is stored on the candidate row (candidates are
regenerated wholesale each run, so this is a new column pair via migration:
`score`, `score_breakdown` jsonb) and *shown* in the confirm UI. Decision support,
never auto-confirmation — auto-confirming above a threshold would delete the
product's deliberate confirmation step.

Scoring runs inside the pure detection core (it is a ranking concern, and "rank,
don't filter" already lives there), so it is testable without infrastructure.

### 1.7 The name-suggestion seam

At confirm time, suggest a name ("Journey to Kalpetta?") from a one-shot geocoder
lookup on the destination coordinate — behind the same
pluggable-with-offline-fallback pattern as routing (invariant 7's shape): a
`Suggester` interface, a live geocoder implementation, and a null implementation
that suggests nothing and says so. The seam and the null implementation land in
phase 2; whether the live geocoder lands with them is a scoping choice — §3F. The
suggestion is prefill for the existing controlled name input, never auto-applied.

---

## 2. What gets built

- `internal/journey`: pure assembly (legs with `kind`/`gap_kind`, stay-point stops,
  chord distances, provenance fractions), pinned by the golden fixture.
- `roadbook journey <candidate-id>`: CLI printing the assembled journey — the same
  numbers the test pins, on demand (invariant 13's reproduction command).
- Parser + store + migration for `raw_positions`; import ingests them idempotently.
- `GET /candidates/{id}/journey` in `openapi.yaml`, server + TS client regenerated.
- `web/app/adventure/[id]/`: server-component detail page (name, dates, distance,
  duration, stop list, provenance line "N% observed / M% inferred") linking from
  the phase 1 list; `route-map.tsx` client island rendering legs per §1.1–1.2.
- Migration + loader for `countries`; countries-crossed on the detail page.
- Scoring in the detect core; `score` + `score_breakdown` on candidates, surfaced
  in the confirm UI.
- `Suggester` seam with null implementation; confirm UI prefill.

Out of scope, explicitly: routing and gap classification beyond `unknown` (phase 3 —
the field exists, its other values do not occur); GPX/GeoJSON export, poster, replay
(post-phase 3); speed-colored routes, viewport fetching, slim arrays (rejected,
final); any timeline-browsing surface.

---

## 3. The real choices

**A. The observed/inferred visual channel (original design, invariants 5 and 8).**
Recommendation: encode the distinction redundantly in *line style and weight*, not
hue alone — observed legs solid, saturated, wider; gap legs dashed, thinner, muted
grey. Dashes read as "sketched in" at every zoom level, survive every colorblindness
class, and print. Hue stays in reserve so phase 3 can introduce routed-road and air
legs as visually third and fourth classes without demoting the observed/inferred
distinction to something subtler. A legend on the detail page states the encoding in
words. Alternatives: hue-only (fails colorblind users and grayscale, and spends the
strongest channel on what dashes already say); opacity-only (ambiguous against any
basemap). Rejected outright: speed coloring (§2, conflicts with this channel).

**B. Where legs live: assembled on demand, or persisted?** Recommendation: on
demand, in the Go service, at API-read time — no legs table. The journey is derived
data over immutable inputs (invariant 2), assembly is O(points-in-span) over at most
a few thousand points, and re-assembly with different parameters must stay free; a
persisted copy is a cache that can silently disagree with its inputs, and phase 3
(routing cache) is where persistence earns its place. The response echoes the
parameters that produced it (invariant 3). Cost accepted: assembly runs per page
view — measured against tens of journeys, not thousands.

**C. rawSignals in the assembly pipeline.** The expected-values file records both a
merged (121 pt / 672.6 km) and a trace-only (69 pt / 671.1 km) measurement, and its
`rawSignalsVerdict` reads "adds 52 points, fills zero gaps … excluded from the
pipeline". This brief assumes the *merged* numbers are the pinned contract — they
are the ones stated publicly for the fixture — and reads the verdict as a scoping
finding (rawSignals densify but do not fill gaps, so ingestion is one checkpoint,
not a pillar), not as an instruction to keep rawSignals out of assembly. **Confirm
or correct at this gate**; if trace-only is the contract, checkpoint 1's target
numbers change and ingestion decouples from the golden test entirely.

**D. Default basemap.** The style URL is env-config from day one either way (the
phase 5 provider seam, done cheap now). A default is still needed. Recommendation:
OpenFreeMap's public vector style — no API key, no account, usable offline-ethos
story (it is self-hostable later). Alternatives: raw OSM raster tiles (usage policy
discourages app default traffic); MapTiler/Carto free tiers (API keys in a
self-hosted product's default path). To be validated once at implementation.

**E. Countries data: resolution vs the 1 MB rule.** Natural Earth admin-0 at 1:50m
is ~5 MB GeoJSON — over the repository's 1 MB per-file line, which exists as a
data-safety tripwire and should not gain exceptions casually. Recommendation: commit
the 1:110m polygons gzipped (~250 KB), adequate for country-level attribution of
adventure-scale journeys, and let the loader accept an optional higher-resolution
file from disk for anyone who wants tighter borders — no build-time network fetch
(the phase 1 font lesson). Alternative rejected: fetching 1:50m at setup (a network
dependency in the install path); committing 1:50m regardless (blunts the 1 MB rule).
Border-adjacent misattribution at 110m is possible; the countries line is labelled
as derived, and upgrading resolution is a data swap, not a schema change.

**F. What trails if the phase runs long.** Recommendation: the live geocoder
implementation of the `Suggester` (the seam and null implementation do not trail —
they are the cheap part that prevents rework), and nothing else. Countries, scoring,
and rawSignals are all schema-touching and get *more* expensive later; the geocoder
is an isolated implementation behind an interface that already exists.

---

## 4. Checkpoint order — five vertical slices

1. **Journey assembly, pinned.** `internal/journey` + rawSignals parsing (file
   path only) + `roadbook journey -src <file>`; golden regression test green in
   `make test`. Visible: the CLI reproduces every number in `.expected.json`.
2. **Journey through the stack.** `raw_positions` migration + idempotent ingestion;
   anomaly-flag extensions; `GET /candidates/{id}/journey`; detail page without the
   map — name, dates, distance, duration, stops, provenance line, leg table.
   Visible: click a row on the list, read the journey's numbers; curl proves the
   endpoint first (phase 1's ordering rule).
3. **The map.** `maplibre-gl` pinned and its bundled docs read; the client island
   per §1.1; gap and observed layers per §1.2 and §3A; stop markers; bounds fit;
   legend. Visible: the phase goal — a recognisable route whose unknown parts are
   visibly unknown.
4. **Countries crossed.** PostGIS extension + `countries` migration + bundled
   polygons + loader; point-in-polygon query; countries line on the detail page.
   Visible: "India · Nepal" appears, derived locally.
5. **Confidence and naming.** Scoring in the detect core, breakdown stored and
   shown in the confirm UI; `Suggester` seam with null implementation; geocoder
   prefill if it does not trail (§3F). Visible: the confirm dialog explains *why*
   the machine proposed, and offers a name.

Order rationale: 1 before 2 because the API serves what assembly computes and the
DB path must reproduce the file path's numbers; 2 before 3 because curl-then-UI is
the standing build order; 4 and 5 are independent of 3 but come after so the phase's
namesake lands as early as possible. Each checkpoint ends with something visible
and stops for review, like phase 1's five.

## 5. Verification at phase close

`make test` green everywhere including the golden journey regression (no `data/`
gate); the CLI, the API, and the page agree on every journey number; fixture
detection still 18/1/32 and decisions still attached after all migrations
(re-detection survival re-verified); the map visually confirmed by the maintainer;
no file from `data/`, nothing over 1 MB, in any commit (standing rule).
