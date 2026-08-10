# Phase 6 log — the life map and the Atlas plate

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. Go and `api/openapi.yaml` were not
touched at any point in this phase — the closing `make test` ran cold and
green with a zero diff on the Go side, which is the "presentation only"
claim proven rather than asserted. Public numbers here come from the
committed demo dataset; nothing derived from private data is stated.

## What the phase does

The web app stops looking like a scaffold and becomes the product: the home
page is the life map, a confirmed adventure reads as a day-sliced
narrative, and every surface sits on one designed token system. Stage A
(research) and Stage B (mockup review) produced `RESEARCH.md`, the mockups,
and the binding `DESIGN.md`; Stage C built it in four checkpoints.

- **Atlas plate tokens (CP1).** DESIGN §6 lands once, as Tailwind 4
  `@theme` in `globals.css` — each name is simultaneously a CSS custom
  property and a utility class, so no second source of truth exists. The
  four leg inks are reserved: nothing else uses those hues at line weight,
  and kind is never hue alone (observed solid and widest, routed cased,
  unknown dashed, air round-dotted). Map code reads the same hexes from
  `lib/tokens.ts` because MapLibre paints WebGL and cannot see CSS
  variables. Fonts are committed woff2 (Source Serif 4 display, IBM Plex
  Mono for every figure; body stays a system sans) via `next/font/local` —
  no runtime font requests. One theme, deliberately; the scaffold's
  `prefers-color-scheme` switch was removed and a dark theme stays a
  parked backlog option.
- **The life map (CP2).** `/` is a full-viewport map of every confirmed
  adventure, bounds-fit, with floating chrome. The honesty channel at
  country zoom is T2: observed and routed merge into one known-ground ink
  at overview zooms and split as you zoom in, while unknown and air never
  graduate — their dash and color carry no zoom expression at all. That
  asymmetry is invariant 8 here, and it is pinned by a unit test over the
  exact layer objects the island feeds to `map.addLayer`
  (`lib/life-map-layers.test.ts` — vitest entered the project for this).
  Hover raises a whole adventure via feature-state on a promoted
  `adventure_id`; click navigates; the canvas is `aria-hidden` and
  stripped of focusable children, because the summoned list — a native
  `<dialog>` — is the accessible enumeration. The home server component
  fans out (candidates, then confirmed journeys in parallel); the measured
  cost (~1.6 ms per request, demo home ~28 KB gzipped) is why no aggregate
  endpoint was added and the API stayed untouched.
- **The Roadbook basemap (CP2).** A committed style JSON derived from
  OpenFreeMap Liberty by a committed script (`derive-map-style.mjs`):
  ground recolored to the token palette, POI and shield noise dropped,
  tiles/glyphs/sprites still OpenFreeMap — the README's tile-host note
  remains the product's one third-party fetch. `ROADBOOK_MAP_STYLE` still
  overrides for any operator.
- **The day narrative (CP3).** `sliceDays` in `lib/slice-days.ts` cuts a
  journey into civil days *in the journey's own recorded offsets* — dates
  read straight off the RFC 3339 strings, durations from instants — and
  returns headings, events, per-day km, and per-day leg/stop assignments.
  The narrative sections and the map highlight both consume that one
  output, so they cannot disagree. The midnight rule is the approved one:
  an event belongs to the civil day it starts in; a transit crossing
  midnight appears once with "overnight, ends Day n"; nothing is split,
  because a split would manufacture a position and a km figure inside
  inferred geometry. Events derive structurally from what the API already
  says: a stationary observed leg (one point, or exactly 0 km) is a fix; a
  moving leg is a transit; a gap and the dwell riding inside it are both
  listed; a day fully inside a multi-day dwell reads "no movement
  observed" — with "no observed position" when the stop's location is the
  (0,0) absent value; window edges frame the first and last day. Selecting
  a day repaints opacity on the existing map through a ref — the WebGL map
  is never rebuilt on click.
- **The cover (CP3).** Honest figures only: distance with its provenance
  bar and composition line, dates, days, fixes, countries (labelled as
  derived), truncation and divergence as words. No score — DESIGN §4
  retired it from the cover; it stays in the candidates table and the
  confirm cell. The plate number is derived (position among confirmed
  adventures in date order), never stored.
- **The detail plate (CP3).** The adventure map takes the full Atlas
  encoding — unlike the life map it keeps the four-way split at every
  zoom (DESIGN §2). The frame is the plate: double rule, margin bar with
  the plate label (`DAY n HIGHLIGHTED` when a day is selected), the shared
  fixed-wording legend, and a MapLibre scale bar.
