# Phase 2 log — one adventure on a map

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. Counts derived from the private
dataset are not stated here; the golden journey's numbers are public because
the fixture that reproduces them is committed
(`testdata/journey-27jul2026.anon.json` / `.expected.json`), and "the
reference output" for detection still means what
`prototype/detect_fixture.py` prints for the fixture in `data/`.

## What the phase does

Open a confirmed adventure and see the route, with gaps visibly marked as
unknown. Straight lines only; routing is phase 3.

- `internal/journey` assembles a candidate's span into an ordered list of
  legs — each observed or a gap, never one undifferentiated line — as a pure
  function over immutable observations. Gap legs carry a `GapKind`
  (`unknown` in this phase; `road` and `air` exist for phase 3). Stops are
  inter-activity dwells; distances are chords on one Earth radius
  (`geo.HaversineM`, R = 6371008.8 m). The committed golden fixture pins the
  whole pipeline — 121 merged points, 672.6 km, 7 observed and 6 gap legs,
  33.0% inferred — and runs in every `make test` with no `data/` gate.
  `roadbook journey` is the reproduction command for every figure.
- The parser now emits `rawSignals` position fixes as a domain type; a new
  `raw_positions` table stores them content-hash idempotently, like every
  observation. Anomaly parameters live at assembly time and flag rather than
  delete: exact (0,0) always; an accuracy ceiling exists but defaults off.
- `GET /candidates/{id}/journey` assembles on demand — legs are derived data
  and are never persisted — and echoes the parameters that produced the
  assembly. The detail page renders the journey's numbers, leg table, stops,
  and provenance line; the map island (`route-map.tsx`) draws observed legs
  solid and gaps dashed grey via separate MapLibre layers, with a legend
  stating the encoding in words. The basemap style URL is environment
  configuration with a keyless default.
- `roadbook countries` loads Natural Earth admin-0 polygons (1:110m bundled
  in the binary, higher resolutions accepted from disk, never fetched) into
  a PostGIS `countries` table used for exactly one thing: an indexed
  point-in-polygon query attributing a journey's drawn points to countries,
  ordered by first appearance, rendered on the detail page as a line
  labelled derived.
- Detection candidates now carry a confidence score, 0–100, computed in the
  pure detect core from four named weighted components (distance from home,
  destination dwell, observation density, span duration) with per-run
  recorded weights and anchors. A component that cannot be measured has its
  weight redistributed — absence never reads as low confidence. The stored
  per-component breakdown is shown when confirming; nothing filters or
  auto-confirms on it.
- The name-suggestion seam (`internal/suggest`): a `Suggester` interface
  with a null implementation that suggests nothing and says so (the
  default), and a Nominatim reverse-geocoder enabled explicitly at serve
  time. A suggestion prefills the confirm input only while it is empty;
  typed text always wins.

Verification at close: all suites green from a cold cache, the golden
journey regression among them; CLI, API, and detail page agree on every
journey number for the same window; detection from the database reproduces
the reference output after all four migrations; every stored decision
re-attached across re-detection, none orphaned; the map confirmed visually
by the maintainer; `git add -A --dry-run` clean of `data/` and of anything
near 1 MB throughout.

## What broke, and why each fix took its form

**1. The golden fixture could not reproduce its own numbers.** The first
`expected.json` carried display-rounded values — minute-truncated window
bounds, 1-dp distances — and a `worstMinutes` list that turned out to be
measured on the trace-only point set while sitting in a file describing the
merged set. Rounded bounds could not even reconstruct which points were
selected. The fix re-measured everything at full precision on the preserved
merged set, replaced `worstMinutes` with `worstGapSeconds`, recovered
`thinSpacingSeconds` by fit (unique on (29, 30]), and pinned the pipeline
semantics in `journey-27jul2026.CONTRACT.md`. Form: amend the fixture to
what the measurement actually was, rather than implement Go to reproduce a
list its own pinned input cannot produce — a golden test that cannot be
re-derived from its input is pinning a transcription, not a pipeline.

