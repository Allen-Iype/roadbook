# Phase 3 design brief — routing fills the gaps

Written before any code, per the working agreement. Goal (PLAN.md): gaps become
road-following inference, clearly labelled as inference. The phase 2 screen is
visibly transformed, and the routed distance is reported alongside the source's
own figure.

Closed decisions this brief builds on, not reopens: one Router interface with
three implementations (PLAN, logged); validation against Google's distance with
air legs excluded (PLAN, logged); observed legs are never routed (invariant 6);
persistence enters here as the routing cache while journeys, countries, and
scores stay derived-on-read (phase 2 LOG, carried forward); hue was held in
reserve in phase 2 (its BRIEF §3A) precisely so this phase could spend it; the
golden fixture's pipeline semantics are pinned by
`testdata/journey-27jul2026.CONTRACT.md` and routing must consume its gap legs
without disturbing them.

---

## 1. Concepts this phase introduces

### 1.1 Interface-based pluggability, and how to choose the seam

An interface in Go is a contract with no implementation: any type with the
right method satisfies it, and the caller cannot tell implementations apart.
The repository already has one working example — `suggest.Suggester`, one
method, a null implementation as the default, a network implementation as
opt-in. Routing gets the same shape because invariant 7 demands it: a
self-hoster without an OSM extract must still see their adventures.

The design work is not the interface syntax; it is choosing *where the seam
sits*. Three criteria, in order:

1. **The pure core stays pure** (invariant 1). Routing is network I/O, so no
   router can be called from inside `journey.Assemble`. The seam must sit
   outside the pure boundary, consuming Assemble's output.
2. **The seam speaks domain types only** — `domain.LatLng` in, geometry out.
   Nothing OSRM-shaped escapes the OSRM client, exactly as no parser type
   escapes `internal/timeline` (invariant 4).
3. **The narrowest surface that does the job.** A router answers one question:
   "what road connects these two points?" One method. Batch orchestration,
   caching, and classification are *not* the router's job — a fatter interface
   would force every implementation (including Null) to reimplement them.

The recommended seam:

```go
// internal/route
type Router interface {
    Route(ctx context.Context, from, to domain.LatLng) (Route, error)
}
type Route struct {
    Points    []domain.LatLng // the road geometry, from near from to near to
    DistanceM float64
    DurationS float64
}
```

with a sentinel `ErrNoRoute` distinguishing "the road network cannot connect
these points" (a data answer, cacheable) from an operational error (network
down, timeout — not cacheable, reported and skipped). Three implementations:
`Null` (always `ErrNoRoute` — see §3B for why), and one OSRM HTTP client used
in two configurations (self-hosted URL, public demo URL). Honesty note: that
is two concrete types, not three; the self-hosted and public cases differ in
URL, timeout, and politeness interval, not in protocol, and duplicating the
client to satisfy a count would be theatre. The invariant's substance — three
ways to run the product — holds.

### 1.2 Batch processing with a cache

Routing runs offline, as a CLI command (`roadbook route`), never in the serve
path. This mirrors how detection already works — `roadbook detect` is a batch
command, `roadbook serve` only reads what it produced — and the cache is the
only bridge between the two worlds:

```
roadbook route ──▶ Router (OSRM) ──▶ route_cache table
                                          │
roadbook serve ──▶ Assemble ──▶ apply cache ──▶ API ──▶ map
```

The serve binary never dials a routing service. At journey-read time it looks
up each unknown gap's endpoints in the cache: hit → the gap becomes a `road`
leg carrying routed geometry; miss → it stays `unknown` and renders as it does
today. Batch re-runs are idempotent the way imports are: already-cached pairs
are skipped, so running `roadbook route` twice costs one run plus a set of
cache hits. The command reports what it did — gaps found, cache hits, newly
routed, no-route answers, failures — because a batch job that says nothing
cannot be trusted to have done anything.

### 1.3 Graceful degradation as a design property

Error handling asks "what do we do when this fails?" Graceful degradation
asks a different question at design time: "what does the product look like at
every level of capability?" — and makes each level a designed state rather
than a failure state. The levels here, best to worst:

1. Self-hosted OSRM, full coverage: gaps become routed roads.
2. Public endpoint: same, slower, rate-limited, someone else's server.
3. OSRM answers `NoRoute` for a gap (patchy OSM coverage — rural and mountain
   roads, exactly where adventures happen): that gap stays `unknown`, drawn
   dashed grey. PLAN states this explicitly: designed behaviour, not a bug.
4. Null router / no routing run at all: the entire phase 2 rendering, which
   was already honest.

