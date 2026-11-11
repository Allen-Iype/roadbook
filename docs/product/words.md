# Words — the rule and the word list

Written 2026-11-02, step 1 of the "what the user gets" copy pass. The audit
behind it is `copy-audit.md` (every user-visible string, by screen, with
file and line). Nothing in the UI has changed yet; step 2 is the side-by-side
copy, step 3 the code.

## The rule

**Say what the user gets, not what Roadbook does.** For every sentence, ask
"so what does the user get?" until you reach something a traveller would
actually want. Lead with that; drop the mechanism or put it after.

Example: "Roadbook detects trips from your location history" → "Every trip
you've taken, found for you. No manual logging."

Audience: anyone who travels or rides, not engineers. If a word would need
explaining to a non-technical friend, replace it with the word from the list
below, everywhere, from now on.

Scope: everything a person sees — landing, sign-in, export guide, upload,
waiting and progress, maps, trip review, the adventure page, sharing,
uploads and delete, error messages, empty states, button labels, page titles,
meta descriptions, the text drawn on downloaded images. Nothing internal is
renamed: code, database, API, `openapi.yaml`, tests' identifiers and the
engineering docs keep their names. Only visible text changes.

Two things the words must not do:

- **Promise more than the product does.** Plainer words keep every privacy
  statement exactly as true as it is today. The typed confirmation word on
  "Delete everything" stays `delete`.
- **Hide the recorded/drawn distinction.** CLAUDE.md invariant 8: any
  geometry on a map carries its kind. The legend stays, four entries, visible
  on phones. Only its words are on the table.

## The word list

Technical word a person sees today → the plain word that replaces it. "Where"
is the screen it is met on first; the audit has every line.

### Trips and what you do with them

| today | from now on | where | note |
|---|---|---|---|
| candidate, candidates | **trip**; one not yet looked at is a trip **to review** | header, /candidates, upload "See your candidates", welcome fold, eyebrow | Decision 5: one word. |
| adventure, adventures | **trip**, trips; a kept trip is a **kept trip** when the state matters | nav, page names, meta, landing step 3, every page | Decision 5: testers found "adventure" confusing too. "Adventures" (nav) → **Trips**. |
| Candidates (page name, nav) | **Trips to review** | header, nav on the map | Decision 6. The count form is "Trips to review · 12". |
| confirm, confirmed | **keep**, kept | buttons, eyebrow, status line, triage notes, delete dialog | The landing already says "Keep the ones that mattered". |
| dismiss, dismissed | **leave out**, left out | buttons, eyebrow, triage notes | "Skip" was rejected: it sounds like "later", and a dismissed trip does not come back on its own. |
| undecided | **to review** / **left** | header count, select-all, empty states | |
| decision, decisions | **choice**, choices | orphaned-decisions line, delete dialog, aria "bulk decision" | |
| score | score | /candidates cards | Kept: plain English, taught in place by its note; the breakdown labels are already words. |
| plate, PLATE II, plate number | **removed (decision 1, amended 2026-11-10)** | /adventures intro and covers, eyebrow, map margin, sharing text, welcome fold, shared-view footer | |
| life map | **your map** ("← Map" in links) | "← Life map", welcome fold, empty states | |
| atlas | kept in the two taglines only ("an atlas of your own travels") | landing and sign-in taglines | Everywhere else ("your atlas fills…") it goes. |

### Where the line came from

| today | from now on | where | note |
|---|---|---|---|
| Observed / Routed / Unknown / Air (the legend labels) | **unchanged**, but the legend moves behind an (i) button on every map — **decision 2** | every map, the landing, the downloaded image | The labels and their one-line descriptions are read only by whoever taps the (i). Running text stops mentioning them. |
| observed (in prose: "640 observed", "observed track") | **recorded** | hover card, photo place lines, "How this route was drawn" | |
| routed | **by road** | same | |
| inferred | **drawn in** | shared-view footer | Already the word the bar caption uses. |
| provenance, the provenance bar | **the bar under the distance**; its note is titled **How much was recorded** | /adventures intro, the adventure page caption's Term | The bar itself has no label on screen; it needs none. |
| Fix, "one recorded position" | **Single point** — "one recorded spot"; inside the (i) with the rest | trip-page legend, downloaded image | |
| observation(s), recorded position(s), path points, points, raw | **point(s)**; the precise ones are **precise points** | upload phases, delete dialog, "How this route was drawn", /imports columns | One word for the thing the phone wrote down. |
| track ("1,210 km track") | **travelled** ("1,210 km travelled") | /candidates cards | |
| leg, legs | **stretch** | sharing dialog, divergence note | The word how-its-built.html already uses. |
| window, imported window, "the record begins/ends", span | **your file** ("already under way when your file begins") | cover flags, narrative edges, downloaded image | |
| divergence, "distance flagged", ground, "Google's ground figure" | **differs from Google's distance**; "by road" for ground | shelf flag, adventure-page flag and fold | |
| mode, "modes are guesses", activity data | **how you travelled, as Google guessed it**; "Google didn't record how" | "How this route was drawn" | |
| home base | **home** | life-map empty state | |
| measured ("91% of it measured") | **recorded** | downloaded image caption | |

