# Copy pass — decision log

Three lines each, written as the work happens. The seven decisions that
shaped the word list are in `words.md`; this log records what each commit
settled on the way.

## 2026-11-02 — Global strings and the legend behind a button

- **Chosen:** one `LegendInfo` client component (button "What the lines
  mean" + the same native popover as `Term`, inline fallback) replaces the
  printed `LegKindLegend` on all four maps; the trip page's Fix and Stop
  markers move inside it, and the bar's caption under the headline
  distance is gone. The maintainer's review of the first build (on the
  phone, 2026-11-02) removed the bar's one-line note from the popover as
  well: the bar stays drawn, unexplained except by the closed fold at the
  foot of the trip page — the recorded/drawn split is not something the
  person is asked to read. One `Unreachable` component
  carries the "isn't answering" sentence with the `roadbook serve` hint
  only when `session.mode === "off"` (decision 7). Nav reads Trips · Trips
  to review · n · Uploads on the header and the map panel. Six Go 404
  messages say "this trip isn't in your list any more — reload the page";
  the seven web fallbacks say "couldn't … (error n)". The date-named
  default becomes "Trip of 24 April 2026" through `fmtDateRange`.
- **Rejected:** keeping the legend's names as `Term` buttons inside the
  popover (a popover in a popover; the descriptions are already there);
  one shared "Legend" string constant for the button label in `lib/`
  (the component exports it, and the e2e helper opens it by role).
- **Would change our mind:** a tester not finding the (i) — then the
  button grows a word ("Legend") or the trip page's margin prints the
  four lines again.
- **Verified (first commit):** tsc, lint (0 errors), vitest 128, `go test ./internal/api
  ./internal/timeline`, Playwright 122 passed / 22 skipped at 390/768/1280
  against `next dev -p 3017` over the roadbook-p16 API; screenshots at
  390 and 1280 with the popover open on all four maps.

## 2026-11-02 — Landing, waitlist replies, sign-in

- **Chosen:** the landing's legend is a closed `<details>` (the page runs
  no script); the three sentences from copy.md §2; `how-its-built.html`
  takes the same style block in the same commit. The waitlist replies lose
  "seats", "your line" and the stale "entry email is the only way in".
  Sign-in is the wordmark and the button — the tagline, the reassurance
  sentence and then the visible heading went at the maintainer's review
  of the first build (the button says it; the heading is screen-reader
  only); "spots" and "the list" in the full state, and one fixed sentence on a provider error — the Go
  side logs Google's code and no longer puts it in the message.
- **Rejected:** showing Google's error code in small print under the
  sentence (nobody who sees it can act on it; the operator has the log).
- **Would change our mind:** a sign-in failure that a person could fix
  themselves (a blocked third-party cookie, say) — then the sentence
  gains the one instruction, not the code.
- **Reported, not done (maintainer, 2026-11-02, on the landing):** the
  page still feels cluttered — eight blocks, two pictures of a trip, two
  "Make mine" buttons. A layout cut (one picture, one button, one-line
  steps and promises, the waitlist box reduced to a field) waits until
  the copy pass closes, then gets its own mockup-first step.

## 2026-11-02 — Welcome

- **Chosen:** copy.md §5 as approved: the lede says what the person gets,
  the sample map is "Sample · three trips across Iceland", the four-step
  fold becomes three ("Your trips are found" absorbs import and detection),
  the walkthrough stamps read "Checked on a phone" / "Not yet checked on a
  phone", Takeout's small files are "info files", and both links to the
  uploads page say Uploads. No Term is left on the page except "Original
  photos"; the `candidate` and `plate` Terms are unused here now.
