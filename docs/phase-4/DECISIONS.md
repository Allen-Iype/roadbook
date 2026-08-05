# Phase 4 decisions

Three lines each: what was chosen, what was rejected, what would change our
mind. Written as decisions are made. The brief's §3 recommendations were
approved at Gate 1 (2026-08-05) and are not restated here; entries below
record decisions made during implementation.

## 2026-08-05 — display offset derived from wall-minus-GPS when both clocks are present

Chosen: when a photo carries both `DateTimeOriginal` (civil wall clock) and a
GPS UTC instant, the civil offset stored for display is `wall − gps` rounded
to the nearest 15 minutes (every real UTC offset is a 15-minute multiple),
accepted only within ±14 h — both values are measurements in the same file,
so the derivation invents nothing.
Rejected: leaving GPS-sourced instants with offset 0 (displays a
five-and-a-half-hour-wrong wall time beside every photo in this data), and
trusting `OffsetTimeOriginal` over the derivation when both exist (the two
should agree; the derivation is used first because it is computed from two
sensor readings rather than a writable tag, and a disagreement beyond the
rounding window falls back down the ladder).
Would change our mind: a camera observed writing a wall clock set to a
different zone than its GPS position (a traveller who never changes the
camera clock) — the derived offset is then that camera's own convention,
which is still what the photographer's other timestamps mean.

## 2026-08-05 — sidecar position reads geoData, then geoDataExif; (0,0) is absent everywhere

Chosen: the sidecar parser takes `geoData` first, falls back to
`geoDataExif` when `geoData` is absent or exactly (0, 0), and both count as
`pos_source: sidecar`; an exact (0, 0) also voids an EXIF-decoded position —
the same "Null Island means unset" rule the anomaly filters already apply to
raw positions.
Rejected: reading only `geoData` (Takeout emits (0,0) there while
`geoDataExif` still holds the camera's real reading — observed in real
exports), and distinguishing the two sidecar fields in provenance (both are
Google's copy; the EXIF-vs-copy distinction the brief §3D draws is already
carried by `exif` vs `sidecar`).
Would change our mind: a real sidecar whose `geoDataExif` disagrees with the
photo's own EXIF block — would demote `geoDataExif` entirely, since the
embedded EXIF is then authoritative and the copy is provably drifted.
