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

## Life-map data path: server-component fan-out, no new endpoint

- **Chosen:** the home server component calls GET /candidates then the
  confirmed journeys in parallel; openapi.yaml is untouched this phase.
- **Rejected:** a GET /adventures/geometry aggregate — measured on the demo
  instance at ~1.6 ms/request and ~28 KB gzipped for all three journeys,
  the fan-out needs nothing from a new API surface at charter scale.
- **Would change our mind:** a real instance whose home payload exceeds
  ~1 MB gzipped or blocks render >500 ms on the fan-out (trigger recorded
  in BRIEF §2).

---

Stage C brief choices (maintainer approved recommendations, 2026-08-08).

## Midnight rule: events belong to the day they start in

- **Chosen:** a transit crossing midnight appears once under its start day
  with an "overnight" note; per-day km sums by start day; the map highlight
  uses the same assignment (one sliceDays function feeds both).
- **Rejected:** splitting legs at midnight — it manufactures a synthetic
  position and a proportional km split inside inferred geometry.
- **Would change our mind:** a real journey where start-day assignment
  makes a day's total actively misleading; none in the corpus.

## Basemap: Roadbook-tuned light style JSON as default

- **Chosen:** Maputnik edit of OpenFreeMap Liberty muted to the token
  grounds, committed under web/public/, ROADBOOK_MAP_STYLE still overrides;
  tiles/glyphs keep pointing at OpenFreeMap.
- **Rejected:** raw Liberty (palette competes with the leg inks) and
  Positron (upstream-abandoned).
- **Would change our mind:** timebox risk in BRIEF §5 — if style work
  drags, ship Liberty and land the custom style later in the phase.

## Summoned list: the native dialog element, not shadcn (CP2, 2026-08-10)

- **Chosen:** the platform `<dialog>` with `showModal()` — modal focus
  trap, Escape, backdrop, and focus-return to the opener are browser
  behaviour, specified and shipped everywhere Roadbook runs; the whole
  component is ~70 lines with zero new dependencies.
- **Rejected:** installing shadcn for its Sheet. The BRIEF assumed shadcn
  was "already in the stack", but web/package.json carries none of it —
  the CLAUDE.md stack line named it as the sanctioned idiom, and pulling
  a component system plus Radix into the bundle for one overlay inverts
  its cost. Also rejected: a hand-rolled toggled div (the focus
  management `<dialog>` gives for free is exactly what gets forgotten).
- **Would change our mind:** the second or third overlay surface (menus,
  sheets, toasts) — that is the point to install shadcn properly and fold
  this dialog into its idiom.

## UI screenshots committed per checkpoint, demo instance only

- **Chosen:** a committed visual record of the UI under `docs/screens/`,
  phase-prefixed sets (`phase6-baseline`, `phase6-cp2`, …) so the practice
  outlives this phase — captured exclusively from the compose demo
  instance (fictional data) by the committed `capture.js`, each file
  under 1 MB.
- **Rejected:** screenshots from the real-data dev server (data safety —
  they show real places and cannot enter git); no record at all (the
  pre-phase-6 look becomes unrecoverable from any running instance once
  the web image is rebuilt at CP2 acceptance); a phase-scoped
  `docs/phase-6/screens/` (maintainer: a permanent top-level home whose
  old sets can simply be deleted if the repo gets heavy — they stay
  regenerable from the era's commit).
- **Would change our mind:** repository weight — the remedy is deleting
  old sets, recorded in `docs/screens/README.md`.

## Fonts: Source Serif 4 display + IBM Plex Mono data, system body

- **Chosen:** OFL woff2 files committed, loaded via next/font/local; body
  text stays a system sans stack.
- **Rejected:** Google Fonts at runtime (network + privacy), Iowan Old
  Style (not redistributable), a third committed face for body.
- **Would change our mind:** nothing foreseen; faces are swappable behind
  the two font tokens.