- **Maintainer's review of the first build:** the block between the steps
  and the photos line was still cluttered — a subheading ("Don't have the
  file yet?"), an intro paragraph and two boxed folds with a stamp each.
  Now: the three steps (step 1 says "how is below"), the button, two plain
  fold lines (On Android · On iPhone), the photos line. The check date
  moved inside each fold as its last line.
- **Maintainer's second review:** the photos door leaves the start page
  altogether — "someone who doesn't have a Timeline is not our audience
  at this point". The section, its "Which photos work" fold and the
  "Never had Timeline on?" line are gone; the photos panel stays on the
  Uploads page, where photos add detail to trips already found. The
  `photos` and `never-enabled` anchors are retired, so an image dropped
  into the Timeline picker is now a wrong file pointed at "What the right
  file looks like", and the Go message for it names the Uploads page.
- **Rejected:** dropping the verification stamps altogether (a stale
  guide must still say so, phase 7's rule; only its place and register
  changed); removing the photos panel from Uploads too (not asked; photos
  on a found trip are a different thing from photos as the only source).
- **Would change our mind:** a tester asking what "Takeout" is — then the
  word goes and only "an older Google download" stays.
- **Verified:** tsc, lint, loop-pages and legend specs (40 passed at three
  widths); screenshots at 390 and 1280.

## 2026-11-02 — Upload and waiting, with the Go messages

- **Chosen:** copy.md §6 as approved. Web: "Reading your file…", "Finding
  your trips…", "Nothing new in this file", "See your trips", "That isn't
  your Timeline file" without the bracketed kind, "Roadbook couldn't read
  this file", "listed under Uploads", and the photo verdicts in words (a
  photo "had a location", "info file", "a copy from a chat"). Both
  uploaders take an `operator` flag from the page's session so "the
  server log has details" is said only on an instance without sign-in
  (decision 7); the start page reads the session for it. Go: every
  rejection in `internal/timeline/parse.go` is one sentence about the
  file plus the hint "export it again from Google Maps on your phone —
  the start page shows how"; the three old-Google-download kinds share
  one sentence; the keys seen in an unrecognised JSON file stay in the
  message in brackets (useful when someone reports it) but the parser's
  own error text is gone from the not-valid-JSON case; `imports.go`'s
  four operator-facing failures go through a new `ownSide` helper that
  appends the server-log clause only when `s.Auth == nil`. The photo
  uploader's three Go messages ("can't be opened", "more than N files at
  once", "larger than N MB"). `Kind` keys and error types unchanged; the
  Go tests pin kinds, not sentences, and pass as they were.
- **Rejected:** mapping kinds to sentences in the web layer instead of
  changing Go (two places to keep in step; decision 3); keeping Google's
  menu path in the hint (it was the Android path only, and the start page
  has both).
- **Would change our mind:** a hosted person hitting "something went
  wrong on this Roadbook's side" with nowhere to turn — then the sentence
  gains a contact, which is a hosting decision, not a word.

## 2026-11-03 — The map

- **Chosen:** copy.md §7 as approved. The three empty states say "No
  trips found — yet", "No trips yet" (self-hosted only, after a
  command-line upload) and "12 trips are waiting for you · Keep the real
  ones and they're drawn here"; the fourth (kept trips that could not be
  loaded) keeps the "check the API and database" clause only on an
  instance without sign-in. The hover card and the summoned list print
  dates as "24–26 April 2026" through `fmtDateRange` and the hover card
  loses "— 640 observed"; the list's dialog is "Kept trips" / "Trips (3)".
  The plate note says "Your kept trips are numbered in the order you
  travelled"; the score note ends "It only sets the order; you decide".
- **Rejected:** removing the bar from the hover card and the list rows
  (decision 2 keeps the bar drawn, uncaptioned, everywhere a distance is).
- **Would change our mind:** a tester reading "12 trips are waiting for
  you" as a to-do rather than an invitation — then "12 trips found so far".

## 2026-11-03 — Trips to review

- **Chosen:** copy.md §8 as approved. The heading is "Trips to review",
  the intro "Roadbook found these trips in your history. Keep the real
  ones; leave out the rest." with no taught term; the buttons are Keep ·
  Leave out · Save · Cancel · Change · Keep as is (capitalised, as
  buttons elsewhere are); a left-out trip reads "left out"; the facts line
  says "from home", "travelled" and "visited 2× before"; the fold keeps
  the rule and the points-too-fast count and drops the run number; the
  selection bar is "Keep or leave out together" with Keep selected / Leave
  out selected; the notes say Kept / Left out / "Nothing changed". The
  CLI hint in the empty state is self-hosted only (decision 7).
- **Rejected:** "Skip" for leave out (sounds like later); lowercase button
  labels (the old style; every other button on the site is capitalised).
