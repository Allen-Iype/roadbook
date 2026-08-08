# Phase 6 · Stage B mockups

Design directions for the phase 6 review, per `docs/phase-6/RESEARCH.md` §8.
Static HTML with inline SVG — no tile server, no network, no webfonts. Open
`index.html` (or any page) directly from disk in a browser.

These are review artifacts, not product code. Nothing here is imported by the
application. Every page carries a footer note saying it is a mockup.

## Contents

| file | what it is |
|---|---|
| `index.html` | links to everything below |
| `direction-a-life-map.html` | Direction A "Atlas plate" — the home page as a life map |
| `direction-a-adventure.html` | Direction A — Westfjords detail with stage narrative |
| `direction-b-life-map.html` | Direction B "Night expedition" — the home page |
| `direction-b-adventure.html` | Direction B — Westfjords detail with day narrative |
| `honesty-channel.html` | T1 vs T2 treatments, both directions, overview and detail zoom |
| `tokens-a.html` | Direction A token sheet |
| `tokens-b.html` | Direction B token sheet |

## Data provenance

All content is the committed fictional demo dataset (`testdata/demo/`,
Reykjavík persona, April–June 2026). No real location data was used. Route
geometry, distances, dates, scores, and the divergence figure were captured
from the demo compose instance on 2026-08-07:

```
docker compose exec api wget -q -O- http://localhost:8080/candidates
docker compose exec api wget -q -O- http://localhost:8080/candidates/1/journey
docker compose exec api wget -q -O- http://localhost:8080/candidates/2/journey
docker compose exec api wget -q -O- http://localhost:8080/candidates/3/journey
```

(the demo quickstart in the README reproduces that instance from a clean
checkout: import `testdata/demo/demo.json`, detect, confirm the three
candidates, run the Iceland routing walkthrough).

Overview panels on `honesty-channel.html` show the Westfjords gaps in their
pre-routing state (unknown chords) — the state a fresh install without a
router serves; the gap endpoints are in the journey response either way.

Coastline: Natural Earth 10 m admin-0 (public domain), Iceland only,
simplified and projected to Web Mercator at build time. The repository
already embeds the 110 m edition (`internal/countries/`); the 10 m edition
appears only as simplified path data inside these HTML files.

Type is deliberately system-stack (Iowan Old Style/Palatino serif, system
grotesk, ui-monospace) so the files render offline anywhere; the token
sheets name self-hosted implementation candidates, to be decided in Stage C.
