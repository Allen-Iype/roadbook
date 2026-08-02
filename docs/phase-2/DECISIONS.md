# Phase 2 decisions

Three lines each: what was chosen, what was rejected, what would change our
mind. Written as decisions are made.

## 2026-08-02 — golden fixture amended to full precision

Chosen: `journey-27jul2026.expected.json` stores exact timestamps, raw gap
seconds, and 4-dp distances with the haversine radius pinned; `worstMinutes`
replaced by `worstGapSeconds` measured on the preserved merged set
(`data/bus_27jul_points.json`); params reduced to the two the pipeline
exercises, with `thinSpacingSeconds: 30` marked recovered-by-fit (unique on
(29, 30]) and semantics pinned in `journey-27jul2026.CONTRACT.md`.
Rejected: keeping display-rounded values (minute-truncated window bounds could
not reconstruct the point selection, and rounding is what let a trace-only
measurement sit undetected in a merged-set file); implementing Go to reproduce
a list its own pinned input cannot produce.
Would change our mind: the July measurement script surfacing with different
semantics — the file then follows the script, not the fit, and the CONTRACT
gets corrected rather than defended.

## 2026-08-02 — one Earth radius for the whole repository

Chosen: journey distances use `geo.HaversineM` (R = 6371008.8 m), and the
fixture's 4-dp values were recomputed at that radius before any Go existed —
the radius is hard-coded in `prototype/detect_fixture.py`, the surviving
reference implementation from the measurement era.
Rejected: pinning R = 6371.0 km (my first amendment pass): the July 1-dp
values cannot distinguish mean-radius variants, and a second radius constant
in one repository invites exactly the dual-implementation drift Dawarich's
history warns about.
Would change our mind: nothing foreseeable — both candidates fit the data,
so the tie goes to the constant the codebase already owns.

## 2026-08-02 — stops are inter-activity dwells in checkpoint 1

Chosen: `journey.Assemble` reports a stop where consecutive Google activities
leave a pause ≥ `MinStopDwellSeconds` — the semantics the golden fixture pins
(CONTRACT.md §8), with movement as first→last displacement.
Rejected for now: the Dawarich stay-point sweep (BRIEF §1.3) as the primary
stop detector — the fixture cannot pin it (its rest halt is an activity
boundary, not a cluster), and shipping two stop definitions at once would
make the golden numbers ambiguous again.
Would change our mind: a journey whose real halts fall inside one long
activity (no boundary to dwell between) — the sweep then becomes necessary,
lands as a separate named-parameter pass, and the fixture gains its expected
output.

## 2026-08-02 — the accuracy filter defaults to off

Chosen: `MaxAccuracyM` exists as a named parameter but defaults to 0
(disabled); exact (0,0) rejection is unconditional. Both flag at assembly
time and never touch stored rows.
Rejected: Dawarich's 100 m default — measured against the golden journey, 40
of its 121 in-window fixes are worse than 100 m and they are the highway
densification (cell fixes en route), not noise; a 100 m default silently
deletes the very data rawSignals ingestion exists to add.
Would change our mind: a journey where low-accuracy fixes visibly disfigure
the drawn route — then the default becomes a measured threshold, recorded
with the run that measured it.

## 2026-08-02 — layer-per-class stands, but for z-order, not dasharray

Chosen: separate gap/observed/stop layers filtered on feature properties.
The brief's supporting claim that `line-dasharray` cannot be data-driven is
wrong for the installed MapLibre (6.1.0 types it `CrossFadedDataDrivenProperty`);
the decision survives on its other ground — paint order between classes
comes from layer order, free and explicit.
Rejected: one layer with data-driven paint expressions — saves nothing at
three classes and buries z-order in expression evaluation.
Would change our mind: enough leg classes (phase 3 adds road and air) that
per-class layers stop being enumerable — then expressions earn their
complexity.

## 2026-08-02 — MapLibre 6.1.0, pinned exact; OpenFreeMap liberty as default basemap

Chosen: `maplibre-gl` 6.1.0 with no version range (the Dawarich lesson:
unpinned map libraries drift unauditable). v6 facts read from the bundled
typings before any map code: ESM-only, named exports only (`MapLibreMap`
alias, no default export), `setData` now returns a Promise. Basemap style
URL comes from `ROADBOOK_MAP_STYLE` server-side with OpenFreeMap's liberty
style as default — no API key, no account, self-hostable later (the phase 5
provider seam, done cheap now).
Rejected: raw OSM tiles as default (usage policy discourages app traffic);
keyed free tiers (a key in a self-hosted product's default path).
Would change our mind on the default: OpenFreeMap availability proving
unreliable in practice — the env var is the escape hatch either way.

## 2026-08-03 — MapLibre's worker is served from public/, pointed at by setWorkerUrl

Chosen: `setWorkerUrl("/maplibre/maplibre-gl-worker.mjs")` at module scope,
with the worker and its single import (maplibre-gl-shared.mjs) copied from
the pinned package into `public/maplibre/` by a `copy:maplibre` npm script
hooked into predev/prebuild; the copies are gitignored, so they always match
the installed version.
Rejected: relying on MapLibre's own worker-URL derivation — v6 derives it
from `import.meta.url`, which under Turbopack points into the chunk
directory where no worker file exists; the failure is fully silent (style
never finishes loading, `load` never fires, the map stays blank — no console
error). Also rejected: committing the two dist files (vendored generated
code that drifts on upgrade).
Would change our mind: Turbopack gaining first-class support for
dependency-internal workers, or MapLibre shipping a bundler-safe worker
entry — then the copy script goes.
