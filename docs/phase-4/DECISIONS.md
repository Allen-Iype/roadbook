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

## 2026-08-05 — store gains DB-backed tests: real SQL, running by default

Chosen: integration tests against a real scratch Postgres (created, migrated
with the embedded migrations, and dropped per test by
`internal/store/storetest`), run by default — `make test` resolves a
database itself (env var → local Postgres → Docker `compose.test.yaml` →
visible skip), so skipping is the exception, not the norm; scope is the
load-bearing behaviours (decision re-attachment across re-detection, import
idempotency, the photo round-trip and its conflict path) plus temp-dir unit
tests for `PhotoFiles`, while the pure core keeps the test weight.
Rejected: mock-based unit tests (the store's risk is the SQL against real
Postgres; mocking pgx tests the mock), an opt-in env-var gate (a suite
behind an unset variable never runs), and a coverage crusade over every
query (thin CRUD that live checkpoint proofs already exercise).
Would change our mind: nothing foreseeable removes the harness; hosting for
other people would instead *raise* its importance — every store query would
gain a user filter, a missed one is a cross-user data leak, and this
harness is what makes that change safe to attempt.

## 2026-08-05 — photo delete is row first, file second

Chosen: `DELETE /photos/{id}` removes the database row, then the thumbnail
file, and a file-step failure surfaces as an error after the row is already
gone — asserted by a failure-injection test, not just the happy path.
Rejected: file first (a crash between steps leaves a row pointing at a
missing file — a permanently broken image on every page render), and
wrapping both in a pretend-transaction (the filesystem cannot join a
database transaction; pretending it can hides the failure mode instead of
choosing it).
Would change our mind: nothing — the asymmetry is inherent: an orphaned
file is unreachable garbage, detectable and sweepable by comparing the
photos directory against content hashes; a broken page is user-visible
damage with no sweep.
