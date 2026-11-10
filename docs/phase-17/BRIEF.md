# Phase 17 — Tester fixes: brief

Four issues raised from the first testers, fixed as one small phase on the
branch `phase-17`, one pull request that closes all four:

| issue | what the tester met | size |
|---|---|---|
| #4 | An iPhone Timeline export is refused at upload | parser, Go only |
| #1 | Trips to review lists the oldest trip first | one page |
| #2 | "PLATE 2" / "PLATE II" means nothing to a traveller | web + landing |
| #3 | A kept trip's name can't be changed on its page; no place for your own words | schema, API, web |

Order of work: #4 (it blocks every iPhone user completely), then #1 and #2
together (small, web only), then #3 (the only schema change). A STOP after
each, for review on the scratch stack.

## 1. Concepts this phase introduces

### Two shapes of the same Timeline export

**What.** Google Maps writes the on-device Timeline export in two shapes.
The Android app writes an *object*: `{"semanticSegments": [...],
"rawSignals": [...], ...}`. The iPhone app writes an *array* of the same
segments with no wrapper: `[{...}, {...}]`. Inside, the segment kinds are
the same four (visit, activity, timeline path, memory), but the encoding
differs:

| field | Android | iPhone |
|---|---|---|
| root | object, `semanticSegments` key | array of segments |
| a position | `"12.34°, 56.78°"` | `"geo:12.34,56.78"` |
| `distanceMeters`, probabilities | numbers | numbers written as strings (`"1234.5"`) |
| a path point's time | `"time"`: an RFC 3339 timestamp | `"durationMinutesOffsetFromStartTime"`: minutes after the segment's `startTime`, as a string |
| visit kind | `INFERRED_HOME` | `Inferred Home` |
| activity mode | `IN_PASSENGER_VEHICLE` | `in passenger vehicle` |
| raw signals | `rawSignals` (last ~30 days) | absent |
| place id | `placeId` | `placeID` (unused by Roadbook either way) |

