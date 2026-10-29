# Phase 16 decisions

Written as decisions are made. Three lines each: what was chosen, what was
rejected, what would change our mind.

## 2026-10-28 — Phase 16 is reading load

- **Chosen:** the next phase addresses the maintainer's finding after the
  first live walk — the UI is "mostly words", confusing, unread. Named
  "reading load"; measured by `web/scripts/copy-count.cjs` before and after.
- **Rejected:** folding this into phase 15 (hosting), whose target is a
  zero product-code diff; a full redesign (the Atlas direction is kept).
- **Would change our mind:** nothing; the evidence is the maintainer's own
  walk and the measured baseline (RESEARCH §2).

## 2026-10-28 — Implementation after phase 15 CP3; planning in parallel

- **Chosen:** research, brief and shape work run alongside phase 15 CP3
  (durability); code starts only when CP3 is on main. CP3 landed the same
  day (`edcbeb7`), so the condition is met before CP1 begins.
- **Rejected:** pausing phase 15 for the UI — the live instance accepted
  real sign-ins with no backup chain yet.
- **Would change our mind:** nothing retroactively.

## 2026-10-28 — The work lives on a branch

- **Chosen:** `phase-16-ux`, created by the assistant, every commit the
  maintainer's; `docs/PLAN.md` edited on main; rebase after each main
  commit; merge at accepted checkpoints. The host pulls main only.
- **Rejected:** working on main beside phase 15's checkpoints — the
  dry-run review and the host's pull become harder to reason about.
- **Would change our mind:** phase 15 finishing entirely; then the branch
  is just a branch and merges as usual.

## 2026-10-28 — Atlas stays; treatments within it are explored

- **Chosen:** DESIGN.md binding; treatment options within the token
  system compared at a STOP; one wording amendment proposed (BRIEF §5C).
- **Rejected:** reopening the direction (a dark theme, a new type
  system) — the complaint is information design, not aesthetics.
- **Would change our mind:** a walk showing the aesthetic itself puts
  people off; none has.

## 2026-10-28 — Watched walks join at checkpoint boundaries, not before CP1

- **Chosen:** the maintainer's call after reading the findings: proceed
  on the measured audit and his own input (RESEARCH §9); walks (RESEARCH
  §1) happen on the live instance as they can be arranged and join at
  the next checkpoint boundary; the closing walk is CP4's acceptance.
  This supersedes the planning STOP's "walks within a week or two before
  design starts".
- **Rejected:** holding the phase for the walks — the research already
  located the words, and the maintainer has used the product.
- **Would change our mind:** a walk before CP2 contradicting §9.1; then
  CP2's shape is re-decided at its STOP.

## 2026-10-28 — Shape review as code on the branch, not static mockups

- **Chosen:** the maintainer's call ("start implementing and see the next
  UI"): the mockup bucket's questions are answered by working code on the
  branch, reviewed on a phone at each checkpoint's STOP against a scratch
  demo stack. The gate's purpose — nothing reaches main unreviewed — is
  unchanged.
- **Rejected:** `docs/phase-16/mockups/` as phases 6 and 9 did — the
  research fixed the treatments' direction, and the open questions (a
  disclosure's feel, a popover on a phone) are better judged in hand.
- **Would change our mind:** the CP1 STOP rejecting the shape rather than
  its details — then CP2 returns to the mockup gate before its code
  (BRIEF §5A fallback).

## 2026-10-28 — The map pages are out of scope

- **Chosen:** the life map, `/adventures` and `/candidates` are left as
  they are except for the plate-number caption and the teaching control
  where a load-bearing term appears; their own improvements go to the
  next iteration (maintainer, RESEARCH §9.3).
- **Rejected:** the research's candidate-card treatment and the
  `/candidates` copy items beyond folding the run line.
- **Would change our mind:** a walker stopping on `/candidates` in a way
  the bulk-triage tools of phase 11 do not explain.

## 2026-10-28 — The adventure page is the trip, not the reconstruction

