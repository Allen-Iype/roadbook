# Phase 20 — Far trips without a long stop — brief

Issue #8. Written 2026-11-13, before any code. Nothing here is decided
until the maintainer has read it; §4 holds the choices and a
recommendation for each.

Detection proposes a trip only where the traveller stayed: a visit of at
least 60 minutes more than 100 km from home. A tester's export showed two
real trips that rule misses. This brief measures why, on all three
private datasets, and proposes a second way for an away-span to qualify:
it left one home, was beyond 100 km for a sustained time, and came back
to the same home. The existing rule is not changed; the new one is added
beside it.

Every figure below is printed by
`python3 docs/phase-20/measure/measure.py <export>…` (§2 gives the exact
command). The private datasets are not in the repository; on a clone
without them only the demo line can be reproduced.

---

## 1. Concepts this phase introduces

### What the destination rule is for, and what it cannot see

Detection works in two steps (`internal/detect/detect.go`, `Run`). First
it finds **away-spans**: maximal runs of observations farther than
`NEAR_M` (25 km) from every home base active at that moment. Then it asks
each span for a **destination**: a visit of at least `MIN_DWELL_MIN` (60)
minutes, farther than `FAR_KM` (100) from home. No such visit, no
candidate.

The second step exists because of bug 3. With two homes 300 km apart, the
middle of the road between them is more than 100 km from both, so every
drive from one home to the other looked like a trip: 42 false candidates
on the archive. Requiring a place where the traveller *dwelt* removed all
42, because nobody dwells in the middle of a corridor.

The rule leans on one thing: Google's own decision to record a `visit`
segment, and how long it made it. That is the part that fails. A day
spent walking around a town can be recorded as six visits of 20 minutes
each. A phone that records little while far away leaves a visit of 10
minutes where the traveller spent a day. In both cases the traveller was
far from home for hours, the data says so, and the rule cannot see it
because it only reads visit durations.

### Two different things a span can be

The 42 false candidates and the missed trips look alike to the current
rule: both are spans that go beyond 100 km with no long visit there. They
differ in one fact the detector already has and does not use: **where the
span starts and where it ends.**

- A **home-to-home** span leaves one base and arrives at another. Being
  far from both in the middle is geometry, not travel.
- A **round trip** leaves a base and returns to the same one. If it was
  beyond 100 km in between, the traveller went somewhere and came back.
  There is no corridor to explain it.

The observation just before a span and the one just after it are, by the
definition of a span, within 25 km of an active base. Which base each is
nearest to is a lookup, not a new threshold. A span that touches the
first or last observation of the file has no "before" or "after", so it
can be called neither; it is the cut-off case bug 4 already marks.

### Time beyond FAR, measured from every observation

The second new idea is to measure presence far from home from *all*
observations in the span (visits, activity endpoints, path points), not
from visit durations alone. A **far run** is a maximal run of consecutive
observations each farther than `FAR_KM` from home; its length is the time
from its first observation to its last. "Continuously beyond 100 km for
at least 30 minutes" is a statement about where the phone was, however
Google chose to cut that time into visits and activities.

A minimum is needed because one stray position can sit beyond 100 km on
its own. Outlier rejection drops a point that is impossibly fast from
both neighbours, but not every bad point is that bad.

### Where this sits in the code

Inside `detect.Run`, at the point where a span with no qualifying dwell
is dropped today (`if len(dwells) == 0 { continue }` and the `FAR_KM`
test after it). The new path is tried only when the old one fails, so
every candidate that exists today is produced by the same code with the
same inputs as before. The core stays pure (invariant 1), the new
threshold is a named parameter recorded with each run (invariant 3), and
the same rule is written into `prototype/detect_fixture.py` so the Go and
Python detectors still agree field for field.

---

## 2. What was measured

```
python3 docs/phase-20/measure/measure.py \
    data/fixture-2025-04.segments.timeline.json data/Timeline.json \
    data/ios-timeline.as-android.json testdata/demo/demo.json
```

The script is a measurement port of the reference detector: parsing,
outlier rejection, bases and spans are the reference's, and it reproduces
the pinned counts (18, 66, 60, 3) before measuring anything else. It
reads its inputs and writes nothing.

### 2.1 The issue's first question: is the stop missing, or too short?

Both, and on the two trips the issue names:

