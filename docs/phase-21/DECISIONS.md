# Phase 21 — decisions

Three lines each: what was chosen, what was rejected, what would change
our mind. Written as each was made.

## 2026-11-16 — Home from nights is not built

- **Chosen:** nothing is built for #23. Every dataset in hand yields
  home bases from Google's label (1, 1, 4 and 4), so a fallback for
  "the label yields none" would run on no data and its thresholds could
  not be chosen from evidence.
- **Rejected:** building it from other projects' thresholds (nothing in
  the repository could show it works); adding places that hold many
  nights as extra bases (takes the fixture from 18 to 16 and changes
  about half the candidates on both long files at 8 nights).
- **Would change our mind:** an export, from a user or a tester, where
  the label yields no base. That is #23's trigger.

## 2026-11-16 — A stay at a home outside its labelled period is a trip

- **Chosen:** it stays a candidate, as today. The measurement found that
  a labelled home goes on being slept at outside the period its label
  covers (419 of 1,840 such nights on the archive, 702 of 2,349 on the
  iPhone export), and that 17 of the archive's 21 candidates longer than
  a week hold five or more of those nights. The maintainer's decision:
  that can be considered a trip for now.
- **Rejected:** extending a base's period from the nights spent there.
  Every threshold measured changes between 5 and 33 of the archive's 67
  candidates, finds fewer of Google's groupings than today, and has no
  flat part to choose a value from.
- **Would change our mind:** a real user reporting such a stay as not a
  trip. It is fixed then, with that report as the case to measure
  against; the kept and dismissed candidates on an instance are the
  other reference, not yet read.

## 2026-11-16 — Phase 20's reading of the tester's long trips is corrected

- **Chosen:** phase 20's brief and log said the tester's spans run long
  because of "a home Google never labelled". Measured, it is a labelled
  home outside its period, and it lengthens two of that export's trips.
  The phase 20 log carries a dated correction; its brief is left as the
  record of what was believed then.
- **Rejected:** rewriting the phase 20 brief.
- **Would change our mind:** nothing.
