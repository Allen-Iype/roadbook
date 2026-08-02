# journey-27jul2026 — pipeline contract

Semantics behind `journey-27jul2026.expected.json` that a fresh implementation
will get wrong and the numbers alone will not catch. Every statement here is
grounded in the preserved July artifact (`data/bus_27jul_points.json`, the exact
121-point merged set the measurement used) or in the fixture itself — not in any
later reimplementation. The script that produced the measurement was not found
(searched: this repository, its git history and stashes, the pre-restart
`roadbook-archive` repository, and local scratch directories); its design spec
survives as `roadbook-archive/docs/SPEC.md` and is cited where used.

## The pipeline

1. **Window.** An input, not derived: `[2026-07-27T19:46:35, 2026-07-28T07:12:22]
   +05:30`, both ends inclusive. These are exact activity boundaries in the
   fixture — the start of the first `IN_BUS` activity and the end of the second.
   In the product the window will come from a candidate's span; the fixture
   records the window it was measured with.
2. **Point selection.** Every `timelinePath` point (69) and every `rawSignals`
   position (121) whose timestamp lies inside the window.
3. **Thinning.** One time-sorted stream of both sources; walk in order, keeping a
   point iff it is at least `thinSpacingSeconds` after the last *kept* point
   (first point always kept). **Keep-earliest, no source preference** — the
   40 trace / 81 raw composition falls out of timestamps alone. Result: 121
   points matching `data/bus_27jul_points.json` timestamp-for-timestamp; the
   spacing is unique on **(29, 30] seconds** (29 keeps 122 points, 31 keeps 120).
4. **Equal-timestamp tie-break.** No trace point and raw position share a
   timestamp in this fixture, so the data cannot adjudicate an order. Pinned by
   convention so implementations are deterministic: **at equal timestamps, trace
   before raw.** Marked UNEXERCISED — the fixture will not catch a violation.
5. **Leg split.** Walk the kept points; a time delta > `gapThresholdMinutes`
   emits a gap leg holding exactly its two endpoints; otherwise the point
   extends the current observed leg (`roadbook-archive docs/SPEC.md`, stage 3).
   Gives 7 observed and 6 gap legs. One observed run has only 3 points and still
   counts as a leg; whether tiny runs should stand is an open design question
   (phase 2 brief, outstanding questions) — the measured pipeline kept them.
6. **Distance.** Spherical haversine, **R = 6371.0 km exactly**, summed over
   consecutive kept points. Evidence this is what was measured: with
   R = 6378.137 the chord sum is ≈673.4 km, incompatible with the recorded
   672.6; only R ≈ 6371 reproduces all four one-decimal 2026-07-29 values.
7. **Gap metrics.** `inferredKm` = sum of endpoint chords over gap legs;
   `inferredPctOfDistance` = inferred / total; `worstGapSeconds` = gap durations
   sorted worst-first, in raw seconds (rounding is display's job).
8. **Rest halt.** The dwell between the first bus activity's end (21:29:58) and
   the second's start (21:57:39). `displacementKm` is the straight-line distance
   from the first to the last merged point inside the halt — **not** the path
   sum. The data adjudicates: displacement 0.1635 rounds to the recorded 0.2;
   path sum 0.7337 does not.

## Why worstMinutes was wrong

The former `worstMinutes` [36, 36, 36, 29, 28, 28] is the top six of the
**seven** trace-only gaps (their rounded durations: 36, 36, 36, 29, 28, 28, 27 —
the seventh being the 28.0-minute rest-halt silence at 21:30, which rawSignals
covers). The merged set this file pins has **six** gaps, durations
2160/2145/2144/1717/1664/1604 s, rounding to [36, 36, 36, 29, 28, **27**]. The
mix survived review because both lists happened to be six entries long. The
former `rawSignalsVerdict` counts (120 positions, WIFI 62) were likewise taken
under the minute-truncated window; under the true window they are 121 / WIFI 63.

## Amendment record (2026-08-02)

| field | was | now | ground |
|---|---|---|---|
| window.start/end | 19:46 / 07:12 | 19:46:35 / 07:12:22 | fixture activity boundaries; first artifact point is 19:46:35 |
| durationHours 11.4 | — | durationSeconds 41147 | timestamp arithmetic |
| worstMinutes [36,36,36,29,28,28] | — | worstGapSeconds [2160,2145,2144,1717,1664,1604] | artifact gap edges |
| params joinSlackSeconds 120 | — | removed | SPEC.md stage 2 semantics; zero points in either 120 s margin |
| params minGeometryPoints 4 | — | removed | SPEC.md journey tiering; 3-point run counts as a leg |
| params thinSpacingSeconds | — | 30, RECOVERED BY FIT, unique on (29,30] | artifact timestamps |
| distances, 1 dp | — | 4 dp, haversine R=6371.0 pinned | artifact coordinates; R adjudicated by the 1-dp values |
| movementKm 0.2 | — | displacementKm 0.1635 | artifact points inside the halt |
| restHalt 21:29 / 21:57 | — | 21:29:58 / 21:57:39 | fixture activity boundaries |
| rawSignalsVerdict 120 / WIFI 62 / gapsFilled | — | 121 / WIFI 63 / movingGapsFilled | fixture rawSignals under the true window |
| knownSourceDefect longitudes 78.0243547, 78.1411 | — | 46.5243547, 46.6411 | the fixture's anonymised frame (both values present in the anon fixture) |
