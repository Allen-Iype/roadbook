# Phase 3 log — routing fills the gaps

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. Counts derived from the private
dataset are not stated here; the two committed golden fixtures
(`testdata/journey-27jul2026.*`, `testdata/journey-30apr2026.*`) carry the
public numbers, and every journey figure reproduces via
`roadbook journey`.

## What the phase does

Gaps become road-following inference, clearly labelled as inference. The
phase 2 screen is transformed: unknown gaps that the road network can
explain draw as dashed sky-blue roads, flights draw as dashed violet
great-circle arcs, and what remains unexplained stays dashed grey. The
routed distance is reported against Google's own figure.

- `internal/route` is the routing seam (invariant 7): a one-method `Router`
  interface; `ErrNoRoute` as a cacheable data answer distinct from
  operational errors; `Null` (the default — fills nothing, so the product
  degrades to phase 2's honest rendering); one OSRM HTTP client serving
  both the self-hosted and public configurations. `Apply` is a pure
  function decorating an assembled journey from cache lookups — the
  golden fixtures keep pinning the whole pure pipeline because `Assemble`
  never routes.
- Air classification lives in the pure core: a gap whose implied speed
  (endpoint chord over duration) meets `AirSpeedMinKmh` is `air` — never
  routed, excluded from road validation, drawn as a client-generated
  great-circle arc while the API reports only the two timestamped
  endpoints it measured. Speed, not Google's mode: the 30 Apr golden
  fixture pins a journey with two flights of which Google labels only one
  FLYING.
- Routing is batch-only. `roadbook route` (confirmed adventures by
  default) asks the router what the cache cannot answer and persists every
  data answer in `route_cache` (migration 00005), keyed by the e4
  fixed-point rounded endpoint pair plus profile — the rounded coordinate
  is itself what gets routed, so a hit is exact. Negative answers are
  remembered; `-refresh` replaces; every batch invocation is recorded as a
  route run with its counts, router, and dataset identity. The serve
  binary has no router wired into it at all — it reads the cache, and a
  gap without an answer stays visibly unknown.
- Road legs carry routed geometry in a separate `routed_points` field; the
  leg's `points` still holds exactly its two timestamped endpoints, because
  routed vertices are inference, not measurement. Countries attribution
  remains measurement-only: routed geometry and arcs never add a country.
- Validation: the ground reconstruction (observed + routed + unknown
  chords) is compared against Google's ground figure (activity distances
  minus FLYING ones — air leaves both sides or neither); divergence beyond
  the named `DivergenceWarnPct` is flagged, server-side, as a conversation
  starter and never a gate. No Google ground figure means
  nothing-to-compare, which is not zero divergence.
- Teleport rejection at assembly (found mid-phase, below): clusters of
  mislocated fixes are excluded from assembly by a named-parameter rule,
  flagged and counted, rows untouched.
- `docs/phase-3/OSRM.md` is the self-hosting runbook, amended from a real
  build; extract acquisition and preprocessing are the maintainer's manual
  steps, and nothing in the repository fetches at build, install, or run
  time.

Verification at close: all suites green from a cold cache with both golden
regressions running; CLI, API, and detail page agree on every figure for
the golden window with the cache applied; file-mode detection still matches
the reference output and a post-migration re-detection re-attached every
stored decision with none orphaned; the batch re-run is all cache hits with
zero network; the serve binary demonstrably cannot dial a routing service;
the map confirmed visually by the maintainer at each checkpoint;
`git add -A --dry-run` clean of `data/` and of anything near 1 MB
throughout, with OSRM's multi-gigabyte artifacts living under `data/osrm/`
where the standing rule already guards them.

## What broke, and why each fix took its form