- **The day trip.** The span has 79 observations beyond 100 km, the
  longest far run lasting 205 minutes, and *no visit at all* beyond
  100 km. The farthest visit of any length is closer than 100 km. The
  stop is missing from the data.
- **The long-distance trip.** The span holds visits more than 2,000 km
  from home, of 4, 10 and 10 minutes. The stops are present and far too
  short. The longest far run at that distance is 58 minutes: the phone
  recorded little while it was away.

Across all rejected spans that reach beyond 100 km, the split is about
even:

| dataset | rejected spans reaching beyond FAR | no visit beyond FAR | a visit there, under 60 min |
|---|---|---|---|
| archive | 42 | 24 | 18 |
| iPhone export | 30 | 11 | 19 |
| fixture | 0 | – | – |
| demo | 0 | – | – |

So no change to the visit-duration threshold alone can be the answer:
for half of these spans there is no visit to measure.

### 2.2 The rejected spans, by where they start and end

| dataset | reaching beyond FAR | same base out and back | home to home | cut by the file's edge |
|---|---|---|---|---|
| archive | 42 | 3 | 39 | 0 |
| iPhone export | 30 | 25 | 5 | 0 |

The archive's 42 are the same 42 that bug 3 was named for, and 39 of them
are home-to-home. The tester's export is the opposite shape: 25 of 30 are
round trips. The two datasets needed opposite things from one rule, and
the start-and-end fact is what tells them apart.

### 2.3 The three rules, compared

Counts are candidates *added* to today's.

**Rule 1 — lower `MIN_DWELL_MIN`** (a visit beyond FAR of at least D
minutes):

| D | archive: added (home to home) | iPhone: added (home to home) |
|---|---|---|
| 45 | 3 (2) | 4 (0) |
| 30 | 5 (4) | 10 (1) |
| 20 | 7 (6) | 14 (1) |
| 10 | 16 (15) | 17 (1) |
| 0 | 18 (17) | 19 (1) |

On the archive nearly everything it adds is a false home-to-home trip: a
fuel or meal stop in the corridor. And at no value does it find the day
trip in §2.1, which has no far visit.

**Rule 2 — "far enough, or a flight"** (the issue's suggestion: a
`FLYING` activity in a span that reaches beyond FAR): adds **0** on every
dataset. Neither missed trip contains a flying activity; the
2,000 km one is recorded as a subway ride and a 21-hour "passenger
vehicle" activity. This is the hazard already on record: modes are
guesses that fail at extremes. A rule keyed on the mode label cannot fix
a case where the label is wrong.

**Rule 3 — continuously beyond FAR for at least T minutes**, split by
class:

| T (min) | archive: same base | archive: home to home | iPhone: same base | iPhone: home to home |
|---|---|---|---|---|
| 0 | 3 | 39 | 25 | 5 |
| 10 | 2 | 36 | 23 | 4 |
| 20 | 2 | 34 | 23 | 4 |
| 30 | 2 | 34 | 22 | 3 |
| 45 | 2 | 30 | 22 | 3 |
| 60 | 2 | 28 | 21 | 2 |
| 90 | 1 | 25 | 19 | 1 |
| 120 | 1 | 24 | 14 | 1 |
| 180 | 1 | 13 | 10 | 0 |

Without the same-base condition, rule 3 is bug 3 again: 34 false trips on
the archive at T = 30. With it, the home-to-home columns are excluded by
construction and the same-base columns are what is added. Those are flat
from 10 to 45 minutes on both datasets (2 and 22–23), which is where a
threshold should sit. The step from 0 to 10 is single stray positions
(two on the iPhone export, one on the archive).

### 2.4 The recommended rule on each dataset

Rule 3, same base only, T = 30:

| dataset | today | with the rule | Google's groupings reaching ≥ 100 km: found today → with the rule |
|---|---|---|---|
| fixture | 18 | 18 | none in the file |
| archive | 66 | 68 | 38 → 39 of 45 |
| iPhone export | 60 | 82 | 26 → 28 of 28 |
| demo | 3 | 3 | none in the file |

The archive's remaining 6 unfound groupings are not misses to fix: 4
overlap a home-to-home span (Google calls a stay at the other home a
trip; this project decided it is not), and 2 overlap no away-span beyond
100 km at all.

