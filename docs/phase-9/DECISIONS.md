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
