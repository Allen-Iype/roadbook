# Phase 14 — log: route image and journey summary

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

A confirmed adventure can now be taken off the page. Its cover, its shared
view, and the command line state the same summary of what the trip was —
time away, time on the move by the source's account, stops, places passed
through down to the region, farthest from home for the owner — every
figure printed by `roadbook journey -candidate N`. And the plate itself
downloads as a file, in two formats that are different objects: the
**image**, a 2400×1600 picture of the plate with its basemap, route, and
printed margin including the legend and the basemap's licence credit; and
the **overlay**, a transparent 1080×1920 story-portrait of the route and
figures alone, for the person's own photo. Both are rendered in the
browser from the served figures and the one layer list both maps draw.
Nothing in the pure detection or journey core changed; the two goldens,
the demo pin, and the corpus pin are byte-identical at every checkpoint,
and that identity was the phase's standing regression.

By checkpoint:

- **CP1 — regions.** `internal/states` embeds Natural Earth's 1:10m
  admin-1 file verbatim (gzipped, 12.2 MB, checksummed against its
  upstream commit) — the one named exception to the 1 MB dry-run rule,
  amended in CLAUDE.md by name. Migration 00014, `roadbook states`
  with the countries loader's `-src`/`-if-empty` shape, the compose
  startup line, `Journey.states` in the contract on both generated
  sides, attribution from measured points only, the cover's region line
  in local names with their diacritics, and the journey command printing
  countries and states (closing a pre-existing invariant-13 gap).
- **CP2 — the summary.** `journey.Summarize`, pure and beside `Assemble`:
  span, civil days by the narrative's own rule ported to Go, stops and
  dwell, pace over moving observed legs only (absent, never zero, for a
  fixes-only journey); `ModeKm` gains hours. `JourneySummary` in the
  contract; the CLI prints the block; the cover and shared view render
  it in the traveller's words after the review rework; the cover's day
  count is the served figure with a cross-language parity test on
  captured demo journeys.
- **CP3 — the image.** `lib/route-layers.ts`, one layer list for the
  on-screen map and the offscreen export map; `lib/legend.ts`, the fixed
  wording as data; `lib/plate-image.ts`, the pure op-list builder with
  the legend unconditional; `lib/plate-export.ts`, the offscreen MapLibre
  render with its drawing buffer preserved, the credit read from the
  loaded sources, fonts read off the document, the 2D composition;
  "Download as image" in the plate margin on the owner page and the
  shared view with worded failure states; the download e2e including
  the blocked-basemap path.
- **CP4 — the overlay.** Brief §9 addendum first, on the maintainer's
  decision at the CP3 review that both formats are offered. The
  thumbnail's projection lifted into a shared function; the builder's
  second format drawn from the same features; a translucent paper halo
  under the route and paper casing around every glyph as the overlay's
  own ground; "Download as overlay" beside the first button; an e2e that
  blocks every tile request and still downloads.
- **CP5 — close.** README (regions, the summary, both formats and what is
  never on either), this log, the cold pass.

## What was verified

- `make test` cold (`-count=1`) at every checkpoint, with the
  regressions confirmed as run: fixture 18/1/32, archive, both journey
  goldens, the demo pin, the corpus pin, the states pin (4,589 rows,
  every code unique, every row named), the store attribution test.
- Pure core untouched: the goldens' byte-identity is the proof for
  `internal/journey` and `internal/detect`; CP3, CP4, and CP5 carried
  zero Go and zero contract diff — the image needed nothing the API did
  not already serve.
- CLI = API = page = shared view on all three demo adventures for every
  new figure (countries, regions, the summary block, per-mode hours), the
  phase-2 ritual repeated per figure.
- Web: tsc, lint (0 errors), production build, vitest 107 (69 before the
  phase, +38: the layout builder's table, the layer list, summary
  parity); e2e 97 passing and 20 skipped across three viewports against
  a scratch compose stack in the demo state, including the three
  download walks and the tap-target case the phase added.
- The exported files, looked at: the three demo plates and a Day-2
  highlight; the three overlays composited over a dark striped ground
  and a bright one before any real photo. The maintainer's eyeball of a
  plate in his own browser at the CP3 STOP. The overlay on a real photo
  in a story editor is the one check that could only be approximated
  (carried, below).
- The screenshot record: `docs/screens/phase14-cp2-*`, `phase14-cp3-*`
  with half-size plate copies, `phase14-cp4-*` with the overlays at full
  size.

## What broke, and the shape of each fix

