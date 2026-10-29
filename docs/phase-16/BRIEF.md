# Phase 16 brief — reading load

Fewer words between a person and their routes. The research stage
(`RESEARCH.md`) measured where the words are and the maintainer sorted the
findings (RESEARCH §9); this brief says what gets built, the real choices
inside it, and the order. Gate 1 is the maintainer's agreement to it.
`DESIGN.md` (phase 6) stays binding throughout; the one amendment this
brief proposes is named in §5C.

## 1. Concepts this phase introduces

**Reading load.** The words a page shows before the thing a person came
for. It is measured, not felt: `web/scripts/copy-count.cjs` counts visible
words, sentences, hover-only titles and vocabulary hits per page at phone
and desktop width (RESEARCH §2 is the baseline). The phase's close prints
the same table. Where in this codebase: the script; the LOG carries both
tables.

**The account of the trip and the account of the drawing.** The adventure
page today tells two stories in one column: what the trip was (days,
distance, stops, places, time away) and how the route was reconstructed
(fixes, trace and raw points, routed and unknown kilometres, Google's
figure, divergence). A person wants the first; invariants 8 and 13 require
the second to exist and be reachable. The phase separates them: the trip
leads, the drawing's account is one tap away, and the floor stays visible
(§1 next). Where: `web/components/adventure/adventure-view.tsx` (`Cover`,
`SummaryBlock`, `ModeLine`, `Divergence`).

**Progressive disclosure with a floor.** One visible line; the rest on
tap, in a native `<details>` so it works with no script, on a phone, with
a keyboard, and in the shared view. The floor is what may shrink but never
hide: the legend with its four kinds, the provenance bar under every
headline distance, the divergence flag when flagged, the truncation marks.
These are invariant 8 applied to a page; a disclosure that hid one of them
would be the "confident solid line" in words. Where: every surface §3
touches; `e2e/adventure.spec.ts` and `legend-wording.spec.ts` pin the
floor.

**Teaching in place.** A term is explained where it is first met, by a
control on the term itself, never on a glossary page. The mechanism is one
component (§5D) used for the small set of load-bearing terms RESEARCH §4
names: plate and plate number, the provenance bar, the four leg kinds,
candidate. Pipeline terms are not taught; they are removed from user pages.

**Review on the branch.** Phases 6 and 9 reviewed new page shapes as static
HTML mockups at a STOP before any code. This phase reviews them as working
code on the `phase-16-ux` branch, served by a scratch stack, at the same
STOP — the maintainer's call (RESEARCH §9, DECISIONS). What the gate
protects is unchanged: nothing reaches main before review. What changes is
the artifact reviewed: the next UI itself, on a phone, with demo data.
The cost is the one the mockup gate avoided — code written for a shape
that the review rejects — and the branch contains it; §5A records the
fallback.

## 2. Scope sources

- RESEARCH §2 (the measurements) and §4 (the vocabulary split).
- RESEARCH §9, the maintainer's response: the front door must make a
  person want the next step; the adventure page is the trip; the map
  pages are fine for now; the phone user has equal weight; the vocabulary
  split is agreed.
- `docs/PLAN.md` backlog: teaching the atlas vocabulary (absorbed here).
- Pilot evidence to come: the watched walks (RESEARCH §1) join at a
  checkpoint boundary, never mid-checkpoint (the phase 9 rule); a walk
  before CP2 shapes the front door, a walk after CP2 is the first "after".

## 3. Every item, sorted

The phase 9 buckets, binding.

**Refinement in system — brief, then build.**

- Adventure cover reorganised around the trip (§4 CP1).
- The drawing's account into a disclosure; the floor stays visible.
- Narrative rows as a leg-kind glyph plus one plain line; coordinates
  leave the rows (they stay in the CLI, `roadbook journey`).
- The legend's descriptions visible at phone width, in a compact form.
- Caveats as mark plus one line: divergence, truncation, "modes are
  guesses", the export caption.
- `/welcome` cut to the payoff, one primary action, walkthroughs folded
  (§4 CP2). `landing/index.html`: the four kinds explained once, custody
  in one paragraph, the hero's action first.
- `/imports`: plain-word column headers and intros; no CLI commands on a
  page a hosted account sees.
- `/candidates`: the run line folded into a disclosure; nothing else
  (RESEARCH §9.3).
- A one-line caption for the plate number and the provenance bar where
  each is first met (`/adventures`, the cover).

**Shape decisions reviewed on the branch at a STOP (the mockup bucket,
carried as code — §1).**

- The teaching control (§5D) and which terms carry it.
- The folded front door's structure and its single primary action (§5E).
- The narrative row's form (glyph + line).
- The legend's wording (§5C — a DESIGN amendment, decided at the CP1
  STOP).

**Charter-level, excluded, on record.**

- Auto-confirm or any removal of the confirmation step (PRODUCT.md).
- Hiding the legend, the provenance bar or a leg kind anywhere, including
  the exported image and the overlay (invariant 8).
