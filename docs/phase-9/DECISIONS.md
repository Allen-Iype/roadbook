# Phase 9 decisions

Written as decisions are made. Three lines each: what was chosen, what was
rejected, what would change our mind.

## 2026-08-17 — Phase 9 is UI refinement, not ingestion

- **Chosen:** the next phase is UI refinement ("cleaner UI, better
  outlook"), maintainer's call at the charter discussion.
- **Rejected:** the ingestion phase (photos-as-source / GPX) going next —
  it stays gated on audit returns, none of which have arrived.
- **Would change our mind:** an audit return arriving with a concrete
  ingestion need before this phase's Gate 1 passes.

## 2026-08-17 — Phase number 9

- **Chosen:** this phase takes the number 9, sequentially; the ingestion
  phase takes the next free number when it is chartered.
- **Rejected:** holding 9 for ingestion because earlier planning notes
  used that label prospectively — numbers follow charter order, not
  forecasts.
- **Would change our mind:** nothing; it is a label.

## 2026-08-17 — Every scope item sorted into three buckets

- **Chosen:** the brief sorts all work into refinement-in-system /
  mockup-gated new IA / charter-level-excluded, and the bucket determines
  the process (build, mockup-STOP, or exclusion on record). BRIEF §3.
- **Rejected:** treating the phase as one undifferentiated UI to-do list —
  that is how a new IA element (the grid) would slip in unmocked, or auth
  work would slip in unchartered.
- **Would change our mind:** nothing for this phase; the fork is the
  working agreement applied to UI work.

## 2026-08-17 — Gate 1 passed

- **Chosen:** BRIEF.md approved as written; all §5 recommendations stand
  (mock `/adventures` + summoned-list contrast, zero-km wording
  presentation-side, sweep before mockups). No itch-list items were added
  at review; poster/print stays held per §3.
- **Rejected:** widening scope at the gate without pilot evidence.
- **Would change our mind:** the first pilot report arriving — its items
  sort into §3 at the next checkpoint boundary.

## 2026-08-17 — CP1: one browser, three widths, no stack management

- **Chosen:** Chromium only, viewports 390/768/1280 as Playwright
  projects, suite read-only against an already-running stack
  (`ROADBOOK_E2E_URL`, default the compose demo on 127.0.0.1:3000); the
  suite never boots or mutates anything.
- **Rejected:** three browser engines (the assertions are CSS layout
  truths — engine spread buys nothing for three binary downloads) and a
  webServer block that boots compose (slow, and it would hide "which
  stack am I testing?" — explicit is safer with three pilot stacks up).
- **Would change our mind:** an engine-specific layout bug reaching a
  pilot user's browser.

## 2026-08-17 — CP1 finding: decided-state buttons miss the 44 px rule

- **Chosen:** the harness's first catch — "change" / "keep as is" measure
  40 px (text-xs line + py-3), because phase 7's verified 44 px only ever
  covered the undecided state's text-sm buttons on a scratch stack. Pinned
  as `test.fail()` with the reason in the spec; the CP2 sweep fixes the
  product and removes the marker.
- **Rejected:** fixing the product inside CP1 — the checkpoint's contract
  is "no product changes; the deliverable is the guard", and honouring it
  keeps the checkpoint reviewable as exactly that.
- **Would change our mind:** nothing; this is the harness doing its job.

## 2026-08-17 — Measure the DOM atomically, not per element

- **Chosen:** layout measurements happen in one `evaluateAll` snapshot;
  per-button `isVisible()`/`boundingBox()` round-trips are banned in this
  suite.
- **Rejected:** the per-element style — it raced React hydration and
  produced runs where every button read as hidden (so the assertions
  vacuously passed) and runs where they read 40 px (so it flaked red).
- **Would change our mind:** nothing; an atomic read is strictly more
  truthful about a settled page.

## 2026-08-17 — CP2: fix-glyph item resequenced by severity (maintainer)

- **Chosen:** at the CP2 discussion the maintainer re-read the "fix-glyph
  legibility" item as three distinct problems, ordered by severity: (1) a
  single-point observed leg painted NOTHING on the detail map — observed
  evidence rendering as absent, an invariant-8 bug, not legibility; (2)
  the flag amber measured 3.07:1 on paper and 2.91:1 on land — under the
  3:1 glyph bar on the divergence box's own background; (3) the narrative
  fix chip (observed red, 6.52:1 — no contrast problem) needed only size
  and weight. All ratios: `node web/scripts/contrast.mjs`.
- **Rejected:** treating the backlog line as one small legibility tweak.
- **Would change our mind:** nothing; the measurements decide.

## 2026-08-17 — CP2: fixes become dots on the detail map only

- **Chosen:** every fix (shared predicate `isFixLeg`, exported from
  lib/slice-days.ts so narrative and map cannot disagree) draws as a 4 px
  observed-ink dot under the black stop dots, joins the day-highlight
  dimming, and earns a "Fix" entry in the plate-margin legend.
- **Rejected:** fix dots on the life map — at T2 overview zoom they are
  clutter on merged known-ground, and the detail plate is where a reader
  interrogates evidence.
- **Would change our mind:** a zoomed-in life map reading where a missing
  gate dot misleads; revisit when the grid/covers work touches thumbnails.

## 2026-08-17 — CP2: flag amber darkened to #8A5C0B (DESIGN §6 amended)

- **Chosen:** darken the token rather than restyle every flag site: one
  value change fixes the glyph bar failure everywhere and clears text AA
  too (5.18:1 paper / 4.91:1 land — reproduction:
  `node web/scripts/contrast.mjs`). DESIGN.md §6 carries the dated
  amendment; the "amber marks, ink words" policy stays as written.
- **Rejected:** the CP4-precedent restyle (words to ink-2 per site) — more
  edits for a weaker result, and the marks would stay marginal at 3.07:1.
- **Would change our mind:** the darker ochre reading as brown/decoration
  in the maintainer's browser review.

## 2026-08-17 — CP2: zero-km routed legs read "Stationary gap"

- **Chosen:** presentation-side wording per BRIEF §5.2: below the named
  threshold STATIONARY_ROUTED_KM = 0.25 (day-narrative.tsx) a routed
  transit reads "Stationary gap — start and end coincide (routed 0.0 km)";
  the figure stays visible, muted. E2e regression:
  e2e/narrative-wording.spec.ts.
- **Rejected (deferred):** the pipeline-side min-chord floor — stays a
  carried item; the trigger (a distorted *figure*, not a noisy line of
  prose) is recorded in the BRIEF.
- **Would change our mind:** exactly that trigger.

## 2026-08-17 — CP2: what the render checks caught

- **Chosen:** two fixes landed from checks that had never run before: the
  home page's designed "API is not reachable" state was dead code — an
  unreachable API makes the generated client THROW (connection refused is
  a rejected fetch, not an HTTP {error}), landing on the generic error
  boundary; now caught and rendered as designed. And the divergence line
  dropped two inter-tag spaces ("62.7 km· Google's", "172.0 km(-63.5%)")
  — the phase 7 SWC trap; spacing now rides in string expressions.
- **Rejected:** trusting either surface without rendering it — both
  claims had been code-reviewed and both were wrong on screen.
- **Would change our mind:** nothing; this entry is the argument for the
  harness.

## 2026-08-17 — Playwright harness is CP1, before any product change

- **Chosen:** build the committed viewport suite first (maintainer's call
  at the brief discussion); the refinement sweep runs assertion-driven
  against it, and later checkpoints extend it as surfaces land.
- **Rejected:** harness at phase close (the draft's original order) — it
  would mean doing the small-screen pass by hand and then writing the
  harness that should have driven it, and risks the harness getting
  squeezed thin at close.
- **Would change our mind:** nothing foreseeable; the accepted cost is
  spec churn on surfaces CP4 later reshapes, and layout assertions mostly
  survive redesign.

## 2026-08-17 — CP3 mockup execution: what got mocked, and how

- **Chosen:** per BRIEF §5.1 — option (b) mocked fully
  (`docs/phase-9/mockups/adventures-grid.html`: grid of plate covers,
  plus exhibits for cover density, pilot-scale repetition, and the empty
  state), option (a) as the cheap contrast (`summoned-grid.html`, its
  costs stated on the page), option (c) not mocked. Cover art reuses the
  CP2 route-thumbnail projection verbatim; geometry captured from the
  demo API; covers propose plate numbers with chronological order, an
  amber flag mark when divergence is flagged, and no score (DESIGN §4).
- **Rejected:** proceeding to any grid code before the STOP passes; and
  waiting on the first pilot user's still-absent report — the maintainer
  chose to proceed, with the report folding in at review if it arrives
  (BRIEF §2 marks the grid provisional on it).
- **Would change our mind:** the review itself — that is what the gate
  is for; every proposal above is decided there, not here.

## 2026-08-17 — CP3 review: option (b), cover B, newest first; triage reopened

- **Chosen:** the grid lives at a new /adventures page (option b). Cover
  density B (name, dates, days, distance with provenance bar, countries,
  ⚑ mark when flagged). Order newest first, plate numbers kept and
  chronological — so display order and numbering deliberately disagree,
  the number is the atlas register, not the sort key. The summoned list
  stays exactly as it is: quick map navigation, while /adventures is the
  browsing surface.
- **Rejected:** option (a) the dialog-as-grid (its costs page made the
  case); chronological display order (the mockup's proposal — maintainer
  preferred recency); cover densities A and C.
- **Would change our mind:** on ordering, pilot feedback that the
  number/order disagreement confuses; nothing else foreseeable.

## 2026-08-17 — Triage-as-gallery reopened at the review; extra mockup added

- **Chosen:** the maintainer reopened BRIEF §3's "triage stays a table"
  at the CP3 review. Per the bucket rule that makes it a mockup-gated
  question: triage-gallery.html added to the CP3 set — T1 today's table,
  T2 table plus a shape-thumbnail column (the middle path), T3 full
  gallery — all three on undecided candidates with the decide
  affordances present, plus a 66-candidate density exhibit and a
  trade-off table. Assistant's recorded position: T1/T2 — triage is a
  deciding surface and the gallery dresses a work queue in the confirmed
  atlas's clothes; T2 captures the gallery's one real gain (shape recall
  while deciding) without losing the queue. Maintainer decides at the
  STOP.
- **Rejected:** building anything for triage before that decision; and
  settling the question in prose — it reopened precisely because the
  covers looked good on a screen, so a screen is where it gets decided.
- **Would change our mind:** the decision itself, at the STOP.

## 2026-08-17 — Triage: T3, the full gallery

- **Chosen:** triage becomes the gallery (maintainer's call at the
  mockup STOP, on triage-gallery.html): tiles with full-width route
  art, date headline, score chip, facts line, and the decide cell in
  the card. The queue properties the table carried (column-scannable
  scores, rows-per-screen) are consciously traded for shape-led recall.
- **Rejected:** T1 (today's table) and T2 (table + shape column) — the
  assistant's recorded recommendation; overruled at the STOP, which is
  the gate working as designed.
- **Would change our mind:** pilot evidence that deciding 66 candidates
  in a gallery is materially slower or error-prone — the trade-off
  table's rows are the things to watch; T2 remains the documented
  fallback and shares all of T3's parts.

## 2026-08-17 — CP4 execution choices

- **Chosen:** (1) the shell seam is a pass-through: both route-group
  layouts render children only — headers stay per-page because the life
  map floats its own chrome; the seam is file layout, which is all BRIEF
  §6 asks. (2) The reserved slot is ROADBOOK_INSTANCE_LABEL, read
  server-side at request time (compose can set it without a rebuild);
  the public shell never reads it — a visitor has no account to show.
  (3) Triage cards wear a single rule; the double-rule plate frame stays
  reserved for confirmed covers on /adventures, so the undecided queue
  never wears the atlas's clothes. (4) The card facts line corrects the
  mockup's "visits" figure — the mock had labelled the observation count
  "visits"; the product line uses days / km away / km track / stops and
  the repeat count ("2× visited") the table's Visits column actually
  carried. (5) /welcome's product-story evolution shipped copy-level
  (the atlas joins the pitch and the what-happens-next steps); a richer
  visual front door stays with the standalone marketing site as a launch
  item.
- **Rejected:** a shared header in the (app) layout (would force the
  life map's floating chrome into a shell it deliberately escapes);
  baking the instance label at build time (compose operators set env at
  run time).
- **Would change our mind:** on (5), maintainer appetite at review — the
  copy delta is deliberately minimal and says so here.
