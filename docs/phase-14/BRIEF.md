# Phase 14 — Route image and journey summary: design brief

Status: DRAFT for Gate 1 review.

Charter position: PLAN.md "The road to strangers", phase 14 (chartered
2026-09-18). Two features the maintainer wants in hand before hosting resumes
in phase 15: a confirmed adventure's atlas plate downloadable as a PNG, and a
journey summary that states what the trip was — on the cover, on the shared
view, and from the command line, identically. The dual-mode charter binds as
ever: everything here works single-user and authless on a plain
`docker compose up`, and auth-off stays byte-identical to today.

Nothing in this brief reaches the pure detection or journey core. The
summary is a set of *derived* functions beside `Assemble` — the seam
`ModeBreakdown` opened in phase 11 — so the two goldens, the demo pin, and the
corpus pin stay byte-identical, and that identity is the phase's standing
regression.

## 1. Concepts this phase introduces

**Admin-1 boundaries.** Countries are "admin-0" in the cartographic
vocabulary; the next level down — states, provinces, regions, départements,
whatever a country calls its first subdivision — is "admin-1". Natural Earth
publishes admin-1 polygons for the whole world at one scale only, 1:10m:
4,596 features across 251 country codes. Its coarser scales are not
worldwide: 1:50m covers nine large countries (Australia, Brazil, Canada,
China, India, Indonesia, Russia, South Africa, the United States) and 1:110m
covers United States states alone. Invariant 9 rules those out — a state line
that appears for Indian journeys and vanishes for Icelandic ones is a regional
assumption dressed as a feature — so the choice is 1:10m or nothing. Sizes,
measured this session (reproduction: download the three
`ne_*_admin_1_states_provinces.geojson` files from the
`nvkelso/natural-earth-vector` repository's `geojson/` directory and run
`gzip -9c FILE | wc -c`):

| scale | features | raw | gzipped |
|---|---|---|---|
| 1:10m | 4,596 | 40.7 MB | 12.2 MB |
| 1:50m | 294 (9 countries) | 2.3 MB | 0.73 MB |
| 1:110m | 51 (US only) | 0.18 MB | 0.04 MB |

The number the framing conversation carried — "~5 MB gz worldwide" — was
wrong; the real figure is 12.2 MB, and §3B decides with the real one.
Attribution works exactly as countries do today: one indexed `ST_Contains`
query over the drawn leg points, grouped per polygon, ordered by first
appearance along the journey, so a states line reads in travel order. The
admin-1 file gives every feature an `iso_3166_2` code (e.g. `IS-5`), a
unique `adm1_code`, a local `name` with diacritics and a `name_en` (missing
for 7 features — `name` is the fallback), and the parent's `adm0_a3`.

**Canvas capture of a WebGL map.** MapLibre paints into a WebGL canvas. Read
that canvas back after the frame is presented (`toDataURL`, `drawImage`)
and you get nothing — the browser is allowed to discard the drawing buffer
once composited, and MapLibre defaults to letting it. This is the phase 6
screenshot trap in a new coat: the committed record showed blank plates
while the live pages were fine. Two honest remedies exist. Either ask for the
buffer to be preserved when the WebGL context is created
(`canvasContextAttributes: { preserveDrawingBuffer: true }` in MapLibre 6,
at a small per-frame cost), or read the pixels inside the frame that draws
them (a `render` event handler runs before the buffer is released). This
phase does neither to the on-screen map: it renders a *second, offscreen*
map instance at the export's exact pixel size with the buffer preserved,
reads it once, and destroys it — so the interactive plate keeps its default
performance, the image has deterministic dimensions independent of the
viewer's window, and the browser's cap on live WebGL contexts is touched for
seconds, not for the page's life.

Two more browser facts shape the design. A 2D canvas that has drawn a
cross-origin image without CORS approval becomes *tainted* and refuses
`toBlob` with a `SecurityError`; MapLibre already requires CORS-served tiles
for WebGL, so the default basemap composes cleanly, but a self-hoster's
`ROADBOOK_MAP_STYLE` pointing at a tile server without CORS headers will
fail — the export must say so in words, not swallow it. And canvas text uses
whatever fonts are *loaded*: the plate's serif and mono are self-hosted
through `next/font/local`, which registers them under generated family
names exposed as CSS variables; the exporter must read those names from the
document and await `document.fonts.load()` before drawing, or the margin
renders in a fallback face on first use.