Every level renders truthfully with no error page anywhere in the stack. This
is the same property invariant 7 bought for the product and `suggest.Null`
bought for naming; the routing cache is what makes level 3 and 4 free at
serve time.

### 1.4 OSRM, extracts, and why routing is offline in the first place

OSRM (Open Source Routing Machine) answers routing queries against a
preprocessed snapshot of OpenStreetMap data. The preprocessing is the cost:
you download a regional extract (`.osm.pbf`, hundreds of MB to GB), run
`osrm-extract`/`osrm-partition`/`osrm-customize` against a profile (car), and
serve the result with `osrm-routed` — an HTTP API on localhost. The
preprocessing takes minutes to hours and RAM proportional to the region; the
queries afterwards take milliseconds. This asymmetry is why routing is a batch
step with a cache and not a runtime dependency: the expensive thing (a running
OSRM with the right extract) needs to exist only while `roadbook route` runs,
and can then be turned off. §4 states the manual-steps boundary.

---

## 2. What gets built

- `internal/journey`: air classification in the pure core — a gap whose
  implied speed (endpoint chord over gap duration) meets `AirSpeedMinKmh`
  gets `GapAir` at assembly time. One new named parameter. Nothing else in
  the pinned pipeline changes.
- `internal/route` (new): the `Router` interface, `ErrNoRoute`, `Null`, the
  OSRM HTTP client, and `Apply` — a pure function decorating a Journey's
  unknown gaps from a set of cache lookups (testable with a fake, no DB).
- Migration `00005_route_cache.sql`: the cache table and the route-run
  record (§3C, §3G).
- `internal/store`: cache read (batch lookup by key set), cache write, and
  route-run recording.
- `roadbook route`: the batch command — enumerate journeys, collect unknown
  gaps, dedupe keys, consult cache, ask the router for the rest, write
  results, report, and record the run.
- `api/openapi.yaml` + `make generate`: routed geometry and the validation
  figures on `Leg`/`Journey` (§3E, §3F) — additive fields only.
- `web/app/adventure/[id]/route-map.tsx` + detail page: road and air as the
  third and fourth visual classes, great-circle arcs, updated legend, the
  routed-vs-Google line.
- `docs/phase-3/OSRM.md`: the self-hosting runbook (commands the maintainer
  runs; the repo fetches nothing).

---

## 3. The real choices

**A. Where routed geometry enters the model.** Recommendation: after
`Assemble`, as decoration. `route.Apply(j, lookups)` takes the assembled
journey and a map of cache results keyed by rounded endpoint pair, and
returns a journey whose matched unknown gaps are now `road` legs. Assemble
itself never routes and never sees a Router — the golden fixture keeps
pinning the entire pure pipeline, and `roadbook journey -src` (file mode, no
DB) keeps working unchanged. The API handler and the DB-backed CLI path both
run Assemble-then-Apply, so CLI and page continue to agree. Rejected:
injecting a Router into Assemble (network I/O inside the pure core, and the
golden test would need a network stub to stay meaningful); routing in the
frontend (business logic in the Next layer, architecture violation).
Contract note: a gap leg still carries exactly its two timestamped endpoints.
Routed geometry rides in a *new, optional* `routed_points` field of plain
coordinates — routed vertices have no timestamps, and inventing them to
satisfy `TimedPoint.t` would fabricate measurements. No existing API field
changes meaning.

**B. What the null router does.** Recommendation: `Null` returns `ErrNoRoute`
for every pair — it fills nothing, caches nothing, and every gap stays
`unknown`, drawn as the dashed straight line phase 2 already draws.
Invariant 7's "draws straight lines and says so" is satisfied by the
*product's* degraded rendering, not by the router fabricating output.
Rejected: Null returning the straight line as a `road` result — that would
label uninspected geometry as road inference, which is precisely the lie
invariant 8 exists to prevent; the `road` kind must mean "a routing engine
produced this."

**C. The cache: key, precision, schema, invalidation — and why persistence
is right here.** Recommendation:

- *Key:* the rounded endpoint pair plus profile, stored as fixed-point
  integers: `(from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile)`,
  unique. E4 fixed-point (coordinate × 10⁴, rounded) rather than floats
  because the key must match by equality and 4-decimal values are not exactly
  representable in binary floating point; integer keys make hits exact by
  construction. Directional, not symmetrized — one-way roads exist.
