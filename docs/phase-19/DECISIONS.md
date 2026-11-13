# Phase 19 — decisions

Three lines each: what was chosen, what was rejected, what would change
our mind. Written as the decisions were made.

## 2026-11-13 — A night is a 22:00–06:00 window, held by overlap

- **Chosen:** for every pair of consecutive civil dates a journey
  touches, a window from 22:00 to 06:00 on the traveller's own clock. The
  stop covering the most of it holds the night if that is at least three
  hours; failing that, the night is "on the move" when moving observed
  legs cover three hours; failing that, it is not known. All three
  figures are `NightParams`, served with the journey.
- **Rejected:** the stop at 03:00 (a short logged walk at 02:50 would
  take the night from the stay around it); the stop at midnight (a
  half-hour halt across 00:00 would be a night).
- **Would change our mind:** testers reporting a night placed where they
  did not sleep — then the window's hours, which are parameters.

## 2026-11-13 — The minimum is three hours

- **Chosen:** 3. Measured before building (BRIEF §2.2): nights placed at
  a stop on the maintainer's archive are 451 / 444 / 434 at two, three
  and four hours; on a tester's iPhone export 209 / 191 / 174.
- **Rejected:** 2 (places more iPhone nights, and calls a two-hour late
  halt a night when nothing longer was recorded); 4 (drops late
  arrivals).
- **Would change our mind:** iPhone testers seeing too many "not
  recorded" nights for stays they remember — then 2, for everyone, since
  the archive barely moves.

## 2026-11-13 — Nights that cannot be placed are counted and said so

- **Chosen:** a journey of N civil days has N−1 nights, always; the pass
  walks the same date range `Summarize` counts (`civilRange`), so the two
  cannot disagree. A night with too little recorded is its own class.
- **Rejected:** listing only the nights that could be placed (reads
  cleaner, and makes a five-day trip look like it had three nights).
- **Would change our mind:** nothing short of the page reading badly at
  the web stop; then the wording changes, not the count.

## 2026-11-13 — No bridging across a silence

- **Chosen:** not built. Measured (BRIEF §2.3): it would place 7 more
  nights of 568 on the archive and none on the other datasets, because a
  halt between two activities is already one stop however long the
  silence inside it.
- **Rejected:** the competitive study's rule (same place either side of
  up to 16 silent hours), which needs a distance and a time parameter.
- **Would change our mind:** a source whose stops are cut at every
  silence (a GPX or photo-only journey) — measured again there.

## 2026-11-13 — The score is not touched

- **Chosen:** nights are served and shown; the triage score keeps its
  destination-dwell component as it is.
- **Rejected:** letting the score see nights (reorders every user's
  "Trips to review" and moves the score pins, with no complaint about the
  order to answer).
- **Would change our mind:** evidence that nights rank trips better than
  dwell hours — its own change, with a before-and-after table.

## 2026-11-13 — The API serves the facts; the page words them

- **Chosen:** `Journey.nights` carries date, kind, the stop's index and
  the deciding hours. A night has no place of its own: one at a stop
  points at the stop, which has its town since phase 18. The page groups
  and words; `roadbook journey` prints the same facts as lines.
- **Rejected:** a ready-made sentence from Go (wording belongs to the
  page — phase 18's decision); a place copied onto the night (a second
  copy of the stop's name to keep in step).
- **Would change our mind:** a second surface that needs the sentence and
  cannot run the page's helper.

## 2026-11-13 — Gate: STOP 0 passed

- The maintainer accepted all six recommendations. Branch `phase-19`,
  milestone "Phase 19"; #14 scoped to the Go half, #22 the page, #23
  (backlog) home from nights.

## 2026-11-13 — #14: which clock a night is read on

- **Chosen:** the offset of the last recorded timestamp whose own wall
  clock is at or before the window's opening; the journey's first
  timestamp's offset when none is. Inside one time zone it is that zone;
  after a flight, later nights use the new clock (tested).
- **Rejected:** the window start's offset for the whole journey (wrong
  for every night after a change of zone); UTC (a 22:00 window at +05:30
  would open at 03:30 local).