**The plate as an exported image.** Strava's "share your activity" image is
the reference the maintainer named: route over a basemap, a few figures, a
logo. Roadbook's version of that object already exists on screen — the atlas
plate: map framed by a double rule, marginalia printed below (legend, scale,
plate label). The exported image is that plate made portable: the same map
at a fixed size, the margin carrying what a page would otherwise carry
around it — the adventure's name, dates, distance with its provenance bar,
the legend, and the basemap attribution. Two of those are not optional.
The legend, because a route image with red, blue, grey, and green lines and
no key is exactly the confident undifferentiated line invariant 8 forbids,
and an image travels without the page that would have explained it. The
attribution, because OpenFreeMap's tiles are OpenMapTiles rendering
OpenStreetMap data, and both licences require the credit to travel with any
derived image (the on-screen map shows it in MapLibre's attribution control;
the image must show it in ink). The attribution text is read from the
loaded style's sources at export time — the same string the on-screen
control displays — never hardcoded, so a self-hoster's different basemap
carries its own credit automatically.

**What "reproducible figure" means for an image (invariant 13).** The
invariant says every public number must trace to a command that regenerates
it. An image carries two kinds of content. *Figures* — the name, dates,
kilometres and their split, day count, countries, states — are data, and
every one of them must be printed by `roadbook journey -candidate N`
identically, as the cover's figures are today. *Rendering* — the basemap
pixels, the projection, the scale bar's length — is not a claim about the
journey; it is a picture of one, and the standard there is "correct and
tested", not "byte-reproducible from the CLI". The brief keeps the boundary
sharp: the image never computes a figure of its own. It draws figures the
API already served and the CLI already printed; anything new the summary
wants (civil days, pace, dwell) is a Go function first, printed by the CLI
second, served third, drawn last.

**Measured versus source-asserted time.** Phase 11 drew the line for
distance: the assembler *measures* geometry; Google's per-mode distances are
the source's *assertions*, shown subordinate and never summed with
measurements. Time has the same two kinds. A journey's *span* is measured —
the window's start to its end, from the data's own timestamps. "Time in
transit" is not: the only per-transit durations are Google's activity
records, whose modes are guesses with recorded failures. The summary keeps
the line: span is a headline figure, transit time rides on the mode line as
the source's own claim, in the source's own labels. Pace follows the same
discipline — an average over *recorded* stretches only, with its caveat
stated (§3C).

## 2. What gets built

