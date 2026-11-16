# Phase 22 — decisions

Three lines each: what was chosen, what was rejected, what would change
our mind. Written as each was made.

## 2026-11-16 — One stray position near home does not end time away

- **Chosen:** in `findSpans`, a run of observations near home stays
  inside the span when it is at most `MAX_STRAY_OBS` (1) observations,
  none of them a visit, and an away observation follows less than
  `MAX_STRAY_MIN` (30) minutes after the latest end among the span's
  observations so far. Both detectors carry it and agree field for
  field. Issue #28.
- **Rejected:** waiting for a report (the position is false, not a
  matter of definition, and phase 17 already sets it aside when a trip
  is drawn); setting stray positions aside before detection with phase
  17's pass (it works on an assembled journey and its thresholds were
  measured there); joining across any short time near home (from 60
  minutes it swallows real returns).
- **Would change our mind:** a trip that gains a stretch its owner says
  was a real return home.

## 2026-11-16 — The bound is on the away observations either side, and is 30 minutes

- **Chosen:** the brief first measured a gap by its own length, which
  for one point is zero, so "under 5 minutes" selected every single
  point near home, including one recorded in the middle of hours of
  silence. The time that says a point cannot be a return home is the
  time between the away observations either side of it. At 30 minutes
  the tester's export goes from 79 candidates to 77, with 11 trips'
  spans changed (two pairs made one) by a median of 1.7 hours; the
  archive keeps 67 with one trip 0.4 hours longer. The tester's result
  is the same from 30 to 60 minutes.
- **Rejected:** no bound (29 trips on the tester's export, a median of
  4.6 hours, much of it possibly a real time at home; these were the
  figures first approved, corrected in the brief's §7 before any code);
  15 minutes (6 trips, and the count is not yet flat); 60 minutes (the
  same on the tester's export, two more of the archive's trips moved,
  for no gain).
- **Would change our mind:** an export whose stray positions sit in
  sparser data, where 30 minutes catches too few.

## 2026-11-16 — MAX_STRAY_OBS is 1

- **Chosen:** one. Allowing two changes no count on any dataset at 30
  minutes, and a pair of positions near home minutes apart is weaker
  evidence of a stray than one. The parameter exists so the count is
  not a literal.
- **Rejected:** two, for symmetry with the pair phase 20 found: that
  pair is handled where it mattered, by `MIN_HOME_MIN`.
- **Would change our mind:** an export where stray positions come in
  pairs between tightly spaced away observations.

## 2026-11-16 — Phase 20's chain rule stays

- **Chosen:** `reachedDest` still looks through a time near home shorter
  than `MIN_HOME_MIN`. The two rules overlap only for a single point
  with away observations under 30 minutes apart; the chain rule also
  covers pairs and longer gaps, which this phase does not join.
  `TestReached`'s cases for a stray pair pass unchanged.
- **Rejected:** removing the chain rule (the archive's false trip from
  phase 20 was a pair of positions: not joined here).
- **Would change our mind:** nothing measured.

## 2026-11-16 — Private count pins re-pinned; the measurement scripts keep their figures

- **Chosen:** four private counts moved with the trips and were set to
  the values their tests printed, old figures in comments. The needle
  count on the tester's export rose from 99 to 111, with 96 set aside
  where 88 were: the stray positions now inside trips are found by
  phase 17's pass when the trip is drawn. The measurement scripts of
  phases 21 and 22 run the reference detector, so both now switch this
  rule off to print the figures their briefs record.
- **Rejected:** letting the scripts drift from the documents that cite
  them.
- **Would change our mind:** nothing.