- **Would change our mind:** a journey whose recorded offsets are wrong
  at the source; then the offset comes from position, which needs a time
  zone map the project does not carry.

## 2026-11-13 — #14: a night's date is a plain string in the contract

- **Chosen:** `Night.date` is `type: string`, "YYYY-MM-DD", described in
  words. It is a civil date in the traveller's offset, the same form the
  web's day slicing already uses.
- **Rejected:** `format: date`, which the Go generator turns into a time
  value with a zone to get wrong on the way out.
- **Would change our mind:** a consumer that validates formats strictly
  and wants the annotation.

## 2026-11-13 — #14: what the first run showed

- The demo's six nights are all at a stop for the full eight hours
  (`TestDemoNights`). On the maintainer's archive: 568 nights, 444 at a
  stop, 45 on the move, 79 not known; the 16-month fixture 123 / 91 / 22
  / 10; a tester's iPhone export 264 / 193 / 9 / 62
  (`nights_private_test.go`, counts only). The iPhone total is five more
  than the brief's measurement, which read the window's edges only; the
  brief carries the note.
- One trip the maintainer had already read for place names in phase 18
  shows two nights, each at the stop that review listed as its overnight
  stay.

## 2026-11-13 — #22: how the Nights row is worded

- **Chosen:** the count, then where: nights at a stop grouped by the
  stop's town, most nights first (ties in the order reached), the first
  three towns by name and the remainder as "elsewhere"; then "no town
  nearby", "on the move", "not recorded". A group of more than one night
  carries its number after the words, in brackets — "near Trimbak (2)" —
  and a row with a single group carries none, the count beside it says
  it. The numbers always add up to the count
  (`web/lib/nights.ts`, `nightsSummary`). Absent on a day trip.
- **Rejected:** the number in front ("2 near Trimbak" reads well, "2 13 km
  from Barshi" does not); every town listed (the longest trip measured
  has dozens of night halts; the block is one line); grouping by stop
  (consecutive nights in one town are usually different stops).
- **Would change our mind:** testers wanting the full list of night halts
  — then a list of its own under the days, not a longer row.

## 2026-11-13 — #22: a night is said once, on the row or the line it belongs to

- **Chosen:** the stay that held a night says so on its own row:
  "Stayed the night" for a stay from the evening on (with "through to
  Day N" only when it ends later than the next day), "Stayed all day and
  the night" for a full day inside a stay, and "overnight" after the
  duration for a stay that began after midnight and held the night
  before. Recorded travel through a night is "travelling through the
  night" on the day line, and a night the record cannot place is "night
  not recorded" there. The older "overnight into Day N" on the day line,
  which only knew that some stretch crossed midnight, now appears only on
  a day that has no night of its own to describe.
- **Rejected:** keeping "overnight into Day N" beside "Stayed the night"
  (the same night said twice, in two places, in different words); a moon
  glyph on the row (a new mark the legend would have to teach).
- **Would change our mind:** readers missing which day a "not recorded"
  night belongs to — then it gets a row of its own between the days.

## 2026-11-13 — #22 review: the trip page's three slots carry keys

- **Chosen:** the elements the trip page makes on the server and hands
  to the client view (`PhotosSection`, `EditTrip`, `ShareLinks`) each
  carry a `key`. Found at the web stop: the maintainer's Firefox-based
  browser showed React's "each child in a list should have a unique key"
  twice on every load of a kept trip in development. It predates this
  phase (it reproduces without the nights changes) and Chromium, the only
  engine the browser tests run, does not report it reliably.
- **Rejected:** leaving a development-only warning (it hides the next
  real one); wrapping each slot in a fragment in the view (the same
  effect, further from the cause).
- **Would change our mind:** nothing; noted for the log that the browser
  tests do not read the console, and run one engine.