Among the 22 the tester would gain is a round trip reaching more than
900 km whose longest far visit is 54 minutes: six minutes under the
threshold. That one alone says the gap is real and not confined to short
outings.

### 2.5 What the new candidates would score

Their destination-dwell component is zero, so they rank low. Computed
from the same weights as `score.go`: all but two of the 24 new candidates
score between 24 and 31; the two long-distance ones score about 54 and
67. Existing candidates would score a median of 44 (archive) and 35
(iPhone) on the other three components alone, before their dwell is
added. The new ones go to the bottom of "Trips to review", which is the
existing principle: rank, do not filter.

### 2.6 Two things the measurement showed that this phase does not fix

- **The tester's spans are too long.** The span holding the
  long-distance trip is 14 days, because it starts and ends at a place
  60 km from the derived home where the traveller repeatedly stays for
  days: a home Google never labelled. The trip would be found, with the
  right destination and the wrong dates. That is #23 (home from nights),
  and the same cause explains why that export has 518 rejected spans
  where the archive has 124.
- **Some outings would appear as two candidates.** On the tester's
  export, two days each yield a pair of short same-base spans to the same
  distance, hours apart. A round trip of 250 km cannot fit in either
  half, so something put the phone "at home" in the middle of the day.
  That matches the fallback positions phase 17 called needles, which are
  handled when a journey is drawn and not in detection. Not verified
  here. It is a finding for its own issue.

---

## 3. What gets built

### Go

- **`internal/detect/detect.go`**
  - `Params.MinFarMin` (`MIN_FAR_MIN`, default 30), with a `-min-far-min`
    flag on `roadbook detect` beside the existing ones.
  - For each span: the base nearest the observation before it and the
    base nearest the observation after it (among bases active at each
    moment, the same set `homeDistM` uses), and its far runs.
  - A span with no qualifying dwell becomes a candidate when it is not
    cut by the file's edge, leaves and returns to the same base, and has
    a far run of at least `MinFarMin` minutes. `MIN_OBS` and `MIN_HRS`
    apply as they do today.
  - `Candidate.Reached bool`: true when the candidate qualified by the
    new path. Its destination is chosen by choice C below. `Stops` keeps
    its meaning (dwelt visits in the span) and may be 0.
- **Tests, written first, one per failure this brief names**
  (`detect_test.go`, hand-built observations, no private data):
  a round trip with a 40-minute far run and no visit → candidate, marked
  reached; the same trip between two different bases → none (bug 3 held);
  a single stray far position → none; a 20-minute far run → none; a
  qualifying span at the file's edge → none; a span that qualifies both
  ways → identical to today's candidate, not marked.
- **Migration 17**: `candidates.reached boolean not null default false`.
- **`api/openapi.yaml`**: `Candidate.reached`, and the `dest_km`
  description, which today reads "farthest place dwelt, never merely
  passed through", states both cases. Go and TypeScript regenerated.
- **`roadbook detect`** prints the mark in its candidate table, the way
  it prints the cut-off mark (invariant 13).

### The reference detector

- **`prototype/detect_fixture.py`** gains the same rule and the
  `reached` field, and takes its output path as a second argument. Today
  the path is fixed, so running it on the archive overwrites the
  fixture's expected file unless the script is copied first.
- **`regression_test.go`** compares `reached` and guards `MIN_FAR_MIN`
  in the expected file's parameters like the other six.
- **The expected files are regenerated by the maintainer**, not by
  Claude: they live in `data/`, which the working agreement says is never
  modified by a tool. The commands, and a diff of old against new to read
  first, are prepared at STOP 1. Expected: the fixture's file gains the
  new field and parameter and nothing else; the archive's gains 2
  candidates; the iPhone's gains 22. `repeat` on later candidates may
  rise where a new candidate's destination is within 60 km; no other
  field of an existing candidate can change.

### Web

- A reached trip says so where the distance is shown. The card and the
  score breakdown already say "N km from home" and show destination
  dwell as 0 h; the addition is a short line that no long stop was
  recorded there. The words are settled on real pages at the web stop,
  under `docs/product/words.md`.
- No new page, no new component. Playwright as it stands.

### Not changed

