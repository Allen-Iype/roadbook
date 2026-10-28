# Phase 16 research — reading load

Stage A of the phase 16 design process, on the phase 6 and phase 9 pattern:
the evidence, the measurements and the vocabulary inventory that ground the
Stage B treatment mockups. Nothing here is a decision; decisions happen at
the Stage B review and the Gate 1 brief, and land in DECISIONS.md.

## 0. The problem, pinned

Stated by the maintainer after the first live walk of the hosted product
(2026-10-28): the product is mostly words. People find them confusing and
do not read them. What they came for is the map and their routes; the words
stand between them and it.

Evidence in hand before this stage:

- The maintainer's own walk on 2026-10-28 stopped at "what is this plate
  written on each tile", and could not find the sign-in link on the landing
  page (fixed the same day). `docs/PLAN.md` backlog, first entry.
- The only non-operator to complete the whole loop (phase 10 pilot) stopped
  deciding candidates 10 of 79 in. His stated remedy inverted the curation
  model; the in-system remedies shipped in phase 11. The friction itself is
  partly reading friction: every card carries five figures and a score.
- A populated instance gives no introduction (phase 10 LEDGER finding):
  `/welcome` greets only zero-import instances.

Decisions the maintainer made at the planning STOP (2026-10-28):

- **Sequencing.** Implementation starts after phase 15 CP3 (durability)
  lands. This stage, the mockups and the brief run in parallel with CP3.
- **Direction.** The Atlas plate (`docs/phase-6/DESIGN.md`) stays the base
  and stays binding. The phase may put treatment options side by side as
  mockups rather than assume today's rendering of it is final. A change of
  direction (a dark theme, a different type system) is not in scope.
- **Evidence.** One or two watched walks (§1) before design work starts.
- **Branch.** All phase 16 work lives on `phase-16-ux`; main keeps taking
  phase 15 checkpoints.

What this phase is not: a feature phase. The target is a zero Go diff and
an untouched `openapi.yaml`, as phases 6, 9 and 14 were. If a treatment
needs a served field that does not exist, that is a named exception at a
STOP.

## 1. Watched-walk protocol

Written before any walk so every walk is logged the same way and the
closing walk compares with the opening ones. The private log lives in
`docs/private/pilot/LEDGER.md` under "Phase 16 — watched walks"; nothing
identifying a person enters it.

**Who.** One or two people who have not used the product. A person with a
Google Timeline export on their phone is best; a person without one walks
the demo path (a demo instance with the fictional dataset loaded, or the
committed `testdata/demo/demo.json` uploaded in front of them).

**Where.** Their own phone, on the hosted instance or a scratch instance
reachable from it. The phone is the primary funnel (phase 7 BRIEF §1.5);
a laptop walk is a second data point, not a substitute.

**Route.** Landing page → sign-in → `/welcome` → upload → wait for
detection → `/candidates`, decide at least three → the life map → one
adventure page, scroll to the end → one share link or one image download.
About twenty minutes. The observer does not help; if the person is stuck
for a minute, the observer notes where and then unsticks them with one
sentence, logged.

**Four fixed questions**, asked when the person reaches each thing, before
they have scrolled past it:

1. On `/adventures` or the life-map list: "What do you think the number on
   this tile is?" (the plate number)
2. On any cover: "What is this bar under the distance?" (the provenance bar)
3. On a candidate card: "What is this page asking you to do?"
4. On the adventure page, at the divergence line or flag: "What is this
   telling you?"

**What is logged**, per walk, in the LEDGER template:

- Read or skipped, per surface: did they stop and read, skim, or scroll
  straight past. Watch the thumb.
- Every question they asked, verbatim.
- Every misclick or dead end (tapped something that did nothing, went back).
- The answers to the four fixed questions, verbatim.
- Where they stopped if they stopped.
- At the end: "What is this product for?" in their words.
- Device, browser (in-app browser or not), the path taken (own export or
  demo), the time taken.

**What it is for.** The log turns "mostly words" into "these words, on this
surface, were skipped, and this term was misread". Stage B mocks against
that list. The same walk, with a different person, repeats at the phase's
close (CP4) as its acceptance test, alongside the counts in §2.

## 2. Reading load, measured

`web/scripts/copy-count.cjs` renders every page of a demo stack at phone and
desktop width and counts what a person is shown: words, sentences, elements
with hover-only `title` text, and hits on the vocabulary list in §4. It is
the phase's one number, taken before and after.

Reproduction (any demo stack; puppeteer-core installed in a scratch
directory, as for `docs/screens/capture.js`):

