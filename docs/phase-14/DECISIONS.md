# Phase 14 — decision log

Three lines each: chosen, rejected, what would change our mind. Written as
decisions are made, per the working agreement. Entries before Gate 1 record
what the brief's drafting settled; the gate's own entry follows the review.

## 2026-09-18 — Roadmap housekeeping before the brief

- **Chosen:** PLAN.md marks self-serve deletion (phase 13 CP5), bulk triage
  and the per-mode breakdown (phase 11 CP4) as done in place, with their
  original reasoning kept as history; adds phase 14 (this) and phase 15
  (hosting on a real platform, absorbing phase 12's unreached proof and
  phase 13's live-Google walk) to the roadmap; records phase 12's early
  close and phase 13's outcome under their sections; moves "reproducible
  stats panel" and "leg-average speed" out of the non-gating refinements
  into phase 14. A stray fragment of the GPX/GeoJSON bullet, orphaned by
  earlier inserts, was reattached.
- **Rejected:** deleting the done entries — the backlog is where a future
  reader learns *why* auto-confirm was declined and why deletion waited for
  accounts.
- **Would change our mind:** nothing; this is bookkeeping.

## 2026-09-18 — The admin-1 size fact, measured, supersedes the framing figure

- **Chosen:** the brief decides the dataset's home on measured numbers —
  Natural Earth 1:10m admin-1 is 12.2 MB gzipped verbatim (4,596 features),
  not the "~5 MB" the framing conversation carried; 1:50m covers nine
  countries, 1:110m the US alone. The recommendation stays "embed
  verbatim", now with the real cost stated and a named amendment to the
  1 MB dry-run rule proposed alongside it.
- **Rejected:** carrying the remembered figure into the brief; slimming or
  quantising to dodge the rule (no variant lands under 1 MB, and quantised
  rings risk invalid polygons); the coarser scales (invariant 9).
- **Would change our mind:** the maintainer refusing a 12 MB blob at the
  gate — then the disk-optional shape (§3B option 3) is adopted without
  schema change.

## 2026-09-18 — Where the summary lives: beside Assemble, never inside

- **Chosen:** `journey.Summary` and the mode-hours extension are pure
  functions reading an assembled journey, on the seam `ModeBreakdown` opened
  in phase 11; goldens pin `Assemble` byte-for-byte and stay untouched. The
  civil-day rule is ported to Go and the web's `sliceDays` count gains a
  parity test against it, so the one figure the frontend already derives
  cannot drift from the served one.
- **Rejected:** adding summary fields to `Assemble`'s output (would invite
  golden churn and blur measurement and derivation); computing summary
  figures in the web layer (invariant 13 — no CLI would print them).
- **Would change our mind:** a figure that cannot be derived from what
  `Assemble` exposes — the brief's answer is to drop the figure, not open
  the core.

## 2026-09-18 — The home-relative figure stays off stranger-facing surfaces

