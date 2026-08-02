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