```
ROADBOOK_SCREENS_URL=http://127.0.0.1:3016 \
NODE_PATH=<scratch>/node_modules node web/scripts/copy-count.cjs
```

Baseline, 2026-10-28, the committed code at `ecd2c81` served by `next dev`
against a scratch demo API (three confirmed adventures, no undecided
candidates — so `/candidates` reads light here; on a fresh upload it shows
every candidate undecided, with the bulk-triage bar):

| page | phone 390 (words / sentences / titles / terms) | desktop 1280 |
|---|---|---|
| landing | 851 / 73 / 0 / 31 | 866 / 77 / 0 / 35 |
| welcome | 1105 / 80 / 0 / 56 | 1120 / 84 / 0 / 60 |
| home (life map) | 16 / 1 / 3 / 4 | 31 / 5 / 3 / 8 |
| adventures | 70 / 6 / 3 / 11 | 85 / 10 / 3 / 15 |
| candidates | 110 / 8 / 3 / 13 | 110 / 8 / 3 / 13 |
| adventure 1 | 605 / 58 / 6 / 83 | 605 / 58 / 6 / 83 |
| adventure 2 | 426 / 42 / 6 / 44 | 426 / 42 / 6 / 44 |
| adventure 3 | 547 / 53 / 6 / 73 | 547 / 53 / 6 / 73 |
| imports | 176 / 14 / 0 / 15 | 190 / 14 / 0 / 17 |

`/signin` is not in the table: with accounts off it redirects home. Read
from source it carries three to five sentences and one button.

Reading the table:

- **The front door is a thousand words.** `/welcome` and the landing page
  together are nearly two thousand words before a person has uploaded
  anything. They are the pages a stranger meets first, and they are the
  pages written most carefully: pitch, two phone walkthroughs, custody
  statements, caveats with verification dates. The care is the problem; a
  stranger on a phone reads none of it.
- **The adventure page speaks the pipeline's language.** Four to six
  hundred words for a three-day demo trip, and the vocabulary hits are the
  highest of any page: observed, routed, unknown, fix, dwell, window,
  source, divergence, reconstruction. A real multi-week trip adds a row per
  event. The map and the photos are the payoff; the left column is the
  reconstruction's own account of itself.
- **The map pages are already quiet.** The life map and `/adventures` carry
  under a hundred words. They are the model: the thing itself, a legend, a
  few figures.
- **Teaching is hover-only.** Every `title` count in the table is text a
  phone never shows. The legend's four descriptions are hidden below the
  `sm` breakpoint, so a phone sees four names and no explanation.

What the words are doing, by surface (read from source, 2026-10-28):

| surface | the words are mostly | visual element |
|---|---|---|
| landing | pitch; the four line kinds explained twice (legend, then "the honesty rule"); custody in three paragraphs; the way in | plate SVG, legend samples, three screenshots |
| `/welcome` | pitch; "what you need"; Android and iPhone walkthroughs with verification notes; photos section; "what happens next" in four steps; upload copy | plate SVG, legend |
| life map | empty and error states only; one shows a CLI command | the map |
| `/adventures` | one subtitle, the legend | covers |
| `/candidates` | run line ("Detection run 7 · 3 outliers dropped · FAR 100 km"), five figures per card, score breakdown with weights, keyboard hint | route thumbnail per card |
| adventure page | cover figures with provenance caveats; divergence line or box; summary list; one row per event with FIX / OBSERVED / ROUTED / DWELL / WINDOW chips; share dialog; export caption | the map, photos, provenance bar, legend |
| `/imports` | two upload panels with intros; a table with schema column names (Format · Window · Visits · Activities · Points · Raw · Skipped · New); the deletion dialog | none |

## 3. Teaching in place — what to look at

Products that put a categorical legend and figures in front of ordinary
people without prose. Three questions per reference, answered at Stage B
against their current public materials (links recorded then, as phase 6
did), not from memory here:

1. Where does the explanation live: inline beside the thing, on tap, or
   once on first visit?
2. What is shown before anything is told?
3. How is a caveat carried: a mark and one line, or a paragraph?

References to answer them on: Komoot (surface-type legend under a route —
already on record in phase 6 as field proof that texture plus legend reads
at a glance); Strava (activity figures with no explanation of how they were
measured, a tap for the rest); Polarsteps (the trip page is pictures first,
one line of text per step); Apple Maps and Google Maps (the legend is
nowhere and everything is still readable, because the encoding is
conventional — the atlas vocabulary is not conventional, which is exactly
why it needs teaching).

