# Phase 23 — Recorded stretches drawn along the road — brief

Issue #9. Written before any code, and measured before it was designed.
Figures come from two test files under `docs/phase-23/measure/` that
print counts, distances and ratios only; how to run them is in each
file's header.

The charter decision this needed was made on 2026-11-16: a recorded
stretch may be drawn along the road its points lie on and stays
"Observed"; gaps stay their own kind; recorded points are never altered.
There is no third line kind. The wording of invariant 6 in `CLAUDE.md`
is the maintainer's to change, and this brief assumes it will read
roughly: *recorded points are never altered or discarded; a recorded
stretch may be drawn along the road its points lie on, and where no
road fits it is drawn point to point; gaps remain their own kind.*

---

## 1. Concepts

### What is drawn today

A journey is a list of legs (`internal/journey`). A **recorded stretch**
(`LegObserved`) is a run of points with no silence longer than 20
minutes between them; it is drawn as straight lines from point to
point. A **gap** (`LegGap`) is such a silence; the batch command
`roadbook route` asks a router for the road between its two ends and
caches the answer, and the map draws that in the "Routed" ink.

### Routing and matching are different questions

*Routing* asks: what is the best road from A to B? It knows nothing
between them. *Map matching* asks: here is a sequence of noisy
positions with times; which roads was this most likely travelling
along? It weighs every point against its neighbours, so one position
that sits on the wrong side road does not drag the line there and back.
OSRM answers both (`/route` and `/match`) from the same prepared data.

For a recorded stretch, matching is the right question. The measurement
below confirms it.

### Why it needs a guard

Positions in a Timeline export are mostly WiFi and cell, not GPS. A
matcher handed a position 300 m off the road will sometimes take a side
road to reach it. The result is a line much longer than the points
justify, and on a map it looks wrong at a glance. A matched line is
therefore kept only when it is not much longer than point to point.

### Where it sits

Matching is inference, so it stays out of `Assemble`, exactly as gap
routing does: `Assemble` is pure and its goldens pin it. The matched
geometry is fetched by the batch command, cached in the database, and
laid alongside a leg's points by `route.Apply`. The server never calls
a router.

---

## 2. What was measured

### 2.1 How far apart recorded points are (`hops_test.go.txt`)

| | demo | Android fixture | archive | iPhone export |
|---|---|---|---|---|
| trips | 3 | 18 | 67 | 77 |
| recorded stretches | 18 | 1,317 | 4,852 | 2,260 |
| gaps | 15 | 1,299 | 4,785 | 2,183 |
| recorded km · gap km | 752 · 773 | 22,205 · 9,066 | 48,839 · 44,003 | 30,268 · 58,943 |
| points per stretch, median · p90 | 3 · 9 | 4 · 22 | 2 · 14 | 2 · 26 |
| hop length, median · p90 | 9.3 · 18 km | 1.1 · 6.5 km | 0.9 · 5.4 km | 0.7 · 2.5 km |
| share of recorded km in hops over 5 km | 100% | 56% | 50% | 44% |

Three things follow.

- By count, points are close together. By distance, about half of what
  is drawn as recorded lies in hops of more than 5 km. That is where a
  straight line most visibly leaves the road, and where snapping helps
  most.
- A trip alternates constantly between recorded and gap: the archive's
  67 trips hold 4,852 stretches and 4,785 gaps. The typical stretch is
  two points.
- The iPhone export has 7 hops of more than 50 km *inside* a recorded
  stretch, 4,879 km between them. Nothing here should be matched; the
  guard in §2.2 excludes them, and they are a finding for the drawing
  pipeline of their own.

### 2.2 What matching makes of the fixture (`match_test.go.txt`)

Local OSRM, India data, driving profile, the 851 stretches of the
fixture that have two or more points and at least 200 m.

| search radius | matched as one piece | no match | points not placed | road km ÷ point-to-point km |
|---|---|---|---|---|
| 50 m | 820 | 26 | 516 of 10,316 | 1.22 |
| 100 m | 842 | 5 | 171 of 10,316 | 1.21 |
| 250 m | abandoned: single requests ran for minutes | | | |

At 100 m, by how much longer the matched line is than point to point:

| road ÷ point-to-point | stretches |
|---|---|
| under 1.1 | 377 |
| 1.1 to 1.3 | 155 |
| 1.3 to 2 | 175 |
| 2 or more | 135 |

One stretch in six comes back at least twice as long as its points
justify. Those are not short in-town scraps only: of 440 stretches
between 2 and 20 km, 166 come back at least 1.5 times as long.

What a guard keeps (100 m):

| keep the match when road ÷ point-to-point is under | stretches kept | recorded km drawn along roads |
|---|---|---|
| 1.2 | 484 | 76% |
| 1.3 | 532 | 82% |
| 1.5 | 591 | 87% |
| 2.0 | 707 | 94% |

The curve has no flat part. A hill road is honestly 1.4 times its
chord; a wrong side road is too. The number cannot be chosen from this
table alone: see choice B.

**Routing each hop on its own**, the alternative to matching: 1.45
times point to point over the same stretches, against 1.21. Worse, as
§1 predicts.

**The matcher's own confidence figure** is near zero throughout (median
0.04). It is not usable as the guard.

### 2.3 Not measured yet