Home bases, outlier rejection, span finding, the score's weights, journey
assembly, routing, nights, places. Kept decisions match candidates by
span and destination (`detect.SpanRef`), and no existing candidate's span
or destination moves, so every kept or dismissed trip stays attached.

---

## 4. The choices

### A. The rule — recommended: same base out and back, with time beyond FAR

The three candidates are measured in §2.3.

- *Lower the dwell minimum.* Rejected: it cannot see the day trip, and
  on the archive almost all it adds is false.
- *"Or a flight."* Rejected: adds nothing; the label is wrong exactly
  where it is needed.
- *Same base, time beyond FAR.* Recommended. It finds both named trips,
  all 28 of Google's groupings on the tester's export, and one more on
  the archive, and by construction it cannot admit a home-to-home span.

What it does not cover: a trip that leaves one home, goes far with no
long stop, and ends at a *different* home. Those stay unfound (3 on the
iPhone export, 34 on the archive at T = 30, and the archive's are the
corridor). Telling a detour from a corridor needs a measure of how far
off the straight line between two homes the span went. That is a further
rule with its own false-positive study, and nothing a tester has reported
needs it.

### B. The minimum time beyond FAR — recommended: 30 minutes

§2.3: the count is flat from 10 to 45 minutes on both datasets. 30 sits
in the middle of the flat part, clear of the stray-position step below
10. At 60 the long-distance trip in the issue would qualify only through
an unrelated outing in the same 14-day span (its own far run is 58
minutes), which is luck, not a rule. The value is a named parameter,
recorded with each run, and a flag.

The alternative shape, a minimum *count* of far observations instead of
minutes, was considered and not recommended: observation density differs
several-fold between Android and iPhone exports, and minutes mean the
same thing on both.

### C. What a reached trip's destination is — recommended: the farthest brief visit, else the farthest point of a qualifying far run

Today "destination" means the farthest place *dwelt*. A reached trip has
none. Options:

1. **The farthest visit beyond FAR of any length; if there is none, the
   farthest observation inside a far run that met the minimum.**
   Recommended. A visit, however short, is Google saying the traveller
   stopped. On the long-distance trip this gives the place 2,600 km out
   (two 10-minute visits) and not a point 2,250 km out that happens to
   sit in the longer run. Where there is no visit, restricting to a
   qualifying run keeps a lone stray point from becoming the
   destination.
2. The farthest observation in the span. Simplest; lets one bad point
   set the distance.
3. The farthest observation in qualifying runs only. Safe; under-reports
   the long-distance trip by 370 km.

Whichever is chosen, `reached` is served so the page never presents such
a destination as a place stayed.

### D. Whether to mark these trips at all — recommended: yes, store and serve `reached`

Without a mark, "N km from home" on a card would mean two different
things with nothing to tell them apart, and the contract's description of
`dest_km` would be false for some rows. The cost is one boolean column
and one schema field. The alternative, deriving it on the page from "0
stops", is wrong: `stops` counts dwelt visits anywhere in the span, and a
reached trip can have long stops within 100 km.

### E. Spans cut by the file's edge — recommended: do not admit them by the new path

A cut-off span has only one known end, so "same base" cannot be shown.
The old path still admits cut-off spans with a real dwell, marked
truncated as today. None of the three datasets has a span this choice
excludes, so it costs nothing measured, and it keeps the new rule's
guarantee whole.

### F. The 22 new candidates on the tester's instance — recommended: accept, and say so to the tester

82 candidates is more to review than 60. Pilot evidence already says a
long list stops people (79 candidates, stopped after 10), and bulk triage
was built for that. The new ones score lowest and sit last. The
alternative, raising the bar until fewer appear, is filtering by another
name, and §2.3 shows no natural higher threshold: the count falls
steadily past 60 minutes with no second flat part.

---

## 5. What is not in this phase

