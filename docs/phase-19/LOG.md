# Phase 19 — log: nights

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

A trip now says how many nights it had and where each was spent.

- **#14 — the pass, the contract, the command.** `journey.Nights`
  (`internal/journey/nights.go`) is a pure pass over an assembled journey.
  For every pair of consecutive civil dates the trip touches it opens a
  window from 22:00 to 06:00 on the traveller's own clock and puts the
  night in one of three classes: at a stop (the stop covering the most of
  the window, if at least three hours), on the move (recorded travel
  covering three hours), or not known. The three figures are
  `NightParams`, served as `Journey.night_params` beside
  `Journey.nights`. `roadbook journey -candidate N` prints one line per
  night with its class, its hours and the stop's town. Detection, journey
  assembly, routing and the score are unchanged; there is no migration.
- **#22 — the trip page.** The trip block has a "Nights" row: the count,
  then where, grouped by the town of the stop that held each night
  (`web/lib/nights.ts`). In the day narrative the stay that held a night
  says so, and a night on the move or a night the record cannot place is
  said on its day's line.

A night has no place of its own. One spent at a stop points at the stop,
which has carried its town since phase 18.

By checkpoint:

- **STOP 0 — brief**, with the measurement that shaped it. All six
  recommendations accepted.
- **STOP 1 — Go** (4b276a1), with the brief and decisions.
- **STOP 2 — web** (fcad9fd).

## What was measured before anything was built

`measure/nights_test.go.txt`, counts only, at the three-hour minimum:

| dataset | nights | at one stop | on the move | not known |
|---|---|---|---|---|
| demo | 6 | 6 | 0 | 0 |
| Android fixture | 123 | 91 | 22 | 10 |
| maintainer's archive | 568 | 444 | 45 | 79 |
| tester's iPhone export, as built | 264 | 193 | 9 | 62 |

Two things the measurement settled. The competitive study's rule for
bridging a silent night when the same place lies either side would have
placed 7 more nights of 568 on the archive and none elsewhere, so it was
not built. And the minimum barely matters on Android (434 to 451 nights
placed on the archive between four hours and two) but does on the iPhone
export (174 to 209), which is why it is a served parameter.

## What was verified

- **`make test` with the database suites, and `make vet`.** The detection
  pins and both journey goldens are unchanged.
- **The rule's edges, as table tests** (`nights_test.go`): a day trip, a
  late arrival, exactly the minimum and a minute under, two stops in one
  window, a half-hour halt across midnight, an overnight bus, a
  stationary recorded run, a stop against travel, a stay across three
  days, the journey's own offset, a change of offset mid-trip. Every case
  also checks that nights equal civil days minus one.
- **The demo's six nights, through the API** (`TestDemoNights`), with the
  contract's shape: a stop index exactly on a night at a stop, hours
  absent on an unknown one, an empty list on the wire as `[]`.
- **Private data, counted and never printed**
  (`nights_private_test.go`): the table above.
- **Web:** tsc clean, lint 0 errors, vitest 159 (the demo's three Nights
  rows pinned), Playwright 139 passed and 26 skipped at 390, 768 and
  1280 on the demo.
- **Read by the maintainer** on two of their own trips in a browser, from
  a local review database: a three-day trip and the longest in the
  archive (67 nights: 63 at a stop, 1 on the move, 3 not known, the same
  figures the command prints).

## What broke, and the shape of each fix

1. **The brief's iPhone figure was five nights short.** The measurement
   script walked the dates between the window's two edges. The summary's
   civil-day rule also reads stops that reach past the window, and the
   pass was built on that rule so the two could never disagree
   (`civilRange`, shared). The built figure is the one pinned; the brief
   carries a dated note beside its table.
2. **A night's date came out of the generator as a time value.** Declared
   `format: date`, the Go generator made it a date type with a zone to
   get wrong on the way out. It is a plain "YYYY-MM-DD" string, the form
   the web's day slicing already uses.
3. **"2 13 km from Barshi".** The first wording of the Nights row put
   each group's number in front, which collides with a far town's
   distance. The number now follows in brackets, and only when it is more
   than one.
4. **The same night was said twice.** The day line already said
   "overnight into Day 2" whenever some stretch crossed midnight, which
   is usually the gap around an overnight stay. With "Stayed the night"
   on the row below, the line's phrase now appears only on a day that has
   no night of its own to describe.
5. **A development warning on every kept trip, seen only in Firefox.**
   At the web stop the maintainer's Firefox-based browser showed React's
   missing-key warning. It was first put down, wrongly, to stale
   dev-server state, because Chromium showed it once and then not again.
   In a Firefox engine it reproduced on every load, and it reproduced
   without this phase's changes: the trip page hands three server-made
   elements to the client view, and none had a key. They have keys now.
   The lesson is in the next section.

## Carried out of the phase

- **The browser tests run one engine and do not read the console.** That
  is how item 5 lived since phase 17. A Firefox project, or a failure on
  console errors, is a small change of its own.
- **Seen, not chased:** on the first Firefox load after a dev-server
  start, the dev server's own hot-reload chunk failed to load
  (`ChunkLoadError`, the HMR client). The second page loaded clean. It is
  the development server's, not the app's, and a built app has no such
  chunk.
- **Home from nights (#23)** — backlog; a detection change with its own
  brief.
- **Still waiting on testers, from phase 18:** villages (#13), the coast
  rule, the wording distances, the blank stay row. From this phase: the
  three-hour minimum on iPhone exports, and whether three named towns are
  enough on a long trip.
- **Host deploy after the merge:** app and demo stacks rebuilt; no
  migration, no data step; checked from outside.
