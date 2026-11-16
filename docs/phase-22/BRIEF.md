# Phase 22 — A stray position near home cuts a trip — brief

Written before any code, and measured before it was recommended. Every
figure is printed by `docs/phase-22/measure/measure.py`, which runs the
reference detector and prints counts and durations only.

---

## 1. Concepts

### How a span ends

`findSpans` (`internal/detect/detect.go`) walks the observations in time
order. An observation farther than `NEAR_M` (25 km) from every active
home is away; a run of them is a span; the first observation that is
not away ends it. One observation is enough.

### A stray position near home

Phase 17 found that an iPhone export holds positions that are not where
the phone was: for a moment the record jumps to a place the phone knows
well, often home, and jumps back. That phase called them needles and
set them aside when a trip is *drawn*. Detection never learned about
them. Outlier rejection does not catch them either: it drops a point
only when the speed to *both* neighbours is impossible, and a jump of
40 km between observations minutes apart is not.

So one such position in the middle of time away ends the span and
starts another. This brief calls the observations near home between two
consecutive spans a **gap**.

### What phase 20 said, and what is true

Phase 20's log carried this out as "an outing cut in two appears as two
trips". Measured, that is nearly absent. What the stray positions do is
different: they cut the *edge* off a trip.

---

## 2. What was measured

### 2.1 The gaps

| | Android fixture | archive | iPhone export |
|---|---|---|---|
| spans | 25 | 310 | 2,379 |
| gaps shorter than 5 minutes | 0 | 23 | 966 |
| of those, a single observation | 0 | 16 | 923 |
| of those, holding a visit | 0 | 0 | 0 |

A gap under 5 minutes is almost always one observation and never a
visit. It is a point, not a stop at home.

### 2.2 Two cards for one outing: nearly never

Gaps with a candidate on *both* sides:

| gap length | archive | iPhone export |
|---|---|---|
| under 5 min | 0 | 1 |
| 5 to 30 min | 0 | 0 |
| 30 to 120 min | 0 | 4 |

At 30 minutes and more the gap is long enough to be a real return home
between two outings; this measurement does not tell those apart. One
pair under 30 minutes, on one export, is the whole of what phase 20's
log described with any confidence.

### 2.3 A trip with its edge cut off: common on the iPhone export

Gaps under 5 minutes with a candidate on *one* side: 4 on the archive,
32 on the iPhone export. The span on the other side is too short or too
near to be a candidate itself, so it is dropped, and the trip starts
late or ends early.

Joining spans across a gap of one observation shorter than 5 minutes:

| | archive | iPhone export |
|---|---|---|
| candidates | 67 → 67 | 79 → 77 |
| made longer | 3 | 29 |
| median time added | 0.7 h | 4.6 h |
| under 1 h · 1 to 6 h · over 6 h | 2 · 1 · 0 | 3 · 16 · 10 |
| two made one | 0 | 2 |
| destination distance changed | 0 | 0 |
| Google's groupings found | 39 of 45, unchanged | 28 of 28, unchanged |

The fixture and the demo have no such gap and do not change.

### 2.4 The result does not depend on the threshold

Joining across gaps of at most two observations and under 30 minutes
gives the same counts on both files (67 → 67 with 3 longer; 79 → 77
with 29 longer). Joining across *any* gap, whatever it holds, starts to
swallow real returns home from 60 minutes: the tester's candidates fall
to 75, then 73 at two hours.

### 2.5 What the measurement cannot say

Whether the hours added are the trip's own. On the tester's export much
of the time just outside a trip is daily life at a home outside its
labelled period (phase 21), which by decision is time away. Joined to
the trip it makes one longer trip, which is consistent with that
decision and may still surprise the person reading it. Ten trips gain
more than six hours. Nobody has reported a trip starting late or ending
early.

---

## 3. What would be built

- **Go** (`internal/detect`): in `findSpans`, a run of observations near
  home does not end a span when it is at most `MAX_STRAY_OBS`
  observations, lasts less than `MAX_STRAY_MIN` minutes, holds no visit,
  and has away observations on both sides. Two named parameters,
  recorded with the run. Table tests for each condition.