**(a) The journey summary.** A pure `journey.Summary` beside `Assemble`
(never inside it), returning: span duration; civil-day count in the
journey's own offsets, by the rule the narrative already uses (start day
owns the event; a day is a date in the timestamp's recorded offset; the
count is the dates from the window's first day to its last, inclusive);
stop count and total dwell hours; leg-average pace over observed legs with
the caveat named; and the existing mode breakdown extended with per-mode
hours. Plus states, attributed by the store the way countries are. Served
on `Journey` (openapi first, both sides regenerated), printed by
`roadbook journey -candidate N` as one summary block, and rendered as a
summary block on the cover and on the shared view. The distance-from-home
figure (the candidate's `dest_km`, farthest place *dwelt*) joins the owner's
cover and the CLI's candidate mode; §3E decides its presence on the shared
view.

**(b) "Download as image".** One control in the plate margin, on the owner's
adventure page and on the shared view, that produces a PNG of the plate:
map with route, stops and fixes; margin with name, dates (with the
truncation words when they apply), distance with provenance bar and split,
countries and states, day count, the legend in its fixed wording, and the
attribution line. The image shows what the plate shows — a highlighted day
exports highlighted and labelled so. Photos are not drawn (they are DOM
markers, not map layers; the control's copy says so). One format: 2400×1600
pixels (a 1200×800 plate at pixel ratio 2), landscape like the on-screen
plate. Other aspect ratios are a follow-up on request, not a guess.

**(c) The states dataset**, wherever §3B puts it, loaded by `roadbook states`
with the same `-src` / `-if-empty` shape as countries, and the compose
startup line extended in the same way.

**Not in this phase:** GPX/GeoJSON export (own backlog entry); poster paper
formats and print CSS; photos on the image; life-map export (the life map is
a different disclosure — its own decision if asked); per-leg instantaneous
speed (rejected on the record — the observation density cannot support it);
any change to `Assemble`, detection, or the goldens; hosting anything.

## 3. The real choices

### A. Render path for the image

- **(1) Offscreen MapLibre render composed on a 2D canvas, in the browser —
  recommended.** A hidden container sized 1200×800, a second `MapLibreMap`
  with `pixelRatio: 2`, `preserveDrawingBuffer: true`, the same style URL
  and the same layer code the plate uses (the layer builder is extracted
  into a shared function so the two maps cannot drift), `fitBounds` to the
  same bounds, wait for `idle`, draw the canvas onto a 2D canvas that adds
  the paper margin and the marginalia, `toBlob`, download, remove the map.
  It is WYSIWYG with the plate people already look at — roads, place names,
  coastlines from the tuned basemap — which is what "Strava-like" means.
  Costs: browser-only (no CLI produces the image; the figures on it are CLI
  figures, the pixels are a rendering — §1); tiles are fetched at export
  time (the page already fetched them; the offscreen instance re-uses the
  browser cache); a CORS-less self-hosted basemap fails, visibly; the
  WebGL trap is real and the remedy is the one MapLibre documents.
- **(2) A tile-free plate rendered in Go.** `roadbook plate -candidate N -o
  plate.png`: route on paper with coastlines from the countries polygons
  already embedded, text set from the self-hosted fonts, byte-reproducible,
  goldens possible, no third-party licence on the image at all. Genuinely
  attractive on invariant 13 grounds — and rejected for this phase on
  product grounds: without roads and place names an inland journey is a red
  line floating on beige, less informative than the plate on screen, and the
  Go work (font rasterising, anti-aliased dashed and dotted strokes, a
  projection, layout) is a rendering engine the project does not otherwise
  need. It stays the recorded alternative for a future poster format, where
  a bare-coastline plate is the *intended* look.
- **(3) The welcome-plate SVG generalised and rasterised in the browser.**
  The route thumbnail builder already draws any journey as SVG in the Atlas
  encoding; add coastlines (from the countries API) and text, draw the SVG
  onto a canvas, export. No WebGL, no tiles, no CORS. Rejected for the same
  informativeness reason as (2), plus SVG-to-canvas has its own font trap
  (fonts must be embedded in the SVG as data URIs or text falls back). It
  is, however, the natural *fallback* if (1) proves unreliable in real
  browsers: the margin composition is written against a "map bitmap"
  input, so swapping the bitmap's source from a MapLibre readback to a
  rasterised SVG changes one function.

### B. Where the admin-1 dataset lives

The 1:110m countries file was committed verbatim and embedded because at
209 KB provenance was free. Phase 2's decision record anticipated today:
"would change our mind: a wanted source file that tops 1 MB even gzipped —
slimming then returns as the price of admission." The wanted file is 12.2
MB gzipped, and slimming does not bring it under the bar: geometry with
seven properties is 10.0 MB; quantising coordinates to three decimals
(~110 m) gives 5.9 MB, two decimals (~1.1 km) 3.7 MB — and quantised rings
can self-intersect, which `ST_Contains` on an invalid polygon answers
wrongly or not at all. No variant is under 1 MB. So the options are:

- **(1) Embed the upstream file verbatim, gzipped (12.2 MB) — recommended,
  with a recorded exception to the 1 MB dry-run rule.** Provenance stays
  "this file, gzipped": a checksum against the upstream commit, no derived
  geometry, no validity risk. `roadbook states` needs no arguments and no
  network; `states -if-empty` joins the compose startup line beside
  countries, so a fresh instance is browser-complete with states from its
  first page — phase 8 retired the last operator CLI step deliberately, and
  this option keeps it retired. The costs, stated plainly: one 12 MB blob in
  a public repository whose largest object today is 280 KB, a 42 → ~54 MB
  server image, and an amendment to CLAUDE.md's data-safety rule from "no
  file over 1 MB" to "no file over 1 MB other than the reference datasets
  named here" — the check stays mechanical (a large file is still reviewed
  by name against a short list), and the rule's purpose, keeping real
  location data out, is untouched by a public-domain boundary file. The
  maintainer's earlier, unvetoed leaning was "in the binary" at a believed
  ~5 MB; this recommendation holds at the real 12 MB, and says so.
- **(2) Embed a slimmed, quantised derivative (~4–6 MB) with a committed
  generator.** Halves the blob at the price the phase 2 record named — the
  committed file no longer byte-matches upstream — plus a generator script
  needing the 40 MB upstream as input and a geometry-validity pass at load.
  A 2× saving for a new failure class. Rejected.
- **(3) Disk-optional: not in git, not in the binary.** `scripts/states-setup.sh`
  downloads the pinned upstream file (URL at a release commit, sha256
  verified) into gitignored `data/states/`, the compose line loads it when
  present, and the summary omits states when it is not, stating so on the
  owner's cover. Zero repository cost; honest degradation. Rejected because
  it re-introduces an operator step into the quickstart and makes the
  summary differ between instances forever, for a 12 MB saving. It stays
  the recorded alternative, and it needs no schema change to adopt later:
  `-src` already exists.
- **(4) 1:50m or 1:110m for the countries they cover.** Rejected by
  invariant 9 before size enters the discussion.

### C. "Time taken" — which duration is the journey's

- **(1) Span as the headline, transit time as the source's claim —
  recommended.** The summary states "away D days H h" (window end minus
  start, with the truncation words when the window is cut) and, on the mode
  line, "in transit as the source recorded it: H h M min — <mode> H h M min,
  <mode> H h M min". Both are printed by the CLI. Photo-sourced journeys say
  "no transit record", never zero, as the mode line does today.
- **(2) Observed-leg time as "moving time".** Sum of observed leg
  durations — measured, but a strange figure: it includes pauses shorter
  than the gap threshold and excludes every routed or unknown stretch, so
  it under-counts a sparse journey wildly. Rejected as a headline; it is
  the denominator of pace (below), where its meaning is exact.
- **(3) Span only.** Honest but thinner than the data allows; the pilot
  asked for per-mode figures and time is the natural second column.

Pace, the same way: **leg-average pace** = observed kilometres over observed
leg hours, reported as "N km/h over recorded stretches (pauses under 20 min
included)" — the caveat names the gap threshold the number depends on, by
its parameter, so the wording tracks the value. No new threshold is
introduced; a leg with zero duration contributes nothing to either sum. Per-
leg pace in the day narrative is the obvious extension and is *not* in scope:
one figure with one caveat first.

