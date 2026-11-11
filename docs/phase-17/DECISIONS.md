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

## 2026-11-04 — #4: one decoder for both shapes; the root token chooses

- **Chosen:** `Parse` reads the first token: `{` is the Android object,
  `[` the iPhone array. Both feed the same segment decoder, widened to read
  numbers written as strings (`flexFloat`), `geo:` positions, and path
  points timed by `durationMinutesOffsetFromStartTime` (resolved against
  `startTime` in its own zone; negative or past `endTime` skipped and
  counted). Success records `phone-timeline-ios`.
- **Rejected:** a second segment struct and loop for iPhone — the shapes
  share every field Roadbook reads; two copies would drift.
- **Would change our mind:** Google diverging the two shapes in a field
  Roadbook reads, beyond what a decoder type can absorb.

## 2026-11-04 — #4: Google's spellings normalised in the parser, for both shapes

- **Chosen:** visit kinds and activity modes go through one helper (words
  joined by `_`, upper-cased): `Inferred Home` → `INFERRED_HOME`,
  `in passenger vehicle` → `IN_PASSENGER_VEHICLE`. Idempotent on Android
  input, so the Android fixtures and goldens are unchanged. `Home` (a
  place the person labelled) becomes `HOME`, which detection does not use
  as a base signal today — unchanged on purpose.
- **Rejected:** teaching detection both spellings (the pure core would
  carry a source's quirk; invariant 4).
- **Would change our mind:** a third spelling that this rule mangles —
  then a table, still in the parser.

## 2026-11-04 — #4: verified against the reference detector, independently

- **Chosen:** the tester's file is converted to the Android shape by a
  separate Python script (not committed: its output holds real
  coordinates; it lives only in the session and its output in `data/`),
  the reference detector runs on that (a copy writing
  `data/ios-candidates.json`, so `data/fixture-candidates.json` is never
  touched), and `TestIOSRegression` compares the Go iPhone path to it
  field for field. Result: 4 home bases and 60 candidates, all fields
  equal. Committed tests: an inline iPhone sample, and the demo written in
  both shapes (`testdata/demo/demo-ios.json`, same generator; visits and
  activities identical, points within 30 s, same three trips).
- **Rejected:** pinning numbers from the Go run alone (that tests the code
  against itself).
- **Would change our mind:** nothing.

## 2026-11-10 — #4: needles on iPhone maps fixed inside phase 17 (STOP 1b)

- **Chosen:** a second rejection pass in journey assembly, by shape: a
  cluster more than 20 km from both neighbours, the neighbours within a
  quarter of the shorter hop of each other, at most 120 minutes apart.
  Named parameters, a counter, rows untouched. On the trips: tester 67 → 4
  needles, Android fixture 0 → 0 with nothing set aside, archive 3 → 0
  (`internal/journey/needles_test.go`). Inside phase 17 because #4 is not
  fixed while an iPhone user's maps look broken.
- **Rejected:** lowering the 900 km/h teleport limit (needles run 24–471
  km/h); a list of recurring bad places (recurrence does not separate
  them); filtering in the parser (inputs are passed through as they are)
  or in detection (needles are under 100 km; detection stays with the
  reference); a 60-minute window (13 needles left) or 180 (took hour-long
  clusters from the archive).
- **Would change our mind:** a real trip missing a stretch it recorded —
  then the window shortens, or the pass asks for a single-point cluster.

## 2026-11-10 — Snapping to roads is its own phase

- **Chosen:** the maintainer asked whether recorded points could be
  matched to OSM roads so routes follow the road. Not in phase 17: a PLAN.md
  backlog entry, "Snapped lines", to be briefed on its own.
- **Rejected:** doing it now. Matching cannot remove a needle (it draws it
  along real roads), so the filter comes first either way; and drawing
  recorded stretches along computed roads changes invariant 6 ("observed
  legs are never routed"), which needs its own decision and its own line
  kind, visibly distinct from both recorded and routed.
- **Would change our mind:** nothing in this phase.

## 2026-11-10 — #1: newest first on Trips to review

- **Chosen:** the review list's default order is newest first
  (`span_start` descending, a sorted copy, the same rule as the Trips
  shelf). The toggle reads "newest first · by score"; score order is
  unchanged. The Go API's order (`seq`, oldest first) is untouched: the
  order is a presentation choice, made where the page is rendered, as the
  Trips shelf already does.
- **Rejected:** changing the API's `ORDER BY` (a change to the contract
  every client and Go test reads, for what the issue describes as a page's
  order); an "oldest first" third option (nobody asked).
- **Would change our mind:** a tester working through a long queue
  wanting the old order back; then it becomes a third toggle link.

## 2026-11-10 — #2: the plate numeral removed everywhere

- **Chosen:** "PLATE n" / "PLATE II" leaves the covers, the trip page's
  eyebrow and map margin, the exported image and overlay, and the
  landing's sample covers (with its `.plate-no` style in all three landing
  pages). The order it encoded is read from the dates; the e2e order checks
  read a `data-start` hook. `roman()` and the "plate" teaching term went
  with their last users. DESIGN §6 and words.md decision 1 amended, dated.