### Files, uploads and what happens to them

| today | from now on | where | note |
|---|---|---|---|
| import, imports, imported, importing | **upload(s)** for the thing; **read** for the act ("Reading your file…") | nav, /imports h1, phase lines, delete text, error text | |
| Imports (page name, nav) | **Uploads** | header, nav on the map, welcome links | |
| export (noun: "a Timeline export", "your export") | **your Timeline file**, then **the file** | welcome, /imports panel heading, rejections, delete dialog | The verb stays where it is Google's menu: "Export Timeline data". |
| Timeline, Google Maps Timeline | Timeline | everywhere | Kept: Google's own name for the feature, and the word on the phone's menu. First mention on a page says "your Google Maps Timeline". |
| detection, detect, "detection ran", "detection failed", "detection step" | **finding your trips**; "Roadbook couldn't look for trips in this file" | phase lines, empty states, /imports status, welcome fold | |
| run, "Run 12", "latest run", re-detection, parameters | dropped | /candidates fold, orphaned line, API errors | Operator figures. The fold keeps the rule in words and loses the run number. |
| format, "detected format", "(zip)" | **kind of file**; the bracketed key goes | upload rejection panel, /imports column | "This looks like a ZIP file, not your Timeline file." |
| JSON, .json, "valid JSON", semanticSegments, top-level keys | **the Timeline file**; `.json` only where a person must spot it in Downloads or Files | rejections, welcome "the right file" | Rejections stop quoting parser output. |
| archive, extract, decompress, gzip, ZIP | **a ZIP file** — "open it and upload the Timeline file inside" | rejections, welcome | |
| Takeout, "Semantic Location History", Records.json, KML, XML | **an older Google download** (Takeout), named once | rejections, welcome "the right file" | Takeout is Google's product name; a person who has one knows the word. Name it once and say what to do instead. |
| server-side format, current format | **older format / the phone's own file** | welcome, rejection links | |
| binary, encrypted backup, decode(d), decodable, pixels, masquerading | **can't be read** / **can't be opened** | rejections, photo errors, photo-strip | |
| sidecar, metadata, "Takeout .json file", "metadata applied", "unpaired sidecar" | **Google's info file** (the small file beside each photo) | welcome photos fold, photo verdicts, results panel | |
| geotagged, "carries a usable position", "no position" | **has a location** / **no location** | welcome, photo verdicts, photo tiles | |
| capture time, "time source", exif / sidecar / filename (shown raw) | **taken** ("taken 24 Apr 10:12, from the photo") | photo tiles, strips, map popups | The source is said in words: from the photo · from Google's info file · from the filename. |
| positioned, "position from exif", "strip only" | **on the map** / **not on the map** | photo tiles | |
| thumbnail | **preview** | welcome, delete dialog, errors | |
| HEIC, JPEG | **the iPhone format (HEIC)**, JPEG | welcome photos fold, photo errors | Format names stay where a picker shows them; the first mention says whose format it is. |
| intake | **upload** | welcome verification note | |
| stream(s) | **goes straight to** | welcome "where your data goes" | |
| ingested, "observations it brought in", "retained copies of your uploaded exports", canonical copy | **read in**, **the places it recorded**, **the files you uploaded**, **your original files are untouched** | life-map empty state, delete dialog | Same promises, plain words. |
| batch, request, "per-request", "file part" | **at once** ("too many at once") | photo errors | |
| Visits / Activities / Path points / Positions / Skipped / New (table headers) | **Stops / Journeys / Points / Precise points / Unreadable / New** | /imports table | Figures unchanged; what `roadbook import` prints. |
| verified / unverified / "drafted from documentation" | **checked on a phone 11 Aug 2026** / **not yet checked on a phone** | export walkthroughs, photos fold | The stamp stays visible (a stale guide must say so); the QA register goes. |
| in-app browser | **the browser inside WhatsApp** | Android walkthrough note | |
| demonstration timeline, demonstration data, fictional | **sample** ("Sample trips. Yours will be drawn the same way.") | welcome plate caption and alt text | |

### The service, the account, sharing

| today | from now on | where | note |
|---|---|---|---|
| instance, "this instance" | **this Roadbook** | sign-in full state, API errors | Mostly done in phase 16; two left. |
| seat, seats | **spot** | sign-in full state, joined.html, delete text | The landing says spots. |
| waitlist, "Join the waitlist" | **the list**, "Join the list" | sign-in full state | The landing says list. |
| provider, "the provider reported" | **Google** ("Google didn't finish signing you in") | sign-in error | Google's raw code (`access_denied`) no longer shown. |
| "Sign in to continue" | **Sign in with Google** | sign-in button | The landing promised it; the button should say it. |
| API, "the Roadbook API is not reachable", roadbook serve, database | **Roadbook isn't answering right now. Try again in a minute.** | every page's failure state | The `roadbook serve` hint stays only when the session is auth-off (the person is the operator). |
| "the server log has details" | **something went wrong on this Roadbook's side** — then the operator hint in auth-off only | upload failures, /imports status | A hosted person cannot read a log. |
| HTTP n | **(error n)** | fallback error strings | Number kept for the report; the protocol word goes. |
| rendering, "Rendering…", render | **making the image** | download buttons and status | |
| overlay, "Download as overlay" | **Download for a story** (see-through background, 1080 × 1920) | adventure page | |
| basemap | **the map** | download status | |
| PNG, transparent, margin | **image**, **see-through background**, dropped | download description | |
| revoke, revoked | **stop sharing**, "turned off" | sharing list and dialog, shared not-found | |
| "keeps only a fingerprint of this link" | **Roadbook can't show this link again** | link-created panel | Same fact. |
| read-only | **shared with you** | shared view status | |
| "honest about what was recorded and what was inferred" | dropped; the legend says it | shared-view footer | |
| rows, "your line is deleted" | **your address** | joined.html | |
| "created nothing" | **you don't have an account yet** | sign-in full state | |

