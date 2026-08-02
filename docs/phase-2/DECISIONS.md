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