- Any figure the CLI cannot reproduce (invariant 13); any new served
  field (zero Go diff, `openapi.yaml` untouched — a need for one is a
  named exception at a STOP, as the cap was in phase 15).
- The life map, `/adventures` and `/candidates` beyond the captions above
  (deferred to the next iteration, RESEARCH §9.3).
- A dark theme (parked), the unified timeline (own gate), auth surfaces
  beyond phase 13, a first-visit tour (state machinery for one sentence).

## 4. What gets built — checkpoints

Every checkpoint ends at a STOP on a scratch demo stack
(`docker compose -p roadbook-p16 …`, phone width first), merges to main
only when accepted, and extends the Playwright suite before it is called
done.

**CP1 — the adventure page.** `adventure-view.tsx`, `day-narrative.tsx`,
`legend.tsx`, `lib/legend.ts`, a new `components/term.tsx`.

- Cover: eyebrow, name, dates · through (countries, regions) as today;
  then the trip block as the hero — days, the headline kilometres with
  the provenance bar under it and its caption ("how much of this was
  recorded, how much drawn in"), time away, on the move, stops, farthest
  from home (owner only, as today). The kind-kilometre line, the points
  line, Google's figure, the mode line and the divergence figures move
  into one `<details>`: "How this route was drawn". When the divergence
  is flagged, the flag and one line stay outside it.
- Narrative: each row is the legend's line sample for its kind (the
  glyph carries the channel), a time, and one line in the traveller's
  words — "Recorded, 82 km", "Drawn along roads, 160 km", "Not recorded,
  straight line 40 km", "Flight, 300 km", "Stayed 3 h", "Journey begins",
  "Journey ends". Overnight and truncation notes stay, shortened.
  Coordinates leave the rows. Day headings unchanged ("Day 3 · Sunday 24
  May · 80 km", DESIGN §3).

  *Amended at the CP1 STOP (2026-10-29).* The maintainer rejected the
  per-event row as a shape: it is still the reconstruction's account.
  A day is now a heading (its kilometres over a provenance bar), one
  line in the traveller's words (the record's edges, an overnight, the
  count of stays, "nothing recorded"), and a row per stay. Transit and
  position rows leave the page; `roadbook journey` keeps every leg. The
  "How this route was drawn" disclosure moves to the foot of the page.
  What a day cannot yet say — which regions it crossed, where each stay
  was — needs data the system does not hold (the Timeline export carries
  no place names; nothing stored names a stop); that is phase 17
  "places", briefed separately, and the day line and stay row carry its
  two slots. DECISIONS 2026-10-29 has the inventory's conclusion.
- Legend: descriptions visible at phone width in a compact two-column
  form; wording per §5C.
- Teaching control on: PLATE n in the eyebrow and the plate margin, the
  provenance bar caption, the four legend entries.
- The shared view inherits all of it by construction (same component).
  The exported image's margin reads `lib/legend.ts`, so a wording change
  reaches it; `lib/plate-image.test.ts` updates in the same commit.
- Suite: `adventure.spec.ts` gains the floor assertions (legend, bar,
  disclosure closed by default, flag visible when flagged);
  `legend-wording.spec.ts` changes to "visible at every width";
  `narrative-wording.spec.ts` regex follows the new wording.
- STOP: the maintainer on a phone; §5C decided here.

**CP2 — the front door.** `app/(public)/welcome/page.tsx`,
`landing/index.html`, `lib/rejection-anchors.ts` only if a section id
changes (the vitest walk is the regression).

- `/welcome`: the plate first with one line; "your own, in three steps"
  as three short lines (export · upload · confirm) with the one primary
  action, the Timeline file picker, directly under them; "Don't have the
  file yet?" opens the walkthroughs as a `<details>` per platform, with
  the verification stamps kept; "Start from photos" as the second door,
  one line and its picker; what the right file looks like, custody and
  what happens next folded. Every `WELCOME_SECTIONS` id keeps a real
  target; a rejection link that targets a folded section opens it
  (one small effect on the hash, the page stays otherwise static).
- Landing: the hero keeps the plate and the sign-in action; the four
  kinds appear once (the legend), "the honesty rule" becomes one
  sentence under it; custody one paragraph with a disclosure for the
  rest; "the way in" unchanged in substance.
- Suite: `loop-pages.spec.ts` welcome checks follow the new structure
  (anchors, both pickers present, primary action is a 44 px target at
  390).
- STOP: the maintainer on a phone from the landing. On acceptance the
  brief proposes the host deploys CP1+CP2 and phase 15 CP4 (the stranger
  proof) runs on them — the maintainer's call, logged in phase 15
  DECISIONS.

**CP3 — the rest of the loop.** `/imports` headers and intros; the
`/candidates` run line folded; the plate-number caption on `/adventures`.
Copy diet pass over every surface against RESEARCH §4 with the script
run after each. Suite extended where structure changed.

