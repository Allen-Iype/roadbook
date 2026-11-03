# Copy — current text and proposed text, screen by screen

Written 2026-11-02, step 2 of the "what the user gets" copy pass. The rule
and the word list are in `words.md`; the audit with every line number is in
`copy-audit.md`. No code has changed. Each screen is one approval; step 3
applies them one screen per commit, in this order.

Conventions. **Now** is the text on screen today; **Then** is the proposal.
Strings not listed on a screen are unchanged. Data (names, dates, numbers)
is shown with the demo values. Where a line differs between a hosted
account and a self-hosted one (decision 7), both are given. Privacy
statements keep every fact they state today; where one is reworded, the
facts are listed under it so the check is easy.

The one layout change this pass needs is the (i) button that replaces the
visible legend on every map (decision 2). It reuses the existing `Term`
popover (button plus native popover, `<details>` fallback); nothing else
moves. Where a screen would need more than that, it is reported under
"Needs more than words" at the end, not done.

---

## 1. Everywhere

**Header and the map's panel** (`site-header.tsx`, `(app)/page.tsx`)
- Now: ROADBOOK · Adventures · Candidates — 12 undecided · Imports · Sign out
- Then: ROADBOOK · Trips · Trips to review · 12 · Uploads · Sign out

**Site title and description** (`app/layout.tsx`)
- Now: Roadbook — The adventures you actually took.
- Then: Roadbook — Every trip you've taken, on one map.

**Loading screen** (`app/loading.tsx`)
- Now: Loading candidates…
- Then: Loading…

**Error screen** (`app/error.tsx`)
- Now: Something failed while rendering. / {the error} / Try again
- Then: Something went wrong on this page. / {the error} / Try again

**Roadbook unreachable** (the map, Trips, Trips to review, Uploads, the trip page)
- Now: The Roadbook API is not reachable. Start it with `roadbook serve` and reload.
- Then, hosted: Roadbook isn't answering right now. Try again in a minute.
- Then, self-hosted: Roadbook isn't answering right now. Start it with `roadbook serve` and reload.

**The legend** (every map: the welcome plate, the map, Trips, the trip page)
- Now: a visible list — Observed — the phone recorded it · Routed — drawn along roads · Unknown — nothing recorded, straight line · Air — a flight (plus Fix and Stop on the trip page).
- Then: one small (i) button labelled **What the lines mean**, opening the same list; on the trip page the list also carries **Single point — one recorded spot** and **Stop**. Labels and descriptions unchanged.