- The archive and the iPhone export through the matcher. The fixture
  run took several minutes; the archive is four times the stretches.
- Agreement with Google's own distances. Each activity in the export
  carries a distance, and the trip page already warns when the drawn
  distance diverges from it by more than 15%. That is the one outside
  reference for "is the matched line the right length", and STOP 1
  reports it.
- Anything outside India: the demo's trips are in Iceland and that
  extract exists locally.

---

## 3. What gets built

### Go

- **`internal/route`**: a `Matcher` interface beside `Router` (one
  method: a sequence of timed points in, a road geometry and its length
  out, or "no match"). Implementations: OSRM `/match`, and a null one
  that matches nothing. Invariant 7's shape, unchanged: pluggable, and
  the product degrades to today's drawing.
- **A cache table** (migration 18, `match_cache`): keyed by a hash of
  the stretch's own points, the profile and the search radius; holds
  the status (matched / no match), the geometry and the length. The
  guard is applied when reading, not when writing, so changing the
  ratio needs no new requests.
- **`roadbook route`** also matches: for each trip, every recorded
  stretch not yet in the cache. Flags for the radius and for switching
  matching off. A run records what it did, as routing runs do.
- **`route.Apply`**: a recorded leg whose cached match passes the guard
  gains `SnappedPoints`, alongside its own points. Kind unchanged.
  Pure, as today.
- **Named parameters**: `MATCH_RADIUS_M` (100), `MATCH_MAX_RATIO`,
  `MATCH_MIN_M` (200), served with the journey like the others.

### Contract

`Leg` gains an optional snapped geometry; `Journey` gains the
parameters and the count of recorded km drawn along roads. Generated on
both sides.

### Web

A recorded leg with a snapped geometry is drawn along it, in the same
ink and style as any recorded leg. The exported image does the same.
The legend's words do not change. "How this route was drawn" gains one
line: how much of the recorded distance follows roads and how much is
point to point.

### Documents

`DESIGN.md` §6 gets a dated note; `landing/how-its-built.html` says
"recorded stretches are drawn along the road where one fits"; the
README's walkthrough likewise.

### Not changed

`Assemble`, detection, the score, nights, places, every figure computed
from recorded points.

---

## 4. The choices

### A. What is matched — recommended: each recorded stretch as a whole

Against routing hop by hop (measured worse, and four times the
requests), and against matching a whole trip in one request (a trip is
half gaps; the matcher would be asked to bridge silences it has no
evidence for, which is what gap routing is for).

### B. The guard — recommended: 1.3 as the default, decided by eye at STOP 2

82% of the fixture's recorded distance at 1.3; 87% at 1.5. Because the
guard is applied on reading, the maintainer can look at the same trips
at both values on a scratch stack before it is fixed. The parameter is
served, so a trip page can always say what was used.

### C. The search radius — recommended: 100 m

Fewer failures and fewer unplaced points than 50 m, the same overall
length, and 250 m is unusable.

### D. Distances — recommended: keep them from the recorded points

The trip's kilometres, pace and the divergence warning stay as they
are. The matched length is reported beside them in "How this route was
drawn" and nowhere else. Changing the headline distance to the road
length is a separate decision with its own consequences (every
confirmed trip's number moves).

### E. Who gets it — recommended: wherever the routing batch runs, and say so

This is the choice that matters most. There is no router on the hosted
instance: hosted users' gaps are drawn unrouted today (phase 15 §3F,
still open). Matching needs the same router. Built as §3 describes, it
reaches a self-hoster with OSRM and the maintainer's own instance, and
not the tester who raised #9.

The options for the hosted instance, none of them in this phase's
build:

1. **Leave it** as gap routing is left. Honest, and the feature is
   invisible to hosted users.
2. **The public OSRM server.** It is a demonstration service with a
   fair-use limit, a trip is hundreds of requests, and it would send
   users' positions to a third party, where the privacy page says a
   user's history stays in their own account.
3. **OSRM on the host, India only.** The prepared India data is 19 GB
   on disk; the host has 80 GB free and 8 GB of memory, 6 available.
   Whether it fits in memory is not measured. Trips outside India would
   stay as today.

Recommended: build §3 against the reference deployment, and make option
3 its own checkpoint at the end, measured on the host before anything
is promised. It would switch on gap routing for hosted users at the
same stroke, which closes §3F.

---

## 5. Not in this phase

- A third line kind (decided against).
- Changing the headline distance (choice D).
- The 50 km hops inside recorded stretches on the iPhone export (§2.1).
- Walking, cycling or train profiles: driving only.
- Villages (#13), home from nights (#23).

## 6. Checkpoints

- **STOP 0 — this brief.** A to E. And the `CLAUDE.md` edit to
  invariant 6, which only the maintainer makes.
- **STOP 1 — Go.** Tests first. Matcher, cache, batch, `Apply`,
  contract, the command. Reported with: the archive and the iPhone
  export through the matcher, the demo in Iceland, and matched length
  against Google's activity distances.
- **STOP 2 — web.** The maintainer looks at their own trips on a
  scratch stack at two guard values and fixes B.
- **STOP 3 — the hosted instance.** Option E.3 measured on the host;
  go or no-go.
- **Close.** Log, plan, PR, host rebuild (migration 18), a note on #9.