- **The reference detector**: the same rule; expected files regenerated
  beside the pinned ones as in phase 20.
- **`reachedDest`**: unchanged. Its chain rule (`MIN_HOME_MIN`) stays;
  it also looks through longer gaps than this joins.
- **Pins**: fixture 18 and demo 3 unchanged; archive 67 with three
  candidates' dates moved; iPhone 79 → 77. The private count pins move
  again.
- **No migration, no contract change, no web change.**

---

## 4. The choices

### A. Join across a stray position — one observation, under 5 minutes

The narrowest rule that the data supports, and §2.4 says widening it to
two observations and 30 minutes changes nothing today.

### B. Wait for a report

The rule of phase 21: a behaviour nobody has complained about stays
until someone does. In its favour here: §2.5. Against: this is not a
question of definition. The position is false, phase 17 already treats
it as false when drawing, and the tester is about to read these 79
trips for #8.

### C. Set stray positions aside before detection, the way drawing does

One definition of a needle for both. Larger: phase 17's pass works on
an assembled journey, not on raw observations, and its thresholds were
measured there.

**Recommended: A**, with B a reasonable answer if the maintainer reads
§2.5 as the larger risk.

---

## 5. Not in this phase

- A home used outside its labelled period (phase 21, decided).
- Any change to how a trip is drawn.
- Snapped lines (#9), villages (#13), home from nights (#23).

## 6. Checkpoints

- **STOP 0 — this brief.** A, B or C. If A: an issue, a milestone and a
  branch `phase-22`.
- **STOP 1 — the detectors**, tests first, with a diff of old against
  new reference output for the maintainer to read.
- **Close**: log, plan, PR, host rebuild (no migration), a note on #8 if
  the tester's list is about to change again.

---

## 7. Correction, 2026-11-16 — before any code

Found while turning §3 into a rule, after choice A was approved. §2.3
and §2.4 overstate the effect, and §3's "lasts less than
`MAX_STRAY_MIN` minutes" does not mean what it reads as.

### What was wrong

A gap's length was measured from its first observation's start to the
latest end among its observations. A gap of one point therefore has
length zero whatever happened around it. "Under 5 minutes" selected
every single point near home, including one recorded in the middle of
hours of silence, when the traveller may well have been at home.

The measure that says "this point cannot be a return home" is the time
between the away observations either side of it. By that measure the
single points are mostly not tightly bracketed:

| time between the away observations either side | archive | iPhone export |
|---|---|---|
| under 30 min | 8 | 62 |
| 30 to 120 min | 8 | 415 |
| over 120 min | 0 | 376 |

### The effect under a bound that can be defended

Joining across one point near home, not a visit, with away observations
either side less than B minutes apart:

| | archive | iPhone export |
|---|---|---|
| B = 15 | 67 → 67, none longer | 79 → 79, 6 longer |
| B = 30 | 67 → 67, 1 longer | 79 → 77, 11 longer (median 1.7 h; 1 over 6 h), two pairs made one |
| B = 60 | 67 → 67, 3 longer | the same as B = 30 |
| no bound (§2.3's figures) | 67 → 67, 3 longer | 79 → 77, 29 longer (median 4.6 h; 10 over 6 h) |

The fixture and the demo do not change at any bound. Google's groupings
found do not change at any bound. Allowing two points changes nothing
at B = 30.

So the supportable claim is 11 trips on the tester's export and 1 on the
archive, by a median of under two hours, with two pairs of cards made
one each. Not 29 trips by 4.6 hours.

### The rule, restated

A run of observations near home does not end a span when it is one
observation (`MAX_STRAY_OBS`), is not a visit, and the away observations
either side of it are less than `MAX_STRAY_MIN` (30) minutes apart. The
count on the tester's export is the same from 30 to 60 minutes; on the
archive it moves by two candidates' dates across that range.

### Whether A still stands

It is the same rule in kind, narrower, and §2.5's worry mostly goes
away: with the away observations half an hour apart, the time added is
adjacent travel, not a day at another home. The benefit is about a
third of what the maintainer approved. That is a different decision
from the one that was made, so it goes back to the maintainer.