- **Chosen:** "furthest from home" (the candidate's `dest_km`) appears on
  the owner's cover and in the CLI's candidate mode only; the shared
  response and therefore the exported image carry no home-relative figure.
  Surfaced while drafting: it is the one summary figure that names a place
  the owner never chose to share.
- **Rejected:** showing it everywhere on the grounds that a ring is a weak
  disclosure — the shared plate's promise is "this adventure, whole, and
  nothing else".
- **Would change our mind:** an explicit owner opt-in at share creation, if
  anyone asks for it; not built speculatively.

## 2026-09-25 — Gate 1: brief approved as written

- **Chosen:** the maintainer's "let's start the implementation" taken as
  approval of the brief as written, with every §3 recommendation standing —
  offscreen MapLibre capture composed on a 2D canvas; the admin-1 file
  embedded verbatim at 12.2 MB gzipped with a named exception to the 1 MB
  dry-run rule (the CLAUDE.md wording lands in CP1's diff, where the
  maintainer reviews it); span headline with source-asserted transit hours
  and observed-leg pace; export on the shared view; the home-relative
  figure owner-only. One landscape format; a highlighted day exports
  highlighted.
- **Rejected:** re-asking the four gate questions one by one — each had a
  recommendation in the brief, and the review returned none overturned.
- **Would change our mind:** any of them raised at a checkpoint STOP; the
  dataset one in particular reverts to the disk-optional shape with no
  schema change (§3B option 3).

## 2026-09-25 — CP1: the admin-1 file, pinned and embedded verbatim

- **Chosen:** `internal/states/ne_10m_admin_1_states_provinces.geojson.gz` is
  the upstream file at nvkelso/natural-earth-vector commit `117488dc`
  ("v5.1.0 packaging"), `gzip -9n` so the archive is reproducible; both
  checksums are in the package doc and the CLAUDE.md rule now names the file
  as the one exception to the 1 MB line. Loaded by `roadbook states` with
  the countries loader's `-src`/`-if-empty` shape; `states -if-empty` joins
  the compose startup command.
- **Rejected:** a slimmed derivative (provenance loss, no rule relief);
  a fetch step (the countries precedent: never at build or install time).
- **Would change our mind:** the maintainer refusing the blob at a STOP —
  then `-src` plus an operator download script, no schema change.

## 2026-09-25 — CP1: what a state row is

- **Chosen:** key = Natural Earth's `adm1_code` (unique; the file's ISO
  3166-2 column repeats eight codes, so it is carried, not keyed); name =
  the local Latin-script `name` with diacritics, `name_en` only as fallback
  — a deviation from the brief's "name_en then name", decided on the data:
  the English column glosses Iceland's regions as "Southern", "Western",
  "Capital", where a traveller saw Suðurland, Vesturland, Höfuðborgarsvæði;
  country code = `iso_a2` when it is a real two-letter code, else `adm0_a3`
  — the countries loader's own fallback, so CYN and SOL line up. Seven
  upstream placeholder features with no name in any column
  (`ATA+99?`, `RUS+99?`, …) are dropped at parse: 4,596 → 4,589 rows, pinned
  by the offline test. No foreign key from states to countries: the two
  tables load independently and an operator's own admin-0 file may name a
  different set — `country_code` is a label.