**CP4 — close.** The watched walk repeated on a phone on the deployed
UI; `copy-count.cjs` after-table in the LOG beside the baseline; screenshots
`docs/screens/phase16-cp4-*` (and `capture.js` PAGES unchanged, no new
URLs); DESIGN.md §6 amendment recorded if §5C passed; README's walkthrough
paragraphs follow the new pages; LOG.md.

## 5. The real choices

**A — The review vehicle: mockups or code.** *Chosen (maintainer, RESEARCH
§9):* code on the branch, reviewed at the STOP on a scratch stack.
*Rejected:* static HTML mockups first — the research already fixed the
treatments' direction (§9.1, §9.2), and the shape questions left are ones
a phone in hand answers better than a static page. *Fallback, recorded
now:* if the CP1 STOP rejects the shape rather than its details, CP2
returns to the mockup gate before its code.

**B — Where the drawing's account goes.** *Recommended:* a `<details>`
on the page, closed by default, the flag outside it. *Rejected:* a
separate "how this was drawn" page (a navigation element, a new URL, a
second place for the same figures); deleting it (invariant 13 wants every
figure reproducible, and these are the figures the CLI prints). *Would
change our mind:* a walker opening the disclosure and being confused by
it — then its own copy gets the diet, not its placement.

**C — The legend's wording.** *Recommended:* amend DESIGN §6 to plain
words that keep the four names: "Observed — your phone recorded it ·
Routed — drawn along roads · Unknown — nothing recorded, straight line ·
Air — a flight". The legend is the product's one teaching fixture, and
its current descriptions carry two of the pipeline terms the phase removes
("recorded fixes", "great-circle arc"). The four kind names, the sample
lines, the order and the legend's presence everywhere do not change; the
exported image and overlay take the new wording through `lib/legend.ts`.
*Rejected:* keeping the wording and teaching "fix" and "great-circle"
instead — teaching a term the page could simply not use. *Would change
our mind:* the maintainer preferring the atlas register of the current
descriptions; then only "recorded fixes" → "recorded positions" changes.
Decided at the CP1 STOP, logged as a dated amendment in DESIGN §6 with the
phase 9 flag-amber precedent.

**D — The teaching control.** *Recommended:* one `Term` component: the
term rendered as a `<button>` with a dotted underline, opening a short
note (one or two sentences, no figures) as a native popover
(`popover` attribute, `popovertarget`), falling back to an inline
`<details>` where the attribute is unsupported; dismiss by tap outside or
Escape; 44 px hit area at phone width; the note text lives in one module
beside `lib/legend.ts` so the CLI docs and the page cannot drift.
*Rejected:* a once-per-visit card ("how to read a plate") — state to
remember the dismissal, a cookie or storage question, and it teaches
before the person has met the term; hover tooltips — invisible on touch,
which is the problem being fixed; a glossary page — nobody arrives there.
*Would change our mind:* the popover attribute misbehaving in the in-app
browsers the pilot phones use (WhatsApp's, Instagram's); the `<details>`
fallback then becomes the only form.

**E — The front door's form.** *Recommended:* one folded page with one
primary action, phone-first. *Rejected:* a stepper — it hides the upload
control behind steps for a returning person with the file in hand, and
needs client state on a page that is deliberately static (phase 7
DECISIONS); the research's stepper treatment is retired on that ground.
*Would change our mind:* a walker who cannot find the file picker on the
folded page.

## 6. What stays fixed

The Atlas plate (DESIGN §1–§6 except the §6 wording in §5C); "Day" not
"Stage"; no score on a cover; the summoned list; no home markers; the
honesty channel on every map at every zoom; the fixed legend presence;
every figure on a page printed by `roadbook journey -candidate N`; the
shared view showing exactly what the owner sees minus the home-relative
figure; `/welcome` static and ungated; zero Go diff; `openapi.yaml`
untouched; real data never near the record.

## 7. Verification standard

Per checkpoint: `tsc`, lint (0 errors), `next build` clean; vitest green;
the Playwright suite green at 390/768/1280 against the scratch stack, the
phone project first; the floor assertions in `adventure.spec.ts`;
`copy-count.cjs` run and the table kept in DECISIONS for that checkpoint;
`web/scripts/contrast.mjs` over any new token pair (none expected);
screenshots at CP4 only. At close: cold `make test` with zero Go diff;
goldens and demo pins byte-identical (nothing in the pure core is
touched); the walk on a phone; LOG.md written — the phase is not complete
without it.

## 8. Sequencing and the branch

Phase 15 CP3 landed on main (2026-10-28); implementation may begin. All
work on `phase-16-ux`; `docs/PLAN.md` edited on main; the branch rebases
after every main commit and merges at each accepted checkpoint. The host
pulls main only. The `frontend-design` skill is loaded before CP1's first
component is designed.