### D. Export on the shared view, or owner-only

- **(1) Both — recommended.** The image discloses nothing the shared page
  does not: same plate, same figures, same attribution. A person the owner
  chose to share with gets the same "Download as image"; revoking the link
  revokes the page and the button with it. (The maintainer heard this
  recommendation at framing and did not veto it.)
- **(2) Owner-only.** Rejected: it would make the shared view a lesser plate
  for no privacy gain, since a viewer can screenshot the page anyway.

### E. The distance-from-home figure on the shared view and the image

Newly visible in writing this brief. "Furthest from home: N km" is the
one summary figure that names *home*. The route is on the page; a stated
distance from the farthest dwelt place draws a ring on which home lies. One
figure, a wide ring — small, but it is the only fact on the plate about a
place the owner did not choose to share.

- **(1) Owner's cover and CLI only; absent from the shared view and the
  image — recommended.** The shared plate stays "this adventure, whole", and
  says nothing about anywhere else. Consistent with D: the image discloses
  exactly what the shared page does.
- **(2) Everywhere.** Rejected: a stranger-facing surface should not carry a
  home-relative figure the owner never opted into showing.

## 4. The plate as an image — design

DESIGN.md is binding: paper, land, sea, ink, rule, the four leg inks with
their non-color channels, flag amber for warnings only, Source Serif 4 for
display, IBM Plex Mono for figures, the fixed legend wording, the provenance
bar, the plate-margin signature. The freedom in this phase is composition
only. Where the atlas register asks for tracked capitals on the plate label
and middle-dot figure strings, that is the design system's own vocabulary
(DESIGN §6), not a default — it stays.