- **The a11y pass (CP4).** Contrast was computed against the tokens, not
  eyeballed. Everything passes AA except the flag amber as running text
  (3.07:1) — so the phase's contrast policy is *amber marks, ink words*:
  every honesty warning is an amber ⚑/◂/▸ glyph (glyphs need 3:1) beside
  ink text. The audit also caught dark-scaffold leftovers that were
  near-invisible on paper: `text-amber-500/80` (1.9:1) and
  `text-emerald-400` (1.7:1) in the photos section, swept to tokens and
  the existing confirm green. A `prefers-reduced-motion` rule zeroes
  transitions and animations. Keyboard walks were driven, not assumed:
  home → List dialog → adventure link (CP2), and page top → day heading in
  five tabs, Enter toggling the highlight with a visible focus ring (CP4).

## What broke, and why the fixes took their form

- **The blank map that was not a map bug.** Next 16 dev treats
  `127.0.0.1` and `localhost` as different origins and silently blocks
  dev resources cross-origin: pages render but never hydrate — dead
  islands, blank map, zero console errors. Every CP1 map symptom traced
  to this; production builds were never affected. The fix is config, kept
  permanently: `allowedDevOrigins: ["127.0.0.1"]` plus
  `logging.browserToTerminal` (which is also how it was found — the
  browser console forwarded into the dev terminal). The lesson recorded:
  when dev-only breakage looks impossible, suspect the dev server's
  origin model before the code.
- **Null island, again.** Journeys with zero-point dwells serve stop
  location (0,0); the map drew them off West Africa and blew the fitted
  bounds out to half a hemisphere. Pre-phase-6 bug, found at CP1 via two
  reference screenshots. Fixed where the convention lives —
  `stopFeatures` skips zero locations, because (0,0) is absent, not a
  place — and CP3 carries the same convention into words: those dwells
  say "position not observed".
- **Screenshots that lied.** Puppeteer's `fullPage` mode composites
  beyond the viewport, and a WebGL canvas without `preserveDrawingBuffer`
  (MapLibre's default) yields only the sliver that intersected its last
  presented frame — blank or partial maps in the committed record while
  the live pages were fine. `capture.js` never uses `fullPage` now: it
  grows the viewport to the page height and takes a plain composited
  capture. An hour went to this masquerading as an app bug.
- **`next start` with standalone output.** Unsupported combination: 500s
  on static chunks and silently dead hydration. Local production tests
  run `node .next/standalone/server.js` or the compose image, and the
  trap is recorded so it is not rediscovered.
- **MapLibre expression constraints.** `["zoom"]` is only legal feeding a
  top-level `interpolate`, so the hover width multiplier lives inside
  each zoom stop's output rather than wrapping the expression. The
  transparent 16 px hit layer exists because 2–3 px inks are hopeless
  pointer targets and `queryRenderedFeatures` reads geometry buckets, not
  pixels.
- **The demo told the truth about fixes (CP3).** The plan assumed fix
  events would be synthesized from gap endpoints; the demo payload showed
  the assembler already emits stationary single-point observed legs — the
  airport-gate representation from phase 3. Deriving events structurally
  (no invented display thresholds) made the narrative match what the
  pipeline actually claims, including listing the Westfjords 80 km road
  gap *and* the 38 h dwell riding inside it.
- **Real data found the tie (CP3).** An overnight dwell ends 06:48 and
  the observed run departs 06:48; construction order put the departure
  first. A one-line comparator tie-break (a dwell's end precedes anything
  sharing its instant) with the case pinned in the table tests — found by
  reading the rendered page against the real archive, which is exactly
  what the real-data dev server is for.
- **Per-day km follows the rule, not the mockup.** The Stage B mockup
  showed the Westfjords days as 160/—/80 km; the approved start-day rule
  sums both road legs under Day 1 (240 km) because the second leg departs
  before midnight. The rule governs — the map highlight uses the same
  assignment by construction, and a heading that disagreed with the
  highlighted geometry would be the worse dishonesty. Flagged at CP3
  review and accepted.

## Verification

- 36 vitest tests green (invariant-8 layer regression; sliceDays tables:
  midnight, offsets, overnight, westward day-regression, ties, honest
  empty days; date-range formatting), `tsc` clean, lint zero errors,
  production build clean.
- Cold-cache `make test` green at close (store, backup, journey goldens,
  detect demo + archive regressions); `git diff` on `*.go`, `go.mod`,
  `api/openapi.yaml` empty across the phase.
- Acceptance on the compose demo instance throughout: Westfjords renders
  Day 1 / the honest Day 2 / Day 3; the air weekend draws both flights as
  dotted arcs with the routed legs cased; screenshots committed as the
  `docs/screens/` `phase6-baseline`, `phase6-cp2`, and `phase6-cp3` sets,
  demo data only.
- Keyboard-only walks driven in a real browser at CP2 and CP4; contrast
  ratios computed for every token pair in use.