Anti-reference, kept from phase 6: Google Timeline's own UI, which explains
nothing and hides its confidence entirely. Invariant 8 is what separates
Roadbook from it; the phase cannot buy quiet by hiding.

## 4. Vocabulary inventory

Every term a newcomer meets, with the desktop hit count across all pages
from §2, sorted into load-bearing (the product's own vocabulary, to be
taught in place) and leakage (the pipeline's vocabulary, to be cut from
user pages and left to the CLI and the docs).

**Load-bearing — teach, do not soften** (the maintainer's instruction,
PLAN backlog 2026-10-28):

| term | hits | where first met | what it needs |
|---|---|---|---|
| plate, plate number | 15 | `/adventures` tiles, cover eyebrow | one line, once: "each adventure is a plate, numbered in the order you travelled" |
| observed / routed / unknown / air | 40 / 11 / 24 / — | the legend, every map | the legend's descriptions visible on phones; the sample line next to the word |
| provenance bar | 1 (the word); the bar itself on every cover | covers, life-map list | a caption on first sight; it has no label anywhere today |
| candidate | 13 | `/candidates`, header count | the page should say what it asks: "is this a trip you took?" |
| legend | — | every map page | it is the teaching element; it must not itself need teaching |
| stop, Day | 7 | adventure narrative | already plain words |

**Leakage — cut from user pages, keep in CLI and docs:**

| term | hits | where | note |
|---|---|---|---|
| fix, fixes | 42 | cover, narrative chips, legend ("recorded fixes") | the legend's own description leaks it; "recorded positions" says the same |
| dwell, dwelling | 43 | narrative rows, summary | "stayed" / "Stopped" |
| window (opens / closes / imported window) | 16 | cover truncation sentences, narrative | the truncation marks ◂ ▸ carry it; one line |
| inferred | 22 | legend, landing | kept in the legend's fixed wording (DESIGN §6); elsewhere "drawn along roads" |
| source, "as the source recorded it", source-asserted | 9 | mode line, summary | "by Google's record" or a mark |
| reconstruction, divergence, "ground reconstruction" | 6 | divergence box | the flag plus one line; the paragraph moves behind a disclosure |
| detection run, outliers dropped, FAR 100 km | 7 | `/candidates` header | operator figures; none helps a person decide |
| trace, raw, "N points (X trace + Y raw)" | 7 | cover points line | not a user figure |
| score, weights, "weight redistributed" | 3 | confirm box | stays in triage (DESIGN §4) but plain-worded |
| instance | 6 | `/welcome`, `/imports`, `/signin` | "this Roadbook" |
| sidecar, Takeout | 4 | `/welcome` photos, `/imports` | needed in the walkthrough only, explained there once |
| Format · Window · Visits · Activities · Points · Raw · Skipped · New | — | `/imports` columns | schema names on a user table |

The legend's fixed wording is a DESIGN §6 rule ("Observed — recorded fixes ·
Routed — inferred along roads · Unknown — straight line, nothing inferred ·
Air — great-circle arc"). "Recorded fixes" and "great-circle arc" are on the
leakage list. Changing the wording is a DESIGN amendment decided at the
Stage B STOP, with `e2e/legend-wording.spec.ts` and `lib/plate-image.ts`
(the exported image carries the same words) updated in the same change.
Nothing else about the legend — presence, four entries, the sample lines —
is open.

## 5. Principles for the brief (draft, confirmed at Gate 1)

1. **Show before tell.** The product can draw. An explanation of a concept
   starts with the thing itself — a route thumbnail, a legend sample, a
   plate — and one line.
2. **Progressive disclosure, with a floor.** One visible line; the rest on
   tap. The floor is the invariants: the legend present with its wording,
   a provenance bar beside every headline distance, the divergence flag
   visible. Those may shrink; they never hide.
3. **Teach where the term appears**, not on a glossary page. A person who
   has not met the word yet does not go looking for it.
4. **Pipeline language stays in the CLI and the docs.** Every figure on a
   user page is still CLI-reproducible (invariant 13); it is the words
   around the figure that change.
5. **A copy budget per surface, measured.** §2's table is the before; the
   LOG prints the after with the same command.
6. **Nothing hardcoded about any user.** Teaching copy names no region, no
   dataset, no person (invariant 9); the demo persona appears only in the
   mockups and the record.

## 6. Stage B plan — treatment mockups at a STOP