The iPhone column is measured, not assumed: the tester's real file
(private, `data/`) was read for structure only — key paths and value
shapes, never values. 35,733 segments: 8,603 visits, 7,981 activities,
19,094 timeline paths (132,644 points), 55 memories. Every visit carries
`semanticType`, which settles the competitive study's open question
(`docs/research/competitive-study.md`, "whether genuine current iOS exports
always carry `semanticType`").

**How Roadbook reads it today, and why it fails.** `internal/timeline`
`Parse` peeks the head of the file (`sniff`), then insists on `{` as the
first token. An array root fails that check and is reported as
`malformed-json` — "this file can't be read — if it was edited or converted,
start from a fresh export". That message blames the user for a file that is
exactly what Google gave them. (Before the copy pass the same failure read
"not valid JSON (expected "{", got [)" — the tester's screenshot.)

### Normalising at the parser boundary

**What.** The fix lives entirely in `internal/timeline`. The parser learns
the second shape and emits the *same* domain types it emits for Android
(`internal/domain`: `Visit`, `Activity`, `PathPoint`), with the same
spellings: `Inferred Home` becomes `INFERRED_HOME`, `flying` becomes
`FLYING`, a `geo:` string becomes a `LatLng`, an offset of 12 minutes becomes
`startTime + 12m` in `startTime`'s own time zone.

**Why there.** Invariant 4: parsers emit domain types and nothing else, so
adding a source requires zero changes to detection. Detection compares
spellings literally — `internal/detect/bases.go:50` keeps only visits whose
`SemanticType` is `"INFERRED_HOME"`; `internal/journey/journey.go:329`
recognises a flight by `"FLYING"`. If the parser passed `Inferred Home`
through, an iPhone user would get no home base, and every observation would
count as "away" — one endless false trip. Normalising in the parser keeps
that knowledge in the one package that knows Google's spellings.

**Rejected:** teaching detection both spellings (spreads one source's quirk
into the pure core, and every future source would add more); converting the
iPhone file to the Android shape in the browser (business logic in the Next
layer, and a second copy of the parser to keep in step).

### Needles: fallback positions in the iPhone export (added 2026-11-10, STOP 1b)

**What.** Once STOP 1 read the tester's file, its maps showed needles: a
line that runs 20–40 km to one place and straight back. The points are in
the export itself. Over the whole file there are 13.2 out-and-back jumps
per 1,000 path points against 0.3 on the Android fixture; 96 % land on five
recurring places; 829 sit inside a single path segment, where Google's own
offsets only increase, so they are not a timing error in the new parser.
The likely cause is a fallback position (a cell tower or an area centre)
standing in for the phone while it had no proper fix.

**How they reach the map.** Every map — life map, covers, review gallery,
trip page, shared view, downloaded image — is drawn from one assembly,
`journey.Assemble` (`internal/journey/journey.go`, called from
`internal/api/server.go`). Assembly already rejects "teleports": clusters
whose entry or exit implies more than 900 km/h. The needles imply 24–471
km/h (median 131), so they pass.

**The fix.** A second pass over the same clusters, after the teleport
pass: drop a cluster when both hops to it exceed 20 km, its two neighbours
lie within a quarter of the shorter hop of each other, and the neighbours
are at most 120 minutes apart. A real trip out and back leaves a trail of
points, so no single cluster is far from both its neighbours; a needle has
no trail. Three named parameters (`ExcursionMinKm`, `ExcursionReturnRatio`,
`ExcursionMaxMinutes`) with the reasoning in their comment, a counter
(`RejectedExcursion`), rows untouched.

**Why there.** Not in the parser: it must hand the file through as it is
(invariant 2 — inputs are never altered, and rejection is a derived
judgement that may change). Not in detection: the needles are under 100 km
and never a destination, and detection is held field for field to the
reference detector. The map layer is where they do harm, and it already
has the same kind of filter.

**Rejected:** a lower speed limit (24 km/h needles exist; it would cut real
driving and flights); a list of "known bad" places (exact-coordinate
recurrence does not separate them — home points recur more); snapping the
points to roads (it would draw the needle along real roads; see §4).

### Owner-only words on a kept trip

**What.** A kept trip is a row in `decisions` (`migrations/00001_init.sql`):
the action (kept or left out), the name, and an anchor that re-attaches the
decision to the same trip after re-detection. Issue #3 adds one nullable
column, `notes`, and a way to edit the name and notes from the trip's own
page.

**How, and why a separate endpoint.** Today a name changes only by deciding
again: `POST /candidates/{id}/decision` with `action: confirmed` rewrites
action, name and anchor together (`internal/api/server.go` `DecideCandidate`
→ `store.UpdateDecision`). Reusing it for notes would mean every keep, leave-
out or bulk decision has to carry the notes back or blank them. A second
operation, `PATCH /candidates/{id}/decision`, changes only `name` and
`notes`, only on a kept trip, and never touches the action or anchor. The
existing decide path leaves `notes` alone, so leaving a trip out and keeping
it again does not lose what was written.

**Who sees notes.** The owner only (maintainer's decision, 2026-11-04). A
share link shows the name, as today; the shared view's contract
(`SharedAdventure` — "nothing about the owner") is unchanged. Sharing notes
can be added later as an explicit choice.

**Where the data goes.** Backup and restore carry the new column
(`internal/backup`); delete-my-data removes whole rows and needs nothing.

## 2. What gets built

### #4 — iPhone export accepted (STOP 1)

- `internal/timeline/parse.go`: the first token chooses the path — `{` the
  existing Android path, `[` a new array path that streams one segment at a
  time. Numeric strings decode through a small string-or-number type.
  `parseLoc` accepts `geo:`. One normalisation helper (upper-case, spaces to
  underscores) for visit kinds and activity modes, applied to both shapes.
  Path points get their time from the offset; a point past the segment's end
  is skipped and counted. `Stats.Format` = `phone-timeline-ios` (the
  `imports.detected_format` column is free text — no migration).
- An array that is neither (no segments, e.g. a My Activity export) is still
  refused with its existing kind.
- Tests: an inline iPhone sample in `parse_test.go`; the demo generator
  (`testdata/demo/gen`) writes the same fictional record in the iPhone shape
  too, `testdata/demo/demo-ios.json`, and a parity test proves both shapes
  parse to identical observations — so the demo's 3/1/0 pin holds for
  iPhone by construction. A private regression test reads
  `data/ios-timeline.json` when present and skips otherwise, like the
  archive's.

### #4, continued — needles on iPhone maps (STOP 1b)

- `internal/journey/journey.go`: the excursion pass and its three
  parameters (defaults 20 km, 0.25, 120 minutes; `ExcursionMinKm` 0 turns it
  off); the cluster builder and the walk shared with the teleport pass.
- `cmd/roadbook`: `journey` gains `-excursion-km`, `-excursion-ratio`,
  `-excursion-min`, and the anomaly line counts excursions.
- Tests: unit cases in `journey_test.go` (a single needle, a slow needle, a
  run of four, a recorded out-and-back kept, a longer absence kept, onward
  travel kept, a short hop kept, the switch, the input untouched); the
  goldens unchanged; `needles_test.go` counts needles on the trips of each
  private dataset before and after — the source of every figure below.

| dataset | trips | needles before | after | points set aside |
|---|---|---|---|---|
| tester's iPhone export | 60 | 67 on 7 trips | 4 on 4 trips | 63 |
| Android fixture | 18 | 0 | 0 | 0 |
| maintainer's archive | 66 | 3 on 2 trips | 0 | 3 |

The window was measured, not chosen: at 60 minutes 13 needles stayed; at
180 the pass began taking multi-point clusters with an hour's stay from
the archive, which may be real errands. Stated blind spot: a real trip of
over 20 km out and back within two hours, with nothing recorded on the way,
disappears from the map. The four needles left are longer than two hours.

### #1 — newest first (STOP 2)

`web/app/(app)/candidates/page.tsx` sorts a copy by `span_start`, newest
first — the same approach the trips shelf already uses
(`web/app/(app)/adventures/page.tsx`). "By score" is unchanged. The API order
(`ORDER BY seq`, chronological) stays: it is what detection produced, and the
CLI and other pages read it.

### #2 — the numeral goes (STOP 2)

"PLATE n" is removed from the trip covers, the trip page's eyebrow and map
margin, the downloadable image and the landing covers. The covers already
carry the trip's name and dates; the shelf's order ("the latest first") is
proven in tests by dates instead of numerals. `docs/product/words.md`
decision 1 named this exact trigger ("a tester asking what the numeral
means"); it is amended, and DESIGN.md §6 gets a dated amendment (the plate
label leaves the marginalia). "plate" stays only as an internal code word.

### #3 — edit name and notes (STOP 3)

- `migrations/00015_decision_notes.sql`: `notes text` on `decisions`.
- `api/openapi.yaml`: `PATCH /candidates/{id}/decision`, body `{name,
  notes}`; `Decision` gains `notes`. Go and TypeScript regenerated.
- Store: `EditDecisionText` — `WHERE user_id = … AND id = … AND action =
  'confirmed'`; cross-tenant test beside the existing ones.
- Trip page: an "Edit" control by the heading opens the name and a notes
  box, Save / Cancel; notes show under the heading as plain text. Hidden on
  shared views. This also makes the triage bar's existing sentence ("Rename
  any from its card or its page") true.

## 3. The real choices, with a recommendation

**A. Where the iPhone shape is understood.** Recommended: in
`internal/timeline`, second format slug, same `Parse` entry (above).
Rejected: a separate `internal/timelineios` package — the two shapes share
the segment model, `parseLoc`, the outlier-free domain output and the
rejection kinds; splitting them doubles the tests for no isolation gain.

**B. How notes are edited.** Recommended: the separate `PATCH` (above).
Rejected: notes through the existing decide call (every decide must round-
trip them); a separate `trip_notes` table (one optional text per decision
doesn't need a join, and backup/restore would gain a table).

**C. Limits.** Recommended: name at most 120 characters, notes at most 4,000
(counted as characters, not bytes, so Malayalam or Hindi aren't penalised),
enforced in Go and declared as `maxLength` in the spec. Existing names are
unaffected — the limit applies to edits, and the decide path keeps its
present rule (non-empty). Rejected: no limit — the first free text in the
schema should have one before it is ever displayed in an image or a share.

## 4. What is not in this phase

- Sharing notes on a share link (owner-only by decision).
- Photo captions, a rich-text editor, Markdown — notes are plain text.
- Raw-signal data on iPhone: the iPhone export has none.
- Places or names derived from `placeID` (phase 18 "places" covers names).
- Anything on the life map's own list order (not in #1).
- Snapping recorded points to roads (map matching against OSM). Asked
  for at STOP 1b; it cannot remove needles, and drawing recorded stretches
  along computed roads changes invariant 6 — its own phase, with a brief
  (PLAN.md backlog, "Snapped lines").

## 5. Checkpoints

- **STOP 0** — this brief. Gate entry in DECISIONS.
- **STOP 1** — #4: `make test`; the real file parsed by the CLI into a
  scratch database and uploaded through the browser on the scratch stack.
- **STOP 1b** — needles: `make test` (goldens unchanged, needle counts
  pinned), the affected trips looked at before and after on the scratch
  stack.
- **STOP 2** — #1 + #2: vitest, Playwright at 390/768/1280, screenshots.
- **STOP 3** — #3: `make test` (database suites), migration up and down,
  Playwright edit-and-reload on the scratch stack.
- **Close** — LOG.md, PR (`Closes #1`…`#4`), host deploy after merge.
