# Phase 9 log — UI refinement

Closed 2026-08-18. BRIEF.md, DECISIONS.md, and this log are the phase's
three artifacts. `docs/phase-6/DESIGN.md` (as amended §6) was binding
throughout; the Go side was untouched from first commit to last — the
close-out `git diff` across the phase's commits shows zero `.go` lines,
which is the "presentation only" claim proven rather than asserted.

## What the phase built

- **CP1 — the Playwright layout harness** (`web/e2e/`, three viewport
  projects at 390/768/1280, read-only against a running stack). Built
  first, deliberately: the refinement sweep then ran assertion-driven.
  It earned its keep on day one by measuring the decided-state
  "change" / "keep as is" buttons at 40 px — phase 7's 44 px proof had
  only ever seen the undecided state's larger buttons.
- **CP2 — the refinement sweep**: map fix dots (one `isFixLeg` predicate
  shared by narrative and map — a latent invariant-8 bug, 1-point
  observed legs painted nothing); flag amber re-inked #B97F10 → #8A5C0B
  (the old ink measured 3.07:1 on paper, under the 3:1 glyph bar;
  `web/scripts/contrast.mjs` is the reproduction; DESIGN §6 carries the
  dated amendment); 44 px targets everywhere the harness could reach;
  stationary-gap wording (presentation-side, BRIEF §5.2's recommendation);
  the hover-card route thumbnail (`lib/route-thumb.ts`, pure SVG); and
  the first-ever render of every empty state — which found the home
  page's designed API-down state was dead code (below).
- **CP3 — the mockup gate** (`docs/phase-9/mockups/`): option (b) mocked
  fully, option (a) as contrast, option (c) refused. The review decided:
  a new `/adventures` page; cover density B; newest first with
  chronological plate numbers (the number is the atlas register, not the
  sort key); the summoned list unchanged. The review also reopened
  "triage stays a table" — handled per the bucket rule with a
  supplementary mockup (T1 table / T2 shape column / T3 gallery, on
  undecided candidates, with a 66-candidate density exhibit), on which
  the maintainer chose T3 over the recorded T1/T2 recommendation. The
  gate worked exactly as designed: the question reopened because covers
  looked good on a screen, so a screen is where it was decided.
- **CP4 — shells, atlas, gallery**: route groups `(public)`/`(app)` land
  the auth-ready seam as file layout (both layouts are pass-throughs —
  the constraint shapes where files live, never what gets built); the
  reserved header slot is `ROADBOOK_INSTANCE_LABEL`, request-time,
  app-shell only; `/adventures` renders plate covers exactly as decided;
  triage became the T3 gallery with the unchanged `DecideCell` island in
  each card (cards wear a single rule — the double-rule plate frame stays
  reserved for confirmed covers); `/welcome` took a copy-level story
  delta (the atlas joins the pitch), with the richer visual front door
  deliberately left to the marketing-site launch item.
- **CP5 — close**: this log; README walkthrough updated for both new
  surfaces; `docs/screens/phase9-cp4-*` captured from the demo instance
  (eight pages — `/welcome` and `/adventures` join the record);
  cold-cache `make test` green (all DB suites, both goldens, demo and
  archive regressions, 62 vitest); e2e 76 passed / 8 viewport-skips.

## What broke, and why each fix took its form

- **The designed API-down state was unreachable** (CP2). openapi-fetch's
  `{error}` covers HTTP responses only; a connection-refused fetch
  *throws*, so the page's designed copy for "API not reachable" could
  never render — the throw landed on the generic error boundary. Fix:
  catch to `null` and branch on it (`safe()` in the home page, `.catch(()
  => null)` elsewhere), because the distinction that matters is "no
  answer" versus "an HTTP answer", and only a value can carry it into
  JSX. Both new CP4 pages were built with the catch from the start.
- **Playwright measurements race hydration** (CP1, rule recorded in
  DECISIONS): per-element `isVisible()`/`boundingBox()` round-trips
  produced both vacuous passes and flaky reds; measurement became one
  atomic `evaluateAll` snapshot per assertion.
- **Focus set before hydration is dropped** (CP5, the same trap in a new
  costume). The keyboard spec focused a cover link immediately after
  `goto`; React's reconciliation replaced the node and focus fell back to
  `<body>` — `clickUntil`'s sibling. Fix: re-issue focus until it sticks,
  with the comment naming the trap; a human tabbing after paint never
  races this, so the product needed no change.
- **SWC still drops inter-tag spaces** (CP2, third occurrence in project
  history): the divergence line rendered "62.7 km· Google's". Spacing now
  rides inside string expressions on that line; the phase 7 log's `{" "}`
  idiom remains the general rule.
- **The mockup's facts line mislabelled a figure**: "91 visits" was the
  observation count; the table's Visits column had always meant repeat
  visits to the destination. The product card uses the correct fields —
  a reminder that mockups are review artifacts, and the schema wins.

## Verification (BRIEF §7)

tsc, lint (0 errors), 62 vitest, prod build clean with every URL
unchanged; e2e green at every checkpoint from CP1 on, extended to each
new surface as it landed (57 → 76 passing); cold `make test` green with
zero Go diff; goldens byte-identical; keyboard walk on the grid is now a
committed spec, not a session ritual; no new color pairs were introduced
after CP2's flag re-ink (every new surface composes existing tokens);
no new motion (`prefers-reduced-motion` unaffected); screenshots in
`docs/screens/`, demo data only. Acceptance: CP2 and CP4 in the
maintainer's browser; CP3 at the mockup review.

## Carried out of the phase

- The first pilot user's report never arrived; the grid shipped shaped
  without it (BRIEF §2's provisional items were decided at the review).
  Standing fallback, recorded with the T3 decision: if deciding 66
  candidates in a gallery proves slow or error-prone in pilot use, T2
  (table + shape column) is the documented retreat and shares all of
  T3's parts.
- Poster/print view: still held for pilot evidence. Dark theme: still
  parked (DESIGN §7). Unified timeline: still deferred behind its own
  mockup gate.
- The iPhone walkthrough stamp on `/welcome` still waits on the friend's
  device report (phase 8 side-item, unchanged).