Token plan (all existing): ground `#F5F2E8` paper / `#EFECE2` land /
`#D9E2E4` sea; `#26251F` ink, `#6E6A5E` secondary, `#C9C3B2` rule; leg inks
observed `#A81E22` / routed `#2A5DA8` cased / unknown `#8A8375` dashed / air
`#3F7069` dotted; flag `#8A5C0B` glyph-only. Type: Source Serif 4 semibold
for the name, IBM Plex Mono for every figure, the system sans for the two
short prose lines (legend descriptions, attribution).

Layout, 1200×800 logical, left-aligned marginalia under a full-width map:

```
┌────────────────────────────────────────────────────────────────┐ paper 32
│ ╔════════════════════════════════════════════════════════════╗ │
│ ║                                                            ║ │
│ ║        basemap · route in the four inks · stops · fixes    ║ │ map
│ ║                                                            ║ │ ~540
│ ║  ▁▁▁▁▁ 20 km                                               ║ │ scale
│ ╚════════════════════════════════════════════════════════════╝ │ double rule
│ <adventure name>                         PLATE <n> · FULL ROUTE │ serif 40 / mono caps
│ <dates> · <country> · <state, state> · <n> days                │ mono 15
│ <N> km drawn — <p>% measured    ▮▮▮▮▮▮▮▮▮▮▮▮▮▮▯▯▯▯▯▯▯▯▯▯▯▯▯  │ serif 28 + bar
│ observed <a> · routed <b> · unknown <c> · air <d> km           │ mono 13
│ ── Observed  ═══ Routed  – – Unknown  ···· Air   ● Stop  • Fix │ legend
│ <attribution from the loaded style's sources>       Roadbook   │ sans 12 secondary
└────────────────────────────────────────────────────────────────┘
```

(Angle-bracketed slots are placeholders, not figures — the record's real
values come from the demo at CP3, per invariant 13.)

Principles. The map is the memorable thing; the margin is quiet and set in
the plate's existing grammar. No logo mark, no rounded overlay card floating
on the map, no gradient, no drop shadow — the Strava export's own furniture
is exactly what the atlas identity is not. The truncation warning, when it
applies, is one line under the dates with the amber ⚑ glyph and ink words,
as on the cover. States are listed after their country, truncated to the
line with "+ n more" when they overflow — an image has one line, the page
has as many as it needs. The scale bar is drawn in the margin composition
from the offscreen map's own bounds (metres per pixel at the centre
latitude), because MapLibre's ScaleControl is DOM, not canvas; its formula
gets a unit test.

Self-critique against the generic default: the reflex version of this
feature is a screenshot of the map with a translucent stats card in a
corner and a wordmark — the object every fitness app ships. The plate
inverts it: the map is framed, not overlaid; every figure is in the margin
where printed maps put them; the legend and attribution are typeset, not
tucked. What was removed on review: a plate-number roman numeral in the
corner of the map itself (redundant with the label line) and a second
distance figure for Google's own total (a validation number, not a claim
the image should make).

## 5. Constraints carried in

- Invariant 13 as §1 reads it: every figure on the image and the cover is
  printed by `roadbook journey -candidate N`; the phase-2 ritual — CLI, API,
  page agree exactly on the demo — is repeated for every new figure.
- Invariant 9: states worldwide or not at all — settled by §3B's option set
  (only 1:10m qualifies).
- Invariant 8: the legend is on every image, in the fixed wording; the
  layout builder's tests assert it cannot be omitted.
- Invariant 3: the pace caveat names the parameter it depends on; the
  summary echoes no new thresholds because it introduces none.
- Invariants 1 and 2: `Summary` is pure and reads the journey it is given;
  `Assemble` is untouched, goldens byte-identical at every checkpoint.
- Invariant 10: `Journey` gains `summary` and `states` in openapi.yaml
  first; both generators run; nothing hand-mounted. The shared response
  carries the same `Journey`, minus the owner-only figure (§3E).
- Data safety: the 1 MB dry-run rule is amended in CLAUDE.md, by name, if
  §3B(1) is taken; nothing under `data/` moves; the exported PNGs of the
  demo journeys are the only images this phase commits (`docs/screens/`),
  and only if each is under the informal 400 KB note — a 2400×1600 map PNG
  may not be; then the record is a downscaled copy or none, never an
  exception.
