# Phase 19 — Nights: where each one was spent — brief

Issue #14. Written 2026-11-13, before any code. Nothing here is decided
until the maintainer has read it; §4 holds the choices and a
recommendation for each.

A trip is remembered by its night halts. Phase 18 gave stops a town; this
phase counts a trip's nights and says where each was spent: "3 nights: 2
near Trimbak, 1 on the move". It is one pure pass over an assembled
journey, with its own named parameters, served beside the summary and
printed by `roadbook journey`.

The competitive study's proposal P2 has two halves. This phase is the
first only. The second, deriving home from nights when Google's home
label is missing, is a detection change with its own pins and is not in
this phase (§5).

---

## 1. Concepts this phase introduces

### A night, as a window on the trip's own clock

A stop already has a start and an end, and a trip already has civil days
(`Summary.CivilDays`: the calendar dates the trip touches, each timestamp
read in its own recorded offset). A trip that touches N civil days has
N − 1 nights between them. That count needs nothing new.

What is new is asking where each of those nights was spent. A night here
is a fixed window on the local clock: from `NIGHT_START` (22:00) on one
civil date to `NIGHT_END` (06:00) on the next. The pass looks at what the
journey holds inside that window and puts the night in one of three
classes:

- **at a stop** — one stop covers at least `MIN_NIGHT_HOURS` of the
  window. If several do, the one with the most hours holds the night.
- **on the move** — no stop does, and recorded travel (observed legs)
  covers at least `MIN_NIGHT_HOURS`. An overnight bus or a night drive.
- **not known** — neither. The phone recorded too little that night to
  say.

The third class is not a failure to hide. It is the same honesty the
route already has: a night nobody recorded is counted and said to be
unrecorded, the way a gap is drawn dashed.

*Why a window and not a single instant.* The simplest rule would be
"where were you at 03:00". It fails at boundaries: a stop that ends at
02:50 because the phone logged a short walk loses the night to a
ten-minute activity. An overlap of hours is stable against that.

*Why not midnight.* "Which stop contains 00:00" calls a midnight arrival
after an evening drive a night at the destination (correct) but also
calls a 23:30–00:30 fuel stop a night (wrong). The minimum-hours rule
removes the second.

### Where this sits in the code

A new pure function in `internal/journey`, beside `Summarize` and
`NameStops`, taking an assembled `Journey` and a `NightParams`:

```go
func Nights(j Journey, p NightParams) []Night
```

It runs after `journey.Assemble` and does not change it, so both journey
goldens and every detection pin stay byte-identical by construction
(invariant 1: no I/O, no clock; invariant 3: every threshold named and
served with the result). A `Night` carries the evening's civil date, its
class, the holding stop's index when there is one, and the hours that
decided it. It carries no name: the stop already has its town from phase
18, and the page joins the two.

### The clock for a night

Every timestamp in a journey keeps the offset it was recorded in. A night
window needs one offset. The rule proposed: the offset of the last
recorded timestamp at or before the window opens (a leg point or a stop
edge), and the window start's own offset when nothing precedes it. For a
trip inside one time zone this is simply that zone. For a trip that flies
across zones, each night is read on the clock the traveller was on when
it began.

---

## 2. What was measured

`docs/phase-19/measure/nights_test.go.txt` (a throwaway test, kept out of
the build by its suffix; the header has the three commands). It runs
detection and assembly on the committed demo and on the private datasets,
and prints counts only.

### 2.1 How the nights fall, at a three-hour minimum

| dataset | trips | nights | at one stop | on the move | a short stop only | nothing recorded |
|---|---|---|---|---|---|---|
| demo | 3 | 6 | 6 | 0 | 0 | 0 |
| Android fixture | 18 | 123 | 91 (74 %) | 22 | 6 | 4 |
| maintainer's archive | 66 | 568 | 444 (78 %) | 45 | 29 | 43 |
| tester's iPhone export | 60 | 259 | 191 (74 %) | 9 | 36 | 22 |

*Added at STOP 1:* the pass as built counts a trip's civil days the way
the summary does, which includes stops that reach past the window's
edges; the script above read the window's edges only. On the iPhone
export that is 264 nights, 193 at a stop, 9 on the move, 62 not known;
the other datasets are unchanged. The built figures are the ones pinned
in `internal/journey/nights_private_test.go`.

"A short stop only" and "nothing recorded" are both the "not known"
class; they are split here to show why. Three of every four nights can be
placed at a stop. Overnight travel is the next largest class: 8 % of the
archive's nights and 18 % of the fixture's.

Of the archive's 444 placed nights, 343 are covered by one stop for the
whole eight hours.

### 2.2 The minimum is not a sensitive choice on Android; it is on iPhone

Nights placed at one stop, by `MIN_NIGHT_HOURS`:

| dataset | 2 h | 3 h | 4 h |
|---|---|---|---|
| Android fixture | 92 | 91 | 88 |
| maintainer's archive | 451 | 444 | 434 |
| tester's iPhone export | 209 | 191 | 174 |

The archive moves by 17 nights of 568 across the range. The iPhone export
moves by 35 of 259: its stops are shorter, because that export breaks a
stay into more pieces.

### 2.3 Bridging a silence is not needed

The study's proposal bridges a night across a silence of up to 16 hours
when the same place lies on both sides. Measured as "no stop time in the
window, but the stops before and after are within 1 km":

| dataset | nights it would add |
|---|---|
| Android fixture | 0 |
| maintainer's archive | 7 of 568 |
| tester's iPhone export | 0 |

A second variant, several short stops at one place adding up to the
minimum, adds 1 night on the iPhone export and none elsewhere. A halt
between two activities is already one stop however long the silence
inside it, which is why a rule for bridging has almost nothing left to
do.

