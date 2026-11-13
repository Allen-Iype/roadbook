# Phase 20 — log: far trips without a long stop

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

Detection has a second way to find a trip (#8).

Until now a time away from home was a candidate only when the export
held a visit of at least `MIN_DWELL_MIN` (60) minutes beyond `FAR_KM`
(100). A traveller who was far from home for hours with no such visit in
the file was missed: on a tester's export, 2 of the 28 trips Google
itself grouped at 100 km or more.

- **The rule** (`reachedDest`, `internal/detect/detect.go`). A span with
  no dwelt destination is still a candidate when it went out from one
  home and came back to the same one, and was continuously beyond
  `FAR_KM` for at least `MIN_FAR_MIN` (30) minutes. It is tried only
  when the dwell rule fails, so no existing candidate can be lost or
  have its destination changed.
- **Same home, asked of a chain.** A time near home shorter than
  `MIN_HOME_MIN` (30 minutes) does not count as having come home; the
  test looks through it to the neighbouring span and compares the
  observations either side of the whole chain. Spans are not joined.
- **The destination** is the farthest visit beyond `FAR_KM` of any
  length; failing that, the farthest observation in a far run that
  meets the minimum.
- **The mark.** Such a candidate carries `reached` (migration 17, the
  contract's `Candidate.reached`). `roadbook detect` prints the column
  and a summary line; "Trips to review" ends the card's facts with "no
  long stop there", which opens a note, and "How these N were found"
  names both ways when the run recorded the new parameter.
- **The reference detector** has the same rule and takes its output path
  as a second argument. The Go detector matches it field for field,
  `reached` included.

By checkpoint:

- **STOP 0 — brief**, with the measurement. Recommendations A to F
  accepted.
- **STOP 1 — the detectors** (ed416c6), with the brief and decisions.
- **The correction** (a9e7ccd): the chain. See "What broke", item 1.
- **STOP 2 — web** (8b4fb04).

## What was measured

`measure/measure.py`, counts only:

| dataset | candidates before | now | reached | Google's groupings at 100 km or more, found |
|---|---|---|---|---|
| demo | 3 | 3 | 0 | none in the file |
| Android fixture | 18 | 18 | 0 | none in the file |
| maintainer's archive | 66 | 67 | 1 | 38 → 39 of 45 |
| tester's iPhone export | 60 | 79 | 19 | 26 → 28 of 28 |

Three things the measurement settled before any code.

- "Far enough, or a flight", the rule the issue proposed, adds nothing
  on any dataset: neither missed trip has a flying activity in the file.
- Lowering `MIN_DWELL_MIN` cannot find a trip with no far visit at all,
  and on the archive most of what it adds are journeys between two
  homes: bug 3 again.
- Where a span starts and ends separates the two cleanly. The count is
  flat for `MIN_FAR_MIN` from 10 to 45 minutes, and for `MIN_HOME_MIN`
  from 15 to 120.

The archive's 6 groupings still not found are not misses: 4 are stays at
another home, and 2 overlap no time away beyond 100 km.

## What was verified

- **`make test` with the database suites.** The fixture's 18 candidates,
  the demo's 3, the photo corpus and both journey goldens are unchanged.
- **Go against the reference detector, field for field**, on the
  fixture, the archive and the iPhone export, first against freshly
  generated output through `ROADBOOK_EXPECTED_DIR` and then against the
  pinned files once replaced.
- **Old reference output against new** (`measure/diff_expected.py`): no
  candidate removed, the order of existing ones kept, every added one
  marked reached. On existing candidates only `repeat` moves, because a
  new trip to the same place makes an old one a repeat.
- **The rule's edges, as table tests** (`TestReached`, ten cases): no
  visit at all; a brief visit ahead of a farther point passed; under the
  minimum, and the same data with the minimum lowered; one stray far
  position; a span cut by the end of the file; a journey between homes
  with a short stop in the corridor; the same cut by a stray pair near
  the first home, and with `MIN_HOME_MIN` lowered; an outing cut in two.
- **The mark through the store** (`TestReachedRoundTrips`).
- **Private counts, never printed** (needles, nights, stop names on the
  archive and the iPhone export): re-pinned to the new trip totals.
- **Web:** tsc clean, lint 0 errors, vitest 159. Playwright was not run:
  it reads the demo, which has no reached trip.
- **Read in a browser on the archive**, from a local review database,
  while the web stop was built. That is where item 1 below was found.

## What broke, and the shape of each fix

1. **The rule as first built called a journey between two homes a
   trip.** The archive gained two candidates at STOP 1, and one was a
   journey from one home to another. Partway along, the export holds two
   positions back near the first home, minutes apart; outlier rejection
   keeps them because each is slow relative to the other. They ended the
   span, and its first half read as out and back. The fix widens the
   question, not the span: the same-home test looks through a short time
   near home. Joining the spans would have moved the dates of existing
   candidates; catching pairs in outlier rejection would have been a
   second change to every dataset. The brief had named stray home
   positions as out of scope because they split outings; they also split
   transits. The span was flagged as doubtful at STOP 1 and should have
   been read before the commit, not after.
2. **A stated range was wrong.** The first measurement of the fix said
   the count was flat for `MIN_HOME_MIN` from 30 to 180. It looked only
   at candidates the fix could remove. The full sweep, now in
   `measure.py`, shows the iPhone count rising from 180 as longer chains
   end where they began. The documents carry 15 to 120.
3. **An existing test was issue #8.** "Passing through without dwelling
   is not an adventure (bug 3)" was a round trip from the only home with
   a 30-minute visit. It now expects one reached candidate under a name
   that says so; bug 3 keeps its own case and gained two.
4. **Three figures in the brief were corrected** as they were built:
   stray positions on the iPhone export (two, not three), the example
   trip of more than 900 km (a leg of a chain between homes, withdrawn
   as evidence), and the checkpoint counts (§7).
5. **Code came before tests at STOP 1**, against the brief's own order.
   Reported at the stop; the tests exist and cover the brief's cases.
6. **The branch was first cut from the wrong commit**, a docs branch
   that happened to be checked out. Re-pointed at main before any
   commit.

## Carried out of the phase

- **An outing cut in two by a stray position near home is two trips.**
  `TestReached` records it as current behaviour. It is the detection
  side of phase 17's needles; no issue yet.
- **A far detour inside a journey between two homes is not found.** The
  three spans the iPhone export lost to the correction are that case.
- **The tester's spans run long** because of a home Google never
  labelled: home from nights (#23).
- **The landing page** still says a trip needs "somewhere you stayed".
- **Waiting on the tester:** whether the 19 added trips are ones they
  recognise, and whether the two named in #8 now appear.
- **Host deploy after the merge:** app and demo stacks rebuilt;
  migration 17 applies at start-up; existing candidates read
  `reached = false` until detection runs again on that account.
