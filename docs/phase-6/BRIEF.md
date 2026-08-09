# Phase 6 — implementation brief (Stage C)

Builds DESIGN.md. Scope: the home page becomes the life map, the adventure
detail becomes a day-sliced narrative, and the whole web app moves onto the
Atlas plate token system. The Go side is untouched — §2 records the
measurement that keeps it that way. The confirm/dismiss flow, photo upload,
and imports keep their behaviour; they are restyled, not rebuilt.

## 0. Concepts this phase introduces

**Design tokens as CSS, via Tailwind 4 `@theme`.** Tailwind 4 is configured
in CSS, not JavaScript: an `@theme { --color-observed: #A81E22; … }` block
in `globals.css` declares tokens that become both CSS custom properties and
Tailwind utilities (`text-observed`, `bg-paper`). DESIGN §6 lands there
once, and every surface reads from it. Rejected: a parallel hand-rolled
variables file (two sources of truth) and per-component hex literals (the
phase-3 lesson about named parameters, applied to color).

**Self-hosted fonts with `next/font/local`.** Font files live in the repo,
are served from our origin, and Next generates the `@font-face` CSS with
size-adjusted fallbacks at build time (no layout shift, no request to any
third party). Rejected: Google Fonts at runtime — a network dependency and
a privacy leak on a self-hosted product; the tile-server note in the README
is already the one deliberate exception, and it stays the only one.

**A full-viewport map page.** The life map is the page: the map island
fills the viewport and the header, legend, and list control float above it.
This is new relative to the framed map on the detail page — the layout is
`position:fixed` layers over the island, and the island must own resize
handling. The detail page keeps its in-flow framed plate.

**Zoom-interpolated paint = T2.** MapLibre paint properties accept
expressions evaluated per zoom:
`"line-color": ["interpolate", ["linear"], ["zoom"], 6, KNOWN_INK, 9, OBSERVED_RED]`.
The life map's observed and routed layers converge on the known-ground ink
below a threshold zoom and split as you zoom in — one layer set, no zoom
handlers, no duplicate geometry. Unknown and air layers carry no such
expression; their dash and color are constant at every zoom. That asymmetry
IS invariant 8 here, so it gets a regression test: a unit test over the
style-layer objects asserting that no color stop of the unknown or air
layers ever equals the known-ground ink, and that their dash arrays are
present at all zooms. Rejected: two layer sets toggled at a zoom breakpoint
(a visible pop, double the layers) and client-side restyling on `zoom`
events (work the style engine already does declaratively).

**Map-as-navigation mechanics.** Each adventure's legs become features
carrying `adventure_id`. Hover uses `queryRenderedFeatures` + feature-state
to raise the card and thicken the route; click navigates. The map is
`aria-hidden` except as a pointer surface: the summoned list is the
accessible enumeration (DESIGN §5), so keyboard and screen-reader users
never need the canvas. Rejected: DOM markers per adventure (phase 4 used
DOM markers for photos — right for point glyphs, wrong for route hit areas).

**The summoned list as a real dialog.** The "List (n)" control opens a
focus-managed panel: focus moves in, Escape and backdrop close it, focus
returns to the control. shadcn's Sheet (Radix under it) does exactly this
and is already in the stack. Rejected: a hand-rolled toggled div (focus
management is precisely the part that gets forgotten) and the native
`<dialog>` element (fine, but Sheet matches the existing component idiom).

**Day slicing, client-side, pure.** A journey's legs and stops already
carry instants with their recorded UTC offsets. `sliceDays(journey)` in
`web/src/lib/` cuts them into civil days in the journey's own offsets and
returns day headings, events, and per-day distance — one pure function
feeding both the narrative and the map highlight, so they cannot disagree.
It gets unit tests (vitest, dev-dependency only — the first web test
runner; the pure-function seam is exactly what it exists for). Rejected:
slicing in Go (presentation arithmetic does not belong in the data
contract) and testing via the page (a date-boundary bug deserves a table
test, not a screenshot).

## 1. What gets built

- `/` becomes the life map: bounds-fit union of confirmed adventures'
  routes, T2 paint, hover card (name, dates, distance + provenance bar,
  route thumbnail), click-through to the adventure, corner wordmark, quiet
  workbench links ("Candidates — n undecided" · "Imports"), summoned list,
  persistent plate-margin legend, and the two empty states as invitations
  (no imports → quickstart pointer; imports but no confirmations →
  candidates pointer with count).
- `/candidates` (new route) receives the existing triage table unchanged in
  behaviour — decide cell, score column, breakdown — restyled.