1. **The dataset was 12 MB, not 5.** The framing conversation carried
   "~5 MB gz worldwide" for admin-1; measured, the 1:10m file is 12.2 MB
   gzipped, and no slimming lands under 1 MB (quantised rings can also
   self-intersect, which PostGIS answers wrongly). The brief decided on
   the real number: embed verbatim with a named exception to the rule,
   provenance intact, rather than a derivative with a new failure class.
   The lesson is the one the repository already knew — measure before
   the brief, not after.
2. **The English names were the wrong names.** `name_en` glosses
   Iceland's regions as "Southern", "Western", "Capital". The row's name
   is the local Latin-script `name` with diacritics — what the road sign
   said — with `name_en` as the fallback; seven upstream placeholder
   features with no name in any column are dropped at parse.
3. **A coastal point fell outside its polygon.** The store attribution
   test first placed a point at Vík, which sits outside the 1:10m
   coastline; the test moved inland (Selfoss), and the cover's "derived
   from route points" note already says what that means for a real
   border-hugging track.
4. **A fixes-only journey showed "0 km/h".** The pace divided observed
   km by observed hours over every observed leg, and a stationary one
   contributes zero km over real minutes. Pace now runs over MOVING
   observed legs only — a fix is not a stretch — and is absent, never
   zero, when none exist; the Westfjords loop reads "no average speed",
   which is the truth.
5. **The summary spoke the pipeline's language.** The first block said
   "observed stretches", "dwelling", "fixes". At the review it was
   reworded into the traveller's words — Time away · On the move ·
   Stopped · Through · Farthest — with the honesty terms staying on the
   provenance lines above, where the headline distance is explained.
   "On the move" is the source's own transit time, the maintainer's
   choice between two honest candidates.
6. **The cover computed a figure the CLI did not print.** The day count
   was the web's `sliceDays` length. It is now the served `civil_days`,
   with a vitest asserting the two rules agree on real demo journeys —
   the frontend derives no figure the API does not serve.
7. **42-pixel buttons, again.** The new export button and the existing
   "Create a share link" both measured 42 px: `text-xs` is a 16 px line,
   and `py-3` adds 24. Both now use `py-3.5` (46 px), and the tap-target
   spec gained the adventure page so the phase-9 finding stays caught.
8. **Two writing specs raced.** The download spec minted a share link on
   the same first adventure the share spec works, and both counted
   "Revoke" buttons under `fullyParallel`. The download spec now works
   the last adventure and proves revocation by the stranger's URL going
   dead. Rule recorded in the spec: writing specs must not share an
   adventure.
9. **A blocked basemap and an unreachable one look the same.** From
   inside a browser a CORS refusal and a network failure are both a
   fetch that rejected (MapLibre reports status 0). The wording names
   both, the host, and that nothing was downloaded; the e2e case aborts
   every tile request, which is the same path.
10. **The overlay's dateline lost a region by one character.** At 12 px
    mono over 460 px the budget is 63 characters; the demo's two regions
    made 64 and truncated to "+ 1 more". 11.5 px gives 66. Seen only
    because the composite was looked at, not just sampled.
11. **The download walks outran the suite's budget.** On a freshly
    restarted stack the shared-view walk (mint, open, render with tiles,
    revoke) passed the 30 s per-test timeout twice while the download
    wait alone allowed 90 s — the two limits disagreed. The two download
    walks set a 120 s budget of their own; the suite-wide limit stays,
    because a 30 s red on a layout spec is information.
12. **The kickoff note said "1200×800 logical" for the offscreen map.**
    That is the plate; the map inside it is the slot (1136×528), smaller
    by the paper margin as the wireframe draws it. Built to the wireframe,
    recorded in DECISIONS so the discrepancy is not re-litigated.

## Carried out of the phase

- The overlay on a real photograph in a phone's story editor — the
  maintainer's check; the figure-block panel is the recorded fallback if
  halos fail there, one op away.
- A square 1080×1080 overlay for feeds — one constant in the
  parameterised builder, added when someone asks.
- Poster and print formats (backlog, on appetite); the tile-free Go
  plate stays the recorded alternative for that use.
- GPX/GeoJSON export with the confidence classes preserved (own backlog
  entry).
- The scratch stack `roadbook-p14` on 3014/8094 served every checkpoint;
  tear it down with `-v` when no longer wanted as the e2e target.
- Phase 15, hosting on a real platform: brief before any code, absorbing
  phase 12's unreached durability proof and phase 13's live-Google walk.
