# Phase 21 — Home from nights — brief

Issue #23. Written before any code. It ends at a stop: the measurement
below does not support building what the issue describes, and what it
found instead is a question about what a trip is, which is the
maintainer's to answer.

Every figure is printed by `docs/phase-21/measure/measure.py`, which runs
the reference detector unchanged and swaps only the list of home bases.
It prints counts and durations; no coordinate, date or place.

---

## 1. Concepts

### Where home comes from today

`deriveBases` (`internal/detect/bases.go`) reads one thing: visits the
export types `INFERRED_HOME`. It buckets them on a grid of about 5 km,
merges groups within 10 km, keeps groups of 8 or more, and calls each a
home base. A base is *active* from its first labelled visit to its last,
padded by 45 days each side. That span is its period.

"Away" means farther than `NEAR_M` from every base active at that moment.
So a base matters only inside its period. Outside it, the same building
is an ordinary place, and time spent there is time away.

### What the issue proposed

Two projects in the competitive study derive home from where nights are
spent. Issue #23 asks for that as a fallback: when the label yields no
base at all, find the places that hold the nights and use those.

### A night, for this purpose

The same definition phase 19 uses for a trip's nights, applied to the
whole file: each night from 22:00 to 06:00, on the clock of the
timestamp's own offset, belongs to the visit that covers the most of it,
if that is at least three hours. A night no visit covers that long is
not held by anything.

### What phase 20 thought it had seen

Phase 20's brief (§2.6) said the tester's trips run long because of "a
home Google never labelled", about 60 km from the derived base. That was
an inference from two spans, not a measurement. This brief measures it,
and it is wrong in a way that matters: see §2.3.

---

## 2. What was measured

Four datasets: the demo, the 16-month Android fixture, the maintainer's
8-year archive, the tester's iPhone export.

### 2.1 The fallback the issue describes has nothing to run on

Every dataset in hand yields bases from the label: 1, 1, 4 and 4. A
fallback that runs "only when today's derivation finds nothing" would
run on none of them. Its thresholds (how many nights, over what window)
could not be chosen from evidence, and no test could say whether it
found the right place. The only data without the label is the photo
corpus, which has its own fallback since phase 11.

### 2.2 Nights are a thinner signal than the label

| dataset | nights held by a visit | nights no visit holds |
|---|---|---|
| demo | 91 | 0 |
| Android fixture | 453 | 32 |
| archive | 1,966 | 1,051 |
| iPhone export | 2,555 | 1,212 |

A third of the nights in both long files are held by no visit of three
hours. Home derived from nights alone, on files that also have the
label, does worse than the label by the one outside reference there is:

| dataset | candidates today | nights only (places with 14+ nights) | Google's groupings found, today → nights only |
|---|---|---|---|
| Android fixture | 18 | 16 | none in the file |
| archive | 67 | 54 | 39 → 22 of 45 |
| iPhone export | 79 | 74 | 28 → 21 of 28 |

The fall is not because nights find the wrong places. On the archive the
four places holding 14 or more nights are exactly today's four bases.
It is because of the period: see next.

### 2.3 What is actually there: homes used outside their labelled period

| dataset | nights at the place of one of today's bases | of those, outside that base's period |
|---|---|---|
| Android fixture | 360 | 0 |
| archive | 1,840 | 419 |
| iPhone export | 2,349 | 702 |

Google's label covers a home for a stretch and then stops, while the
person goes on sleeping there: before the label starts, after they have
moved away and come back to visit, or in a gap. Those nights are "away"
to the detector.

This is what phase 20 saw on the tester's export. It is not an
unlabelled place. It is a labelled home outside its period.

And it is not mainly the tester's export:

| | archive | iPhone export |
|---|---|---|
| candidates today | 67 | 79 |
| holding 1 or more nights at a base's place outside its period | 30 | 23 |
| holding 3 or more | 21 | 4 |
| holding 8 or more | 15 | 2 |
| candidates longer than 7 days | 21 | 16 |
| of those, holding 5 or more such nights | 17 | 2 |

On the maintainer's own archive, 17 of the 21 trips longer than a week
are, for five nights or more, a stay at a place that is one of their
homes at another time. On the tester's export it is 2 of 16; the other
14 long trips there are long trips.

### 2.4 Every way of using nights to fix it has a cost, and none is flat

