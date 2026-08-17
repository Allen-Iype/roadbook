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