**1. The map drew a 300 km fiction in observed green — and the maintainer's
eye caught it.** Reviewing checkpoint 3, the maintainer questioned a
straight line they had no memory of travelling. Investigation first
*validated* a different suspect — an air leg whose endpoints turned out to
be two international airports 102 minutes apart, a real flight Google never
labelled — and then found the actual defect: a wrong entry in Google's
geolocation database had repeatedly placed fixes ~300 km away, identically,
across three days. The detector rejects implausible points, but that
filtering lives in detection only; journey assembly re-read the raw window
with no speed test, drew the teleport chords as confident observed track,
and OSRM dutifully routed the fake gaps on their flanks. The fix is
cluster-level, bridge-conditioned rejection in the pure core (see
DECISIONS): point-wise rules — the detector's both-neighbours test,
Dawarich's speed sandwich — pass runs of self-consistent bogus points
wholesale, and an overnight silence launders a run's entry speed, so the
rule judges whole clusters and rejects only where removal restores
plausibility. Form: flagging never deleting, two named parameters
defaulting to the detector's own constant, and conservatism when the data
is contradictory — the map then shows the contradiction rather than an
invented resolution.

**2. The fix's first version called quantization a teleport.** The 30 Apr
golden fixture — banked three checkpoints earlier — failed immediately:
trace timestamps are minute-truncated, so two points in the same minute
130 m apart divide to infinite speed. The edge duration now floors at
`ThinSpacingSeconds`, the stream's own stated temporal resolution, and the
false positive is pinned by its own unit test. Form: derive the floor from
a parameter the pipeline already owns rather than invent a knob — and the
episode is the argument for banking fixtures the moment interesting cases
appear, since the fixture paid for itself within the same phase.

**3. OSRM served nothing twice before it served everything.** The dataset
path resolves relative to the current directory, so a first launch reported
sixteen "Missing/Broken File" warnings for files that existed (and modern
OSRM never creates a bare `.osrm` file — the name is only the family
prefix); the second launch died on "Address already in use" because macOS's
AirPlay Receiver holds port 5000. Both fixes are one line; both are
recorded in the runbook, which is the form that matters — a runbook amended
from a real build instead of intentions.

**4. `no_route` never happened.** The brief expected patchy OSM coverage to
leave gaps visibly unfilled on rural roads; on this dataset, with a current
regional extract, the road network connected every pair. The designed
degraded state is still demonstrated — by the null-router inventory run,
by unit tests, and by the rendering that sits ready — but honesty requires
recording that real data has not yet exercised it.

## What changed about the plan mid-phase

- The brief gained, at Gate 1, the data-capture additions from review
  discussion: OSM snapshot identity (`dataset`) in cache provenance, the
  persisted route-run record, and the commitment to bank an air fixture at
  first contact — all three were used before the phase ended.
- A hosted deployment's routing setup (compose profile plus operator-run
  script) was raised mid-checkpoint-3, and deliberately captured for
  phase 5 instead of built (DECISIONS): the serve binary needs no OSRM, so
  the want is a phase 5 deployment concern with `OSRM.md` as its
  specification.
- `roadbook journey` gained a `-candidate` database mode so the
  CLI-equals-API property stayed provable once journeys became
  cache-dependent; file mode stays pure and cache-free.
- Two pairs created by the teleport fix were routed via the public
  endpoint rather than restarting the local OSRM; the provenance columns
  record the mix, and a later `-refresh` against the self-hosted instance
  normalises it.

## Carried forward

- Short-flight dilution (BRIEF §7) stays parked; the divergence flag is
  the tripwire until a real journey makes it concrete.
- Tiny observed legs now have their extreme case on record — a
  single-point observed leg at an airport gate. Still parked; it renders
  honestly.
- A routed "road" of ~100 m across an overnight halt is technically
  correct and cosmetically odd; if it ever misleads, a minimum-chord floor
  for routing is a named parameter waiting for evidence.
- Divergence flags fire on a substantial minority of adventures, each
  explicable from the map. If living with them proves the threshold noisy,
  `DivergenceWarnPct` retunes via a recorded run, not a code change.
- Phase 5 owns the optional OSRM compose profile and setup script
  (PLAN, phase 5 features; `docs/phase-3/OSRM.md` is the spec).
- Countries stay derived-on-read; nothing measured drags. Same trigger as
  phase 2 left it.