## Decisions — made 2026-11-02

The input behind this pass: non-technical testers said everything was too
technical and they could not work out how to move through the app. The
maintainer's reading: the person wants to see how far they went, from where
to where, by road or by air, on a map that looks good and reads at once.
Whether a stretch was recorded or drawn is not something they ask.

1. **Plate.** Decided as proposed. "PLATE II" stays as the printed numeral on
   covers and map margins (marginalia, aria-hidden, read like a page number);
   the word leaves every sentence. "Provenance" goes everywhere.
   *Rejected:* keeping and teaching both, the 2026-10-28 line. *Would change
   our mind:* a tester asking what the numeral means.
   *Amended 2026-11-10 (phase 17, issue #2):* that happened — the first
   testers asked what "PLATE 2" meant. The numeral is removed everywhere;
   the dates on each cover carry the order. See `docs/phase-17/DECISIONS.md`.

2. **The legend.** The four line styles stay drawn exactly as they are
   (CLAUDE.md invariants 5 and 8 are untouched: the kind is in the geometry).
   The legend — four labels, unchanged, with their one-line descriptions —
   moves behind an (i) button on every map, the trip page included. Every
   sentence in running text about recorded, drawn, unknown, gaps and
   divergence leaves the page; the bar under the distance stays as a thin
   mark with no caption; the account of how the route was drawn stays in its
   closed fold at the foot. This reverses DESIGN §6's "legend permanent in
   the margin" and phase 16 CP1's "descriptions visible at every width" —
   both get a dated amendment at the step 3 STOP, and `legend-wording.spec`
   flips to "reachable at every width". *Rejected:* renaming the labels (no
   longer worth a cross-cutting change once they sit behind a tap); drawing
   one undifferentiated line (a PRODUCT.md and CLAUDE.md amendment, not
   asked for). *Would change our mind:* a tester asking why one stretch is
   dotted — then the (i) is not findable enough.

3. **Go-side messages.** Message text changes; `Kind` keys, error types and
   the openapi contract do not. Each is a small commit with its package
   test. *Rejected:* a web-side map from kind to sentence (two places to
   keep in step).

4. **how-its-built.html is exempt.** It holds the mechanism for the reader
   who wants it. It must still follow decision 2 if its legend changes.

5. **One word: trip.** "Candidate" and "adventure" both go; a trip is a
   trip, "to review" or "kept" when the state matters. *Rejected:* the
   trip/adventure pair (the landing's usage) — testers found "adventure"
   confusing as well.

6. **Nav names: Trips · Trips to review · Uploads.**

7. **Operator hints only for the operator.** When the session is auth-off
   (one person, their own machine) the shell commands and "the server log"
   stay; hosted accounts see the plain sentence. One condition per state;
   named here because it is behaviour, not words.

## Where the words live

For step 3, the words with the widest reach, so the first commits are sized
right:

| word | files that print it |
|---|---|
| legend labels | `web/lib/legend.ts` (one source) → page legend, map margin, `plate-image.ts`, `landing/index.html` (pinned equal by test), `landing/how-its-built.html`, `share-links.tsx` bullets, DESIGN §6 |
| candidate / adventure / confirm / dismiss | `site-header.tsx`, `(app)/page.tsx`, `candidates/page.tsx`, `decide-cell.tsx`, `triage-*.tsx`, `adventure-view.tsx`, `adventures/page.tsx`, `upload-import.tsx`, `photo-upload-import.tsx`, `welcome/page.tsx`, `delete-my-data.tsx`, `terms.ts` |
| import(s) | `site-header.tsx`, `(app)/page.tsx`, `imports/page.tsx`, both uploaders, `welcome/page.tsx`, `delete-my-data.tsx`, `photo-strip.tsx`, `adventure-view.tsx`, `rejection-anchors.ts`, Go messages |
| plate / provenance | `terms.ts`, `adventures/page.tsx`, `adventure-view.tsx`, `share-links.tsx`, `shared/[token]/page.tsx`, `welcome/page.tsx`, `plate-export.tsx`, `plate-image.ts`, `landing/index.html` covers |
| point / position / observation | both uploaders, `delete-my-data.tsx`, `adventure-view.tsx` fold, `imports/page.tsx` headers, `candidates/page.tsx` fold, `plate-image.ts`, Go messages |