`docs/phase-16/mockups/`, the phase 6 and 9 convention: static HTML with
inline SVG, demo data only (geometry from a demo stack's API), system font
stacks, opens from `file://`, `index.html` as the hub listing the review
questions, a README with data provenance. The `frontend-design` skill is
loaded before any of it is designed. Each surface is mocked at phone width
first; the desktop version is the derived one.

| surface | treatment 1 | treatment 2 |
|---|---|---|
| adventure page (cover + Day 1) | **margin notes** — teaching lives in the plate margin as printed marginalia, the Atlas signature carrying the explanation | **disclosure** — figures first; caveats as marks that open on tap; narrative rows as glyph + one line |
| `/welcome` | **stepper** — one step at a time, each with a picture; the walkthroughs as screens, not paragraphs | **one page, folded** — the plate first, the walkthroughs collapsed, custody in one line with a disclosure |
| candidate card | figures cut to the three a person decides on, and the page asking its question | (one treatment; density exhibit at pilot scale as phase 9 did) |
| the teaching element | tap-to-explain on the term (popover) | a once-per-visit "how to read a plate" card, dismissable |

Optional exhibit, the maintainer's "explore options": one alternative
rendering of the plate margin and the cover's type scale, inside the Atlas
tokens. Not a dark theme (parked, DESIGN §7; a direction change per phase 9
BRIEF §3).

Review questions, to be listed in `index.html`:

1. Margin notes or disclosure for the adventure page — or margin for the
   cover and disclosure for the rows?
2. Does `/welcome` become a stepper, and does the stepper also serve
   `/imports` for a returning user?
3. Which teaching element, and does it appear on the shared view?
4. Does the legend's wording change (a DESIGN §6 amendment)?
5. What does the candidate card ask, in words, and which three figures stay?

## 7. Scope, provisionally sorted (the phase 9 buckets)

To be confirmed by the walks (§1) and the review (§6); the brief carries the
final sort.

**Refinement in system — brief, then build:**

- Copy diet on every surface against §2, cutting the §4 leakage list.
- The legend's descriptions visible at phone width.
- Caveats as mark + one line: the divergence box, "modes are guesses", the
  truncation sentences, the export caption.
- `/imports`: plain-word column headers and intros; CLI commands never on a
  hosted page.
- `/candidates`: the run line gone or folded; the score breakdown in plain
  words; the keyboard hint moved to where keyboards are.
- `landing/index.html`: the four kinds explained once; custody in one
  paragraph; the same copy-count discipline (the file is read from disk by
  the script).
- A caption for the provenance bar where it first appears.

**Mockup-gated new IA — decided at the Stage B STOP:**

- The teaching element.
- `/welcome` as a stepper.
- Narrative rows as glyph + line (the FIX / OBSERVED / ROUTED / DWELL /
  WINDOW chips re-expressed).
- Any new navigation element.

**Charter-level, excluded, named so leaving it out is on record:**

- Auto-confirm, or removing the confirmation step (PRODUCT.md: a
  confirmation step is not friction to be removed; declined at phase 11
  Gate 1).
- Hiding the legend, the provenance bar or any leg kind anywhere
  (invariant 8), including on the exported image and the overlay.
- Any figure the CLI cannot reproduce (invariant 13).
- A dark theme (parked); the unified timeline (its own mockup gate); auth
  surfaces beyond phase 13.

## 8. Sequencing, branch and verification

- Phase 15 CP3 continues on `main`. Phase 16 lives on `phase-16-ux`:
  `docs/phase-16/`, `web/`, `landing/`, nothing under `scripts/pilot/`.
  `docs/PLAN.md` is edited on `main`. The branch rebases after each phase 15
  checkpoint; it merges at Gate 1 (docs) and at each accepted checkpoint.
- Phase 15 CP4, the stranger proof, is better run on the improved UI; the
  brief proposes it waits for phase 16's first checkpoint. The maintainer
  decides, recorded in phase 15 DECISIONS.
- Proposed checkpoints: CP1 teaching element + legend on phones + the
  adventure page's copy diet → CP2 front door and landing → CP3 candidates
  and imports → CP4 close: the walk repeated, §2 recounted, LOG.md.
- Verification standard, inherited from phase 9: tsc, lint, build clean;
  vitest green; the Playwright suite green at 390/768/1280 and extended
  per surface (`legend-wording.spec.ts` changes only with a DESIGN
  amendment); goldens byte-identical and a cold `make test` with zero Go
  diff; `web/scripts/contrast.mjs` over any new token pair; screenshots
  `docs/screens/phase16-*` from demo data under 1 MB; `copy-count.cjs`
  before and after in the LOG.
