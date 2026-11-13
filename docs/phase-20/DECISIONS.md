# Phase 20 — decisions

Three lines each: what was chosen, what was rejected, what would change
our mind. Written as the decisions were made.

## 2026-11-13 — A second way to qualify: out and back from one home, with time beyond FAR

- **Chosen:** a span with no dwelt destination is still a candidate when
  the observations either side of it are nearest the same home base and
  it holds a run of consecutive observations beyond `FAR_KM` lasting at
  least `MIN_FAR_MIN`. Tried only when the dwell rule fails, so every
  earlier candidate is produced as before (BRIEF §2.3, §4 A).
- **Rejected:** lowering `MIN_DWELL_MIN` (half the missed spans have no
  far visit to measure, and on the archive what it adds is home-to-home
  stops); "or a flight" (adds nothing on any dataset: the missed trips
  carry no flying activity); time beyond FAR without the same-base
  condition (34 home-to-home spans on the archive at 30 minutes: bug 3).
- **Would change our mind:** a tester's trip that leaves one home, goes
  far with no long stop and ends at another. That needs a measure of how
  far a span strays from the line between the two homes, studied on its
  own.

## 2026-11-13 — MIN_FAR_MIN is 30 minutes

- **Chosen:** 30. The count added is flat from 10 to 45 minutes on both
  datasets that have any (2 on the archive, 22–23 on the iPhone export).
- **Rejected:** 60 (the long-distance trip in the issue has a 58-minute
  far run and would qualify only through an unrelated outing in the same
  span); 0 (admits single stray positions); a count of observations in
  place of minutes (density differs several-fold between Android and
  iPhone exports).
- **Would change our mind:** reviewers dismissing most reached trips as
  noise, by which threshold they fall under. It is a parameter and a flag.

## 2026-11-13 — A reached trip's destination: the farthest brief visit, else the farthest point of a qualifying run

- **Chosen:** the farthest visit beyond `FAR_KM` of any length; if the
  span has none, the farthest observation inside a far run that met the
  minimum. Read from the observation list, so a visit that outlier
  rejection dropped cannot become a destination. (The brief said "the
  farthest visit"; reading it from the surviving observations and not
  from the raw visit list was settled while writing the code.)
- **Rejected:** the farthest observation in the span (one bad position
  sets the distance); qualifying runs only (puts the long-distance trip
  370 km short of two brief visits recorded there).
- **Would change our mind:** a reached trip shown at a distance its owner
  says they never went.

## 2026-11-13 — Reached trips are marked, in the row and the contract

- **Chosen:** `Candidate.Reached`, `candidates.reached` (migration 17),
  `reached` required in the contract, a column in `roadbook detect`.
  `dest_km`'s description states both meanings.
- **Rejected:** inferring it on the page from `stops == 0` (`stops`
  counts long stays anywhere in the span, including within 100 km).
- **Would change our mind:** nothing foreseen; removing it would make one
  figure mean two things.

## 2026-11-13 — A span cut by the file's edge cannot qualify this way

- **Chosen:** the new path needs an observation before and after the
  span. A cut-off span with a real dwell is still found and marked
  truncated, as before. No span on any dataset is excluded by this.
- **Rejected:** treating the known end as both ends.
- **Would change our mind:** monthly exports where the trip in progress
  at the cut is regularly a short-stop one.

## 2026-11-13 — The tester's 22 more candidates are accepted

- **Chosen:** they appear, scored in the mid-20s and so last in the
  review list. Rank, do not filter.
- **Rejected:** a higher threshold to thin them: the count falls steadily
  past 60 minutes with no second flat part to choose.
- **Would change our mind:** the tester's reply on #8.

## 2026-11-13 — An existing test changed meaning, and says so

- **Chosen:** the synthetic case "passing through without dwelling is not
  an adventure (bug 3)" expected no candidate. Its data is a round trip
  from the only home, 111 km out for two hours with a 30-minute visit:
  issue #8 exactly. It now expects one reached candidate, under a new
  name, with a comment recording the old one. Bug 3 keeps its own case
  (the corridor between two homes) and gains a second (a short stop in
  the corridor).
- **Rejected:** stretching the old case until it failed again.
- **Would change our mind:** nothing; the old name described data the
  case did not contain.

## 2026-11-13 — Six private count pins re-pinned with the trip totals

- **Chosen:** the needles, nights and stop-name counts for the archive
  and the iPhone export are counts over every trip, so they moved with 2
  and 22 more trips. Each was set to the value its own test printed, with
  the earlier figures kept in a comment. The fixture's three lines and
  both journey goldens did not move.
- **Rejected:** restricting those tests to trips that are not reached
  (they would stop measuring what a user sees).
- **Would change our mind:** nothing.

## 2026-11-13 — New reference output is written beside the old, not over it

- **Chosen:** the reference detector takes its output path as a second
  argument. Fresh expected files were written to a new folder under
  `data/`, the Go detector was held against them through
  `ROADBOOK_EXPECTED_DIR`, and `diff_expected.py` printed old against
  new. The maintainer replaces the three pinned files after reading the
  diff.
- **Rejected:** letting the tool overwrite the pinned files (the working
  agreement: nothing in `data/` is modified by a tool); writing them
  outside `data/` (they hold real coordinates).
- **Would change our mind:** nothing.
