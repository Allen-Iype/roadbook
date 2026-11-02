# Copy audit — every user-visible string, in the order a new user meets it

Written 2026-11-02 for the "what the user gets" copy pass (the rule and the
word list are in `words.md`). Step 1 of three: audit, then copy, then code.
Nothing in the UI has changed yet.

**Marks.** Each string carries one of:

- **OK** — a traveller would read it without stopping.
- **TECH** — contains a word that would need explaining to a non-technical
  friend (the word is named in the note).
- **HOW** — says what Roadbook does instead of what the person gets.
- **STALE** — no longer true.

A string can carry two marks. Strings that are pure data (a name, a date, a
number, a filename) are listed once per screen as "data" and not marked.
Lines are the file's line numbers at commit 45483eb; strings longer than a
line are abridged with "…".

**What is not here.** There are no emails: Roadbook sends none. The only
message a person ever receives is the hand-written note the maintainer sends
when a spot opens, which is not in the repository. There is no settings page:
the nearest things are the Sign out control in the header and the "Delete
everything" block on the imports page, both audited below. `landing/
how-its-built.html` is audited but exempted from the word list by design
(see `words.md`, decision 4).

---

## 1. Landing — `landing/index.html`

| line | string | mark | note |
|---|---|---|---|
| 6 | Roadbook — every road you've travelled, on one map | OK | page title |
| 7 | Roadbook turns the location history your phone already keeps into a map of every trip you've taken. | OK | meta description; "location history" is Google's own phrase |
| 167 | Roadbook · an atlas of your own travels | OK | |
| 168, 179, 379 | Make mine | OK | |
| 177 | Every road you've travelled, on one map. | OK | |
| 178 | Your phone has quietly kept a record of everywhere you've been. Roadbook turns it into a map of every trip you've taken. | OK | |
| 180 | Sign in with Google · 10 free spots | OK | |
| 181 | Only you can see your history. Delete all of it, any time. | OK | |
| 188 | A life map: three sample trips across Iceland, drawn as routes | OK | alt text |
| 307–310 | Observed — the phone recorded it / Routed — drawn along roads / Unknown — nothing recorded, straight line / Air — a flight | TECH | the four labels; descriptions are plain. Bound by DESIGN §6; see `words.md` decision 2 |
| 312 | All your trips on one map. These are sample trips. Yours will be drawn the same way. | OK | |
| 320–321 | Each trip, day by day. The route, the distance, the days, the countries, and your photos along the way. | OK | |
| 324–347 | PLATE 1 / Iceland / South coast drive / 24–26 April 2026 · 3 days / 700 km | TECH | "PLATE n" marginalia on the covers (aria-hidden); see decision 1 |
| 352 | Roadbook's page for one trip: the name, the distance over three days, the stops, and the route on a map beside it. | OK | alt text |
| 353 | The app itself. | OK | |
| 359–363 | Three steps: Export your history from Google Maps… / Upload that file. Roadbook reads it and finds the trips that stood out — the ones far from home, with somewhere you stayed. / Keep the ones that mattered… | OK, HOW | step 2 describes the mechanism; one clause |
| 365 | No Timeline history? Start from your photos instead… | OK | "Timeline" is Google's product name |
| 371 | Your history stays in your own account… Your export is kept so Roadbook can re-read it later. Photos are never kept, only where and when each was taken… | HOW | "export is kept so Roadbook can re-read it later" is a mechanism; the rest are promises and stay |
| 376–380 | 10 spots, then a list / Roadbook is opening to its first ten people… / If all ten are taken, you'll land back here. Leave your email… | OK | |
| 382–387 | Email / Join the list / Nothing else, and we'll delete your address if you ask. | OK | |
| 395–400 | Source on GitHub / Licensed AGPL-3.0 / Run it yourself / How it's built / Made by one person, for a small pilot. | OK | footer; "AGPL-3.0" is a licence name in a footer, left |

## 2. How it's built — `landing/how-its-built.html`

Deliberately the engineer's page: the landing's footer link for "anyone who
wants to check the claim before trusting a line on a map" (line 182). Every
paragraph is HOW by design. Audited, not rewritten; see `words.md` decision 4.

| line | string | mark | note |
|---|---|---|---|
| 6–7 | Roadbook — how it's built / How Roadbook draws a route: the four kinds of line… | OK | its purpose is the mechanism |
| 182 | The landing page says what Roadbook does. This page says how the drawing is made… | OK | says who it is for |
| 185–192 | Every line says how it is known… Observed / Routed / Unknown / Air paragraphs | HOW | by design |
| 195 | Where the figures come from… | HOW | by design |
| 198–200 | Where your data lives… request log sentence… no public demo | OK | promises; stays |
| 203–204 | Run it yourself… / ← Back to Roadbook | OK | |

## 3. Waitlist replies — `landing/joined.html`, `landing/oops.html`

| file:line | string | mark | note |
|---|---|---|---|
| joined:6, 26 | You're on the list — Roadbook / You're on the list. | OK | |
| joined:27 | When seats open again, one email arrives telling you to sign in — that is the only mail this list ever sends. | TECH | "seats"; the landing says "spots" |
| joined:29 | Change your mind? Reply to that email, or ask any time, and your line is deleted. | TECH | "your line" (a database row) |
| joined:31 | Back to the page | OK | |
| oops:6, 25 | That didn't look like an address — Roadbook / That didn't look like an email address. | OK | |
| oops:26 | Nothing was recorded. Head back and check for a typo — the form needs a working address, because the entry email is the only way in. | STALE | the entry email was retired at phase 15 CP2; sign-in is open |
| oops:28 | Back to the form | OK | |

## 4. Sign-in — `web/app/(public)/signin/page.tsx` and the messages behind it

| file:line | string | mark | note |
|---|---|---|---|
| signin:22 | Roadbook — sign in | OK | title |
| signin:48, 54 | ROADBOOK / A road atlas of your own past | OK | |
| signin:57 | Sign in | OK | |
| signin:60 | This Roadbook keeps each person's adventures their own. Sign in and you'll only ever see yours. | OK | a promise |
| signin:69 | Roadbook is full for now. | OK | |
| signin:71 | Every seat on this instance is taken, so your sign-in created nothing. Join the waitlist and one email arrives when a seat opens. Nothing else is ever sent. | TECH | "seat", "instance", "waitlist", "created nothing" |
| signin:81 | Join the waitlist | TECH | landing says "Join the list" |
| signin:93 | Sign-in didn't complete: {error} | TECH | `{error}` is a Go string, below |
| signin:106 | Sign in to continue | HOW | does not say with what; the landing promised "Sign in with Google" |
| api/auth/callback:34 | sign-in failed | TECH | fallback `{error}` |
| internal/api/auth.go:140 | the provider reported: {Google's code} | TECH | "provider", and Google's raw code (e.g. `access_denied`) |
| internal/api/auth.go:118, 137 | this instance runs without sign-in | TECH | reachable only by hand-typing the route in auth-off mode |
| internal/api/auth.go:66 | no session — sign in first | TECH | JSON 401; the page redirects before a person sees it |

## 5. Welcome and export guide — `web/app/(public)/welcome/page.tsx`

| line | string | mark | note |
|---|---|---|---|
| 32 | Roadbook — start here | OK | title |
| 34 | What Roadbook is, how to export your Google Timeline, and where to upload it. | HOW | meta description |
| 50 | The adventures you actually took. | OK | |
| 53 | Roadbook reads the record your phone already keeps of where it went, draws the trips that stood out, and leaves it to you to say which were adventures. | HOW | three mechanisms; says nothing the person gets |
| 66 | THREE ADVENTURES ACROSS ICELAND · DEMONSTRATION TIMELINE | TECH | "demonstration timeline" |
| 72 | Fictional demonstration data. Yours is drawn the same way, from your own record. | TECH | "demonstration data", "record" |
| 82 | Your own, in three steps | OK | |
| 87–92 | Export your Timeline from Google Maps on your phone — a couple of minutes. / Upload the file here. / Confirm the trips that were adventures; the rest stay off the map. | OK | "confirm" → see word list |
| 106 | Don't have the file yet? | OK | |
| 109 | Google makes the export on the phone itself. Do it on the phone you're reading this on and the file is already where the button above can find it. | OK | |
| 116–118 | On Android / Verified 11 Aug 2026 / Walked 11 Aug 2026 on an Android device from a WhatsApp-opened link: the steps matched the phone's menus and the upload completed in the in-app browser. | HOW | a QA stamp shown to users; "in-app browser". Visible on purpose since phase 7 (a stale guide must say so) — keep the fact, plain the words |
| 121–141 | Open Settings → Location → Location Services → Timeline… / Choose Export Timeline data and confirm. / The phone saves a .json file to Downloads. / Come back here and choose the file… | OK | Google's own menu names; ".json" is what the person sees in Downloads |
| 147–167 | On iPhone / Open the Google Maps app… / Open Timeline settings… / Save the file from the share sheet… / Come back here and choose the file… | OK | |
| 178–187 | Never had Timeline on? Then there is nothing to export — Google holds no location record for you. Start from your photos instead. | OK | |
| 195 | Or start from photos | OK | |
| 198–201 | A photo straight from the camera carries where and when it was taken, and that is enough to find your trips — no export step, the picker opens your photo library. Original photos work best. | OK | |
| 208 | Which photos work | OK | fold title |
| 210–211 | Camera originals, and downloads from your own Google Photos. iPhone photos are usually HEIC — upload them as they are; if the picker offers JPEG, either works. | TECH | "HEIC", "JPEG" — format names a person meets in the picker; keep but say "iPhone's own format" first |
| 215–216 | Photos received in a chat almost never carry a location — messaging apps strip it. Screenshots never have one. | OK | |
| 219–222 | A Google Photos Takeout includes small .json files beside the photos. Upload them together: they can restore a location the photo file lost. | TECH | "Takeout", ".json files beside the photos" |
| 225–228 | On their own, photos need a little context: home is worked out from where your photos recur, so include a spread from ordinary weeks too… | HOW | the instruction (include ordinary weeks) is useful; the reason is mechanism |
| 232–234 | Photo intake verified 26 Aug 2026 with a real camera roll. Not yet walked on a phone — the iPhone notes come from documentation, not a device. | HOW, TECH | QA stamp; "intake" |
| 241 | What the right file looks like | OK | |
| 243–247 | A single .json file, usually named Timeline.json. On Android it lands in Downloads; on iPhone… the Files app. | OK | the person must recognise the file |
| 251–253 | Got a .zip archive? Extract it and upload the .json file inside. | TECH | "archive", "extract" |
| 256–260 | Files from a Google Takeout download — monthly "Semantic Location History" files or Records.json — are the old server-side format, which Roadbook does not read. The phone export is the current one. | TECH | "Takeout", "server-side format" |
| 263–264 | Photos, PDFs and KML files are not location exports — the upload will say so and point back here. | TECH | "KML" |
| 267–269 | Using someone else's Roadbook link on a computer? Export on the phone first and move the file over — or open the same link on the phone… | OK | |
| 274 | Where your data goes | OK | |
| 276–284 | Your file streams to this Roadbook and nowhere else, and is kept here as-is so later versions can re-read it without asking you for the file again. Photos are not kept — only the position, the time and a small thumbnail of each. Everything you upload is listed on Imports, and can be deleted from there. | HOW, TECH | "streams", "re-read", "thumbnail", "Imports" (nav word) |
| 288 | What happens after you upload | OK | |
| 291–293 | Upload. Nothing is recorded until the whole file has arrived — if the connection drops, just try again. | OK | |
| 296–298 | Import. Every visit, journey and recorded position is read in; each geotagged photo contributes the position it carries. | HOW, TECH | "Import", "geotagged" |
| 301–306 | Detection. Roadbook looks for journeys far from home with a real destination — somewhere you stayed, not somewhere you passed through — and lists them as candidates. Home is worked out from the data itself. | HOW, TECH | "Detection", "candidates" |
| 309–312 | You decide. Confirm the real adventures and name them; dismiss the rest. The confirmed ones are drawn on your life map and shelved as plates in your atlas. | TECH | "confirm/dismiss", "plates" |
| 316–320 | Once the upload finishes you can close the tab — Imports keeps every attempt and its outcome. | OK | "Imports" → nav word |
| 403, 416 | Unverified / Drafted 11 Aug 2026 from documentation, not yet verified on a device — Google changes these menus without notice. | HOW | QA stamp; keep the warning, plain the words |
| welcome-plate:18 | A life map drawn from the demonstration timeline: three adventures across Iceland — an observed drive along the south coast, a routed loop through the Westfjords, and a return flight to Akureyri. | TECH | alt text: "demonstration timeline", "observed", "routed" |

## 6. Upload and waiting — `web/components/upload-import.tsx`, `photo-upload-import.tsx`, `web/app/loading.tsx`, the Go rejections

| file:line | string | mark | note |
|---|---|---|---|
| upload:189 | Choose the Timeline file / Choose another file | OK | door button |
| upload:204 | Add a Timeline export | TECH | panel heading on /imports; "export" as a noun |
| upload:212–218 | The file exported from Google Maps Timeline on your phone. Keep this page open while it uploads. Don't have the file yet? | OK | |
| upload:232 | Choose file… / Choose another file… | OK | |
| upload:256–257 | Uploading — 42% (keep this page open) | OK | |
| upload:267 | Uploaded. Importing your observations… | TECH, HOW | "Importing", "observations" |
| upload:269 | Imported. Looking for adventures… | TECH | "Imported" |
| upload:274–278 | Done. This file held nothing new — every observation was already imported. | TECH | "observation", "imported" |
| upload:287 | See your candidates → | TECH | "candidates" |
| upload:295–297 | That file isn't a Timeline export (zip) | TECH | the format key in brackets (`zip`, `not-json`, `json-unrecognised`…) is a machine label |
| upload:301 | Nothing was stored — try the right file. | OK | |
| upload:308–315 | The import failed / {message} / The attempt is recorded on the imports list. | TECH | "import", "imports list" |
| upload:89 | The upload didn't complete — nothing was recorded. Check the connection and try again. | OK | |
| upload:108, 116 | This file was not recognised. / The upload failed (HTTP 502). | TECH | "HTTP n" |
| upload:149 | Your data imported fine, but the detection step failed — the server log has details. | TECH, HOW | "detection step", "server log" — a hosted user cannot read it |
| rejection-anchors:46–58 | How to export the current format from your phone / What the right file looks like / Export a fresh copy from your phone and upload that / Photos import separately — add them in the photos section | TECH | "current format", "import" |
| parse.go:54 | export Timeline data on the phone: Settings → Location → Location Services → Timeline → Export Timeline data | OK | appended to every rejection; fine on Android, wrong menu on iPhone |
| parse.go:94, 280 | the file is empty; … | OK | |
| parse.go:298 | this is a gzip archive, not a Timeline JSON file — decompress it and import the JSON inside | TECH | "gzip", "JSON", "decompress", "import" |
| parse.go:300 | this is a ZIP archive (a Takeout download?) — extract it and import the Timeline JSON file inside | TECH | "archive", "Takeout", "JSON" |
| parse.go:302, 305 | this is a PDF, not a location export / this is an image file, not a location export | OK | |
| parse.go:313 | this is a web page — Takeout's archive_browser.html is a table of contents, not data; import the Timeline JSON file instead | TECH | |
| parse.go:316, 318 | this is KML from the old Timeline interface, which Roadbook does not read / this is XML, not a Timeline JSON export | TECH | "KML", "XML", "JSON" |
| parse.go:323 | unreadable binary data — possibly an encrypted Timeline backup, which cannot be imported; instead | TECH | "binary"; the sentence also ends mid-air ("; instead" + hint) |
| parse.go:325 | not JSON (the file starts with "…") | TECH | |
| parse.go:334–340 | (semantic-history / records-json / my-activity rejections) | TECH | the old-format family; wording not shown here, same register |
| parse.go:160 | JSON, but with no semanticSegments section — top-level keys found: …; this is not a phone Timeline export; … | TECH | key names shown to the person |
| parse.go:405 | the file ends mid-record — the export or the copy of it was cut short; re-export or re-copy the file | OK | |
| parse.go:414 | the file is not valid JSON (…); if it was edited or converted, start from a fresh export — … | TECH | "valid JSON" plus a parser error in brackets; also the message the iOS export gets today (competitive study §3.2) |
| imports.go:38 | an import is already running — watch it on the imports page and retry when it finishes | TECH | "import" |
| imports.go:96 | the file exceeds the N MB upload limit | OK | |
| imports.go:103 | the request carried no file part — choose a Timeline export and try again | TECH | "request", "file part" |
| imports.go:161, 175, 193 | internal error during import — the server log has details / the stored upload could not be reopened: … / storing observations failed: … | TECH | shown in the imports table's status cell |
| photo-upload:175 | Choose photos / Choose more photos | OK | |
| photo-upload:187 | Add photos | OK | |
| photo-upload:194–196 | Photos that carry a time and place; the picker opens your photo library. Original photos work best — keep the screen on while they upload. | OK | |
| photo-upload:228–230 | Uploading 12 files — 42% (keep this page open) | OK | |
| photo-upload:240 | Photos imported. Looking for adventures… | TECH | "imported" |
| photo-upload:245–249 | Done. These photos held nothing new — every position was already imported. | TECH | |
| photo-upload:258 | See your candidates → | TECH | |
| photo-upload:92 | The upload didn't complete — nothing was recorded. Check the connection, keep the screen on, and try again. | OK | |
| photo-upload:140 | Your photos imported fine, but the detection step failed — the server log has details. | TECH, HOW | |
| photo-upload:265 | The import failed | TECH | |
| photo-upload:296–301 | 9 of 12 files carried a usable position (2 sidecars matched). | TECH | "sidecar", "usable position" |
| photo-upload:306 | 3 files added nothing — why | OK | |
| photo-upload:326–332 | no place in the photo — a copy from a messenger has it stripped; use the original / a location but no readable capture time / a Takeout .json file naming no photo in this batch — include the photo it describes / not a supported photo format | TECH | "capture time", "Takeout .json", "batch" |
| photos.go:85, 163, 356, 365 | (sidecar error) / the image could not be decoded — it may be corrupted or a format masquerading as JPEG / over the N-files-per-request limit — upload the rest in another request / larger than the N MB per-file limit | TECH | "decoded", "masquerading as JPEG", "per-request" |
| loading.tsx:10 | Loading candidates… | TECH, STALE | the global loading screen, shown for every page |

## 7. Life map — `web/app/(app)/page.tsx`, `life-map.tsx`, `legend.tsx`, `terms.ts`, `summoned-list.tsx`, `site-header.tsx`

| file:line | string | mark | note |
|---|---|---|---|
| layout.tsx:34–35 | Roadbook / The adventures you actually took. | OK | site title and meta |
| header:35–50 | ROADBOOK / Adventures / Candidates — 12 undecided / Imports | TECH | "Candidates", "undecided", "Imports" |
| header:65 | Sign out | OK | |
| page:59 | The Roadbook API is not reachable. Start it with roadbook serve and reload. | TECH | shown to hosted users too; "API", a shell command |
| page:82 | No adventures found — yet | OK | |
| page:85–88 | Your data imported fine. Detection ran and found no journeys far from home with a real destination — it looks for time spent well away from every home base, staying somewhere, and pass-throughs don't count. | HOW, TECH | "imported", "Detection ran", "home base" |
| page:91–105 | More data may change that: add another export — earlier years, another device — or see Imports for what was ingested. | TECH | "ingested" |
| page:111–129 | No adventures yet / Roadbook draws your life map from the journeys in a Google Timeline export. Data has been imported but detection has not run yet. / roadbook detect -from-db / Or add your data through the browser, which runs detection automatically. | TECH, HOW | reachable only after a CLI import; a shell command on screen |
| page:144–149 | 12 candidates await review / The life map draws confirmed adventures only — none are confirmed yet. Confirm the real journeys and they appear here as routes. | TECH, HOW | "candidates", "confirmed" |
| page:156 | Review candidates — 12 undecided | TECH | |
| page:190–193 | 3 confirmed adventures exist, but no journey could be loaded from the API. Reload; if it persists, check that the API and database are up. | TECH | |
| page:208–231 | ROADBOOK / Adventures / Candidates — n undecided / Imports / Sign out | TECH | the map's own panel repeats the header |
| page:177 | Unnamed adventure | OK | |
| legend.ts:17–20 | Observed — the phone recorded it / Routed — drawn along roads / Unknown — nothing recorded, straight line / Air — a flight | TECH | the labels; see decision 2 |
| legend.tsx:61 | Legend | OK | aria-label |
| terms.ts:16–60 | Plate / The provenance bar / Observed / Routed / Unknown / Air / Original photos / Candidate / Score (each with a one-paragraph note) | TECH | the notes are plain; the terms are the words being taught. "provenance" and "plate" are on the list |
| term.tsx:100 | Got it | OK | |
| life-map:213–216 | 700 km — 640 observed | TECH | hover card: "observed" |
| life-map:225 | Click to open | OK | |
| summoned-list:36, 49, 56 | List (3) / Adventures (3) / Close | OK | |
| summoned-list:40 | Confirmed adventures | TECH | aria-label |

## 8. Trip review — `web/app/(app)/candidates/page.tsx`, `decide-cell.tsx`, `triage-*.tsx`

| file:line | string | mark | note |
|---|---|---|---|
| page:162 | Candidates | TECH | h1 |
| page:170–171 | Each candidate is a trip Roadbook found in your record. Confirm the adventures; dismiss the rest. | TECH | "candidate", "confirm/dismiss" |
| page:99–104 | Roadbook looked and found no trips. / Your data imported fine. A trip is time spent well away from home with a real stay somewhere; short outings and pass-throughs don't count. | OK, TECH | "imported"; otherwise plain |
| page:107–114 | More data may change that: add another export — earlier years, another device. | OK | |
| page:119–127 | No candidates. Add a Timeline export — or import one with the CLI and run roadbook detect -from-db. | TECH | CLI on screen; shown in hosted mode too |
| page:138–144 | ⚑ 2 decisions no longer match a candidate under the current parameters. They are preserved and will re-match if parameters change back. | TECH, HOW | "parameters"; an operator state |
| page:184–205 | Order: by date · by score | OK | |
| page:210 | After each decision, focus moves to the next undecided card. | TECH | keyboard hint (sm+ only); "undecided" |
| page:234 | How these 12 were found | OK | folded |
| page:240–254 | Time spent more than 100 km from home, with somewhere you stayed, counts as a trip. 3 recorded positions too fast to be real were left out first. Run 12. | TECH | "Run 12" |
| page:304 | route shape unavailable | TECH | |
| page:342–345 | score 73 | TECH | taught by a Term; plain English word, kept — see word list |
| page:349–351 | 3 days · 420 km away · 1,210 km track · 4 stops · 2× visited | TECH | "track" |
| decide-cell:109–176 | save / cancel / confirm / dismiss / dismissed / change / keep as is | TECH | "confirm", "dismiss(ed)" |
| decide-cell:217–220 | distance from home / time at the farthest stay / how much was recorded / length of the trip | OK | |
| decide-cell:236 | score 73.2 of 100 — what it is made of | OK | |
| decide-cell:254 | no record of it; its share went to the others | HOW | |
| triage-select:20 | Select this candidate for a bulk decision | TECH | aria-label |
| triage-select-all:25–26 | All 12 undecided selected / Select all 12 undecided | TECH | "undecided" |
| triage-bar:71, 80, 87, 94, 104 | Bulk decision / 4 selected / Dismiss selected / Confirm selected / Clear | TECH | |
| triage-bar:34 | Dismissed 4 candidates. | TECH | |
| triage-bar:37, 47 | Nothing was recorded — {error} | OK | |
| triage-bar:52–63 | Confirmed 4 candidates — 3 with suggested names, 1 named by date. Rename any from its card or adventure page. | TECH | |
| actions.ts:111 | Journey of 2026-04-24 | OK | default name |
| actions.ts:35–245 | decide failed (HTTP n) / bulk decide failed (HTTP n) / upload failed (HTTP n) / delete failed (HTTP n) / creating the link failed (HTTP n) / revoking failed (HTTP n) / deletion failed (HTTP n) | TECH | fallbacks when the API gives no message |
| server.go:125, 313, 351, 429 | no such candidate in the latest run — re-detection may have replaced it; reload the list | TECH | "run", "re-detection" |
| server.go:332–412 | action must be 'confirmed' or 'dismissed' / confirming requires a non-empty name / decisions must not be empty / candidate n appears twice — one decision per candidate | TECH | unreachable through the UI's own controls; listed for completeness |

## 9. Adventures shelf — `web/app/(app)/adventures/page.tsx`

| line | string | mark | note |
|---|---|---|---|
| 163 | Adventures | OK | |
| 126–130 | 3 adventures, the latest first. Each is a plate, numbered in the order you travelled. The bar under each distance shows how much of it the phone recorded. | TECH | "plate"; the bar sentence is plain |
| 69–73 | No adventures yet / Your atlas fills as you confirm candidates — the trips Roadbook found in your record. | TECH | "confirm candidates" |
| 80–81 | Review candidates — 12 undecided | TECH | |
| 193 | PLATE 2 | TECH | marginalia (aria-hidden) |
| 210 | distance flagged | TECH | screen-reader text for ⚑ |
| 216–220 | 24–26 Apr 2026 · 3 days / 700 km | OK | data |
| 175 | Unnamed adventure | OK | |

## 10. Adventure page — `adventure-view.tsx`, `day-narrative.tsx`, `photos-section.tsx`, `photo-strip.tsx`, `route-map.tsx`, `plate-export.tsx`, `share-links.tsx`, `format.ts`, `plate-image.ts`

| file:line | string | mark | note |
|---|---|---|---|
| page:99 | ← Life map | OK | |
| view:278–292 | SHARED ADVENTURE / CONFIRMED ADVENTURE · PLATE II / DISMISSED CANDIDATE / CANDIDATE | TECH | eyebrow: "confirmed", "plate", "dismissed", "candidate" |
| view:318–321 | ⚑ Already under way when the record begins. / Still under way when the record ends. | OK | |
| view:333 | Read-only · shared by its owner | TECH | "read-only" |
| view:339–342 | confirmed 12 Oct 2026 / Undecided — confirm or dismiss in the candidates gallery | TECH | |
| view:381–383 | 700 km over 3 days | OK | |
| view:394 | How much of this was recorded, how much drawn in | OK | the bar's caption |
| view:398–439 | Time away / On the move / no record of it / by Google's count / 43 km/h average where recorded / Stopped / no stops recorded / 4 stops · 31 h / Farthest 420 km from home | OK | |
| view:143–149 | DAY 2 HIGHLIGHTED / PLATE II · FULL ROUTE / FULL ROUTE | TECH | marginalia: "plate" |
| view:156–157 | Fix — one recorded position | TECH | "Fix" |
| view:163 | Stop | OK | |
| view:208–227 | 3 photos the owner added to this adventure, placed by capture time against the drawn route. / 2 photos from the owner's photo imports were taken inside this journey's window — placed by capture time against the drawn route. | HOW, TECH | "capture time", "drawn route", "photo imports", "window" |
| view:461 | How this route was drawn | OK | folded, at the page foot |
| view:478–481 | observed 640.0 km · routed 40.0 km · unknown 12.0 km · air 0.0 km | TECH | inside the fold |
| view:484–498 | 312 recorded points (280 from the timeline + 32 raw) · Google's own distance 702.0 km · countries and regions from the route's recorded points | TECH | inside the fold: "raw", "timeline" |
| view:517 | No mode record — this journey's evidence carries no activity data. | TECH | "mode", "activity data" |
| view:524 | By mode, as Google labelled it (modes are guesses): passenger vehicle 640.0 km, 9 h 10 min · … | TECH | "mode" |
| view:549–558 | Distance check: drawn on the ground 652.0 km · Google's ground figure 702.0 km (−7.1%). Gaps with no road under-count, detours over-count, and a record cut short compares against a door-to-door figure — the legs explaining it are on the map. | TECH, HOW | "ground", "legs" |
| view:576–578 | ⚑ The drawn distance and Google's differ by 7% — the account below says why. | OK | |
| view:616 | Journey of 2026-04-24 | OK | |
| narrative:119–134 | Show the whole route again / Highlight this day on the map / DAY 1 / Fri 24 Apr / 120 km | OK | |
| narrative:204–271 | Nothing recorded this day. / Journey begins · already under way when the record starts / Nothing recorded from Fri 24 Apr 21:10 to Sat 25 Apr 08:00 / Travelling all day / 2 stops / On the move / 43 km/h average where recorded / overnight into Day 2 / Journey ends · still under way when the record stops | OK | |
| narrative:316–337 | Stayed 2 h 10 min / Stayed · through to Day 3 / Stayed all day / Moved on after 9 h | OK | |
| narrative:350–351 | position not recorded / Position not recorded — this stay is not on the map | OK | |
| photos-section:110–119 | Photos / Add photos / Uploading… | OK | |
| photos-section:142 | None yet. Original photos work best. | OK | |
| photos-section:88 | The batch could not be sent — most likely too large in one request. Try fewer photos at a time. | TECH | "batch", "request" |
| photos-section:199–213 | capture time from exif / 2026-04-24 10:12 · exif / no capture time / positioned · exif / position from exif / no position — strip only | TECH | raw source names (`exif`, `sidecar`, `filename`) shown as text; "strip only" |
| photos-section:226–234 | ⚑ 1.2 km from the observed track at this time / outside this journey's timeline | TECH | see format.ts |
| photos-section:254 | sidecar | TECH | pending tile |
| photos-section:272–276 | added / already uploaded / rejected / metadata applied / unpaired sidecar | TECH | "metadata", "sidecar" |
| photos.go:116, 123 | names no image in this upload — include the photo it describes in the same request / HEIC (the iPhone default format) cannot be attached — attaching needs a thumbnail and HEIC pixels cannot be decoded here; convert to JPEG, or download it from Google Photos, which serves JPEG | TECH, HOW | |
| photos.go:49, 212 | photos attach to confirmed adventures — confirm this candidate first | TECH | |
| photos.go:286 | thumbnail file missing from the photos directory — the photo must be re-uploaded | TECH | |
| photo-strip:42–43 | No preview — this format's pixels aren't decodable here | TECH | |
| photo-strip:51–53 | capture time from exif / 2026-04-24 10:12 · exif | TECH | |
| photo-strip:69 | outside the drawn route's elements | TECH | |
| photo-strip:90–94 | 2 photos from your photo imports were taken inside this journey's window — placed by capture time against the drawn route. They stay with their import; deleting is not done from here. | HOW, TECH | |
| route-map:169–171 | no capture time | TECH | map popup |
| format.ts:70–79 | from the observed track at this time / from the routed road at this time / from the straight-line gap at this time / from the stop at this time / over an air leg — not checked against the arc | TECH | "observed track", "routed", "air leg", "arc" |
| format.ts:52 | passenger vehicle / bus / flying … | OK | Google's mode names, lower-cased |
| plate-export:80–91 | Download as image / Download as overlay / Rendering… | TECH | "overlay", "Rendering" |
| plate-export:95–97 | Image: PNG, 2400 × 1600 — the map with its route, and the margin as printed here. Overlay: PNG, 1080 × 1920, transparent — route and figures only, for your own photo or story. Photos are not drawn in either. | TECH, HOW | "PNG", "transparent", "margin" |
| plate-export:109–110 | Rendering the plate — the basemap loads once more at the export size. / Rendering the overlay. | TECH, HOW | "plate", "basemap" |
| plate-export:113 | Downloaded south-coast-drive.png. | OK | |
| plate-export:55 | The export failed (…). Nothing was downloaded. | TECH | "export" = download here |
| share:87, 96 | Sharing / Create a share link | OK | |
| share:101–102 | Not shared. A share link lets anyone who has it open this plate — map, days, countries, and photos — without signing in. | TECH | "plate" |
| share:112–113 | Link created 12 Oct 2026 — open to anyone who has it | OK | |
| share:121 | Revoke | TECH | |
| share:135, 152 | Share this adventure | OK | |
| share:155–169 | Anyone with the link can open this plate, without signing in. They will see: the route map, each leg marked observed, routed, unknown, or air — exactly as it is drawn for you; the day-by-day narrative, distances, and countries; all 3 photos on this adventure, and any you add later — photo imports included. | TECH, HOW | "plate", "leg", the four labels, "photo imports" |
| share:173–174 | Nothing else of yours is reachable through it. You can revoke the link from this page at any time. | TECH | "revoke"; the promise stays |
| share:188–195 | Create link / Creating… / Cancel | OK | |
| share:220–224 | Link created / Copy it now. Roadbook keeps only a fingerprint of this link, so it cannot be shown again — but it can be revoked from this page at any time. | HOW, TECH | "fingerprint", "revoked" |
| share:231–252 | Share link / Copy link / Copied / The link is on your clipboard. / Done | OK | |
| plate-image:290–293 | The record starts mid-journey — it began before the imported window. / Still in progress at the window's edge — the end shown is the cut, not the return. | TECH | on the downloaded image: "imported window" |
| plate-image:301–303 | DAY 2 HIGHLIGHTED / PLATE II · FULL ROUTE / FULL ROUTE | TECH | on the image |
| plate-image:441, 713 | drawn — 91% of it measured | TECH | "measured"; on the image |
| plate-image:491–509, 763–781 | Observed / Routed / Unknown / Air (with legend.ts descriptions) / Fix / Stop | TECH | on the image; follows legend.ts |
| plate-image:535, 791 | ROADBOOK | OK | |

## 11. Shared view — `web/app/(public)/shared/[token]/page.tsx`, `not-found.tsx`

| file:line | string | mark | note |
|---|---|---|---|
| page:25 | Roadbook — shared adventure | OK | title |
| page:50–51 | This Roadbook is not reachable right now. Try the link again in a little while. | OK | |
| page:76–81 | This plate was shared from someone's Roadbook — an atlas drawn from their own location history, honest about what was recorded and what was inferred. How Roadbook works, and how to make your own. | TECH, HOW | "plate", "inferred"; the one sentence a stranger reads |
| page:104–106 | ROADBOOK / A shared adventure | OK | |
| not-found:20–30 | This link doesn't open anything. / Whoever shared it may have revoked it, or the adventure it led to is no longer available. If someone sent you this link, ask them for a fresh one. / What Roadbook is | TECH | "revoked" |

## 12. Imports and delete — `web/app/(app)/imports/page.tsx`, `delete-my-data.tsx`

| file:line | string | mark | note |
|---|---|---|---|
| page:93 | Imports | TECH | h1 and nav |
| page:32 | The Roadbook API is not reachable… | TECH | as §7 |
| page:51–59 | Nothing added yet. / Nothing added yet — upload above, or run roadbook import -src Timeline.json. | OK, TECH | the CLI hint is auth-off only |
| page:111 | Added so far | OK | |
| page:126–136 | Added / Source / Format / Dates / Visits / Activities / Path points / Positions / Skipped / New / Status | TECH | "Activities", "Path points", "Positions", "Skipped" are pipeline figures |
| page:104–105 | Timeline export / Photos | OK | format labels |
| page:220 | everything | OK | dates cell when the file has no bounds |
| page:182–213 | failed / running / completed / detection failed — the server log has details | TECH | |
| delete:50 | Delete everything | OK | |
| delete:55 | Removes every import, adventure, decision, photo, and share link of yours, and forgets your account. Nothing of anyone else's is touched. | TECH | "import", "decision" |
| delete:56 | Removes every import, adventure, decision, photo, and share link. This Roadbook is empty again, as if freshly installed. | TECH | auth-off wording |
| delete:58 | Nothing is stored yet. Deleting forgets your account and frees its seat; you can sign in again any time. | TECH | "seat" |
| delete:59 | Nothing is stored yet — there is nothing to delete. | OK | |
| delete:72, 86 | Delete everything… / Delete everything? | OK | |
| delete:88–98 | This removes, for good: every import and the observations it brought in; every candidate and decision — names included; every photo thumbnail and photo record; every share link — the links stop opening immediately; your account here — signing in again starts from nothing. / the retained copies of your uploaded exports. | TECH | "import", "observations", "candidate", "decision", "thumbnail", "retained copies" |
| delete:101–102 | Your own export files, wherever you keep them, are untouched — they are the canonical copy of your history. There is no undo. | TECH | "canonical" |
| delete:105 | Type delete to confirm | OK | **the typed word stays exactly "delete"** |
| delete:125, 132 | Delete everything / Deleting… / Keep my data | OK | |

## 13. Global — `web/app/error.tsx`

| file:line | string | mark | note |
|---|---|---|---|
| error:17–23 | Something failed while rendering. / {error.message} / Try again | TECH | "rendering"; a raw error message |

---

## Totals

Counted over the rows above.

| screen | rows | OK | TECH | HOW | STALE |
|---|---|---|---|---|---|
| Landing | 21 | 18 | 2 | 2 | 0 |
| How it's built | 6 | 4 | 0 | 2 | 0 |
| Waitlist replies | 7 | 4 | 2 | 0 | 1 |
| Sign-in | 13 | 5 | 7 | 1 | 0 |
| Welcome and export guide | 38 | 20 | 13 | 9 | 0 |
| Upload and waiting | 48 | 17 | 31 | 3 | 1 |
| Life map | 21 | 8 | 13 | 3 | 0 |
| Trip review | 27 | 8 | 19 | 2 | 0 |
| Adventures shelf | 8 | 3 | 5 | 0 | 0 |
| Adventure page | 61 | 23 | 38 | 8 | 0 |
| Shared view | 5 | 3 | 2 | 1 | 0 |
| Imports and delete | 18 | 9 | 10 | 0 | 0 |
| Global | 1 | 0 | 1 | 0 | 0 |
| **all** | **274** | **122** | **143** | **31** | **2** |

Regenerate with `python3 docs/product/count-marks.py docs/product/copy-audit.md`
(a row is counted once under each mark it carries, so OK + TECH + HOW + STALE
exceeds the row count).

What the totals say: the landing is already in traveller words (phase 16
CP2b did that). The problem is everything after the first click. Inside the
app, the technical words are not spread evenly: nine words account for most
of the TECH rows — *candidate, confirm/dismiss, import(s), observed/routed/
unknown/air, plate, provenance, detection, observation/position/fix, seat/
instance*. Replacing those nine, consistently, fixes most rows at once. The
HOW rows cluster in four places: the welcome lede and its "what happens
after you upload" fold, the upload phase lines, the empty states on the life
map, and the sharing dialog.

## Tests that pin visible text

Changing copy will break these, and they must change in the same commit as
the screen they cover (step 3):

| file | pins |
|---|---|
| `web/e2e/share.spec.ts` | 12 |
| `web/e2e/plate-image.spec.ts` | 12 |
| `web/e2e/loop-pages.spec.ts` | 9 |
| `web/e2e/adventures.spec.ts` | 6 |
| `web/e2e/adventure.spec.ts` | 4 |
| `web/e2e/narrative-wording.spec.ts` | 1 |
| `web/e2e/legend-wording.spec.ts` | 1 (the legend's exact words, every page) |
| `web/e2e/auth-pages.spec.ts` | 1 |
| `web/lib/landing-legend.test.ts` | the landing's legend must equal `legend.ts` |
| `web/lib/plate-image.test.ts` | the words drawn on the exported image |

Go-side messages (`internal/timeline/parse.go`, `internal/api/*.go`) are
asserted in their package tests; the brief allows changing message text, not
names, and those tests change with them.