- **Rejected:** keeping the unnamed features under their code (a region
  nobody can be told they crossed); keying on ISO 3166-2 (not unique);
  `name_en` first (worse names for the plate's own readers).
- **Would change our mind:** a user population reading a script the Latin
  `name` column serves badly — then a per-language column joins the row,
  chosen per instance, never per region.

## 2026-09-25 — CP1: the journey command prints the derived lines

- **Chosen:** `roadbook journey -candidate N` now prints the countries and
  states crossed, in journey order, after the mode line — and says which
  command loads the table when a line is empty. File mode (`-src`) prints
  neither: attribution needs the database's polygons.
- **Rejected:** leaving the cover's countries line without a reproduction
  command (a pre-existing invariant-13 gap this checkpoint happened to
  close); attributing in file mode by parsing the embedded polygons in Go
  (a second point-in-polygon implementation to keep in step with PostGIS).
- **Would change our mind:** nothing foreseeable.

## 2026-09-25 — CP1: regions are attributed from measured points only

- **Chosen:** states use exactly the point set countries use — observed
  points and gap endpoints — never routed polyline vertices. Seen on the
  demo: the sparse Westfjords loop attributes to Vesturland alone until a
  fix lands elsewhere, even though its routed road crosses Vestfirðir.
  Inference does not testify to where a person was; a region named from a
  routed guess would be invariant 8 failing in words.
- **Rejected:** attributing from routed geometry (more regions, less
  truth); a second "regions the routed road passes" line (a distinction
  the plate would then have to explain on every surface).
- **Would change our mind:** an explicitly labelled "routed through" line
  if a real user asks — never merged into the attributed list.

## 2026-09-25 — CP2: the summary's shape and where its figures come from

- **Chosen:** `journey.Summarize` beside `Assemble`: span hours, civil days
  (the narrative's rule ported — min to max civil date over window edges,
  leg and stop endpoints, each in its own offset), stop count and dwell
  hours, observed hours and pace (observed km over observed leg hours;
  zero-duration legs contribute nothing; absent, never zero, when no leg
  has a duration). `ModeKm` gains `Hours`, summed whole like km, and their
  sum is the "in transit" figure — the source's account, labelled so on
  every surface. The cover's day count now reads the SERVED `civil_days`
  (was the web's own `sliceDays` length), with a vitest parity assertion on
  real demo journeys so the two rules cannot drift; the adventures grid
  follows. The home-relative figure is the candidate's `dest_km`, rendered
  by the owner's cover and the CLI's candidate mode only.
- **Rejected:** any new field on `Assemble`'s output (goldens byte-identical
  is the phase's standing regression — proven by the untouched golden
  tests); computing pace from Google's activity durations (a guessed
  denominator under a measured numerator); an observed-leg "moving time"
  headline (excludes every gap, so it under-counts sparse journeys — it is
  pace's denominator, where its meaning is exact).
- **Would change our mind:** a real journey where the served day count and
  the narrative's disagree — the parity test is the tripwire, and the Go
  rule would follow the narrative's, never the reverse.

## 2026-09-25 — CP2: the screenshot record, captured at the checkpoint

- **Chosen:** `docs/screens/phase14-cp2-*` captured now, from the scratch
  demo stack, rather than waiting for CP4 as the brief scheduled — the
  record is cheapest at the moment the surface changes. `capture.js` gains
  `ROADBOOK_SCREENS_URL` (default unchanged: the compose demo on 3000) so a
  scratch demo stack on another loopback port can be captured without
  editing the script. Same rule as ever: demo data only, whichever port.
- **Rejected:** a per-phase copy of the script; capturing from the
  maintainer's real instance (never — pixels are location data too).
- **Would change our mind:** nothing; CP3's image work adds its own set.

## 2026-09-25 — CP2 review: the summary speaks the traveller's language

- **Chosen:** at the maintainer's review the block was reworded from the
  pipeline's vocabulary (observed stretches, dwelling, fixes) to the
  traveller's: Time away · On the move · Stopped · Through · Farthest. The
  honesty terms stay on the provenance lines above the block, where the
  headline distance is explained. "On the move" = the source's own transit
  time (the sum of its activity durations, the mode line's figure) —
  Allen's choice between the two honest candidates; the alternative,
  span minus dwell, is measured but counts an unrecorded overnight as
  moving. The average speed beside it stays the measured pace over
  recorded driving. The CLI prints the same lines in the same words.
- **Rejected:** a separate "Distance" row (the headline figure with its
  provenance bar sits directly above and IS that row); switching the
  distance to Google's figure when nothing is routed (the drawn figure with
  its bar is the honest one — the Westfjords loop reads 197 km, 0%
  measured, and says so).
- **Would change our mind:** photo-sourced journeys gaining a measured
  moving-time source — none exists today, so they read "no transit record".

## 2026-09-25 — CP3: one layer list for the plate and its image

- **Chosen:** the route features, the seven style layers, the fit padding,
  and the day-highlight paint moved out of the map island into the pure
  `lib/route-layers.ts`; the on-screen map and the offscreen export map
  both consume it unchanged, and `route-layers.test.ts` pins the list —
  paint order, the non-color channel per kind, no zoom expression, dim
  never hide. The legend's fixed wording moved the same way into
  `lib/legend.ts`, read by the legend component and the image builder.
- **Rejected:** copying the layer list into the exporter (the two maps
  would drift the first time one was touched); reading the on-screen map's
  layers back through `getStyle()` (ties the export to a mounted map and
  to whatever the page had already done to it).
- **Would change our mind:** nothing foreseeable; a third map would join
  the same list.

## 2026-09-25 — CP3: the image is an op-list first, pixels second

- **Chosen:** `lib/plate-image.ts` is a pure layout builder — served
  Journey plus the page's facts (name, plate number, truncation flags,
  highlighted day, the loaded style's attribution, the fitted view) in,
  an op-list of fills, rules, text, the map slot, the provenance bar, and
  the legend row out. `lib/plate-export.ts` paints it. The split exists so
  vitest can assert what the image WILL say without a browser: the
  legend is present in the fixed wording for every input (the invariant-8
  regression — there is no input that omits it), the attribution line is
  present whenever a source declared one, the dateline truncates regions
  with "+ n more", the truncation sentences are the cover's verbatim, the
  scale-bar formula rounds as MapLibre's control does. Every figure is a
  string the cover already prints from the same served values, so
  `roadbook journey -candidate N` reproduces each one; the builder
  computes no figure of the journey's own.
- **Rejected:** drawing straight from the component (nothing to test but
  pixels); measuring text in the builder (needs a canvas — the one line
  that can overflow, the name, shrinks in the painter; the dateline is
  mono so its character budget is exact without measuring).
- **Would change our mind:** a second format (poster, square) — the
  builder would take the format as a parameter; the ops stay.

## 2026-09-25 — CP3: composition follows the cover, not the wireframe's bar

- **Chosen:** the provenance bar runs full-width under the headline
  distance with the split line beneath it — the cover's own arrangement —
  rather than the wireframe's bar beside the figure. People already read
  the cover that way; the image should not teach a second layout of the
  same three lines. Otherwise the wireframe as drawn: map full-width in
  the double rule with the scale bar inside it bottom-left; name in the
  display serif with the plate label right-aligned in tracked capitals;
  the dateline in mono; the legend row with drawn samples and the full
  fixed wording (the image has room the phone margin does not); the
  basemap credit and ROADBOOK on the foot line. The dateline drops the
  cover's fix count: the image has no line to explain it. The name at
  38 px and a lighter scale chip were the two changes after looking at
  the first render.
- **Rejected:** a translucent stats card over the map and a logo mark
  (the fitness-app export the plate is not — BRIEF §4); the summary
  block (time away, on the move, stopped) on the image — the brief's
  list is name, dates, distance with split, places, day count, legend,
  credit, and adding rows the image cannot caveat would crowd it.
- **Would change our mind:** the maintainer's eyeball at the STOP.

## 2026-09-25 — CP3: the offscreen map is the map slot's size, not the plate's

- **Chosen:** the export map is created at the slot's logical size
  (1136×528) with `pixelRatio: 2`, so its canvas lands pixel for pixel in
  the 2400×1600 composition; the plate's margin is paper drawn by the
  painter. The kickoff note said "at 1200×800 logical" — that is the
  plate; the map inside it is smaller by the margin, as the brief's
  wireframe draws it. Same style URL, same bounds, same fit padding as
  the on-screen map; `preserveDrawingBuffer` on this instance only;
  `fadeDuration: 0` so `idle` means every tile is at full opacity;
  destroyed in `finally`.
- **Rejected:** reading the on-screen canvas (blank without the buffer
  kept, and the buffer kept for the page's life is a per-frame cost the
  interactive map should not pay); rendering the map at 1200×800 and
  cropping (wastes tiles and shifts the fitted bounds).
- **Would change our mind:** a real browser where the offscreen readback
  is blank after the documented remedies — then §3A(3), the rasterised
  SVG, with the composition unchanged (BRIEF §8).

## 2026-09-25 — CP3: a status-0 fetch is worded as "blocked or unreachable"

- **Chosen:** every basemap error the offscreen map reports aborts the
  export with words and downloads nothing. A fetch that failed outright
  (MapLibre's AJAXError with status 0) is what a tile server without CORS
  headers looks like from inside a browser — and also what an
  unreachable one looks like; the browser does not distinguish them, so
  the message names both, the host, and that nothing was downloaded. An
  HTTP failure names its status (a transient 429 from the public tiles is
  the expected case; the message says to try again). A tainted 2D canvas
  (SecurityError at toBlob) has its own wording; so do the WebGL and
  timeout failures. The e2e CORS case aborts every basemap request at the
  network layer, which is the same status-0 path.
- **Rejected:** exporting with holes when one tile fails (a partial
  plate presented as the plate); swallowing basemap errors because the
  map "mostly" drew.
- **Would change our mind:** a way to tell CORS refusal from network
  failure in the browser — none exists.

## 2026-09-25 — CP3: the screenshot record carries half-size copies of the export

- **Chosen:** `docs/screens/phase14-cp3-plate-{1,2,3}.png` are the demo's
  three exported plates downscaled to 1200×800 — the full 2400×1600 files
  measure 383 KB, 430 KB, and 549 KB, and two of the three are over the
  informal 400 KB note, so the record takes the downscaled copy (the
  brief's own fallback), never an exception. The page captures
  (`phase14-cp3-*`) come from `capture.js` as usual.
- **Rejected:** committing the full-size PNGs (two over the note); no
  record at all (the whole point of the set is to see what the export
  looked like at this checkpoint).
- **Would change our mind:** nothing; a future format gets the same rule.
