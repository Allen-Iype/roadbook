# Phase 17 — Tester fixes: decisions

Written as decisions are made. Three lines each: what was chosen, what was
rejected, what would change our mind.

## 2026-11-04 — One phase, one branch, one PR for four issues

- **Chosen:** issues #1–#4 as one small phase with one brief, on
  `phase-17`, closed by one pull request; the maintainer commits at each
  STOP.
- **Rejected:** four separate fixes without a brief — #3 changes the schema
  and #4 adds a source format; both are the kind of change a brief exists
  for.
- **Would change our mind:** #4 needing to ship before #3 is ready; then
  STOP 1 goes out as its own PR.

## 2026-11-04 — This is phase 17; Places becomes phase 18

- **Chosen:** the tester fixes take the next number, 17, as every other
  phase has one; "Places" (planned 2026-10-29, brief pending, no code)
  moves to 18. Live references follow (PLAN.md heading with a dated note,
  the code comments marking its slots, the competitive study); phase 16's
  DECISIONS keep "phase 17" as written on the day.
- **Rejected:** a named folder (`tester-fixes`) among numbered ones;
  `phase-16b` (these fixes aren't phase 16's reading-load work — #3
  changes the schema, #4 adds a source format).
- **Would change our mind:** nothing; Places had not started.

## 2026-11-04 — #2: the numeral is removed, not renamed

- **Chosen:** "PLATE n" leaves every surface (maintainer's choice).
- **Rejected:** "Trip 2" (a number with no job a traveller needs); a bare
  numeral (still unexplained).
- **Would change our mind:** someone asking how to refer to a trip by
  number — a share or print use, not a reading one.

## 2026-11-04 — #3: notes are owner-only

- **Chosen:** notes show on the owner's trip page only; share links
  unchanged (maintainer's choice).
- **Rejected:** notes on shared views by default — private words leaving
  without being asked to.
- **Would change our mind:** testers asking to share what they wrote; then
  a per-share choice, never a default.

## 2026-11-04 — #4: proven on the tester's real file

- **Chosen:** the tester's export (`data/ios-timeline.json`, private) is the
  acceptance file; its structure was read without values, and a fictional
  iPhone-shaped demo carries the committed tests.
- **Rejected:** synthetic-only (the spelling of `semanticType` would have
  been guessed, and was different from Android's).
- **Would change our mind:** nothing.