- `/adventure/[id]` becomes cover + plate + day narrative: honest distance
  with provenance bar, dates, days, fixes, countries, divergence flag and
  truncation flags as words, **no score** (DESIGN §4); days per DESIGN §3
  with fixes, transits (routed km with straight-line figure), dwells —
  including position-unobserved dwells stated as such; selecting a day
  highlights its geometry and dims the rest; photos and stops keep their
  placements, restyled.
- App-wide: tokens, fonts, legend and provenance-bar components, plate
  framing on maps, `/imports` restyled.

**Not in scope:** any Go or `openapi.yaml` change (§2), theming machinery
(DESIGN §7), poster/print view, elevation, coverage anything.

## 2. The measured decision: life-map data path

Measured on the demo compose instance (2026-08-08, `docker compose exec
api`, warm): 200 sequential requests in 0.32 s — **~1.6 ms per request**;
a full home fan-out (candidates + 3 journeys) ≈ 6.4 ms sequential. Payload
for all three demo journeys: **141 KB raw, ~28 KB gzipped** (5:1).
Extrapolated to charter scale ("tens", ~30 confirmed): ~50 ms sequential —
less with `Promise.all` — and roughly 300 KB gzipped to the browser.

**Decision: the home server component fans out** — `GET /candidates`, then
the confirmed ids' journeys in parallel — and no aggregate endpoint is
added. Revisit trigger, recorded here: a real instance where the home
payload exceeds ~1 MB gzipped or render blocks >500 ms on the fan-out;
until measured, the API stays as it is.

## 3. The three real choices

**A. Midnight rule (day slicing).** *Recommendation: an event belongs to
the civil day it starts in.* A transit that crosses midnight appears once,
under its start day, with an "overnight, ends {day}" note; per-day
distances sum leg distances by start day; the map highlight uses the same
assignment by construction. Rejected: splitting legs at midnight — it
manufactures a synthetic position inside inferred geometry and invents a
proportional km split; fabricated precision on the product's most
honesty-sensitive page. The Westfjords dwell (Fri 20:00 → Sun 10:00)
renders as: dwell begins under Day 1, "dwelling, no movement observed" as
Day 2's whole content, dwell ends under Day 3.

**B. Basemap.** *Recommendation: a Roadbook-tuned light style JSON,
committed and served from `web/public/`, as the new default.* Start from
OpenFreeMap Liberty's style, in Maputnik: desaturate land/sea to the token
ground colors, drop POI and road-shield noise, keep place labels quiet;
tiles and glyphs keep pointing at OpenFreeMap (unchanged external surface —
the README's tile note already covers it). `ROADBOOK_MAP_STYLE` still
overrides for any operator. Rejected: raw Liberty (its saturated palette
competes with the leg inks — exactly what RESEARCH §4 forbids the basemap
to do) and Positron (upstream-abandoned, and restyling it is the same work
as restyling Liberty without the maintained base).

**C. Fonts.** *Recommendation: Source Serif 4 for display, IBM Plex Mono
for data, system sans for body.* Both OFL-licensed — committing the files
is clean; body text stays a system stack because its role is to be quiet
and it saves ~100 KB of woff2. Rejected: Iowan Old Style (Apple-bundled,
not redistributable — mockup-only), Newsreader (narrower optical range at
display sizes), shipping a third face for body (weight without character).

## 4. Checkpoints

1. **Tokens and shell.** `@theme` tokens from DESIGN §6, fonts via
   `next/font/local`, legend + provenance-bar components, triage table
   moved to `/candidates`, `/imports` restyled, header/nav on all pages.
   Visible: the existing app, re-skinned, nothing at `/` yet changed.
2. **The life map.** Fan-out on the home server component, `life-map.tsx`
   island, Roadbook style JSON, T2 layers + the invariant-8 style test,
   hover card, click-through, summoned list, empty states. Visible: `/`
   is the life map on the demo instance; keyboard-only walk reaches every
   adventure.
3. **The adventure narrative.** `sliceDays` + vitest table tests (midnight
   rule, offset handling), cover rebuild (no score), day sections, day ↔
   map highlight sync, plate margin on the detail map, photos/stops
   restyled in place. Visible: Westfjords demo page shows Day 1/2/3 with
   the honest Day 2; golden fixtures untouched.
4. **Close.** Full `make test` cold (nothing in Go changed — prove it),
   a11y pass (focus-visible walk, reduced-motion, contrast vs tokens),
   README home-page description + screenshots refreshed if any, LOG.md.
   The demo compose instance is the acceptance environment throughout.

## 5. Risks named up front

- The T2 expression test is the phase's honesty keystone; write it in
  checkpoint 2 before styling drifts.
- `sliceDays` date arithmetic across offsets is the likeliest bug source —
  hence the table tests and the single-function seam.
- Style-JSON authorship in Maputnik is unfamiliar territory; timebox it —
  Liberty-as-shipped is the fallback default with the custom style landing
  later in the phase, and nothing else blocks on it.
