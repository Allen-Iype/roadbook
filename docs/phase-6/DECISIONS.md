# Phase 6 — decision log

Three lines per decision: what was chosen, what was rejected, what would
change our mind. Design decisions (direction, T1/T2, vocabulary, score
prominence, list rail) belong to the Stage B review and land in DESIGN.md;
the entries below are Stage B execution decisions.

## Mockup geometry comes from the demo compose instance

- **Chosen:** capture `/candidates/{1,2,3}/journey` from the running demo
  stack and render the real observed points, routed polylines, and air
  chords as inline SVG; every figure on a mockup traces to those responses.
- **Rejected:** hand-drawn illustrative geometry — it cannot stress the
  honesty channel, and it would put numbers on committed pages that the
  repository cannot reproduce (invariant 13).
- **Would change our mind:** nothing; this is invariant 13 applied to design
  artifacts.

## Natural Earth 10 m coastline, embedded as simplified paths

- **Chosen:** NE 10 m admin-0 Iceland (public domain), simplified and
  projected at build time into the mockup files (~10–30 KB per panel).
- **Rejected:** the repo's embedded 110 m edition — Iceland is 20 points at
  110 m and the Westfjords peninsula does not exist in it, and the detail
  screen is *about* the Westfjords.
- **Would change our mind:** if mockups had to regenerate from repo-only
  assets; they are one-shot review artifacts, so provenance in the README
  suffices.

## System font stacks in the mockups

- **Chosen:** Iowan Old Style/Palatino, system grotesk, and ui-monospace
  stacks so every page opens from disk with no network and no font files.
- **Rejected:** webfonts (network — the mockups must open file://) and
  data-URI embeds (license review and file weight for a review artifact).
- **Would change our mind:** Stage C, which should pin self-hosted faces —
  candidates are named on the token sheets.

## Honesty-channel overview panels use the pre-routing state

- **Chosen:** draw the Westfjords gaps as unknown chords in the T1/T2
  overview panels — the state a fresh install without a router serves.
- **Rejected:** routed-state-only panels — the demo has 0 unknown km after
  routing, and without unknown geometry T1 and T2 barely differ at
  overview, which is exactly the decision the panels exist to expose.
- **Would change our mind:** nothing; the state is real (invariant 7's null
  router), and the panels label it.

## Detail screens highlight stage/day 3, not 1

- **Chosen:** show the third day selected on both adventure mockups.
- **Rejected:** day 1 — its transit covers day 3's return along the same
  road, so the dimmed state is invisible underneath and the highlight
  pattern looks inert.
- **Would change our mind:** demo geometry where day 1 has visually
  exclusive geometry; the product behaviour (any day selectable) is
  unaffected.

## The two directions deliberately split the open review questions

- **Chosen:** A mocks "Stage" vocabulary + a visible list rail; B mocks
  "Day" + a summoned list — one rendered example of each answer to review
  questions 3 and 5.
- **Rejected:** mocking one default in both directions — it would collapse
  the comparison the review needs to make.
- **Would change our mind:** nothing; the split is scaffolding for the
  review, not a claim that vocabulary belongs to a direction.

---

Stage B review decisions (maintainer, 2026-08-08). Full statement in
DESIGN.md; recorded here at the moment of decision.

## Direction: A — Atlas plate

- **Chosen:** the Atlas plate direction, tokens per mockups/tokens-a.html.
- **Rejected:** Night expedition as the product's look — but its token
  sheet is kept, and an optional dark theme is parked in the backlog.
- **Would change our mind:** nothing chartered; a theme option is the
  recorded path for revisiting, not a redesign.

## Honesty channel: T2 — zoom-graduated emphasis

- **Chosen:** overview merges observed+routed into one known-ground ink;
  unknown and air keep their marks at every zoom; full split at detail.
- **Rejected:** T1 (always split) — the observed/routed hue pair carries
  little at country scale, and known-vs-unknown is the distinction the
  life map must not lose.
- **Would change our mind:** any state where an unknown leg renders in the
  known-ground style — that is the invariant-8 boundary T2 lives behind.

## Vocabulary: Day

- **Chosen:** "Day n · date · km" headings.
- **Rejected:** "Stage" — the rally register stays in the framing, not the
  labels; plain words win where guests read the screen.
- **Would change our mind:** nothing foreseen.

## Score: retired from the adventure cover

- **Chosen:** confirmed adventure pages carry no score; the candidates
  table and confirm-cell breakdown keep it.
- **Rejected:** quiet-marginalia and score-as-stat treatments — ranking is
  triage machinery, not part of the memory.
- **Would change our mind:** a real use for comparing confirmed adventures
  by score; none exists in the charter.

## Life-map list: summoned

- **Chosen:** map-maximal home; "List (n)" summons the adventure list
  overlay; keyboard/screen-reader/small-screen requirements in DESIGN §5.
- **Rejected:** the always-visible index rail from the A mockup.
- **Would change our mind:** evidence the summoned list hides adventures
  from real users (self-hoster feedback), since the map remains the
  primary path either way.