**2. Two Earth radii nearly entered one repository.** The first amendment
pass pinned R = 6371.0 km; the prototype — the surviving reference
implementation from the measurement era — uses 6371008.8 m, and the July
1-dp values cannot distinguish the two. Form: recompute the fixture's
expected distances at the radius the codebase already owns
(`geo.HaversineM`), because a second distance constant is exactly the
dual-implementation drift the Dawarich comparison warns about, and the tie
goes to the constant with an existing owner.

**3. MapLibre v6 renders a blank map under Turbopack, silently.** v6
derives its web-worker URL from `import.meta.url`, which Turbopack points
into a chunk directory where no worker exists; the style never finishes
loading, `load` never fires, and nothing errors on the console or the
network panel. The fix: `setWorkerUrl` at module scope pointing at copies of
the worker and its one import in `public/maplibre/`, made by a
`copy:maplibre` npm script hooked into predev/prebuild, with the copies
gitignored so they always match the installed, exactly-pinned version. Form:
copy-not-commit (vendored generated code drifts on upgrade), and the
gitignore pattern was written as `web/public/maplibre/` — mid-slash patterns
anchor, the same pattern class as phase 1's near-leak. Two debugging
findings are recorded with the decision: an occluded window suspends
requestAnimationFrame so MapLibre legitimately paints nothing, and
OpenFreeMap tile 429s surface as "Failed to fetch (0)" — neither is an
application bug.

**4. The borrowed accuracy default would have deleted the best data.**
Dawarich filters positions worse than 100 m; measured against the golden
journey, that discards 40 of its 121 merged points — the cell-fix
densification along the highway, which is precisely what rawSignals
ingestion exists to add. Form: `MaxAccuracyM` exists as a named parameter
but defaults to off, and turning it on is a measured decision recorded with
a run, not an inherited constant. The general lesson repeats phase 1's:
defaults imported from another product encode that product's data, not
ours.

**5. PostGIS was not installed for postgresql@18.** The countries
checkpoint's first act found `pg_available_extensions` empty of postgis;
per the working agreement the work stopped there and reported rather than
working around. `brew install postgis` (a large dependency tree: gdal,
geos, proj, sfcgal) made the extension visible and migration 00003 applied
cleanly. Form: stop-on-failed-precondition, again — the migration was
written only after the extension provably existed, so it never grew a
fallback path that would mask a broken install for a future self-hoster.

**6. Natural Earth's ISO codes lie for five countries.** `ISO_A2` is "-99"
for France, Norway, Kosovo, Northern Cyprus, and Somaliland in the admin-0
file. The loader prefers the corrected `ISO_A2_EH`, then falls back to
Natural Earth's own `ADM0_A3` for the two territories with no alpha-2 at
all, and errors on any collision. Form: an offline unit test pins the
bundled file's 177 features and every quirk case by name, because the quirk
is in committed reference data and must not depend on anyone remembering
it.

## What changed about the plan mid-phase

- The brief proposed the Dawarich stay-point sweep as the stop detector;
  checkpoint 1 shipped inter-activity dwells instead, because that is the
  semantics the golden fixture actually pins — its rest halt is an activity
  boundary, not a cluster. The sweep remains available as a future
  named-parameter pass if a journey's halts ever fall inside one long
  activity.
- The brief's claim that `line-dasharray` cannot be data-driven is wrong
  for MapLibre 6.1.0; the layer-per-class decision survives on its other
  ground (z-order between classes from layer order).
- The brief's §5 question — whether tiny observed runs (the golden set has
  a 3-point leg) should stand as observed legs — never bit. They stand;
  the question stays parked until a real journey makes it concrete.

## Carried forward

- Phase 3 consumes `GapKind`: classify gaps as `road` (routed) or `air`
  (implied speed above a named threshold, drawn as great-circle arcs);
  observed legs are never routed.
- Persistence earns its place in phase 3 as a routing cache; journeys,
  countries, and scores all stay derived-on-read until then.
- The score anchors are parameters so a retune after living with real
  scores is a recorded run, not a code change.
- The suggester defaults to null because it needs the network; a bundled
  offline gazetteer would let suggestions default on. Not scheduled.
- Tiny observed legs (3 points) render as observed; revisit only if a real
  journey makes them misleading.