- **Rejected:** keeping the numeral and teaching it better (the copy pass
  set this exact trigger for removal); keeping it only on the exported
  image (one meaning per word: a number nobody can decode is worse on a
  picture with nothing to tap).
- **Would change our mind:** testers asking for a way to refer to a trip
  by number; then a plain "Trip 12", not a roman numeral.

## 2026-11-10 — #3: limits, and the name limit on every write

- **Chosen:** the proposed limits, agreed by the maintainer: a name of at
  most 120 characters, notes of at most 4,000, counted in characters (Go
  counts runes; the browser counts code points; the spec's `maxLength`
  says the same). The 120 applies to every write of a name — the new edit,
  the single confirm and the bulk confirm — so a name can never be saved
  that the edit form then refuses to save unchanged. Notes keep their line
  breaks and inner spacing; trailing space goes; blank notes clear them.
- **Rejected:** a name limit only on the edit (two rules for one field);
  counting bytes (an Icelandic or Malayalam name would hit the limit at
  half the length of an English one).
- **Would change our mind:** a real name over 120 characters, or notes
  that want more than a few paragraphs — then the limit rises, and the
  candidate list stops carrying notes.

## 2026-11-10 — #3: where the words sit, and how they read

- **Chosen:** on the trip page's header, under the dates and above the
  distance: the notes set in the display serif (the book's own face, so
  the traveller's words read as part of the trip, not as interface text),
  then one text control, "Edit name or add notes" (or "Edit name and
  notes" once there are some). Editing replaces both in place: the name in
  the heading's face, the notes box, a character count, Save and Cancel;
  Escape cancels. A slot on `AdventureView` (`ownerWords`) that the shared
  view never fills, on top of the API never sending notes there.
- **Rejected:** an edit control on the Trips covers (a cover is one link);
  a rich-text or Markdown editor (plain text was the brief); notes on the
  shared view (owner-only by decision).
- **Would change our mind:** testers not finding the control, or wanting
  notes per day rather than per trip.

## 2026-11-10 — #3: the e2e that writes, and what it taught

- **Chosen:** `web/e2e/edit-trip.spec.ts` edits the second kept trip, at
  the desktop project only (so three projects never race one row, and
  never share.spec's first trip), and puts the name back and clears the
  notes in a `finally`, so a failing run still leaves the stack as it
  found it. Two traps found on the way: `app/loading.tsx`'s own `h1` sits
  hidden in the DOM while a reloaded page streams in (the spec reads
  `article h1`), and a streamed node first arrives in a hidden container
  with zero height (the line-break check polls). Twenty serial repeats
  green before it was kept.
- **Rejected:** a read-only spec for #3 (it would not prove the write
  path end to end); resetting through the API from the spec (the browser
  side has no API access by design).
- **Would change our mind:** e2e ever pointed at a stack holding real
  data — this spec would then need to be skipped there, like share.spec.