| home evidence | archive: candidates · same span as today · Google found | iPhone: candidates · same span as today · Google found |
|---|---|---|
| today | 67 · 67 · 39 of 45 | 79 · 79 · 28 of 28 |
| today + other places with 8+ nights | 67 · 35 · 32 | 79 · 40 · 27 |
| today + other places with 14+ nights | 67 · 67 · 39 (adds none) | 75 · 49 · 28 |
| today + a period for each run of 3+ nights at a base | 82 · 43 · 35 | 88 · 63 · 26 |
| the same, runs of 8+ nights | 77 · 52 · 37 | 84 · 67 · 27 |
| the same, runs of 14+ nights | 72 · 62 · 37 | 82 · 68 · 27 |
| today's bases, each active for the whole file | 52 · 34 · 22 | 67 · 41 · 20 |

Reading it:

- No row improves on today by Google's groupings on either file.
- The count moves with every step of the threshold. There is no flat
  part to choose a value from, which is what phases 19 and 20 had.
- Each row changes between 5 and 33 of the archive's 67 candidates.
- On the Android fixture, "other places with 8+ nights" adds one base
  (a second place holding 66 nights, never labelled) and takes the pin
  from 18 to 16.
- The iPhone export has 154 visits typed `HOME`, a type today's rule
  does not read. Reading it makes things worse (91 candidates, the
  longest 71 days, Google's found 27).

### 2.5 What the tester's export would gain

Two trips' dates. The thousands of short spans that export produces
(2,379 against the archive's 310) are daily life at a home outside its
period; they are rejected already and cost nothing a user sees.

---

## 3. What this means

The issue assumed a missing-data problem: no label, so no home. The data
shows a different thing. The label is present, and where it stops, the
detector calls a stay at a former or occasional home a trip.

Whether that is wrong is not a measurement. Google's own groupings count
such stays as trips. The project decided in phase 1 that a stay at
another *active* home is not one (bug 3's sibling), and has never
decided the case of a home outside its period. The maintainer has
reviewed the archive's candidates and kept 23; whether the long ones
among them are "I went back to my old home for two weeks" and whether
that belongs in a set of adventures is something only they know.

If such a stay **is** a trip, there is nothing to fix here, and #23 as
written has no data to run on.

If it **is not**, the change is not "home from nights". It is a rule
about periods, it would remove or shorten a quarter of the maintainer's
own candidates, and it needs a reference to be measured against that is
better than Google's groupings, because those disagree with it by
definition. The candidates already kept and dismissed on the
maintainer's instance are that reference, and nobody has looked at them
this way yet.

---

## 4. The choices

### A. Build the fallback as the issue describes — not recommended

It would be code that no dataset exercises, with thresholds taken from
other projects. Invariant 13's spirit applies: nothing in the repository
could show it works.

### B. Add other night places as bases — not recommended

It breaks the fixture's pin for a reason the fixture's owner may not
agree with, changes half the candidates on both long files at 8 nights,
and adds nothing on the archive at 14.

### C. Extend a base's period from nights spent there — not now

This is the change that addresses what §2.3 found. It is not
recommended *yet* because the question in §3 comes first, and because
§2.4 shows no threshold that could be defended.

### D. Stop here, and answer the question first — recommended

1. The maintainer says whether a stay at a home outside its period is a
   trip. One way to answer from evidence: of the archive's 17 long
   candidates in §2.3's last row, how many were kept and how many
   dismissed. That needs a read of the local instance's decisions, which
   has not been done and would be done only on the maintainer's word.
2. #23 is rewritten to say what was found, and stays in the backlog with
   its trigger stated: an export where the label yields no base.
3. If the answer to 1 is "not a trip", a new issue carries choice C with
   the kept/dismissed reference as its measure, and gets its own brief.
4. If the answer is "it is a trip", the tester's two long trips are
   correct as they stand, and phase 20's log line about them is amended.

---

### Decided, 2026-11-16

D, and the answer to §3 is that such a stay is a trip for now. Nothing
is built. It is looked at again when a user reports it as wrong.
`DECISIONS.md` has the record.

## 5. Not in this phase, whatever is chosen

- Snapped lines (#9): waiting on the decision about invariant 6.
- Villages (#13): waiting on testers.
- An outing cut in two by a stray position near home (phase 20's log).

## 6. Checkpoints

- **STOP 0 — this brief.** The maintainer chooses among A to D and
  answers §3's question, or says how they want it answered.
- Nothing further is planned until then. No code has been written; the
  only new files are this brief and the measurement script.
