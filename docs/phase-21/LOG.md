# Phase 21 — log: home from nights, measured and not built

What the phase does, what broke, and why. BRIEF.md and DECISIONS.md are
the other two artifacts; this closes the set.

## What the phase does

It measures, and builds nothing.

Issue #23 asked for home bases derived from where nights are spent, as a
fallback when Google's home label yields none.
`measure/measure.py` runs the reference detector unchanged with the list
of home bases swapped, on the demo, the Android fixture, the
maintainer's archive and a tester's iPhone export, and prints counts.

## What was found

- **The fallback has no data.** All four datasets yield bases from the
  label.
- **Nights are thinner than the label.** A third of the nights in both
  long files are held by no visit of three hours. Home from nights alone
  finds 22 of Google's 45 groupings on the archive where today finds 39,
  and 21 of 28 on the iPhone export where today finds all 28.
- **A labelled home is used outside its period.** 419 of 1,840 nights at
  a base's place on the archive, 702 of 2,349 on the iPhone export.
  Those nights are time away to the detector. 17 of the archive's 21
  candidates longer than a week hold five or more of them; 2 of the
  iPhone export's 16.
- **No rule over nights fixes that cleanly.** The table is in the
  brief, §2.4.
- **Seen in passing:** the iPhone export types 154 visits `HOME`, which
  today's rule does not read. Reading it gives a worse result on that
  export.

## What was decided

A stay at a home outside its labelled period is a trip, until a user
reports otherwise. #23 stays in the backlog with its trigger stated.

## What broke

1. **The reason this phase was proposed was wrong.** Phase 20 inferred
   an unlabelled home from two long spans and this phase was recommended
   on that. Measuring it first, before designing anything, is what
   showed the inference did not hold. The phase 20 log is corrected.
2. **The first measurement compared periods by first and last night.**
   A night at an old home years later then keeps it active for all the
   years between, which is why "nights only" loses so much. It was kept
   in the table because that is the era shape the issue asked for.

## Carried out of the phase

- **#23**, backlog. Trigger: an export whose home label yields no base.
- **Period extension**, not filed. Trigger: a user reports a stay at a
  former home as not a trip.
- No test, migration, contract or host change. Nothing to deploy.
