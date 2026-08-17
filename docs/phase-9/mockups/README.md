# Phase 9 · CP3 mockups — the adventure grid

Mockups for the phase's one mockup-gated IA element (BRIEF §5.1): adventures
as a browsable grid of plate covers. Static HTML with inline SVG — no
network, no webfonts. Open `index.html` directly from disk in a browser.

These are review artifacts, not product code. Nothing here is imported by
the application. Every page carries a footer note saying it is a mockup.

## Contents

| file | what it is |
|---|---|
| `index.html` | links to both mockups and lists the review questions |
| `adventures-grid.html` | option (b): a new `/adventures` page — the full mock, with review exhibits (cover densities, pilot-scale grid, empty state) |
| `summoned-grid.html` | option (a): the existing summoned-list dialog carrying the same covers — the cheap contrast, costs stated on the page |
| `triage-gallery.html` | added at the review: `/candidates` as T1 table (today) / T2 table with a shape column / T3 full gallery, on undecided candidates, with a 66-candidate density exhibit |

Option (c) — the home page as a map-over-grid hybrid — is deliberately not
mocked: it would re-litigate the phase 6 map-is-the-navigation decision,
which is exactly what the mockup gate exists to prevent (BRIEF §5.1).

## Data provenance

All content is the committed fictional demo dataset (`testdata/demo/`,
Reykjavík persona, April–June 2026). No real location data was used.
Journey geometry, distances, dates, provenance splits, and the divergence
flags were captured from the demo compose instance on 2026-08-17, in its
routed quickstart state:

```
docker compose exec api wget -q -O- http://localhost:8080/candidates
docker compose exec api wget -q -O- http://localhost:8080/candidates/1/journey
docker compose exec api wget -q -O- http://localhost:8080/candidates/2/journey
docker compose exec api wget -q -O- http://localhost:8080/candidates/3/journey
```

Cover art was produced from those journey responses with the same
projection `web/lib/route-thumb.ts` uses (equirectangular with the
longitude axis compressed by cos mid-latitude; paint order
unknown → air → road → observed; air as a 24-segment great-circle arc;
routed legs on their routed polylines), decimated to thumbnail pixel
resolution for file size. Stroke widths and dash patterns are
`web/components/route-thumb.tsx` verbatim.

The life map underneath the dialog in `summoned-grid.html` is the phase 6
Stage B plate (`docs/phase-6/mockups/direction-a-life-map.html`), reused
verbatim — including its Natural Earth 10 m Iceland coastline (public
domain).

The "grid at pilot scale" exhibit repeats the three demo covers four times
over. That is a density study, not data; the exhibit says so on the page.
The triage page repeats them to twenty-four tiles for the same reason, and
shows the three demo candidates as *undecided* — a state fiction (their
demo decisions stripped for the mock), the data itself unchanged.

All three demo adventures carry a divergence flag (window truncation
against Google's door-to-door figure — a known property of the demo
dataset), so every cover shows the amber mark; on real data the mark is
the exception, not the rule.

Type is deliberately system-stack (Iowan Old Style/Palatino serif, system
grotesk, ui-monospace) so the files render offline anywhere; the product
uses Source Serif 4 and IBM Plex Mono (committed in `web/app/fonts/`).