- **Would change our mind:** a tester hunting for "where did the one I
  left out go" — then the left-out state gains a line, not a new word.

## 2026-11-03 — Trips

- **Chosen:** copy.md §9 as approved. Heading "Trips"; one line under it
  ("3 trips, the latest first, numbered in the order you travelled") with
  no taught term — the `provenance` entry leaves `lib/terms.ts`, nothing
  reads it now; the empty state is "No trips kept yet · Keep a trip under
  Trips to review and it appears here"; the flag on a cover is read aloud
  as "distance differs from Google's"; "Unnamed trip".
- **Rejected:** keeping the plate Term on the intro line (decision 1
  keeps the printed numeral, not the word in a sentence).
- **Would change our mind:** nothing here; the shelf is the simplest page.

## 2026-11-03 — The trip page

- **Chosen:** copy.md §10 as approved. Eyebrows KEPT TRIP · PLATE II / LEFT
  OUT / TO REVIEW / SHARED TRIP; the flags and the day-by-day edges say
  "your file" ("Trip begins · already under way when your file starts");
  status "kept 12 Oct 2026" / "Not reviewed yet — keep it or leave it out
  under Trips to review" / "Shared with you". Photos: times as "24 April
  2026 10:12 · from the photo" through a new `sourceWords` in
  `lib/format.ts` (the API's enum values said in words), "on the map" /
  "no location — not on the map", place lines "from the route at that
  time" (the three ground kinds collapse to one sentence), results "not
  added" / "info file matched" / "info file without its photo"; the
  uploads strip says how to remove them. Download: "Download the map" /
  "Download for a story", "Making it…", the description in words. Sharing:
  "Stop sharing", "Share this trip", three plain bullets with the same
  disclosure, "Copy it now — Roadbook can't show this link again". The
  fold at the foot: "recorded · by road · gap · flight" in whole
  kilometres, "312 recorded points (32 of them precise)", "How you
  travelled, as Google guessed it", "Google didn't record how you
  travelled", "Distance check: by road … · Google's figure …". The
  downloaded picture: "91% recorded by the phone", "Single point", "Already
  under way when the file begins." Four Go photo messages in words. The
  `candidate` entry leaves `lib/terms.ts`; only `plate`, the four kinds,
  `originals` and `score` remain.
- **Rejected:** dropping the fold's figures to whole kilometres only on
  the page but not the picture (one register everywhere); hiding the
  photo source entirely (a person who uploaded an info file wants to see
  it was used).
- **Would change our mind:** a tester asking what "precise" points are —
  then the bracket goes and the count stands alone.

## 2026-11-03 — Shared view, Uploads and delete

- **Chosen:** copy.md §11 and §12 as approved. The shared view is "A shared
  trip" with the one-sentence footer and "Make your own map →"; its
  not-found page says "stopped sharing". Uploads: heading "Uploads", table
  headers Stops · Journeys · Points · Precise points · Unreadable · New,
  statuses "reading" / "done" / "couldn't be read", and "Roadbook couldn't
  look for trips in it" with the server-log clause only for the operator
  (the table takes the session's mode, decision 7). Delete everything:
  upload / trip / choice / preview / spot in place of import / candidate /
  decision / thumbnail / seat; the list of what goes is the same list; the
  typed word is still `delete`.
- **Rejected:** collapsing the eleven-column table (a design question,
  reported in copy.md's "Needs more than words").
- **Would change our mind:** a tester asking what "precise points" are —
  then that column folds into Points.

## 2026-11-03 — Close

- **Chosen:** DESIGN §6 gains a dated amendment (the legend behind one
  control; labels, order, samples, line styles unchanged; the bar
  uncaptioned; the exported image keeps its printed legend). The README's
  walkthrough names the controls as they now read. `docs/screens/
  copy-pass-*` is the committed after-set, captured from the demo through
  the dev server. The after-table and the breakages are in `LOG.md`. The
  PLAN.md entries (landing cut, "one line") are drafted in LOG.md for the
  maintainer to place on main.
- **Rejected:** merging the branch from this session (the maintainer
  merges, one merge for the whole pass, then the host pulls main).
- **Would change our mind:** the five-second test with friends failing on
  the new words — then `words.md` is the place to change, and each screen
  follows.