- **Home from nights (#23).** It would shorten the tester's 14-day span
  to the real trip. A change to base derivation with its own pins.
- **Needles in detection** (§2.6). To be filed as its own issue if the
  maintainer agrees.
- **Home-to-home detours** (choice A's last paragraph).
- **Trips under 100 km.** Google groups 27 on the tester's export;
  `FAR_KM` excludes them by design, and it is already a parameter.
- **The score.** No weight or anchor changes. Reached trips rank low
  because their dwell is zero, which is true.
- **Snapped lines (#9).** Waiting on the charter decision about
  invariant 6.

---

## 6. Checkpoints

- **STOP 0 — this brief.** The maintainer reads it and decides A to F.
  Two things only the maintainer can do here: run the script with
  `--list` on the archive and say whether the two spans it would add are
  trips they recognise; and decide whether §2.6's second finding gets an
  issue. Then: branch `phase-20`, milestone, #8 off `backlog`.
- **STOP 1 — the detectors.** Tests first, then `detect.go`, the
  prototype, migration 17, the contract, the command's column. Claude
  prepares the regeneration commands and a field-by-field diff of old
  against new expected output; the maintainer reads the diff, regenerates
  the three files in `data/`, and `make test` runs green with the pins at
  18 / 68 / 82 / 3 and both journey goldens byte-identical. The photo
  corpus pin (2 candidates) is checked here; it takes the synthetic-visit
  path and was not measured by the script.
- **STOP 2 — web.** The line on a reached trip, on a scratch stack loaded
  with the maintainer's own data for the two archive trips.
- **Close.** LOG.md, PLAN.md, the PR, the host rebuild (migration 17
  applies at start-up), a comment on #8 asking the tester to look at
  their list again.

---

## 7. Amendment, 2026-11-13 — the rule as first built admitted a false trip

Written after STOP 1 was committed, during STOP 2. Sections 1 to 6 stand
as the record of what was agreed; where this section differs, it is the
rule as built.

### What went wrong

§1 says the observations either side of a span are near a home, and that
when both are nearest the same home "no corridor can explain the
distance". That holds only if the observations near home are real. On
the archive, one of the two spans the rule added was a journey from one
home to another. Partway along, the export holds two positions back
near the first home, minutes apart. The outlier pass (§1 of the phase 1
design: drop a point faster than `MAX_KMH` from *both* neighbours) keeps
them, because each is slow relative to the other. They end the
away-span, so the journey is two spans: the first leaves the first home
and "returns" to it, with far more than 30 minutes beyond `FAR_KM`
between. It qualified. The second half ends at the other home and was
correctly rejected.

This is bug 3 reached by a different road, and the brief's §2.6 had the
warning in it: stray positions near home were listed as something the
phase does not fix, on the reasoning that they only split outings. They
also split transits.

### The correction

"Out and back" is asked of the **chain** a span belongs to. A time near
home that lasts less than `MIN_HOME_MIN` (new parameter, default 30
minutes, measured from its first observation's start to the latest end
among its observations) does not count as having come home: the test
looks through it to the neighbouring span, and repeats in both
directions. The observations compared are the ones either side of the
whole chain. A chain that touches the edge of the file is cut, as a span
was.

What does not change: the spans. They are still found the way step 4 of
the detection rule finds them, with the same dates, observation counts
and track distances; the dwell rule is untouched; a candidate found by
a dwelt destination cannot be affected, because the chain is consulted
only by `reachedDest`. Joining the spans instead would have moved the
dates of existing candidates and is rejected in the decision log.

### Measured (`measure.py`, the lines under "as built")

| dataset | before | §2.4's rule | as built | Google's groupings found |
|---|---|---|---|---|
| fixture | 18 | 18 | 18 | none in the file |
| archive | 66 | 68 | 67 | 39 of 45, unchanged |
| iPhone export | 60 | 82 | 79 | 28 of 28, unchanged |
| demo | 3 | 3 | 3 | none in the file |

- The counts are the same for any `MIN_HOME_MIN` from 15 to 120 minutes
  on both private datasets. Below 15 the false trip is admitted again.
  From 180 the iPhone count rises, as chains grow long enough to end
  where they began.
- The three spans the iPhone export no longer gains are legs of a single
  chain that starts at one home and ends at another. §2 cited the
  longest of them as an example of what the rule finds; it is withdrawn
  as evidence. Whether a far detour inside a journey between homes is a
  trip is the question §5 already left out.
- Both trips named in #8 are still found.

### Still not fixed

A real outing cut in two by a stray position near home is found as two
trips, each a round trip in its own right. `TestReached` has a case
that records this as the current behaviour. It is the detection-side
counterpart of phase 17's needles and wants its own issue.

### Checkpoint figures

Where §6 says 18 / 68 / 82 / 3, read 18 / 67 / 79 / 3.