**The bar under a distance** (covers, the trip page, the hover card, the list)
- Now: drawn, with a caption on the trip page ("How much of this was recorded, how much drawn in") and a sentence on the Trips page.
- Then: drawn, no caption, no sentence, no line in the popover either (the maintainer's review of the first build, 2026-11-02). The closed fold at the foot of the trip page is the only place its figures are read.

**Fallback errors when Roadbook gives no message** (`actions.ts`)
- Now: decide failed (HTTP 500) · bulk decide failed (HTTP 500) · upload failed (HTTP 500) · delete failed (HTTP 500) · creating the link failed (HTTP 500) · revoking failed (HTTP 500) · deletion failed (HTTP 500)
- Then: couldn't save your choice (error 500) · couldn't save your choices (error 500) · couldn't upload (error 500) · couldn't delete (error 500) · couldn't create the link (error 500) · couldn't stop sharing (error 500) · couldn't delete everything (error 500)

**A trip that is no longer in the list** (`internal/api/server.go`, `photos.go`)
- Now: no such candidate in the latest run — re-detection may have replaced it; reload the list
- Then: this trip isn't in your list any more — reload the page

**Default name for a kept trip with no name** (`actions.ts`, `adventure-view.tsx`, `plate-image.ts`)
- Now: Journey of 2026-04-24
- Then: Trip of 24 Apr 2026

---

## 2. Landing — `landing/index.html`

Already in traveller words (phase 16 CP2b). Five lines change.

**Step 2**
- Now: Upload that file. Roadbook reads it and finds the trips that stood out — the ones far from home, with somewhere you stayed.
- Then: Upload that file. Your trips appear, found for you — the ones far from home, with somewhere you stayed.

**Step 3**
- Now: Keep the ones that mattered. Say yes to the real adventures and name them; the rest stay off the map.
- Then: Keep the ones that mattered. Say yes to the real trips and name them; the rest stay off the map.

**Privacy**
- Now: … Your export is kept so Roadbook can re-read it later. …
- Then: … Your Timeline file is kept, so you never have to upload it twice. …
- Facts kept: history stays in your account; nobody else sees it unless you share a link, which you can take back; the file is kept; photos are never kept, only where and when; delete everything any time; no ads, no trackers.

**The legend under the hero map**
- Now: the four-line legend list.
- Then: a closed `<details>` titled **What the lines mean**, holding the same four lines. (The landing is static and script-free; `<details>` is the (i) here. `landing-legend.test.ts` keeps pinning its words to `legend.ts`.)

**Caption under the hero map**
- Now: All your trips on one map. These are sample trips. Yours will be drawn the same way.
- Then: unchanged.

---

## 3. Waitlist replies — `landing/joined.html`, `landing/oops.html`

**joined.html**
- Now: When seats open again, one email arrives telling you to sign in — that is the only mail this list ever sends. / Change your mind? Reply to that email, or ask any time, and your line is deleted.
- Then: When a spot opens, you'll get one email saying so. That is the only email this list ever sends. / Changed your mind? Reply to that email, or ask any time, and we delete your address.

**oops.html**
- Now: Nothing was recorded. Head back and check for a typo — the form needs a working address, because the entry email is the only way in.
- Then: Nothing was saved. Go back and check for a typo — the list needs a working address to reach you.

---

## 4. Sign-in — `web/app/(public)/signin/page.tsx`

**Tagline and the sentence under the heading**
- Now: A road atlas of your own past / Sign in / This Roadbook keeps each person's adventures their own. Sign in and you'll only ever see yours.
- Then: nothing visible — the wordmark and the button are the page (the maintainer's review of the first build, 2026-11-02); "Sign in" stays as a heading only screen readers get.

**The button**
- Now: Sign in to continue
- Then: Sign in with Google

**When all spots are taken**
- Now: Roadbook is full for now. / Every seat on this instance is taken, so your sign-in created nothing. Join the waitlist and one email arrives when a seat opens. Nothing else is ever sent. / Join the waitlist
- Then: Roadbook is full for now. / All the spots are taken, so you don't have an account yet. Join the list and you'll get one email when a spot opens. Nothing else is ever sent. / Join the list

**When Google sign-in fails**
- Now: Sign-in didn't complete: the provider reported: access_denied
- Then: Google didn't finish signing you in. Try again.
- Go side (`internal/api/auth.go:140`): the message becomes "Google didn't finish signing you in" and no longer carries Google's code. The code stays in the server log.

---

## 5. Welcome and export guide — `web/app/(public)/welcome/page.tsx`

**Page description**
- Now: What Roadbook is, how to export your Google Timeline, and where to upload it.
- Then: Every trip you've taken, found from the location history your phone already keeps.

**Heading and lede**
- Now: The adventures you actually took. / Roadbook reads the record your phone already keeps of where it went, draws the trips that stood out, and leaves it to you to say which were adventures.
- Then: Every trip you've taken, on one map. / Your phone has quietly kept a record of everywhere you've been. Upload it and your trips are found for you — no diary, no logging. You choose which ones to keep.

**Under the sample map**
- Now: THREE ADVENTURES ACROSS ICELAND · DEMONSTRATION TIMELINE / (legend) / Fictional demonstration data. Yours is drawn the same way, from your own record.
- Then: SAMPLE · THREE TRIPS ACROSS ICELAND / (i) What the lines mean / Sample trips. Yours will look like this.
- Alt text, now: A life map drawn from the demonstration timeline: three adventures across Iceland — an observed drive along the south coast, a routed loop through the Westfjords, and a return flight to Akureyri.
- Alt text, then: A sample map: three trips across Iceland — a drive along the south coast, a loop through the Westfjords, and a flight to Akureyri.

**Three steps**
- Now: Export your Timeline from Google Maps on your phone — a couple of minutes. / Upload the file here. / Confirm the trips that were adventures; the rest stay off the map.
- Then: Export your Timeline from Google Maps on your phone — a couple of minutes. / Upload the file here. / Keep the trips that mattered; the rest stay off the map.

**Walkthrough stamps**
- Now: On Android · Verified 11 Aug 2026 / Walked 11 Aug 2026 on an Android device from a WhatsApp-opened link: the steps matched the phone's menus and the upload completed in the in-app browser.
- Then: On Android · Checked on a phone 11 Aug 2026 / Checked 11 Aug 2026 on an Android phone, opened from a WhatsApp link: the menus matched these steps and the upload worked from the browser inside WhatsApp.
- Now: On iPhone · Unverified / Drafted 11 Aug 2026 from documentation, not yet verified on a device — Google changes these menus without notice.
- Then: On iPhone · Not yet checked on a phone / Written 11 Aug 2026 from Google's own instructions, not yet checked on an iPhone — Google changes these menus without notice.
- The steps themselves are unchanged: they are Google's menu names.

**Never had Timeline on** — REMOVED with the photos section (maintainer, 2026-11-02: the start page is for someone with a Timeline file; photos stay on the Uploads page)
- Now: Never had Timeline on? Then there is nothing to export — Google holds no location record for you. Start from your photos instead.
- Then: Never had Timeline on? Then Google has nothing for you to export. Start from your photos instead.

**Or start from photos**
- Now: A photo straight from the camera carries where and when it was taken, and that is enough to find your trips — no export step, the picker opens your photo library. Original photos work best.
- Then: Photos straight from the camera know where and when they were taken — enough to find your trips. No export needed; the picker opens your photo library. Original photos work best.

**Fold: Which photos work**
- Now: Camera originals, and downloads from your own Google Photos. iPhone photos are usually HEIC — upload them as they are; if the picker offers JPEG, either works.
- Then: Photos straight from the camera, and downloads from your own Google Photos. iPhone photos come in the iPhone's own format (HEIC) — upload them as they are.
- Now: A Google Photos Takeout includes small .json files beside the photos. Upload them together: they can restore a location the photo file lost.
- Then: If you downloaded your photos from Google Takeout, each one comes with a small info file beside it. Upload them together: the info file can give back a location the photo lost.
- Now: On their own, photos need a little context: home is worked out from where your photos recur, so include a spread from ordinary weeks too, not only the trips. With a Timeline export already in, photos simply add detail to the adventures it found.
- Then: Photos on their own need some ordinary days too, not only the trips — that is how Roadbook works out where home is. If you've already uploaded a Timeline file, photos simply add detail to the trips it found.
- Now: Photo intake verified 26 Aug 2026 with a real camera roll. Not yet walked on a phone — the iPhone notes come from documentation, not a device.
- Then: Photo upload checked 26 Aug 2026 with a real camera roll, on a computer. Not yet checked on a phone — the iPhone notes come from Apple's and Google's instructions.
- Unchanged: the chat and screenshots sentence.

**Fold: What the right file looks like**
- Now: Got a .zip archive? Extract it and upload the .json file inside.
- Then: Got a .zip file? Open it and upload the .json file inside.
- Now: Files from a Google Takeout download — monthly "Semantic Location History" files or Records.json — are the old server-side format, which Roadbook does not read. The phone export is the current one.
- Then: Files from an older Google download (Takeout: monthly "Semantic Location History" files, or Records.json) are a format Roadbook can't read. The file from the phone is the one that works.
- Now: Photos, PDFs and KML files are not location exports — the upload will say so and point back here.
- Then: A photo, a PDF or a map file isn't your Timeline — the upload will say so and point you back here.
- Unchanged: the first paragraph (the person must recognise the file), the "on a computer" paragraph.

**Fold: Where your data goes**
- Now: Your file streams to this Roadbook and nowhere else, and is kept here as-is so later versions can re-read it without asking you for the file again. Photos are not kept — only the position, the time and a small thumbnail of each. Everything you upload is listed on Imports, and can be deleted from there.
- Then: Your file goes straight to this Roadbook and nowhere else. It is kept as it is, so you never have to upload it again. Photos are not kept — only where and when each was taken, and a small preview. Everything you've uploaded is listed under Uploads, and can be deleted from there.
- Facts kept: nowhere else; kept as uploaded; photos not kept; position, time, preview kept; listed; deletable.

**Fold: What happens after you upload** → retitled **What happens next**
- Now: Upload. Nothing is recorded until the whole file has arrived — if the connection drops, just try again. / Import. Every visit, journey and recorded position is read in; each geotagged photo contributes the position it carries. / Detection. Roadbook looks for journeys far from home with a real destination — somewhere you stayed, not somewhere you passed through — and lists them as candidates. Home is worked out from the data itself. / You decide. Confirm the real adventures and name them; dismiss the rest. The confirmed ones are drawn on your life map and shelved as plates in your atlas.
- Then: Upload. Nothing is saved until the whole file has arrived — if the connection drops, just try again. / Your trips are found. Roadbook reads the file and picks out the trips: time well away from home, with somewhere you stayed. Home is worked out from your own data. / You choose. Keep the real trips and name them; leave out the rest. The kept ones are drawn on your map.
- Now: Once the upload finishes you can close the tab — Imports keeps every attempt and its outcome.
- Then: Once the upload finishes you can close the tab — Uploads keeps every attempt and how it went.

---

## 6. Upload and waiting — `upload-import.tsx`, `photo-upload-import.tsx`, `rejection-anchors.ts`, the Go messages

**Timeline panel (on Uploads)**
- Now: Add a Timeline export
- Then: Add your Timeline file

**Progress lines (both doors and both panels)**
- Now: Uploaded. Importing your observations… / Imported. Looking for adventures… / Done. This file held nothing new — every observation was already imported. / See your candidates →
- Then: Uploaded. Reading your file… / Finding your trips… / Done. Nothing new in this file — everything in it was already here. / See your trips →
- Photos, now: Photos imported. Looking for adventures… / These photos held nothing new — every position was already imported.
- Photos, then: Photos read. Finding your trips… / Nothing new in these photos — their places were already here.

**A wrong file**
- Now: That file isn't a Timeline export (zip) / {message} / Nothing was stored — try the right file.
- Then: That isn't your Timeline file / {message} / Nothing was saved — try the right file.
- The bracketed kind goes; the message says what the file is.

**A file that could not be read**
- Now: The import failed / {message} / The attempt is recorded on the imports list.
- Then: Roadbook couldn't read this file / {message} / The attempt is listed under Uploads.

**Connection and other failures**
- Now: The upload didn't complete — nothing was recorded. Check the connection and try again. / The upload failed (HTTP 502). / Your data imported fine, but the detection step failed — the server log has details.
- Then: The upload didn't complete — nothing was saved. Check the connection and try again. / The upload failed (error 502). / Your file was read, but Roadbook couldn't look for trips in it. Try again later.
- Self-hosted adds to the last: " — the server log has details."

**Links back to the guide** (`rejection-anchors.ts`)
- Now: How to export the current format from your phone / Export a fresh copy from your phone and upload that / Photos import separately — add them in the photos section
- Then: How to export the right file from your phone / Export a fresh copy from your phone and upload that / Photos go in separately — under "Or start from photos"

**The rejection messages** (`internal/timeline/parse.go`; the hint appended to each)
- Hint, now: export Timeline data on the phone: Settings → Location → Location Services → Timeline → Export Timeline data
- Hint, then: export it again from Google Maps on your phone — the start page shows how
- empty — now: the file is empty — then: unchanged
- gzip — now: this is a gzip archive, not a Timeline JSON file — decompress it and import the JSON inside — then: this is a compressed file (.gz), not your Timeline file — open it and upload the file inside
- zip — now: this is a ZIP archive (a Takeout download?) — extract it and import the Timeline JSON file inside — then: this is a ZIP file (an older Google download?) — open it and upload the Timeline file inside
- pdf — now: this is a PDF, not a location export — then: this is a PDF, not your Timeline file
- image — now: this is an image file, not a location export — then: this is a photo, not your Timeline file — photos go in under "Or start from photos"
- html — now: this is a web page — Takeout's archive_browser.html is a table of contents, not data; import the Timeline JSON file instead — then: this is a web page (the index of an older Google download), not your Timeline file
- kml — now: this is KML from the old Timeline interface, which Roadbook does not read — then: this is a map file from the old Timeline, which Roadbook can't read
- xml — now: this is XML, not a Timeline JSON export — then: this isn't your Timeline file
- binary — now: unreadable binary data — possibly an encrypted Timeline backup, which cannot be imported; instead — then: this file can't be read — it may be an encrypted Timeline backup, which Roadbook can't open
- not-json — now: not JSON (the file starts with "…") — then: this isn't your Timeline file
- semantic-history, records-json, my-activity — then, all three: this is from an older Google download, which Roadbook can't read — the file from the phone is the one that works (the current three texts are checked at step 3)
- json-unrecognised — now: JSON, but with no semanticSegments section — top-level keys found: …; this is not a phone Timeline export — then: this isn't the Timeline file from the phone
- truncated — now: the file ends mid-record — the export or the copy of it was cut short; re-export or re-copy the file — then: the file ends early — the export or the copy was cut short; export it again
- not valid JSON — now: the file is not valid JSON (…); if it was edited or converted, start from a fresh export — then: this file can't be read — if it was edited or converted, start from a fresh export

**Upload errors** (`internal/api/imports.go`)
- Now: an import is already running — watch it on the imports page and retry when it finishes / the request carried no file part — choose a Timeline export and try again / internal error during import — the server log has details / the stored upload could not be reopened: … / storing observations failed: …
- Then: another upload is still being read — watch it under Uploads and try again when it finishes / no file arrived — choose your Timeline file and try again / something went wrong on this Roadbook's side while reading the file / something went wrong on this Roadbook's side: … / something went wrong on this Roadbook's side: …
- Self-hosted appends " — the server log has details" to the three "something went wrong" lines.

**Photo verdicts**
- Now: 9 of 12 files carried a usable position (2 sidecars matched). / 3 files added nothing — why
- Then: 9 of 12 photos had a location (2 info files matched). / 3 files added nothing — why
- Reasons, now: no place in the photo — a copy from a messenger has it stripped; use the original / a location but no readable capture time / a Takeout .json file naming no photo in this batch — include the photo it describes / not a supported photo format
- Reasons, then: no location in the photo — a copy from a chat has it stripped; use the original / a location but no readable time / a Google info file with no matching photo in this upload — include the photo it belongs to / not a photo format Roadbook can read
- Go side (`photos.go`), now: the image could not be decoded — it may be corrupted or a format masquerading as JPEG / over the N-files-per-request limit — upload the rest in another request / larger than the N MB per-file limit
- Then: this image can't be opened — it may be damaged / more than N files at once — upload the rest separately / larger than N MB
- Now: The import failed (photos)
- Then: Roadbook couldn't read these photos

---

## 7. The map — `web/app/(app)/page.tsx`, `life-map.tsx`, `summoned-list.tsx`, `terms.ts`

**No trips found**
- Now: No adventures found — yet / Your data imported fine. Detection ran and found no journeys far from home with a real destination — it looks for time spent well away from every home base, staying somewhere, and pass-throughs don't count. / More data may change that: add another export — earlier years, another device — or see Imports for what was ingested.
- Then: No trips found — yet / Your file was read, and nothing in it counted as a trip. A trip is time well away from home, with somewhere you stayed; short outings and pass-throughs don't count. / More data may change that: add another Timeline file — earlier years, another phone — or see Uploads for what came in.

**File in, trips not yet looked for** (self-hosted only: reached after a command-line upload)
- Now: No adventures yet / Roadbook draws your life map from the journeys in a Google Timeline export. Data has been imported but detection has not run yet. / `roadbook detect -from-db` / Or add your data through the browser, which runs detection automatically.
- Then: No trips yet / Your file is in, but Roadbook hasn't looked for trips in it yet. / `roadbook detect -from-db` / Or add your file through the browser, which looks for trips on its own.

**Trips waiting to be reviewed**
- Now: 12 candidates await review / The life map draws confirmed adventures only — none are confirmed yet. Confirm the real journeys and they appear here as routes. / Review candidates — 12 undecided
- Then: 12 trips are waiting for you / Keep the real ones and they're drawn here. / Review your trips · 12

**Kept trips that could not be loaded**
- Now: 3 confirmed adventures exist, but no journey could be loaded from the API. Reload; if it persists, check that the API and database are up.
- Then, hosted: 3 kept trips couldn't be loaded. Reload; if it keeps happening, try again later.
- Then, self-hosted: 3 kept trips couldn't be loaded. Reload; if it persists, check that the API and database are up.

**Hover card**
- Now: South coast drive / 2026-04-24 → 2026-04-26 / 700 km — 640 observed / Click to open
- Then: South coast drive / 24 → 26 Apr 2026 / 700 km / Click to open

**The list**
- Now: List (3) / Adventures (3) / Close — dialog label "Confirmed adventures"
- Then: List (3) / Trips (3) / Close — dialog label "Kept trips"

**The map's own panel**: as "Everywhere"; the legend becomes the (i).

**The notes behind tapped words** (`terms.ts`)
- Plate, now: Each confirmed adventure is one plate in your atlas. Plates are numbered in the order you travelled, the oldest first.
- Plate, then (on the numeral only): Your kept trips are numbered in the order you travelled, the oldest first.
- The provenance bar: its note moves into the (i) as the fifth line (see "Everywhere"); the tapped caption is gone.
- Candidate: no longer used on any page; removed.
- Score, now: How much this looks like a trip, from 0 to 100: how far from home, how long you stayed, how much was recorded. It orders the queue; it decides nothing.
- Score, then: How much this looks like a trip, from 0 to 100: how far from home, how long you stayed, how much your phone recorded. It only sets the order; you decide.
- Observed, Routed, Unknown, Air, Original photos: unchanged.

---

## 8. Trips to review — `web/app/(app)/candidates/page.tsx`, `decide-cell.tsx`, `triage-*.tsx`

**Heading and intro**
- Now: Candidates / Each candidate is a trip Roadbook found in your record. Confirm the adventures; dismiss the rest.
- Then: Trips to review / Roadbook found these trips in your history. Keep the real ones; leave out the rest.

**Nothing found**
- Now: Roadbook looked and found no trips. / Your data imported fine. A trip is time spent well away from home with a real stay somewhere; short outings and pass-throughs don't count. / More data may change that: add another export — earlier years, another device.
- Then: Roadbook looked and found no trips. / Your file was read. A trip is time well away from home with a real stay somewhere; short outings and pass-throughs don't count. / More data may change that: add another Timeline file — earlier years, another phone.

**Nothing uploaded**
- Now: No candidates. Add a Timeline export — or import one with the CLI and run `roadbook detect -from-db`.
- Then, hosted: Nothing to review yet. Add your Timeline file.
- Then, self-hosted: Nothing to review yet. Add your Timeline file — or import one with the CLI and run `roadbook detect -from-db`.

**Earlier choices that no longer match**
- Now: ⚑ 2 decisions no longer match a candidate under the current parameters. They are preserved and will re-match if parameters change back.
- Then: ⚑ 2 of your earlier choices no longer match a trip. They are kept.

**Order and keyboard hint**
- Now: Order: by date · by score / After each decision, focus moves to the next undecided card.
- Then: Order: by date · by score / After each choice, the next trip is selected.

**How these were found** (closed fold)
- Now: How these 12 were found / Time spent more than 100 km from home, with somewhere you stayed, counts as a trip. 3 recorded positions too fast to be real were left out first. Run 12.
- Then: How these 12 were found / Time spent more than 100 km from home, with somewhere you stayed, counts as a trip. 3 recorded points too fast to be real were left out first.

**A card**
- Now: route shape unavailable / score 73 / 3 days · 420 km away · 1,210 km track · 4 stops · 2× visited
- Then: no map yet / score 73 / 3 days · 420 km from home · 1,210 km travelled · 4 stops · visited 2× before
- Buttons, now: confirm · dismiss · save · cancel · change · keep as is — state: dismissed
- Buttons, then: keep · leave out · save · cancel · change · keep as is — state: left out
- Score box, now: score 73.2 of 100 — what it is made of / … / no record of it; its share went to the others
- Score box, then: score 73.2 of 100 — what it is made of / … / not recorded; counted with the others

**Selecting several**
- Now: (checkbox) Select this candidate for a bulk decision / Select all 12 undecided / All 12 undecided selected / (bar) Bulk decision · 4 selected · Dismiss selected · Confirm selected · Clear
- Then: (checkbox) Select this trip / Select all 12 to review / All 12 selected / (bar) Keep or leave out together · 4 selected · Leave out selected · Keep selected · Clear
- Notes, now: Dismissed 4 candidates. / Confirmed 4 candidates — 3 with suggested names, 1 named by date. Rename any from its card or adventure page. / Nothing was recorded — {error}
- Notes, then: Left out 4 trips. / Kept 4 trips — 3 with suggested names, 1 named by its date. Rename any from its card or its page. / Nothing changed — {error}

---

## 9. Trips — `web/app/(app)/adventures/page.tsx`

**Heading and intro**
- Now: Adventures / 3 adventures, the latest first. Each is a plate, numbered in the order you travelled. The bar under each distance shows how much of it the phone recorded.
- Then: Trips / 3 trips, the latest first, numbered in the order you travelled.

**Nothing kept yet**
- Now: No adventures yet / Your atlas fills as you confirm candidates — the trips Roadbook found in your record. / Review candidates — 12 undecided
- Then: No trips kept yet / Keep a trip under Trips to review and it appears here. / Review your trips · 12

**A cover**
- Now: PLATE 2 (printed) · Iceland · Westfjords loop · ⚑ (read aloud: distance flagged) · 22–24 May 2026 · 3 days · 197 km
- Then: PLATE 2 (printed) · Iceland · Westfjords loop · ⚑ (read aloud: distance differs from Google's) · 22–24 May 2026 · 3 days · 197 km
- Unnamed, now: Unnamed adventure — then: Unnamed trip

**Legend at the foot**: the (i).

---

## 10. The trip page — `adventure-view.tsx`, `day-narrative.tsx`, `photos-section.tsx`, `photo-strip.tsx`, `route-map.tsx`, `plate-export.tsx`, `share-links.tsx`, `format.ts`, `plate-image.ts`

**Back link**
- Now: ← Life map — Then: ← Map

**Eyebrow over the name**
- Now: CONFIRMED ADVENTURE · PLATE II / DISMISSED CANDIDATE / CANDIDATE / SHARED ADVENTURE
- Then: KEPT TRIP · PLATE II / LEFT OUT / TO REVIEW / SHARED TRIP

**Flags under the dates**
- Now: ⚑ Already under way when the record begins. / Still under way when the record ends.
- Then: ⚑ Already under way when your file begins. / Still under way when your file ends.
- In the day-by-day, now: Journey begins · already under way when the record starts / Journey ends · still under way when the record stops
- Then: Trip begins · already under way when your file starts / Trip ends · still under way when your file stops

**The trip block**
- Now: 700 km over 3 days / (bar) / How much of this was recorded, how much drawn in / Time away 2 d 9 h · On the move 9 h 10 min by Google's count · 43 km/h average where recorded · Stopped 4 stops · 31 h · Farthest 420 km from home
- Then: 700 km over 3 days / (bar, no caption) / the same four lines, unchanged

**Status line**
- Now: confirmed 12 Oct 2026 / dismissed 12 Oct 2026 / Undecided — confirm or dismiss in the candidates gallery / Read-only · shared by its owner
- Then: kept 12 Oct 2026 / left out 12 Oct 2026 / Not reviewed yet — keep it or leave it out under Trips to review / Shared with you

**Map margin**
- Now: PLATE II · FULL ROUTE / DAY 2 HIGHLIGHTED / Fix — one recorded position / Stop / (legend)
- Then: PLATE II · FULL ROUTE / DAY 2 HIGHLIGHTED / (i) What the lines mean — holding the four lines, the bar line, Single point — one recorded spot, Stop

**Day by day**: unchanged except "Journey" → "Trip" (above). Everything else there is already plain: Travelling all day · 2 stops · On the move · Stayed 2 h 10 min · Moved on after 9 h · Nothing recorded this day · overnight into Day 2 · 43 km/h average where recorded · Position not recorded — this stay is not on the map.

**Photos (owner)**
- Now: None yet. Original photos work best. / Add photos / Uploading…
- Then: unchanged
- A tile, now: IMG_4021.jpg / 2026-04-24 10:12 · exif (title: capture time from exif) / positioned · exif (title: position from exif) / ⚑ 1.2 km from the observed track at this time / outside this journey's timeline / no capture time / no position — strip only
- A tile, then: IMG_4021.jpg / 24 Apr 2026 10:12 · from the photo / on the map · from the photo / ⚑ 1.2 km from the route at that time / taken outside this trip's dates / no time / no location — not on the map
- Sources in words: exif → from the photo · sidecar → from Google's info file · filename → from the filename.
- Place lines (`format.ts`), now: from the observed track at this time / from the routed road at this time / from the straight-line gap at this time / from the stop at this time / over an air leg — not checked against the arc
- Then: from the route at that time (first three) / from the stop at that time / during a flight — not checked
- Pending tile, now: sidecar — then: info file
- Results, now: added / already uploaded / rejected / metadata applied / unpaired sidecar
- Then: added / already uploaded / not added / info file matched / info file without its photo
- Reasons (`photos.go`), now: HEIC (the iPhone default format) cannot be attached — attaching needs a thumbnail and HEIC pixels cannot be decoded here; convert to JPEG, or download it from Google Photos, which serves JPEG / names no image in this upload — include the photo it describes in the same request / photos attach to confirmed adventures — confirm this candidate first / thumbnail file missing from the photos directory — the photo must be re-uploaded
- Then: iPhone-format (HEIC) photos can't be added here — convert to JPEG, or download the photo from Google Photos, which gives JPEG / this info file's photo isn't in this upload — add them together / photos go on kept trips — keep this trip first / the preview is missing — upload the photo again
- Batch error, now: The batch could not be sent — most likely too large in one request. Try fewer photos at a time.
- Then: Couldn't send these — probably too many at once. Try fewer photos at a time.

**Photos from uploads (owner)**
- Now: 2 photos from your photo imports were taken inside this journey's window — placed by capture time against the drawn route. They stay with their import; deleting is not done from here.
- Then: 2 photos from your photo uploads were taken during this trip. To remove them, delete that upload under Uploads.
- Facts kept: they belong to the upload; they are not deleted here.

**Photo strips (shared view)**
- Now: 3 photos the owner added to this adventure, placed by capture time against the drawn route. / 2 photos from the owner's photo imports were taken inside this journey's window — placed by capture time against the drawn route.
- Then: 3 photos the owner added to this trip. / 2 photos from the owner's uploads, taken during this trip.
- Strip tile, now: No preview — this format's pixels aren't decodable here / outside the drawn route's elements
- Then: No preview for this format / away from the route
- Map popup: the same words as the tiles.

**Download**
- Now: Download as image / Download as overlay / Rendering… / Image: PNG, 2400 × 1600 — the map with its route, and the margin as printed here. Overlay: PNG, 1080 × 1920, transparent — route and figures only, for your own photo or story. Photos are not drawn in either. / Rendering the plate — the basemap loads once more at the export size. / Rendering the overlay. / Downloaded south-coast-drive.png. / The export failed (…). Nothing was downloaded.
- Then: Download the map / Download for a story / Making it… / The map: a 2400 × 1600 picture of the route with its figures, as shown here. For a story: 1080 × 1920 with a see-through background — the route and the figures only, to lay over your own photo. Photos aren't included in either. / Making the map — it loads once more at full size. / Making the story picture. / Downloaded south-coast-drive.png. / Couldn't make the picture (…). Nothing was downloaded.

**Sharing**
- Now: Sharing / Create a share link / Not shared. A share link lets anyone who has it open this plate — map, days, countries, and photos — without signing in. / Link created 12 Oct 2026 — open to anyone who has it / Revoke
- Then: Sharing / Create a share link / Not shared. A link lets anyone who has it see this trip — map, days, countries and photos — without signing in. / Link created 12 Oct 2026 — open to anyone who has it / Stop sharing
- Dialog, now: Share this adventure / Anyone with the link can open this plate, without signing in. They will see: / the route map, each leg marked observed, routed, unknown, or air — exactly as it is drawn for you; / the day-by-day narrative, distances, and countries; / all 3 photos on this adventure, and any you add later — photo imports included. / Nothing else of yours is reachable through it. You can revoke the link from this page at any time. / Create link · Creating… · Cancel
- Dialog, then: Share this trip / Anyone with the link can see this trip, without signing in. They will see: / the route on the map, exactly as it is drawn for you; / the day-by-day account, distances and countries; / all 3 photos on this trip, and any you add later — photos from your uploads included. / Nothing else of yours is reachable through it. You can stop sharing from this page at any time. / Create link · Creating… · Cancel
- Zero and one photo variants keep their shape: "any photos you add to this trip later, including ones from your uploads." / "the one photo on this trip, and any you add later — photos from your uploads included."
- Facts kept: anyone with the link; no sign-in; map, days, countries, photos including later ones and upload photos; nothing else reachable; can stop any time.
- Created, now: Link created / Copy it now. Roadbook keeps only a fingerprint of this link, so it cannot be shown again — but it can be revoked from this page at any time. / Copy link · Copied · The link is on your clipboard. · Done
- Created, then: Link created / Copy it now — Roadbook can't show this link again. You can stop sharing from this page at any time. / unchanged
- Facts kept: not shown again; can be stopped.

**How this route was drawn** (closed fold at the foot, with the flag above it)
- Flag, now: ⚑ The drawn distance and Google's differ by 7% — the account below says why.
- Flag, then: ⚑ The distance differs from Google's by 7% — "How this route was drawn" says why.
- Fold body, now: observed 640.0 km · routed 40.0 km · unknown 12.0 km · air 0.0 km / 312 recorded points (280 from the timeline + 32 raw) · Google's own distance 702.0 km · countries and regions from the route's recorded points / By mode, as Google labelled it (modes are guesses): passenger vehicle 640.0 km, 9 h 10 min · … / No mode record — this journey's evidence carries no activity data. / Distance check: drawn on the ground 652.0 km · Google's ground figure 702.0 km (−7.1%). Gaps with no road under-count, detours over-count, and a record cut short compares against a door-to-door figure — the legs explaining it are on the map.
- Fold body, then: recorded 640 km · by road 40 km · gap 12 km · flight 0 km / 312 recorded points (32 of them precise) · Google's own distance 702 km · countries and regions from the recorded points / How you travelled, as Google guessed it: passenger vehicle 640 km, 9 h 10 min · … / Google didn't record how you travelled. / Distance check: by road 652 km · Google's figure 702 km (−7%). Gaps with no road under-count, detours over-count, and a trip cut short compares against a door-to-door figure.

**The downloaded picture** (`plate-image.ts`)
- Now: The record starts mid-journey — it began before the imported window. / Still in progress at the window's edge — the end shown is the cut, not the return. / drawn — 91% of it measured / Fix / (legend)
- Then: Already under way when the file begins. / Still under way when the file ends. / 91% recorded by the phone / Single point / (legend unchanged — the picture has no (i), so it keeps its printed legend)

---

## 11. Shared view — `shared/[token]/page.tsx`, `not-found.tsx`

- Title, now: Roadbook — shared adventure — then: Roadbook — a shared trip
- Masthead, now: A shared adventure — then: A shared trip
- Footer, now: This plate was shared from someone's Roadbook — an atlas drawn from their own location history, honest about what was recorded and what was inferred. How Roadbook works, and how to make your own.
- Footer, then: Someone shared one of their trips with you, drawn from their phone's own location history. Make your own map →
- Not found, now: Whoever shared it may have revoked it, or the adventure it led to is no longer available. If someone sent you this link, ask them for a fresh one.
- Not found, then: Whoever shared it may have stopped sharing, or the trip is no longer there. If someone sent you this link, ask them for a fresh one.
- Unreachable: This Roadbook is not reachable right now. Try the link again in a little while. — unchanged.

---

## 12. Uploads and delete — `imports/page.tsx`, `delete-my-data.tsx`

**Heading and panels**
- Now: Imports / Add a Timeline export / Add photos
- Then: Uploads / Add your Timeline file / Add photos

**Nothing yet**
- Now: Nothing added yet. (hosted) / Nothing added yet — upload above, or run `roadbook import -src Timeline.json`. (self-hosted)
- Then: unchanged

**The table**
- Now: Added so far — Added · Source · Format · Dates · Visits · Activities · Path points · Positions · Skipped · New · Status
- Then: Added so far — Added · Source · Format · Dates · Stops · Journeys · Points · Precise points · Unreadable · New · Status
- Status, now: failed (+ message) / running / completed / completed + detection failed — the server log has details
- Status, then: couldn't be read (+ message) / reading / done / done + Roadbook couldn't look for trips in it (self-hosted adds " — the server log has details")

**Delete everything**
- Now (hosted, with data): Removes every import, adventure, decision, photo, and share link of yours, and forgets your account. Nothing of anyone else's is touched.
- Then: Removes every upload, trip, choice, photo and share link of yours, and forgets your account. Nothing of anyone else's is touched.
- Now (self-hosted, with data): Removes every import, adventure, decision, photo, and share link. This Roadbook is empty again, as if freshly installed.
- Then: Removes every upload, trip, choice, photo and share link. This Roadbook is empty again, as if freshly installed.
- Now (hosted, empty): Nothing is stored yet. Deleting forgets your account and frees its seat; you can sign in again any time.
- Then: Nothing is stored yet. Deleting forgets your account and frees its spot; you can sign in again any time.
- Dialog list, now: every import and the observations it brought in; / every candidate and decision — names included; / every photo thumbnail and photo record; / every share link — the links stop opening immediately; / your account here — signing in again starts from nothing. / the retained copies of your uploaded exports.
- Dialog list, then: every upload and every place it recorded; / every trip and every choice — names included; / every photo preview and photo record; / every share link — the links stop working immediately; / your account here — signing in again starts from nothing. / the kept copies of the files you uploaded.
- Now: Your own export files, wherever you keep them, are untouched — they are the canonical copy of your history. There is no undo.
- Then: Your own files, wherever you keep them, are untouched — they are the original copy of your history. There is no undo.
- Unchanged: Delete everything · Delete everything? · This removes, for good: · **Type delete to confirm** (the typed word is still `delete`) · Delete everything · Deleting… · Keep my data.

---

## Needs more than words

Reported, not done:

- **Review is still a gallery of cards with a score.** Testers who could not find their way around will meet the same page shape; this pass renames it. If the shape is the problem, that is the next UI phase's brief, with a watched walk first.
- **The Uploads table** has eleven columns. The headers become words here; whether a traveller needs the table at all is a design question.
- **The sign-in page in auth-off mode** redirects straight to the map, so a self-hoster never sees the new sign-in copy. Correct, and noted so nobody looks for it.
- **"How this route was drawn"** keeps its figures in plain words. If the fold should go entirely (decision 2 says it stays, closed), say so.