- The frontend never computes a figure the API did not serve. The one
  place the web already derives a figure — `sliceDays`' day count — gains a
  parity test against `summary.civil_days` so the two implementations
  cannot drift.

## 6. Verification plan

- **Goldens and pins:** both journey goldens, the demo pin, the corpus pin,
  fixture 18/1/32 and the archive regression — run uncached (`-count=1`) at
  every checkpoint, verified as run, not skipped.
- **Summary:** Go table tests for civil days (the narrative's 25 vitest
  cases ported where they concern day count — midnight, offsets, the
  Westfjords Friday-20:00-to-Sunday-10:00 dwell = 3 days), span with and
  without truncation, dwell sum, pace with zero-duration legs and with no
  observed legs (absent, not zero), mode hours beside mode km. Vitest
  parity: `sliceDays(j).length === j.summary.civil_days` on the fixtures.
- **States:** offline test on the embedded file — 4,596 features, every
  `adm1_code` unique, every feature named (`name_en` then `name`), the 7
  `name_en`-less features recovered; a store test attributing synthetic
  points across two Icelandic regions in journey order; API test that a
  journey's `states` is empty, not absent, before `roadbook states` runs.
- **CLI = API = page:** the demo's three confirmed adventures print the
  summary block; the API's `summary` and `states` match it field for field;
  the cover and shared view render the served values.
- **Image:** vitest on the pure layout builder — legend present with fixed
  wording, attribution line present whenever a source declares one, figures
  taken from the served journey, states truncation, scale-bar formula. e2e:
  clicking "Download as image" on the demo (owner page and shared view)
  yields a PNG download of 2400×1600, decoded, not all-paper (a sampled row
  through the map area has non-ground pixels); a CORS-less style URL
  produces the worded error, not a silent nothing. Maintainer eyeball of
  the actual file — fonts loaded, legend legible, attribution readable.
- **Auth-off regression:** full suite plus e2e against an auth-off demo
  stack, zero diff on every existing surface.
- **Access:** the shared view's export needs no session (it is the same
  page); the owner-only figure is absent from the shared response (API
  test), and therefore from the image it draws.

## 7. Checkpoints

1. **CP1 — states.** Dataset per §3B; `internal/states` mirroring
   `internal/countries` (parse, code and name rules, offline pin);
   migration 00014 (`states` table + GiST); `roadbook states` with `-src`
   and `-if-empty`; compose startup line; `store.StatesForPoints`;
   `Journey.states` in openapi and both generated sides; the cover and
   shared cover line "Iceland · Vestfirðir, Vesturland"; CLI prints it.
   *Visible: the demo's Westfjords adventure names its regions on the page
   and in the terminal, identically.*
2. **CP2 — the summary.** `journey.Summary` and mode hours, Go tests,
   openapi `summary`, CLI block, the cover and shared-view summary block
   (redesigned with the frontend-design pass; the home figure per §3E),
   parity vitest. *Visible: one summary, three surfaces, same numbers.*
3. **CP3 — the image.** Shared layer builder extracted from `route-map`;
   `lib/plate-image` pure layout + tests; offscreen render and composition;
   "Download as image" on both pages with busy and error states; e2e
   download spec. *Visible: a PNG from the demo, opened, every margin
   element present; the same from a share link, signed out.*
4. **CP4 — close.** README (image export, summary, the states statement,
   the CLAUDE.md rule amendment if taken), cold `make test`, e2e vs a demo
   stack, `docs/screens` record where size allows, LOG.md — the phase is
   not complete until the log exists.

## 8. What would change our mind

- If the offscreen capture yields blank or partial bitmaps in a real
  browser the maintainer uses — after the documented remedies — CP3 falls
  back to §3A(3) with the composition unchanged; recorded, not retried
  indefinitely.
- If the 12 MB embed is refused at the gate, §3B(3) is the fallback, adopted
  without schema change; the summary then states "states not loaded" on the
  owner's cover, never silently omits.
- If any summary figure turns out to need something `Assemble` does not
  expose, the answer is a new pure function reading the journey, never a
  change to `Assemble` — and if that is impossible, the figure is dropped
  from the phase.
- If invalid geometry surfaces at states load (an upstream ring that
  `ST_Contains` rejects), `ST_MakeValid` at insert is the fix, recorded
  with the feature that needed it.
