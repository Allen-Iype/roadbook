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

## 2026-08-03 — country polygons committed verbatim, embedded, replaced wholesale

Chosen: the upstream Natural Earth 1:110m admin-0 GeoJSON gzipped unchanged
(209 KB — provenance is "this file, gzipped"), compiled in via `go:embed` so
`roadbook countries` needs no arguments and no network ever; the loader
replaces the table's contents in one transaction, and `-src` accepts a
higher-resolution file from disk.
Rejected: slimming properties before committing (saves ~100 KB but the
committed file no longer byte-matches upstream); upserting by code (stale
rows survive a switch to a source file with a different country set);
fetching at build or install time (the phase 1 font lesson, and BRIEF §3E).
Would change our mind: a wanted source file that tops 1 MB even gzipped —
slimming then returns as the price of admission.

## 2026-08-03 — ISO codes take the ISO_A2_EH → ISO_A2 → ADM0_A3 fallback

Chosen: prefer Natural Earth's corrected `ISO_A2_EH` column, then `ISO_A2`,
then the three-letter `ADM0_A3`; "-99" (Natural Earth's null) never passes.
Measured on the bundled file: France, Norway, Kosovo recover via ISO_A2_EH;
Northern Cyprus and Somaliland land on CYN/SOL; 177 features, zero collisions
— pinned by internal/countries's offline test.
Rejected: `ISO_A2` alone (France and Norway read "-99"); dropping features
without an alpha-2 (a journey through them would silently lose a country).
Would change our mind: upstream assigning real alpha-2 codes to the ADM0_A3
holdouts — the fallback then simply stops firing.

## 2026-08-03 — countries computed at read time, ordered by first appearance

Chosen: attribution runs per journey read — one SQL query, every drawn leg
point unnested against the GiST-indexed polygons, grouped per country,
ordered by the first point that hit it, so the line reads in journey order.
Nothing is persisted, matching legs (BRIEF §3B: derived data stays derived).
Rejected: storing countries on the candidate row (a cache that can disagree
with a reloaded polygon set); alphabetical order (journey order is
information: "India · Nepal" says where it started).
Would change our mind: the per-read query measurably dragging on real
journeys — persistence then joins phase 3's routing-cache discussion.

## 2026-08-03 — score is a weighted mean over present components

Chosen: score = 100 × Σ(weight·normalized)/Σ(weight) over the components
that could be measured — redistribution of a missing component's weight
(BRIEF §1.6) falls out of the denominator instead of being a special case.
Four components, each with a named linear saturation anchor: distance from
home (full at 1000 km), dwell within 60 km of the destination (24 h),
observation density (48 obs/day), span duration (7 days); weights
0.35/0.30/0.20/0.15 because distance and dwell are the two halves of the
detection rule itself. All recorded per run under params.SCORE.
Rejected: scoring absent components zero (absence would read as low
confidence — the lie invariant 8 forbids on the map, in numeric form);
logarithmic normalization (harder to reproduce by hand, and the breakdown's
whole point is hand-checkable arithmetic).
Would change our mind on the anchors: measured score bunching on the full
archive — the anchors are parameters precisely so retuning is a recorded
run, not a code change.

## 2026-08-03 — score columns are nullable; absence is not zero

Chosen: `score`/`score_breakdown` on candidates are NULL for rows stored
before scoring existed; the API omits them and the UI shows a dash ("not
scored"), never 0.
Rejected: backfilling old runs (their params carry no SCORE block — a
backfilled score would claim parameters that run never had, violating
invariant 3); NOT NULL DEFAULT 0 (reads as "no confidence", which is false).
Would change our mind: nothing — old runs are comparison history, and
re-running detection is the supported way to score current candidates.

## 2026-08-03 — Suggester defaults to null; Nominatim is opt-in

Chosen: `roadbook serve` wires the null suggester unless
`-geocoder nominatim` (or ROADBOOK_GEOCODER) says otherwise, with the
instance URL configurable for self-hosted Nominatim; the suggestion
prefills the confirm input only while it is still empty — typed text always
wins, and nothing is ever applied to a decision automatically.
Rejected: geocoder-by-default (a self-hosted product must not make
surprise network calls); suggesting at list-render time (18 lookups per
page view against a policy that allows interactive use — one lookup per
confirm click is the honest shape).
Would change our mind on the default: a bundled offline gazetteer landing
in some later phase — then suggestions can be on by default without a
network dependency.