- *Precision:* 4 decimals ≈ 11 m. Below the source data's accuracy (16 m
  median for true GPS fixes, far worse for WiFi/cell), so rounding discards
  no real information; fine enough that OSRM's road snapping is unaffected
  except between adjacent parallel roads, where the source data could not
  adjudicate anyway. The *rounded* coordinate is what gets routed, so the key
  IS the request and a hit is exact, never approximate. Rejected: 3 decimals
  (~111 m — enough to snap to the wrong road in dense areas); exact floats
  (no coalescing when re-assembly under different thinning picks a
  neighbouring point, plus the float-equality trap).
- *Schema:* `status` (`routed` | `no_route` — negative answers are cached so
  re-runs don't re-ask; an operational failure caches nothing), `geometry`
  jsonb as `[[lat, lon], …]` (lat-first, matching the repo's domain
  convention — the lat/lon flip stays uniquely in the frontend's GeoJSON
  conversion), `distance_m`, `duration_s` (unused by the renderer, kept
  because it enables a future plausibility check — routed travel time
  exceeding the gap's duration proves a wrong route regardless of distance —
  and the backlog's replay animation), `router` (which implementation and
  URL answered), `dataset` (the OSM snapshot identity: OSRM's
  `data_version` when the build exposes it, else the extract name the
  maintainer states at batch time via `-dataset` — without it, a divergence
  puzzle a year from now cannot say which map it was routed against), and
  `routed_at`. Provenance throughout in the spirit of invariant 3.
- *Invalidation:* entries never expire on a clock. `roadbook route -refresh`
  re-asks and replaces (the story for "I built a better extract" and for
  retrying `no_route` after OSM improves). `TRUNCATE route_cache` is always
  safe: the cache is derived data and decisions never reference it. If the
  key precision ever changes, old rows orphan harmlessly and a truncate is
  the honest reset.
- *Why persistence is right here when journeys, countries, and scores stay
  derived-on-read:* those three are pure or local functions over data already
  in Postgres — recomputation is milliseconds, deterministic, and always
  possible, so a stored copy is only a chance to disagree with its inputs.
  Routing output depends on an external process that is slow, rate-limited,
  possibly absent (null default), and non-deterministic across OSM snapshots.
  Recomputation is not free and not always possible — so here the stored copy
  is not an optimization, it is the mechanism that removes the runtime
  dependency. The cache earns persistence because the thing it caches cannot
  be re-derived from what the database holds.
- *Rejected: loading the road network itself into Postgres.* Only route
  *answers* are stored, never OSM road data. Importing the graph (pgRouting
  or hand-rolled) would store gigabytes to serve tens of queries and
  reimplement, worse, the algorithm OSRM's preprocessed files already answer
  in milliseconds — the product needs a few dozen answers, not a routing
  engine. The network stays OSRM's own files on disk (`data/osrm/`), needed
  only while `roadbook route` runs; once the cache is written, OSRM can be
  shut down or deleted and the product is unaffected.

**D. Air classification.** Recommendation: in `Assemble`, in the pure core —
implied speed is arithmetic over data the assembler already holds, and
classifying there means the batch router and the renderer can never disagree
about which gaps are flights. `AirSpeedMinKmh` joins `journey.Params`
(named, echoed in every response; invariant 3), default **250 km/h**: above
any ground transport sustained over a ≥20-minute gap (the fastest trains in
the source's part of the world average well under 100 km/h between stops),
below the implied speed of any flight long enough to matter, even diluted by
airport dwell at the gap's edges. Air gaps are never sent to the router and
never counted in road validation. Known blind spot, stated rather than
hidden: a short flight inside a long silence (boarding + flight + baggage in
one gap) can dilute below the threshold and be classified `unknown`; the
router will then try it, likely produce a road with distance far from the
chord, and the divergence check (§3E) is the tripwire that surfaces it.
Revisit the threshold against real journeys — it is a parameter precisely so
that retuning is a recorded run, not a code change. The golden fixture is a
bus journey; every gap's implied speed sits far below 250, so its
expectations are untouched — checkpoint 1 proves this by running the
unmodified golden test. Rejected: classifying by Google's activity mode
(`FLYING`) — modes are guesses that fail at extremes (a 1,023 km
"motorcycling" activity exists in the source data), and a speed rule over
our own measurements is explainable; mode stays available as a
cross-check, not the rule.

**E. Validation against `google_km`.** The point (PLAN, logged): an
independent check that catches routing producing a plausible road that is not
the road taken. Recommendation:

- *Compared figures:* ground total = observed km + routed km over road legs +
  chord km over remaining unknown gaps, versus Google ground =
  `google_km` minus activities whose mode is `FLYING`. Air must leave both
  sides or neither: our air legs are excluded by construction, and Google's
  flight activities carry their own distance which would otherwise make every
  flight adventure fail validation systematically (PLAN states exactly this).
- *Where computed:* in Go at read time, after `Apply` — divergence needs
  routed values, which exist only post-cache. Derived, never stored, like
  every journey number. New response fields: `air_km`, `unknown_km`,
  `routed_km`, `google_ground_km`, `divergence_pct` (absent when Google
  ground distance is zero), `divergence_flagged`. The flag threshold
  `DivergenceWarnPct` (default 15) is a named parameter echoed with the
  response — the frontend renders the flag, it does not compute it.
- *Where shown:* on the detail page beside the provenance line — both
  figures and the signed percentage, highlighted when flagged. A journey with
  unroutable gaps shows negative divergence (chords under-count roads); that
  reads as designed behaviour because the unknown legs are right there on the
  map saying why. Expect divergence, in both directions, on mountain
  adventures — the check is a conversation starter, never a gate.
- Rejected: validating per-leg against per-activity distances (activity
  windows and gap windows do not align; an aggregate is the honest
  granularity); storing divergence (derived data, phase 2 BRIEF §3B logic).

**F. Rendering the third and fourth classes.** Recommendation: dashes remain
the one non-negotiable inference marker; hue — held in reserve since phase 2
— is spent now to distinguish *kinds* of inference. Observed legs stay
exactly as they are (solid, saturated emerald, wide). Road legs: dashed in a
distinct hue (sky blue), width between observed and unknown — confident
inference, still visibly not measurement. Air legs: dashed violet
great-circle arc, thin. Unknown gaps: unchanged dashed grey. Layer-per-class
stands at four line layers (the phase 2 decision said expressions earn their
complexity when classes stop being enumerable; four is still enumerable, and
z-order from layer order stays free — unknown at the bottom, then air, then
road, then observed, stops on top). The legend states all four encodings in
words. Arc geometry is generated client-side in `toGeoJSON` by great-circle
interpolation between the two endpoints (~64 segments, spherical
interpolation — the Dawarich flights layer is the mechanics reference): the
arc is presentation, not data; the API keeps reporting exactly the two
timestamped endpoints it knows, and `air_km` is the great-circle distance,
which the endpoint chord already is. Rejected: solid lines for routed roads
(spends the solidity channel that means "measured"; a routed leg drawn solid
is phase 2's exact definition of a lie); server-generated arc points (would
put fabricated coordinates in an API that otherwise reports measurements).

**G. Batch scope and public-endpoint politeness.** Recommendation:
`roadbook route` defaults to the *confirmed* adventures of the latest run —
they are the product; routing every candidate multiplies load for rows the
user may dismiss. `-all` widens to every candidate, `-candidate N` narrows to
one. Router selection mirrors the geocoder exactly: `-router none` (default)
| `-router osrm`, with `-router-url` (default: the public OSRM demo server),
`-profile` (default `driving`), and `-router-interval` (minimum spacing
between requests, default 1s — the public demo is a courtesy service; set 0
against localhost). Env fallbacks `ROADBOOK_ROUTER`, `ROADBOOK_ROUTER_URL`
as with the geocoder. `serve` gains no router flag at all — the serve binary
cannot dial a routing service even by misconfiguration. Every batch
invocation is recorded as a *route run* — when it ran, which router and
dataset, the parameters, and the counts from its report (gaps found, cache
hits, newly routed, no-route, failures). This is invariant 3 applied to
routing: a batch that writes derived data records what produced it, and the
accumulating run rows turn "is OSM coverage improving on these roads?" from
anecdote into a query. Rejected: routing at confirm time in the API (a
network call in the serve path is the exact dependency this design exists to
avoid); auto-routing after detection (same, via the back door).

**H. Countries attribution stays measurement-only.** Attribution currently
runs over drawn leg points, which today are all observed measurements (gap
legs contribute their two observed endpoints). Recommendation: keep it that
way — routed geometry and arc points never feed the countries query. A
routed road that clips a border country would otherwise add a country the
user was never observed in, on the strength of an inference the divergence
check exists to doubt; and a flyover must never count as a crossing. Cost
accepted: a genuine unobserved border crossing stays unattributed — an
honest gap, consistent with the product's treatment of every other
unmeasured thing. Rejected: attributing over routed points with a "derived"
label (the label already carries the polygon-resolution caveat; stacking a
second inference on it makes the line unexplainable).

---

## 4. OSRM self-hosting — the boundary

Extract acquisition and preprocessing are the maintainer's manual steps.
Nothing in this repository fetches anything at build, install, or run time —
the phase 1 font lesson, the countries-data decision, and now routing, all
one rule. The repo's contribution is `docs/phase-3/OSRM.md`: a runbook
stating where extracts come from (Geofabrik regional `.osm.pbf`), the
preprocessing commands (`osrm-extract` → `osrm-partition` →
`osrm-customize`, MLD pipeline, car profile), how to serve
(`osrm-routed --algorithm mld`, localhost), disk/RAM expectations, and where
to put the files (`data/osrm/` — inside the gitignored directory, so nothing
can leak; the never-delete rule for `data/` covers the irreplaceable exports,
and OSRM artifacts regenerate from the pbf, but nothing in the repo touches
them either way). The null router keeps the product fully usable when none
of this has been done — that is checkpoint 2's demonstrated state, not a
promise.

---

## 5. Checkpoint order — four slices, each visible, each a STOP

1. **Air legs, end to end.** `AirSpeedMinKmh` in the pure core; gap
   classification at assembly; API fields for `air_km`; the violet
   great-circle arc and updated legend in the renderer. The unmodified golden
   regression stays green (the bus journey classifies entirely below the
   threshold — proof the pinned pipeline is undisturbed); a unit test pins a
   synthetic supersonic gap as `air`. When the first real flight adventure
   is in front of us here, its window gets anonymised into `testdata/` as
   the air-classification golden case — the moment an interesting case is on
   screen is the cheap moment to bank its fixture. *Visible: a confirmed
   adventure containing a flight shows an arc instead of a straight dashed
   line, and its air distance is broken out on the page.* Air comes first because
   routing and validation both consume the classification — built later,
   checkpoint 2 would cache roads for flights and checkpoint 3's validation
   would be wrong for every flight adventure.

2. **The Router seam, the cache, and one routed journey.**
   `internal/route` (interface, `ErrNoRoute`, `Null`, OSRM client, pure
   `Apply`); migration 00005; store read/write; `roadbook route` with its
   report; cache application in the API handler and the DB-backed CLI path.
   First run with the default null router — *visible: the command inventories
   every gap and routes nothing, and the product is byte-identical to phase
   2* (graceful degradation demonstrated, not asserted). Then
   `-router osrm` against the public endpoint for one adventure — *visible:
   that adventure's gaps turn sky-blue on the map, `curl` shows
   `gap_kind: "road"` with `routed_points`, and re-running reports 100% cache
   hits.* Public endpoint before self-hosted: it proves the whole pipeline
   with zero setup cost, and swapping to localhost afterwards is the
   pluggability demonstration.

3. **Self-hosted OSRM and the full batch.** `docs/phase-3/OSRM.md` written;
   the maintainer builds the regional extract by hand per the runbook;
   `roadbook route -router osrm -router-url http://localhost:5000
   -router-interval 0` over all confirmed adventures; `-refresh` semantics
   proven by re-routing one adventure. *Visible: every confirmed adventure
   transformed; gaps OSRM cannot fill demonstrably remain dashed grey on
   real mountain roads — the designed level-3 state on real data.*

4. **Validation against Google.** `google_ground_km` (flying-mode activities
   excluded), the divergence computation and named threshold, the
   routed-vs-Google line on the detail page with the flag. *Visible: every
   adventure page reports "routed X km · Google Y km · Z%", flagged where
   divergent, and the unroutable-gap case reads as explanation, not error.*
   Then the phase log, and the phase is complete when it exists.

---

## 6. Verification at phase close

`make test` green from a cold cache, the golden journey regression among them
and unchanged; CLI, API, and page agree on every number for the golden window
after Apply (same cache, same results); fixture detection still matches the
reference output and every decision still attached after migration 00005 and
a re-detection; the serve binary demonstrably makes no outbound routing
call (no router is even wired into it); the map's four classes confirmed
visually by the maintainer; `git add -A --dry-run` clean of `data/` and of
anything near 1 MB, throughout — with the OSRM artifacts living under
`data/osrm/` where the standing rule already guards them.

## 7. Outstanding questions

**Short-flight dilution.** A flight short enough that airport dwell drags the
gap's implied speed under 250 km/h classifies `unknown` and may route as a
road (§3D). The divergence flag is the detector for now. If a real adventure
hits this, the candidate fixes are a mode cross-check (`FLYING` overlapping
the gap window as a tie-breaker) or a per-gap distance floor — decide then,
against the journey that makes it concrete, and as a named parameter either
way.

**Countries-query cost.** Phase 2 parked "persistence joins phase 3's
routing-cache discussion if the per-read query drags." Nothing has been
measured to drag, routed geometry does not feed the query (§3H), so nothing
changes: countries stay derived-on-read. Parked again, with the same trigger.
