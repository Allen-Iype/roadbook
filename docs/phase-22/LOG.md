# Phase 22 — log: a stray position near home cuts a trip

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

One stray position near home no longer ends time away (#28).

An export, an iPhone's above all, holds positions that are not where the
phone was: for a moment the record jumps to a place the phone knows,
often home, and jumps back. Phase 17 sets them aside when a trip is
drawn. Detection did not know about them, so one in the middle of time
away ended the span. The piece on the other side was usually too short
to be a trip, and the trip started late or ended early.

- **The rule** (`findSpans`, `internal/detect/detect.go`). A run of
  observations near home stays inside the span when it is at most
  `MAX_STRAY_OBS` (1) observations, none of them a visit, and an away
  observation follows less than `MAX_STRAY_MIN` (30) minutes after the
  latest end among the span's observations so far.
- **The reference detector** has the same rule; Go matches it field for
  field.
- **`roadbook detect`** takes `-max-stray-obs` and `-max-stray-min`; both
  are recorded with the run.
- No migration, no contract change, no web change.

By checkpoint: STOP 0 the brief, with a correction (§7) before any code;
STOP 1 the detectors (abea3a4).

## What was measured

`measure/measure.py`, and old against new reference output:

| | demo | Android fixture | archive | iPhone export |
|---|---|---|---|---|
| candidates | 3 → 3 | 18 → 18 | 67 → 67 | 79 → 77 |
| spans changed | 0 | 0 | 1 | 11, two pairs made one |
| median time added | none | none | 0.4 h | 1.7 h |
| destination distance changed | 0 | 0 | 0 | 0 |
| Google's groupings found | none in the file | none in the file | 39 of 45, unchanged | 28 of 28, unchanged |

On the candidates whose span did not change, only `repeat` moves.

## What was verified

- **`make test` with the database suites**, against freshly generated
  reference output through `ROADBOOK_EXPECTED_DIR`. The fixture, the
  demo, the photo corpus and both journey goldens are unchanged.
- **The rule's edges, as table tests** (`TestStray`, seven cases, written
  before the code): joined at 20 minutes; not when the bound is 10; not
  at 40 minutes; two points not joined, and joined when the count is 2;
  a short visit near home still ends the trip; two halves made one.
- **Phase 20's cases pass unchanged**: a stray *pair* is not joined here
  and is still handled by `MIN_HOME_MIN`.
- **Drawing agrees with detection.** On the iPhone export the needles
  found inside trips went from 99 to 111 and the points set aside from
  88 to 96: the positions now inside spans are the ones phase 17's pass
  already knew how to find.
- **Private counts, never printed**: four re-pinned.

## What broke, and the shape of each fix

1. **The defect in the backlog was not the defect in the data.** Phase
   20's log said an outing cut in two appears as two trips. Measured,
   that is one pair on one export. The same cause cuts the edge off 11
   trips there. The brief was written from the measurement, not from the
   log line.
2. **The first figures were three times too large, and were approved.**
   A gap was measured by its own length, which for one point is zero, so
   "under 5 minutes" matched every single point near home, including one
   recorded in hours of silence. The error surfaced while turning the
   brief's rule into code: the minutes parameter would have decided
   nothing. The right measure is the time between the away observations
   either side. The work stopped there, the brief gained §7, and the
   narrower rule went back to the maintainer before anything was built.
3. **The measurement scripts read the detector they measure.** Once the
   rule was in the reference detector, phase 21's and phase 22's scripts
   printed different figures from their briefs. Both now switch the rule
   off.

## Carried out of the phase

- **Single points near home with more than 30 minutes around them**:
  791 on the iPhone export. Some are stray, some are a real time at home
  in sparse data; nothing here tells them apart.
- **A far detour inside a journey between two homes** (phase 20), and
  **a home outside its labelled period** (phase 21, decided).
- **Waiting on the tester** (#8): the added trips, and now whether the
  11 whose dates moved read right.
- **Host deploy after the merge:** app and demo stacks rebuilt; no
  migration; an account's list changes only when detection runs on it
  again.