### 2.4 Nights at a stop with no recorded position

| dataset | placed nights whose stop has no position of its own |
|---|---|
| demo | 6 of 6 |
| Android fixture | 3 of 91 |
| maintainer's archive | 18 of 444 |
| tester's iPhone export | 16 of 191 |

These are the stops phase 18 names from Google's record of the visit. The
demo's nights are all of this kind, so the demo exercises that path
fully. Nothing new is needed: a night's place is its stop's place,
whatever position named it.

### 2.5 How many places a trip's nights have

Per trip with at least one night, the median number of distinct stops
holding a night is 2 or 3 on every dataset (90th percentile 8 to 15).
Consecutive nights in one town are often different stops, so the page
groups nights by the town's name, not by stop: "2 nights near Trimbak".

---

## 3. What gets built

### Go

- **`internal/journey/nights.go`** — `NightParams{StartHour, EndHour,
  MinHours}`, `DefaultNightParams()` (22, 6, 3), `Night{Date, Kind,
  StopIndex, Hours}`, `Nights(j, p)`. Table tests on hand-built journeys
  for each class and each boundary (a stop ending inside the window, two
  stops in one window, a zone change), and the demo's six nights pinned.
- **`api/openapi.yaml`** — `Journey.nights` (required array of `Night`:
  `date`, `kind` as `stop | moving | unknown`, `stop_index` when the kind
  is `stop`, `hours`) and `Journey.night_params`. Regenerated Go and
  TypeScript. The shared view receives the same journey and so the same
  nights.
- **`roadbook journey -candidate N`** prints a nights block: one line per
  night with its class, hours, and the stop's town when it has one
  (invariant 13: the page states nothing the command cannot print).
- **A private-data count test**, like phase 18's: the table in §2.1
  pinned for the three private datasets, skipped when `data/` is absent,
  asserting counts only.

No migration, no new table, no change to detection, assembly, routing or
the score.

### Web

- **The trip block** ("The trip", the summary at the head of the page)
  gains a "Nights" row: the count, then where — "3 · 2 near Trimbak, 1 on
  the move". A pure helper in `lib/` groups nights by their stop's place
  phrase and words the three classes; vitest cases; the demo's lines
  pinned.
- **The day narrative.** The stay row that holds a night says so. Today it
  reads "Stayed · through to Day 3 · near Höfn"; it would read "Stayed the
  night · near Höfn". A night on the move puts "travelling through the
  night" on the day line where "overnight into Day N" is today. Exact
  wording is settled on real pages at the web stop.
- Playwright on the demo at 390, 768 and 1280: the Nights row is present
  and no page scrolls sideways.

The exported image and the trip covers on the Trips page are unchanged in
this phase.

---

## 4. The choices

### A. What decides a night — recommended: the window with a minimum

The three candidates are in §1: hours of overlap with a 22:00–06:00
window (recommended), the stop at 03:00, the stop at midnight. The
window rule is stable at a stop's edges, and the same overlap measured
against recorded travel gives the "on the move" class, which is 8 % of
the archive's nights and 18 % of the fixture's.

### B. The minimum hours — recommended: 3

Section 2.2. Three hours is the study's figure and sits in the flat part
of the range on Android. On the iPhone export every value is a
compromise; 2 hours places 18 more nights there, at the cost of calling
a two-hour late dinner stop a night when nothing longer was recorded.
It is a named parameter, served with the journey, so it can move on
tester evidence without a code change.

### C. Nights that cannot be placed — recommended: count them and say so

A trip of five days has four nights whether or not the phone recorded
them. Two ways to word a trip with one unrecorded night:

- "4 nights: 2 near Trimbak, 1 on the move, 1 not recorded" (recommended)
- "3 nights: 2 near Trimbak, 1 on the move" (only the placed ones)

The second reads cleaner and is wrong about the trip. The first keeps the
count equal to days minus one, which a reader can check against the page.

### D. Bridging a silence — recommended: do not build it

Section 2.3: 7 nights of 568 on the archive, none elsewhere. It would add
a distance parameter and a time parameter to explain 1 % of nights.

### E. The score — recommended: leave it alone

The study suggests letting the score's destination-dwell component see
nights. That changes the order of "Trips to review" for every user and
moves the score pins, for a ranking nobody has complained about. If
nights prove a better signal than dwell hours, that is its own change
with its own before-and-after table.

### F. Who words the sentence — recommended: the page

The API serves each night's class, hours and stop. The page groups by the
stop's place phrase and writes the line, as it does for regions and
towns (phase 18's decision: the API serves facts, wording belongs to the
page). The command prints the same facts as a list.

---

## 5. What is not in this phase

- **Home from nights** (the study's P2b): deriving home bases from where
  nights were spent when `INFERRED_HOME` is absent. A detection change;
  needs its own brief and pins.
- **The score** (choice E).
- **Bridging** (choice D).
- **Nights on the exported image and the covers.**
- **Any new place lookup.** A night's place is its stop's place from
  phase 18. Villages (#13) and the coast rule stay where they are.

---

## 6. Checkpoints

- **STOP 0 — this brief.** The maintainer reads it and decides A to F.
  Then: branch `phase-19`, milestone, and #14 split if the two halves
  want separate issues.
- **STOP 1 — Go.** `Nights`, the contract, the command's block, the
  tests. The maintainer reads the nights block for one or two of their
  own trips, the way phase 18's names were checked.
- **STOP 2 — web.** The Nights row and the narrative wording, on the
  demo.
- **Close.** LOG.md, PLAN.md, the PR, the host rebuild (no migration).