- **Chosen:** the cover leads with the trip's figures (phase 14's summary
  block becomes the hero); the drawing's account moves into one
  disclosure; the floor — legend, provenance bar, divergence flag,
  truncation marks — stays visible (BRIEF §1, §4 CP1).
- **Rejected:** deleting the reconstruction figures (invariant 13 — they
  are what the CLI prints); a separate page for them (a new navigation
  element).
- **Would change our mind:** nothing about the split; the disclosure's
  copy gets its own diet if a walker opens it and is confused.

## 2026-10-28 — Gate 1 passed

- **Chosen:** BRIEF.md approved as written; all §5 recommendations stand
  (disclosure for the drawing's account, legend wording to be decided at
  the CP1 STOP, the `Term` popover control, the folded front door). CP1
  begins in a fresh session from the brief.
- **Rejected:** continuing straight into CP1 in the planning session —
  the brief is the source CP1 builds from, and a cold read of it is the
  check that it is complete.
- **Would change our mind:** nothing; a process choice.

## 2026-10-28 — CP1 built on the branch: the trip leads, the drawing is one tap away

- **Chosen:** the cover is eyebrow · name · dateline (dates, countries,
  regions), then the trip block as the hero — the headline kilometres
  with "over n days", the provenance bar directly under them with its
  caption as a Term, then Time away · On the move · Stopped · Farthest
  (owner only). The "Through" row went: the dateline already says it.
  Days and the point count left the dateline (days are the headline,
  points are the drawing's). Everything about the drawing — the kind
  kilometres, the point counts, Google's figure, the mode line, the
  distance check in full — sits in one `<details>` "How this route was
  drawn", closed by default; a flagged check keeps its flag and one plain
  line above it. The truncation line shortened to one sentence each.
- **Rejected:** keeping any figure out of the disclosure for emphasis
  (then the trip block is not the hero); dropping the stay-in-place
  wording's muted figure (invariant 13 wants it reachable).
- **Would change our mind:** a walker opening the disclosure and being
  confused by it — then its copy gets the diet, not its placement
  (BRIEF §5B).

## 2026-10-28 — On a phone the plate comes before the narrative

- **Chosen:** the view's three grid children are cover, plate, narrative
  in DOM order; a phone stacks them that way, so the map is one swipe
  below the trip block instead of below every day's rows. From `lg:` the
  plate takes the right column across both rows and sticks, as before.
- **Rejected:** leaving the plate last on phones (it was the last thing
  on a page about a map); duplicating the map (one MapLibre instance
  per page, the phase 9 WebGL-context rule).
- **Would change our mind:** the maintainer's phone review preferring the
  narrative first; it is a class change on three elements.

## 2026-10-28 — The Term control is a native popover with a scripted fallback

- **Chosen:** `components/term.tsx`: a `<button popovertarget>` with a
  dotted text-decoration opens a `<span popover="auto">` note — top layer,
  centred, light-dismiss and Escape native, a "Got it" button closes it
  in one tap. Where `popover` is unsupported (read once with
  `useSyncExternalStore`, no effect-driven setState) the same button
  toggles the note inline. Both forms are spans, so a Term sits inside an
  eyebrow `<p>`, a `<dt>` or a list item. Notes live in `lib/terms.ts`
  beside the legend's wording. Hit area 44 px by the site's
  negative-margin padding; the mark is a text-decoration because a
  bottom border sat 14 px under the words in the first build.
- **Rejected:** CSS anchor positioning (not in Safari's stable at the
  pilot phones' versions); a `<details>` fallback inside text (invalid
  nesting inside `<p>`, a hydration mismatch).
- **Would change our mind:** the popover misbehaving in an in-app browser
  on the device walk — then the inline form becomes the only form.

## 2026-10-28 — Narrative rows: glyph · time · one line; the legend as a list

- **Chosen:** each row is the legend's own line sample (`LegSample`,
  exported from the legend component) or the map's fix/stop dot, a time,
  and one line: "Recorded, 82.0 km" · "Drawn along roads, 160.0 km" ·
  "Not recorded, straight line 40.0 km" · "Flight, 300.0 km" · "Stayed
  3 h" · "Position recorded" · "Journey begins / ends" with ⚑ and a
  shortened note when truncated. Coordinates left the rows. The
  stay-in-place rule (phase 9, routed gaps under 0.25 km) now covers
  unknown gaps too: an unrouted stack showed "Not recorded, straight line
  0.0 km" seven times on one demo page. The legend is a `<ul
  aria-label="Legend">`; its descriptions are visible at every width, two
  columns on a phone, one line from `sm:`; each name is a Term.
- **Rejected:** keeping the uppercase chips (a fifth register on the row,
  and the words they carried were pipeline terms); glyph-only rows
  without a kind word (the glyph is aria-hidden; the line names the kind).
- **Would change our mind:** the maintainer's phone review finding the
  glyph column too quiet to carry the channel — then the kind word
  returns in front of the line.

## 2026-10-28 — Legend wording built as recommended, decided at the STOP

- **Chosen (provisional):** `lib/legend.ts` carries BRIEF §5C's wording —
  "Observed — your phone recorded it · Routed — drawn along roads ·
  Unknown — nothing recorded, straight line · Air — a flight" — so the
  maintainer reviews the recommendation in hand. The image builder and
  its tests read the constant; `legend-wording.spec.ts` reads it too.
- **Open for the STOP:** "your phone" on the shared view and on an
  exported image reads to a stranger; "the phone recorded it" is the
  neutral form. The DESIGN §6 amendment is written only after the
  decision.
- **Would change our mind:** the maintainer preferring the atlas register
  of the old descriptions — then only "recorded fixes" → "recorded
  positions" changes.

## 2026-10-28 — CP1 reading load, measured

`copy-count.cjs` against the scratch stack after CP1 (baseline: RESEARCH
§2). Adventure pages only; the rest is CP2/CP3 territory.

| page | before, phone (words / sentences / titles / terms) | after, phone | after, desktop |
|---|---|---|---|
| adventure 1 | 605 / 58 / 6 / 83 | 422 / 54 / 6 / 8 | 426 / 55 / 6 / 8 |
| adventure 2 | 426 / 42 / 6 / 44 | 319 / 37 / 6 / 9 | 323 / 38 / 6 / 9 |
| adventure 3 | 547 / 53 / 6 / 73 | 397 / 51 / 6 / 10 | 401 / 52 / 6 / 10 |

The six titles are the day-heading toggles' (kept: they name the action
of a real button, not a term). The remaining terms are "plate" and the
legend's four names — the load-bearing set, each now a Term. The
sentence count barely moves: rows are sentences, and the row count is
the journey's. Words on the plate margin grew (the legend's descriptions
are now on the phone); the cover and rows lost more.

## 2026-10-29 — CP1 review: the narrative's shape rejected, not its details

- **Chosen:** the maintainer's verdict on the phone: a day must say what
  the person did — the places and regions covered, the distance, the
  stops — not "Position recorded" and "Not recorded, straight line 130
  km". The cover, the disclosure, the legend and the Term control stand.
  Before redesigning, a full inventory was taken (three read-only sweeps:
  parser, schema + API, pipeline; the plan file carries the tables). Its
  conclusion: the Timeline export contains no place names (visits are a
  coordinate plus an opaque Google `placeId` the parser never reads and
  Google's paid API alone could resolve); nothing stored names a stop;
  per-day regions are derivable with one query change but not served;
  per-day kilometres, their kind composition, stays and photos are
  served or derivable in the web today.
- **Rejected:** keeping the event rows and adding places around them;
  re-parsing retained uploads for `placeId` (resolvable only through a
  paid API against its terms).
- **Would change our mind:** nothing about the verdict; the shape below
  is reviewed again at the next STOP.

## 2026-10-29 — Phase 16 stays web-only; phase 17 "places" owns the data

- **Chosen:** the split. Phase 16 builds the day form from what is served
  and keeps its zero-Go-diff charter; a new phase 17 "places" (brief
  before code, its own gate) serves per-leg/per-stop regions and stop
  names from two sources — an embedded offline gazetteer (Natural Earth
  populated places; every instance names its stops with no service) plus
  an opt-in batch Nominatim pass cached like `route_cache`, with the row
  recording which source named it. The day line and stay row carry the
  two slots phase 17 fills.
- **Rejected:** re-scoping phase 16 to carry the Go work as a CP1b (the
  assistant's recommendation — one rework of the page instead of two);
  Google `placeId` as a source.
- **Would change our mind:** phase 17's brief finding the gazetteer too
  coarse to be worth embedding — then Nominatim-only, opt-in.

## 2026-10-29 — The day form: heading + provenance bar, one line, stay rows

- **Chosen:** `day-narrative.tsx`: the heading as before, a
  `ProvenanceBar` under it fed by `dayComposition` (new pure helper in
  `lib/slice-days.ts`, vitest-pinned to sum to the day's km); one line
  built from the day's events — "Journey begins", "1 stop", "Stayed all
  day · nothing recorded from … to …", "Travelling all day", "On the
  move", "overnight into Day 3", "Journey ends", "Nothing recorded this
  day"; a row per stay (stop glyph · time · "Stayed 38 h" / "Stayed ·
  through to Day 3" / "Moved on after 38 h", "· position not recorded"
  when the stop has no points). Photos on the day's legs sit under the
  line; a stay's photos on its row. The stay-in-place threshold
  (`STATIONARY_GAP_KM`, phase 9) left with the transit rows: no transit
  wording remains to demote. `narrative-wording.spec.ts` now pins "no
  0.0 km figure anywhere in the narrative".
- **Rejected:** per-stay durations repeated in the day line (the row has
  them); an empty-DOM placeholder for the phase 17 slots (a comment marks
  them; nothing ships that renders nothing).
- **Would change our mind:** the maintainer's phone review finding the
  line too thin before phase 17 lands — then the day's kind composition
  is said in words under the bar ("mostly recorded").

## 2026-10-29 — CP1 reading load, re-measured after the review

`copy-count.cjs` against the scratch stack with the revised day form
(baseline: RESEARCH §2; the first CP1 build's figures are in the entry of
2026-10-28 above).

| page | before, phone (words / sentences / titles / terms) | first CP1 build | revised day form |
|---|---|---|---|
| adventure 1 | 605 / 58 / 6 / 83 | 422 / 54 / 6 / 8 | 355 / 42 / 6 / 11 |
| adventure 2 | 426 / 42 / 6 / 44 | 319 / 37 / 6 / 9 | 298 / 36 / 6 / 11 |
| adventure 3 | 547 / 53 / 6 / 73 | 397 / 51 / 6 / 10 | 330 / 42 / 6 / 12 |

The term count rose by two or three per page: "stop"/"stops" are in the
script's list and the day line now counts them ("3 stops"). They are
plain words (RESEARCH §4 lists stop and Day as already plain); the list
is left as it is so the before and after stay comparable.

## 2026-10-29 — Review of the revised day form: the flag to the foot, the unplaced stay as a mark

- **Chosen:** the maintainer's two changes on sight. (1) The flagged
  distance check leaves the cover for the foot of the page, directly
  above "How this route was drawn", still outside the disclosure and
  visible with it closed — the floor (BRIEF §1) asks that the flag never
  hide, not that it lead. (2) "position not recorded" leaves the stay
  row: a stay the record holds no position for draws a hollow stop
  marker, with the words as the glyph's title and aria-label. The map
  draws no such stop, so the hollow mark says the same thing the map
  does.
- **Rejected:** dropping the unplaced note entirely (the row would claim
  a place on the map it does not have); a hover-only title on the row
  text (invisible on touch; the mark is visible everywhere, the title is
  the extra).
- **Would change our mind:** a walker asking what the hollow marker means
  — then the plate margin's legend gains the hollow stop beside "Stop".

## 2026-10-29 — The photos empty state: one short line, the why on a term

- **Chosen:** "None yet. Original photos work best." in small text, with
  "Original photos" a Term whose note says what an original carries and
  that a messenger copy has it stripped and cannot be placed
  (`lib/terms.ts` `originals`). The old sentence named EXIF and the
  project's accuracy ranking — the pipeline's account again (maintainer's
  review).
- **Rejected:** a hover-only title (invisible on touch; the Term is the
  phase's one teaching mechanism and works by tap); keeping the longer
  sentence in smaller type.
- **Would change our mind:** upload results showing people still send
  messenger copies — then the note moves onto the upload control itself.

## 2026-10-29 — A day's average pace on the day line

- **Chosen:** the maintainer asked for speed at the STOP. The day line
  gains "49 km/h average where recorded": `dayPaceKmh` in
  `lib/slice-days.ts`, the Go `Summarize` pace rule applied to one day —
  summed observed distance over summed observed duration, moving legs
  only (a fix has no pace), drawn-in stretches and flights excluded
  (no measured duration of their own); absent when the day recorded no
  moving stretch, never zero. Parity vitest: because each leg belongs to
  exactly one day, the days' sums reproduce `summary.observed_pace_kmh`
  on the three demo journeys — the figure is arithmetic on legs the CLI
  prints, the same class as the day's kilometres.
- **Rejected:** a top speed or any point-to-point speed (28–40 positions
  per journey, Wi-Fi/cell fixes, minute-truncated stamps — the phase 3
  false-teleport trap; recorded as rejected in PLAN); speed over routed
  gaps (the router's travel-time estimate is the road's guess, not the
  drive); Google's per-fix `speedMetersPerSecond` (dropped at parse;
  last month only, GPS fixes only — a later evidence-gated item).
- **Would change our mind:** real-data days where a single short
  observed stretch skews the average — then a minimum observed-hours
  floor becomes a named parameter.

## 2026-10-29 — Legend wording decided: plain words, "the phone"

- **Chosen:** BRIEF §5C's amendment stands, with one word changed at the
  STOP: "Observed — the phone recorded it" (not "your phone"), because
  the same legend sits on a shared view and travels on an exported image,
  where the reader is not the phone's owner. The Term notes for Observed
  and the provenance bar say "the phone" too. DESIGN §6 carries the
  dated amendment; `lib/legend.ts` is the one constant; the image
  builder's tests and `legend-wording.spec.ts` read it.
- **Rejected:** keeping the atlas register ("recorded fixes",
  "great-circle arc") and teaching the two terms instead.
- **Would change our mind:** a walker reading "the phone" as someone
  else's phone — then "a phone recorded it".

## 2026-10-29 — CP2, the front door as built: the plate, three steps, one door

- **Chosen:** `/welcome` opens with the title, one sentence and the demo
  plate with its legend; then "Your own, in three steps" (export · upload
  · confirm) with the Timeline file picker directly under it as the one
  filled button on the page ("Choose the Timeline file", full width on a
  phone, 48 px); then "Don't have the file yet?" with the Android and
  iPhone walkthroughs as native `<details>`, their verification stamps on
  the summary line before the fold; the photos door second, one line and
  an outlined button, with "Original photos" a Term; "What the right file
  looks like", "Where your data goes" and "What happens after you upload"
  folded at the foot. The upload islands gained a `variant` prop
  (`panel` for `/imports`, unchanged; `door` for the front door — the
  button and its status alone, no second heading or paragraph). Every
  `WELCOME_SECTIONS` id keeps a real target, unchanged, so
  `lib/rejection-anchors.ts` and its test are untouched; three ids now
  sit on `<details>` elements and `components/open-on-hash.tsx` — the one
  script added to a deliberately static page — opens the fold a rejection
  link lands on, and every fold enclosing it, then scrolls to it. The
  eyebrow ("A road atlas of your own past") and the section jump row are
  gone: the picker is in the first screen and a half.
- **Rejected:** a stepper (BRIEF §5E — client state on a static page,
  and it hides the picker from a returning person with the file in
  hand); opening every walkthrough when a link targets the "export"
  block (two open accordions where one platform applies — the summaries
  name the platforms); repeating the picker after the walkthroughs (one
  primary action; the walkthroughs' last step links back up to it);
  deleting the walkthroughs' verification notes (phase 7 BRIEF §3E — a
  stale date is information).
- **Would change our mind:** a walker who does not find the picker
  (then it moves above the plate); a walker who does not open "Don't
  have the file yet?" and stalls (then the two platform summaries open
  by default on a phone).

## 2026-10-29 — CP2, the landing: the four kinds once, the rule in one sentence, custody folded

- **Chosen:** the "honesty rule" section with its four ink entries is
  removed; the four kinds appear once, in the plate's legend, whose
  wording now follows DESIGN §6 as amended at CP1 (the landing must never
  drift from the product's words) and whose descriptions are visible at
  every width (they were hidden under 480 px — the phone user has equal
  weight, RESEARCH §9.4). The rule itself is one italic sentence under
  the legend; the lede lost its copy of it. "How it works" keeps its four
  steps at one sentence each. Custody is one paragraph, with the full
  statement (the three headings and the no-public-demo note) in a
  `<details>` under it. "The way in" is unchanged in substance. The page
  stays zero-script: `<details>` is HTML.
- **Rejected:** dropping "How it works" or "What you reach" (the brief
  names neither; the steps are a real sequence and the screenshots are
  the page's only look at the product past the hero); rewriting "the way
  in" (phase 15's copy, reviewed on the CP2 walk there).
- **Would change our mind:** the phase 15 CP4 stranger stalling on the
  landing — then the steps go too, and the page is hero, custody, way in.

## 2026-10-29 — CP2 reading load, measured

`copy-count.cjs` against `next dev` on the CP2 code with the scratch
demo API (baseline: RESEARCH §2; CP1's rows above). The landing is read
from disk by the script; the welcome page from the stack.

| page | before, phone (words / sentences / titles / terms) | after CP2, phone | after CP2, desktop |
|---|---|---|---|
| landing | 851 / 73 / 0 / 31 | 597 / 53 / 0 / 25 | 601 / 54 / 0 / 25 |
| welcome | 1105 / 80 / 0 / 56 | 257 / 27 / 0 / 12 | 261 / 28 / 0 / 12 |

The welcome page's folded text is not counted (closed `<details>`
content is not in `innerText`), which is the point of the measure: it
is what a person is shown. Opened, the page carries roughly the
walkthroughs and notes it did before, shortened. Page height at 390 px:
5,276 px before, 2,143 px after (puppeteer, full page).

## 2026-10-29 — CP2 review: the welcome page accepted, the landing returned to the mockup gate

- **Chosen:** the maintainer's verdict on sight: the welcome page is
  "much cleaner" and stands; the landing is not clean — flat hierarchy
  (every section eyebrow + heading + prose), too many devices (rules,
  boxes, numerals, mono caps, middle dots), still too wordy (597 words on
  a phone). CP2 is committed as built; the landing redesign is **CP2b**,
  the BRIEF §5A fallback used deliberately: a shape question goes back to
  the mockup gate. Inspiration first — a reference board before any
  mockup (`LANDING-REFERENCES.md`: eleven pages in three groups, same
  job / same audience / print ancestry, with word counts measured the
  way ours are).
- **Rejected:** holding the welcome page uncommitted until the landing is
  done; redesigning the landing straight into the file.
- **Would change our mind:** nothing about the split; the board's
  principles are decided at the next entry.

## 2026-10-29 — STOP 1: the board's principles stand, plus two of the maintainer's

- **Chosen:** principles 1–5 as the board states them (one thing leads;
  one label treatment and one rule weight; drawn covers, not screenshots;
  a copy budget of about 350 words, custody once; the form is the only
  box). Two more from the maintainer: **(6) written for a traveller, not
  an engineer** — the landing never uses *plate, leg, fix, custody,
  provenance, candidate, civil day, great-circle, colophon*; "atlas" stays
  as the brand word; captions say what the person sees ("All your trips on
  one map", "Each trip, day by day"); the provenance bar may appear as a
  drawing but carries no jargon label. **(7) Copy before design** — the
  full landing copy, at most 350 words, in the order hero · what you get ·
  three steps · privacy · the way in · footer, approved at a STOP before
  any mockup; step 1 says plainly that the history is exported from Google
  Maps and roughly how long that takes. The hero copy is fixed by the
  maintainer: headline "Every road you've travelled, on one map."; button
  "Make mine"; under it "Sign in with Google · 10 free spots"; then one
  privacy sentence. "Make mine" is the one action's name everywhere the
  sign-in appears (masthead, hero, the way in). The hero image is the life
  map — the whole promise — not one adventure's cover.
- **Rejected:** "custody" and the rest of the list as landing words (they
  are the product's and the docs' vocabulary; a stranger reads none of
  them); a different verb per sign-in placement.
- **Would change our mind:** a reader taking "Make mine" for a purchase —
  then the small line under it says "free" before "10 spots".

## 2026-10-29 — The honesty rule leaves the landing for a page of its own

- **Chosen:** the four kinds explained, the rule ("never a line it cannot
  justify") and the drawing's account move to a separate static page,
  "How it's built" (`landing/how-its-built.html`, same tokens, zero
  script), linked from the footer. The legend under the hero map stays on
  the landing, in the plain words DESIGN §6 now carries. The planned
  drift test (Step 3) pins the landing's legend to `web/lib/legend.ts` as
  before; the new page carries the fuller account and is not pinned
  (prose, not the fixed wording).
- **Rejected:** keeping the rule as a sentence under the legend on the
  landing (the CP2 trim did this; it is still the engineer's account on
  the traveller's page); dropping the rule from the public surface
  altogether (it is the product's reason to exist, and the self-hosting
  reader wants it).
- **Would change our mind:** the five-second test (below) showing readers
  do not understand why the lines differ — then one plain sentence
  returns under the legend.

## 2026-10-29 — Product proof: drawn covers plus one real screenshot

- **Chosen:** the three demo adventures as drawn covers replace the phase 9
  screenshots; **one** real screenshot of the app stays as proof that this
  is working software (a current phase-14 capture or a fresh one from the
  demo stack, never phase 9's).
- **Rejected:** covers only (a drawing could be a poster shop; the
  screenshot says it runs); three screenshots (the greyish row the board
  found weak).
- **Would change our mind:** the screenshot reading as clutter next to the
  covers on a phone — then it moves below the steps, smaller.

## 2026-10-29 — One mockup, A1; the review is a five-second test

- **Chosen:** build mockup A1 ("the plate is the page") only. A2 (the
  typographic atlas cover) is dropped: fewer images is the wrong direction
  for a visual product. B (direction B's dark sheet) is dropped: a rule
  amendment with no gain for visitors. The review at STOP 2 is the
  maintainer's phone plus three non-engineer friends sent the A1 link and
  asked "what does this do?" after five seconds; their answers decide.
- **Rejected:** three mockups for comparison (the board already fixed the
  direction; what is left is execution, which one mockup tests).
- **Would change our mind:** the five-second answers missing the point —
  then A1's hero is reworked before any build, not the mockup count.

## 2026-10-29 — Copy approved; mockup A1 built from it

- **Chosen:** the maintainer approved the landing copy with edits (hero
  subline, privacy line, captions, a platform-neutral step 1, the photos
  line, the privacy paragraph, the way-in's second paragraph, the footer's
  second sentence dropped); the approved text measures **327 words**,
  under the 350 budget. One edit of the assistant's stands: "delete all of
  it in one tap" became "Delete all of it, any time", because deletion
  asks the person to type "delete" before the button unlocks. Mockup A1
  (`mockups/landing-a1.html`): masthead with the one action; headline,
  two sentences, "Make mine" full width on a phone, the two small lines;
  the life-map plate at full sheet width with the legend and caption in
  its margin strip; "Each trip, day by day" as the three demo covers in
  the atlas-grid idiom (route art, PLATE n, country, name, dates · days,
  km, the bar with no label) over one screenshot of the app's adventure
  page captured fresh from the demo stack; three steps; privacy; the one
  box with the form; a four-link footer. Six type sizes, one rule (1 px
  ink), the double rule only on the plate and the covers, no eyebrows, no
  inline styles except the covers' bar widths (data).
- **Rejected:** a two-column hero at desktop (text beside the plate — the
  plate shrinks, and the plate is the point); lazy-loading the screenshot
  (a blank box in the first full-page capture; the page has one image);
  a horizontal scroller for the covers on a phone (sideways scroll on a
  page whose test forbids it); the life-map screenshot as "the app itself"
  (the plate above already is the life map — the adventure page shows what
  the covers open into).
- **Would change our mind:** the five-second answers; the maintainer's
  phone.

## 2026-10-29 — A1 gains a desktop layout; the phone layout is unchanged

- **Chosen:** the maintainer, on a laptop: the page was the phone layout
  centred with empty margins. From 1024 px the hero sets in two columns —
  headline, subline, the action and its two small lines on the left, the
  life map on the right as large as the sheet allows — and both are in
  the first screen at 1280×800 (the plate's lower rule lands at 630 px).
  The three covers and the three steps each set in three columns; privacy
  and the way in keep a reading column of about 65 characters. The
  screenshot is capped at 52 rem so it reads as proof, not a second hero.
  Same copy; the phone layout is byte-identical in the DOM and unchanged
  on screen.
- **Rejected:** scaling the type up to fill the width (the phone layout
  centred, only larger); the plate at full sheet width under the words
  at desktop (what the first build did — the page's one thing led, but
  from below the fold).
- **Would change our mind:** the plate reading as a thumbnail on a wide
  laptop — then the left column narrows to 22 rem.

## 2026-10-29 — STOP 2 passed on sight; the landing built from A1

- **Chosen:** the maintainer's "this looks good" on the desktop pass;
  `landing/index.html` is now the A1 page generated from the same source
  as the mockup (the hero plate SVG, the three covers' route art as
  `/adventures` draws it, the approved copy), with the committed
  screenshot `docs/screens/phase16-cp2b-adventure-2.png` staged as
  `shots/app.png` by `deploy-landing.sh` (the three phase-9 shots are no
  longer referenced). `landing/how-its-built.html` is new: the four kinds
  explained, the rule, where the figures come from, where the data lives
  (the custody statement that left the landing), self-hosting; linked
  from the landing's footer, same tokens, zero script, placeholder
  substituted at deploy like the landing. `web/lib/landing-legend.test.ts`
  pins the landing's legend to `LEGEND_ENTRIES`, zero `<script>` and the
  `{ROADBOOK_DOMAIN}` placeholder on both pages (vitest 128). The landing
  README is rewritten for the two pages and the test.
- **Rejected:** keeping the deploy mapping for the three old screenshots
  "in case" (dead staging); a build step to share the stylesheet between
  the two pages (two files, the README's same-commit rule).
- **Would change our mind:** the three friends' five-second answers, still
  to come — a miss on "what does this do?" reopens the hero, not the build.

## 2026-10-29 — CP2b reading load, measured

`copy-count.cjs` on the built landing (read from disk, as the script does):

| page | before, phone (words / sentences / titles / terms) | CP2 trim | CP2b, phone | CP2b, desktop |
|---|---|---|---|---|
| landing | 851 / 73 / 0 / 31 | 597 / 53 / 0 / 25 | 390 / 47 / 0 / 10 | 399 / 49 / 0 / 11 |

The approved copy is 327 words; the script's 390 adds the legend's eight
words, the three covers' names, dates and distances, the PLATE chips and
the form's placeholder — what a person is shown. The vocabulary hits fell
from 31 to 10: the four kind names in the legend, "atlas", "export",
"timeline" and "stop" forms in the steps — none from the banned list.
