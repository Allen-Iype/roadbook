# Competitive study: eight Google Timeline tools → Roadbook

Written 2026-11-01. Investigation only; nothing here is implemented. Each
proposal in §4 waits for the maintainer's approval as its own change.

**Method.** Every project was cloned into a scratch folder outside this
repository and read at the code level, with file:line citations kept in the
working notes; READMEs were treated as claims to check, not facts. Where it
was practical without a heavy toolchain, the tool was run against Roadbook's
committed anonymised demo export (`testdata/demo/demo.json`) and the
project's own sample data: the kurupted viewer (headless Chromium, every
non-CDN host blocked), mahlernim's Python CLI and test-suite, and
visualize-timeline's script. The Java/Docker stacks (reitti, GeoPulse) and
the two Vite apps were read, not run. Roadbook's own parser was run against
the iOS sample file that ships with the kurupted repository. Comparisons are
against what Roadbook does today (`internal/timeline`, `internal/detect`,
`internal/journey`, `api/openapi.yaml`, `web/`), not against its plans.

**Relation to the Dawarich report.** `docs/feature-comparison-dawarich-
roadbook.md` (2026-07-31) is not repeated. Its adopted ideas (format
sniffing and failure taxonomy, streaming parse, decision durability, the
speed-sandwich filter, weighted scoring, air-leg classification, local
countries table, GPX export, poster, replay, elevation, demo dataset) and its
rejections (eleven-mode detection, speed-coloured routes, timeline browsing,
live tracking, fog of war) are taken as settled. Each proposal below says
whether that report already covers it, and §2 notes where a project here does
something differently from Dawarich.

**Data safety.** No real coordinates or personal figures appear in this
document. The only numbers are from the committed demo fixture, the
projects' own source, and Roadbook's source.

---

## 1. The projects

| # | project | licence (from the LICENSE file) | head commit | size | run here? |
|---|---|---|---|---|---|
| 1 | kurupted/google-maps-timeline-viewer | GPL-3.0 | 2026-09-23 (one squashed commit) | one 16,439-line `timeline.html` | yes, headless |
| 2 | jameslashmar/Google-Timeline-Visualiser | MIT (`cities.json` has no licence or provenance) | 2026-07-21, v1.17, no commits since | one 2,742-line `index.html` + 2.3 MB `cities.json` | no (read) |
| 3 | dedicatedcode/reitti | **AGPL-3.0** (the brief said MIT; the file says AGPL) | 2026-11-01, active | Java/Spring, ~76 k lines, 139 migrations | no (read) |
| 4 | tess1o/geopulse | **BUSL-1.1**, "Personal Use" only, hosted service expressly commercial, change to AGPL-3.0 on 2075-01-01 | 2026-10-26, active | Java/Quarkus + Vue, 1,423 Java files | no (read) |
| 5 | mahlernim/google-timeline-visualizer | MIT (Mediabunny dependency MPL-2.0; GeoNames CC BY 4.0) | 2026-10-22, active | Python CLI 1,611 lines + Kotlin Android app + Vite web app | yes, Python |
| 6 | ryangriggs/GoogleTimelineMapper | Apache-2.0 | 2024-06-05, frozen | 413-line `index.html` | no (read) |
| 7 | makinux/GoogleMapTimelineViewer2 | not declared — no LICENSE file at head, no `license` field, `"private": true`; only an AI Studio template header on one file | 2026-05-14 | Vite/React, ~1,700 lines | no (read) |
| 8 | a913cb82/visualize-timeline | MIT | 2026-01-04 | Python 394 lines + Leaflet page | yes, Python |

Licence consequences: ideas from every project may be re-implemented from
description; no code is copied from any of them. GeoPulse (BUSL) and
GoogleMapTimelineViewer2 (no licence declared) are ideas-only in the strictest sense.
reitti shares Roadbook's licence, which changes nothing about the no-copy
rule.

### 1.1 kurupted/google-maps-timeline-viewer

A browser-only replacement for the retired Google Maps desktop Timeline: the
user picks a folder, the file is parsed in memory, and days render on a
Google Maps JavaScript map. It is unusable without a Google Cloud project,
billing and a Maps/Places API key, and it resolves every visit's `placeId`
through the Places API to get names, cities and countries, which is what its
Trips and World views depend on. Its trip detection is the closest thing to
Roadbook's core found in this study: residences ("Places Lived") are derived
from overnight stays clustered within 180 m, split into eras at absences over
65 days, qualifying at 14 nights over 25 days; a trip is a run of items more
than 60 km (the one user knob) from the residence active at that time, with
the drive out and back attached if within 4 hours, a bounce home of under
18 hours tolerated, and a split at 168 hours of separation. Trip cards show
nights per city, flights with local departure and arrival times, and
`City → City` breadcrumbs, but no distance, time or pace. Run against the
demo export, it dropped all 253 visits because an Android visit must carry a
`placeId` (the anonymised demo has none) and flagged two bogus one-day trips
with a hard-coded `"Taiwan"` home-country fallback; with synthetic place ids
added it found exactly the three journeys Roadbook's detector pins and
correctly excluded the demo's sub-60 km day trip. Parsing covers the current
Android object, the iOS array (`geo:` strings and minute offsets), legacy
`timelineObjects` and Takeout year folders, all loaded whole. `INFERRED_HOME`
is never recognised; only `TYPE_HOME`/`HOME` labels are. No sharing, no
image; KML export only.

### 1.2 jameslashmar/Google-Timeline-Visualiser

A single-page tool "built for motorcycle touring". There is no detection and
no notion of home: a "ride" is the date range the user picks. Geometry comes
only from `rawSignals` positions plus `timelinePath` points not within 120 s
of a raw fix; visits are listed as stops (times only) and activities are
never used. From the fixes it computes distance (haversine, gaps over 15 min
skipped), moving time, average moving speed, an unfiltered maximum of
Google's per-fix speed, elevation gain with 4 m hysteresis, bends (same-sign
bearing accumulation reaching 40°, 35° on road-matched geometry, with
reversals over 100° ignored), twistiness in degrees per km with a 0–10 score
and named bands from "arrow straight" to "hairpin heaven", the twistiest
10 km, rally pace notes, a per-day tour table, and per-day weather. Output is
a Canvas-drawn ride card PNG in three social aspect ratios handed to the
browser share sheet, a "flythrough" video recorded from the canvas, GPX, CSV,
and a date-sliced copy of the original export. Two privacy claims do not hold
in code: Open-Meteo is called automatically on every selection with the ride
midpoint, and road matching against the public FOSSGIS Valhalla instance runs
automatically for cards and videos; the whole parsed history is also kept in
IndexedDB. The iOS array is accepted but its offset-only path points are
dropped.

### 1.3 dedicatedcode/reitti

A self-hosted, multi-user location-timeline server (Spring Boot, PostGIS,
Redis, Quartz) built around live tracking, with file import as a second path.
It imports the Android object, the iOS array, Takeout `Records.json`
(streamed), GPX, GeoJSON and FIT; the format is chosen by the user from six
upload cards and validated by extension only. The Android importer turns each
visit into two synthetic 10 m-accuracy points and keeps path points, but
discards `activity` segments, `rawSignals` and `semanticType`. Detection is a
running-centroid stay-point sweep (radius 100 m, 300 s minimum, 300 s maximum
gap, at least 5 points) whose stay location is the densest accuracy-weighted
real point, not the centroid; visits attach to PostGIS "significant places";
trips are the intervals between consecutive visits, capped at 24 hours, with
60-second speed-band mode segments. Every threshold lives in a per-user table
with a validity date, five sensitivity presets, and a one-day preview before
a full recalculation. Home is inferred from nights: each 22:00–06:00 night is
assigned to the place with the most overlap if at least three hours, bridged
across a phone-idle gap of up to 16 hours when the same place lies on both
sides, then a centred 45-day window with a minimum share and hysteresis gives
a home per day and a "this looks like your home" suggestion. The closest
thing to adventure detection is a statistics line, "N nights away never
became a memory". Memories are hand-created date ranges auto-filled with
canned prose and shared by magic link. Default install talks to an
author-hosted geocoder and tile server and checks GitHub nightly. Share
pages carry no `noindex`.

### 1.4 tess1o/geopulse

A large self-hosted, multi-user "Google Timeline alternative" (Quarkus,
PostGIS, Vue) centred on live tracking, with import, sharing, friends,
photos, weather, badges, an AI assistant and a Trip Workspace. Google import
is a Jackson streaming parser that routes on the first JSON token: an array
is the iOS export, an object whose first key is `locations` is Records.json,
any other object is the Android export; the format is chosen by the user and
`timelineObjects` is not handled. It keeps none of Google's semantics: a
visit becomes a synthetic zero-speed point every five minutes, an activity
two endpoints, and `INFERRED_HOME` is parsed then dropped. Its own state
machine then rebuilds stays (50 m, 7 min), trips and data gaps (silence over
3 hours), the last through an ordered rule chain that can continue a stay
across a gap or infer a long-distance trip, both off by default. The one idea
with no counterpart anywhere else in this study: every stay, trip and gap
stores a JSON "explanation" of the reason codes and numbers that produced it,
rendered as one plain sentence each ("Starts where you stopped moving: 4
points slower than 9 km/h for 95 s") with an optional map of the surrounding
fixes. It has no home concept and no adventure concept; MOTORCYCLE is a label
the user picks for the single motorised speed band. Its "Add Missing Data"
dialog synthesises GPS points from user-drawn stays and trips and inserts
them as ordinary points, indistinguishable from recorded ones on the main
map. Share links carry expiry and a password; exports are GPX (per trip, per
day, per stay), GeoJSON, CSV and its own zip. The licence forbids using it to
provide a hosted service.

### 1.5 mahlernim/google-timeline-visualizer

Turns a Timeline export into an animated travel video: a pink trail drawing
itself over a CARTO basemap with a title card and a running kilometre count,
no labels, stops, speeds or statistics. Three implementations kept in parity
by shared fixtures: a Python CLI (matplotlib + ffmpeg), an Android app on the
Play Store (MapLibre Native, MediaCodec H.264, a persistent "Travel Journal"
that accumulates `rawSignals` across imports), and a web app (MapLibre GL,
WebCodecs H.264 via Mediabunny, streaming 64 KiB chunks with a byte-range
index). Its parser accepts the array and object roots, `geo:`, `°`, `latLng`
and E7 coordinates, and `durationMinutesOffsetFromStartTime`, all fixture-
backed; Android and web name legacy Takeout formats in a rejection message.
The Android app has a conservative trip detector (day centroids, home = the
most frequent 0.5° cell, away at 120 km, span at least two days, "strong"
when bracketed by home days on both sides) with confirm/dismiss and offline
GeoNames naming, and an outlier rule that removes a run of up to three points
only when the track leaves and rejoins at over 1,300 km/h with both hops over
500 km, so a real flight is never touched. Its onboarding is the most
complete seen: an in-app button opening Android's Location settings, text
explaining that Timeline is off by default, a freshness policy that nudges
re-export at 14, 24 and 29 days because the detailed window expires, and the
iPhone file name `location-history.json`. The Python CLI ran against the
demo export and produced a video; without a CARTO key the basemap is a
watermark.

### 1.6 ryangriggs/GoogleTimelineMapper (skim)

A 2024 single-file page for Takeout-era monthly `timelineObjects` files:
lists `placeVisit` rows (several files at once), lets the user tick visits
and add a note, and exports the ticked ones as CSV, GPX waypoints or KML
placemarks. Rejects the current phone export. Loads Google satellite tiles
over plain HTTP and asks for the device location on page load. Nothing is
computed; the one transferable idea is stops-as-waypoints with a note.

### 1.7 makinux/GoogleMapTimelineViewer2 (skim)

A Vite/React/Leaflet page generated from an AI Studio template. Parses
`Records.json`, legacy `timelineObjects` and the Android object (path points
and visits; activities ignored); the iOS array fails both the root test and
the coordinate parser although the README gives iOS export steps. Draws a
speed-coloured polyline in five bins from consecutive fixes (the
instantaneous speed Roadbook rejects), animates a marker that interpolates
across silences as if observed, plots geotagged JPEGs as thumbnail markers
with a small ring to spread stacked shots, and exports KML/KMZ with the
thumbnails embedded. No licence is declared anywhere that covers the parsing
code.

### 1.8 a913cb82/visualize-timeline (skim)

A Python script plus a Leaflet page. Reads only `timelinePath` points from
the Android object, removes "glitches" with a 200 km/h ceiling and a
discard-until-recovery rule, thins points, and writes GeoJSON. Run against
the demo export it deleted the 2026-06-14 return flight and three road hops
after it, while the outbound flight on 2026-06-12 survived only because no
recovery point fell inside the 24-hour look-ahead; Roadbook's both-neighbour
900 km/h rule keeps both. Its README has the clearest single sentence on why
the file must come from the phone. Default basemap is Esri's terrain layer.

---

## 2. Comparison tables

Roadbook's column states what the code does on 2026-11-01. "—" means the
feature does not exist in that project.

### 2.1 Input and parsing

| | Roadbook | kurupted | Lashmar | reitti | GeoPulse | mahlernim | skim 6/7/8 |
|---|---|---|---|---|---|---|---|
| Android on-device object (`semanticSegments`) | yes, streamed; visits, activities, paths, `rawSignals` | yes, whole file; a visit without `placeId` is dropped | partial: `rawSignals` + paths only | yes, whole file; activities and `rawSignals` discarded | yes, streamed; everything flattened to points | yes; web streams | 7 paths+visits; 8 paths only; 6 no |
| iOS array (`geo:` strings, `durationMinutesOffsetFromStartTime`, string `distanceMeters`) | **no** — rejected as "not valid JSON (expected {, got [)" | yes | array accepted, offset-only points dropped | yes | yes | yes, fixture-backed | no |
| legacy Takeout `timelineObjects` | rejected with a message | yes, plus year folders | no | no | no (silent zero points) | rejected with a message (Android/web) | 6 and 7 yes |
| Takeout `Records.json` (E7) | rejected with a message | no | no | yes, streamed | yes, if `locations` is the first key | E7 by magnitude | 7 yes |
| GPX / other | photos (EXIF, HEIC, sidecars) | — | GPX, CSV, Apple Health routes | GPX, GeoJSON, FIT | GPX, GeoJSON, CSV, OwnTracks | — | 7 JPEG EXIF |
| format detection | sniff + failure taxonomy | shape test, one generic error | extension/first bytes | user picks a card, extension check | user picks; first JSON token | shape + legacy message | shape |
| re-upload dedup | per-import corpus; no cross-import dedup | — | — | `ON CONFLICT (user, device, second)` | `ON CONFLICT (user, timestamp, point)` | Android Journal: same-file no-op, mismatch guard | — |
| offset-less timestamps | assumed RFC 3339 | moment default | `Date.parse` | ISO offset required | ISO or epoch | flagged; source order kept, streams never interleaved | — |

### 2.2 Detection and home

| | Roadbook | kurupted | Lashmar | reitti | GeoPulse | mahlernim (Android) |
|---|---|---|---|---|---|---|
| unit of analysis | observation | timeline item | user date range | GPS point | GPS point | calendar day centroid |
| stays / visits | Google visits as given; stops = pauses between activities ≥ 5 min; stay synthesis for photo fixes (200 m, 30 min) | Google visits; overnight test by local clock | visits > 3 min listed | running-centroid sweep 100 m / 300 s / ≥ 5 pts; location = densest fix | state machine 50 m / 7 min; arrival cluster rules | — |
| trips / adventures | away-span > 25 km from every active base containing a dwelt visit > 100 km, ≥ 60 min, ≥ 5 obs, ≥ 1 h; ranked 0–100, never auto-confirmed | run of items > 60 km from active residence; transit ± 4 h attached; bounce < 18 h tolerated; split at 168 h; no ranking | — | interval between visits, ≤ 24 h; "nights away ≥ 3" hint | interval between stays; gaps by rule chain; no adventure | away ≥ 120 km, span ≥ 2 d, episode gap ≤ 2 d, destination radius ≤ 250 km; STRONG if bracketed by home |
| home | set of eras from `INFERRED_HOME` clusters (~5 km grid, merge 10 km, ≥ 8 visits, ± 45 d); no user override | set of eras from overnight stays (≤ 180 m, ≥ 14 nights / 25 d, break at 65 d); user "Not a Home"; `INFERRED_HOME` ignored | — | per-day home from nights (22:00–06:00, ≥ 3 h, bridge ≤ 16 h; 45 d window, ≥ 10 nights, ≥ 30 %, hysteresis 5); suggestion with accept/dismiss | none | single most-frequent 0.5° cell |
| anomaly filter | > 900 km/h from both neighbours (detect); teleport clusters flagged (journey) | spurious-home sandwich (< 25 km home between > 150 km items) | none; raw max speed unfiltered | both-neighbour, floor `max(1000 km/h, 3 × median)`; accuracy > 100 m | accuracy ≤ 60 m; 1,200 km/h → UNKNOWN | out-and-back run ≤ 3 pts, both hops ≥ 500 km at > 1,300 km/h, rejoin ≤ 200 km in ≤ 12 h |
| parameters | named, recorded per run, CLI flags | ~30 literals, one user knob | literals | per-user table with validity date, presets, preview | ~60 per-user settings | literals |
| decisions survive re-detection | yes, anchor matching by span + destination | overrides keyed by auto id keep auto values | — | suppressed visits re-matched by centroid; mode overrides by timestamp | gap→stay overrides re-matched by time/distance | confirmed ids persist |

### 2.3 Privacy, output, onboarding, rider features

| | Roadbook | kurupted | Lashmar | reitti | GeoPulse | mahlernim |
|---|---|---|---|---|---|---|
| runs | Go server (self-host or hosted pilot) | browser | browser | server, multi-user | server, multi-user | local: Android, Python, browser |
| raw upload retained | yes, the Timeline file (content-hash) as the import's source; photos never | no (folder handle) | whole history in IndexedDB | no | no (temp ≤ 24 h) | no (Android Journal DB instead) |
| home hidden | distance-from-home absent from shares and exports; life map shows everything to the owner | pin shown | start town labelled on the card | no | no | no |
| hosts contacted by default | OpenFreeMap tiles; Nominatim opt-in; OSRM batch only | Google Maps + Places (every `placeId`), three CDNs | CARTO tiles, Google Fonts, **Open-Meteo automatic**, **Valhalla automatic** on cards | author-hosted geocoder and tiles, GitHub nightly | OSM tiles, OpenFreeMap, Nominatim, Photon, GeoNames, Open-Meteo, GitHub | CARTO after a consent dialog; web: Cloudflare analytics (disclosed) |
| share links | token, revocable, `noindex`, read-only plate | — | — | magic link, view/edit, expiry; no `noindex` | UUID, expiry, password; exposes named stays + full path | — |
| images | plate PNG 2400×1600, overlay 1080×1920; no share sheet | — | card PNG in 3 ratios → `navigator.share` | — | — | Android overview PNG |
| video | — | — | canvas + MediaRecorder (MP4 where supported) | — | in-app replay only | MP4 (ffmpeg / MediaCodec / WebCodecs) |
| geodata export | — (backlog) | KML | GPX, CSV, sliced JSON, pace notes | GPX raw points | GPX per trip/day/stay, GeoJSON, CSV | — |
| onboarding | `/welcome` per-platform walkthroughs with verification stamps; demo dataset; failure messages | README both platforms + 6-step Google Cloud key setup | in-page both platforms + GPX routes | one line per card | none | in-app, Location-settings button, "Timeline is off by default", freshness nudges, `location-history.json` |
| rider-relevant | per-mode km/h (source-asserted), observed pace, stops, day narrative, countries + states | nights per city, flights, MOTORCYCLING colour | bends, twistiness bands, best 10 km, max/avg speed, elevation gain, weather, pace notes, per-day table | speed-band modes, MOTORCYCLE manual label, terrain 3D, nights away | time-weighted trip stats, MOTORCYCLE user label, badges, opt-in map matching | none |

### 2.4 Where a project differs from Dawarich on a settled point

- **Anomaly filter.** Dawarich's speed sandwich uses a fixed floor and a
  3×-median; reitti is the same rule. mahlernim adds a rejoin-distance and
  hop-length condition and removes runs of up to three points; kurupted adds
  a distance-to-home sandwich. Roadbook's journey already flags teleport
  *clusters*, which none of the four do.
- **Stay-point sweep.** Dawarich's running-mean sweep (100 m, 1.5× drift cap,
  60 min gap) is matched by reitti's (100 m, 300 s gap) with one addition:
  the stay's location is the densest accuracy-weighted fix, not the mean.
- **Confidence score.** Dawarich scores visits; mahlernim's two-level
  STRONG/POSSIBLE is the only other scoring seen, and its "bracketed by home
  on both sides" component has no Dawarich counterpart.
- **Home.** Dawarich has no home inference. Two projects derive it from
  where the person sleeps (kurupted, reitti); Roadbook derives it from
  Google's `INFERRED_HOME` label. Neither approach is a superset of the
  other: the label exists only in Timeline exports; nights exist in every
  source.
- **GPX export.** Dawarich exports tracks; GeoPulse exports per trip, per day
  and per stay with stays as waypoints; GoogleTimelineMapper exports waypoints
  with a user note.
- **Replay.** Dawarich's day replay was parked as premature; mahlernim
  documents a complete camera model and an in-browser MP4 path, and
  GoogleMapTimelineViewer2 shows the failure mode to avoid (a marker gliding
  across a silence as if observed).

---

## 3. Where Roadbook stands

### 3.1 Already ahead

Each statement was checked in both codebases.

- **Honesty of geometry.** Roadbook is the only project whose drawn route
  distinguishes observed, routed, unknown and air legs all the way to the
  renderer and the exported image. Every other project draws one
  undifferentiated line, and two (Lashmar's cards, GeoPulse's continuity
  trips) substitute inferred geometry for measured without marking it.
- **Adventure detection with multiple homes over time.** kurupted is the
  only other project with eras; mahlernim has one fixed home; the rest have
  none. Roadbook alone combines eras, a dwelt-destination rule, observation
  and duration floors, and a recorded, ranked score.
- **Uses what the export contains.** Roadbook reads visits with their
  semantic type, activities with distance and mode, path points and
  `rawSignals`. reitti and GeoPulse discard activities and semantics;
  Lashmar and visualize-timeline discard visits and activities; kurupted
  drops visits without a `placeId` and never reads `rawSignals`.
- **Named, recorded parameters and a pure, pinned core.** Every other
  detection or statistic in this study is driven by literals in function
  bodies (kurupted, Lashmar, mahlernim) or by user settings without a run
  record (reitti, GeoPulse).
- **Input rejection.** Roadbook's sniffer names seven wrong inputs with a
  next step; the nearest competitor (mahlernim) names one family.
- **No key, no billing, one default host.** kurupted needs a Google Cloud
  project; Lashmar calls two third parties automatically; reitti and
  GeoPulse default to several external services. Roadbook's serve path calls
  tiles only, with Nominatim opt-in.
- **Offline countries and states** from embedded Natural Earth; everyone
  else uses a remote geocoder or has none (kurupted: US states only).
- **Share links with `noindex` and the home figure withheld**, plus two
  rendered image formats. No other project renders an image that carries its
  own legend.
- **Photos as a first-class source** without a photo server (reitti needs
  Immich; GoogleMapTimelineViewer2 plots photos but detects nothing).
- **A demo dataset the UI can load.** None of the eight ships one.

### 3.2 Behind

Concrete and verified; each maps to a proposal in §4.

- **The iOS Google Maps export is rejected**, with a message that blames the
  user ("if it was edited or converted"). Four of the eight parse it, and the
  shape appears nowhere in Roadbook's docs. Reproduce from the repo root with
  `go run ./cmd/roadbook detect -src <an iOS export>`; the kurupted
  repository ships a synthetic one at `sample_data/Local_format/iOS/`.
- **No road-shape statistic** (bends, degrees per km), which is the figure a
  motorcycle tourer actually brags about; Lashmar has a complete, small
  algorithm for it.
- **Nothing says where the rider slept.** The day narrative counts stops and
  dwell hours; kurupted and reitti count nights per place, and reitti
  bridges the common phone-idle overnight gap.
- **Home derivation has a single evidence source.** When `INFERRED_HOME` is
  absent or sparse (photo-only imports, and possibly some iOS exports) there
  is a synthetic fallback for photos only; kurupted and reitti infer home
  from nights in any source.
- **No per-element explanation.** GeoPulse stores why each stay, trip and
  gap begins and ends and renders it as one sentence; Roadbook's phase 16
  research found exactly this reading problem.
- **No share-sheet hand-off.** The plate and overlay PNGs download; Lashmar
  passes the PNG to `navigator.share` so it lands in WhatsApp in one tap.
  `web/` contains no call to it.
- **No freshness reminder.** The imports page says nothing about the ~30-day
  `rawSignals` window expiring at the source; mahlernim nudges at 14, 24
  and 29 days.
- **No geodata export** (backlog, Dawarich 1.11); five projects have at least
  GPX or KML.
- **No way to edit a candidate's dates or add a note** without dismissing
  it; kurupted keeps the automatic dates and title beside the user's edits.
- **No video and no elevation** (both backlog; mahlernim and Lashmar show two
  working in-browser render paths).

---

## 4. Ranked proposals

Effort: S = under a day, M = a few days with a brief, L = its own checkpoint.
"Phase fit" refers to the current position: phase 16 (reading load) at its
CP4 close with a zero-Go-diff target, then phase 17 (places), brief pending.

### P1. Read the iOS Google Maps export

- **What.** Accept the JSON the Google Maps iOS app writes: a top-level array
  of the same segment kinds, `placeLocation`/`start`/`end`/`point` as
  `"geo:lat,lon"` strings, `distanceMeters` as a string, `placeID` with a
  capital D, and `timelinePath` points carrying
  `durationMinutesOffsetFromStartTime` (string or integer) instead of a
  timestamp, resolved against the segment's `startTime` and rejected past its
  `endTime`. Record it as a second format slug beside `phone-timeline`.
  First, and separately, fix today's rejection message: a `[` root whose head
  contains `"geo:"` should be named as the iOS export, not called invalid
  JSON.
- **Inspired by.** kurupted, reitti, GeoPulse, mahlernim (whose fixtures
  `android-ios-sample.json` and `offset-path-sample.json` are a ready
  specification of the shape; write Roadbook's own synthetic fixtures).
- **Problem for an Indian rider.** An iPhone rider who used Google Maps with
  Timeline on has exactly one file, and today it is refused with a message
  that says they broke it. The pilot found iPhone friends with no Timeline;
  this does not establish that no iPhone rider has one.
- **Effort.** S for the message; S–M for the parser (one package, two
  fixtures, a `probe` run on each). Real iOS exports carry `semanticType`
  (three of the parsers read it), so home derivation needs no change; this
  should be stated as an assumption and confirmed on the first real file.
- **Touches.** `internal/timeline` (sniff, `parseLoc`, `rawSegment` with a
  string-or-number `distanceMeters`, offset handling), `testdata/`,
  `imports.detected_format` values, the welcome page's iPhone walkthrough
  (file name `location-history.json` per mahlernim; verify on a device).
  Zero change to `internal/detect` and `internal/journey` (invariant 4).
- **Phase fit.** Not phase 16 (Go diff). A standalone change between the
  phase 16 close and phase 17 CP1, or alongside 17.
- **Dawarich report.** §1.1 covers sniffing and the three Takeout-era
  variants; it does not mention the iOS array. New.
- **Licence.** Shape knowledge only; fixtures written here.

### P2. Nights: where the rider slept, and home from nights when the label is missing

- **What.** Two pure passes sharing one definition of a night. (a) Per
  adventure, assign each local night (22:00–06:00, at least three hours of
  overlap) to the stop or visit that held it, bridging a night across a
  silence of up to 16 hours when the same place lies on both sides; report
  "3 nights away: 2 near A, 1 near B" in the summary and the day narrative,
  and let the score's destination-dwell component see nights. (b) When
  `INFERRED_HOME` clustering yields no base, derive bases from nights (a
  centred rolling window, minimum nights and share, hysteresis) with the same
  era shape as today, after the existing photo-synthesis fallback.
- **Inspired by.** reitti (`NightCalculator`, `HomeResolver`), kurupted
  (residence eras from overnight stays; the night test by local clock),
  mahlernim ("bracketed by home" as confidence).
- **Problem for an Indian rider.** A tour is remembered as its night halts
  (Manali, Jispa, Leh), not as dwell hours; and riders whose data comes from
  photos or an iOS export with thin labels currently get "no home, no
  adventures".
- **Effort.** M. Every threshold named (`NIGHT_START`, `NIGHT_END`,
  `MIN_NIGHT_HRS`, `MAX_BRIDGED_GAP_HRS`, window, share, hysteresis); the
  fixture, archive and demo pins must stay byte-identical where
  `INFERRED_HOME` exists, because (b) runs only when today's derivation
  yields nothing.
- **Touches.** `internal/detect` (bases fallback, score input), `internal/
  journey` (summary field beside `Summarize`), `api/openapi.yaml` (summary
  and day fields), CLI `journey` print, web day narrative and summary block.
- **Phase fit.** Phase 17: "night N near <place>" is the sentence the
  places phase exists to fill, and its brief is pending.
- **Dawarich report.** Not covered (Dawarich has no home or night concept).
- **Licence.** reitti is AGPL like Roadbook; idea only regardless.

### P3. Bends and twistiness per leg, computed on road geometry and labelled as such

- **What.** For each leg: count bends (signed bearing change accumulated to a
  named angle, reversals over a named angle ignored, hops under a named
  length merged) and turn degrees per kilometre, with a 0–10 band and a short
  label. Compute it on routed geometry (a property of the road OSRM chose,
  stated as "from the map") and on observed legs only where median point
  spacing is below a named threshold (dense `rawSignals` legs), otherwise
  "not measurable". Never on unknown or air legs. Show per day and per
  adventure with the same source caveat the per-mode figures carry.
- **Inspired by.** Lashmar (`computeRide`, `analyseDense`, the band table).
- **Problem for an Indian rider.** "212 bends, mountain grade" is the one
  figure that distinguishes a Ghats ride from a highway slog; Roadbook's
  plate currently cannot say what kind of road the day was.
- **Effort.** M, brief first: the measurability rule and the wording are the
  design; the arithmetic is small. Golden fixtures pin assembly, so this
  lives beside `Assemble` like `ModeBreakdown`.
- **Touches.** `internal/journey` (new pure function), `api/openapi.yaml`,
  CLI print, web plate and day rows, DESIGN §6 if the plate gains a tile.
- **Phase fit.** After phase 17; own small brief. Depends on routed legs
  existing for the adventure (null-router instances get "not measurable").
- **Dawarich report.** Not covered; Dawarich rejected speed colouring, which
  this is not (curvature is not speed and does not use the colour channel).
- **Licence.** MIT; re-derive from the description. Do not touch
  `cities.json`.

### P4. Explanations as data: why each leg, stop and gap begins and ends

- **What.** Each leg, stop and gap carries a short list of reason codes with
  the numbers that fired them (`gap_after_silence {minutes: 47, threshold:
  20}`, `air_by_speed {kmh: 540, min: 250}`, `stop_between_activities
  {minutes: 42}`, `teleport_cluster_rejected {points: 3}`), produced by the
  pure assembler and rendered by the web as one fixed sentence per code, in
  the day narrative's "How this route was drawn" fold and on hover of a leg.
- **Inspired by.** GeoPulse (`explanation` JSON on every stay/trip/gap,
  `TimelineReasons`, one sentence per code, an "explanation map").
- **Problem for an Indian rider.** The phase 16 walk found the plate honest
  but opaque; a dotted stretch that says "no fixes for 47 minutes, longer
  than the 20-minute rule" teaches the legend by example.
- **Effort.** M. Reason emission is deterministic and testable against the
  golden fixture (the codes are a new field, not a change to geometry).
- **Touches.** `internal/journey`, `api/openapi.yaml`, web (`terms.ts`-style
  sentence table), CLI print.
- **Phase fit.** Subject matter is phase 16's, but it is a Go change; fold
  into the phase 17 brief as the narrative's second half, or a phase 16
  follow-on after CP4.
- **Dawarich report.** Not covered.
- **Licence.** BUSL: idea only; the reason vocabulary is Roadbook's own.

### P5. Trip-edge rules: attach the transit, tolerate a bounce, flag bracketing

- **What.** In span assembly: (a) prepend the activity that left home if it
  ended within N hours of the span's first away observation, and append the
  one that returned; (b) a return inside the home ring shorter than N hours,
  between away runs less than M hours apart, does not end the adventure; (c)
  expose `bracketed_start`/`bracketed_end` (a home visit observed before and
  after) as the positive form of today's truncation flags and as a score
  input.
- **Inspired by.** kurupted (4 h transit, 18 h / 36 h bounce), mahlernim
  (bracketing as confidence).
- **Problem for an Indian rider.** Rides start at the front door, not 25 km
  out; a mid-tour loop home to swap luggage currently splits one tour into
  two candidates.
- **Effort.** S–M in code, but it moves the regression pins (fixture 18,
  demo 3) and so needs its expectation files re-derived and reviewed; the
  reference Python detector would diverge, which must be recorded.
- **Touches.** `internal/detect`, testdata expectations, `score.go`.
- **Phase fit.** Later; its own detect brief. Partly answers the open
  "bug 4" item.
- **Dawarich report.** Not covered.
- **Licence.** GPL / MIT; idea only.

### P6. User corrections that keep the automatic result visible

- **What.** Let a confirmed adventure carry an edited start/end date, a
  note, and a per-adventure vehicle label ("I rode this": user-asserted,
  shown as such, never inferred), while the automatic dates and figures stay
  stored and one tap restores them. Later, per-stop corrections ("fuel, not a
  destination") re-matched by time and distance after re-detection.
- **Inspired by.** kurupted (overrides keyed by auto id, "reset to auto",
  hidden list), GeoPulse (user-chosen motorised label; time-window override
  re-matching), reitti (suppressed visits matched by centroid).
- **Problem for an Indian rider.** The detector's boundary is often a day
  off for a ride that started before dawn; and nothing on the plate says it
  was a motorcycle, because Google's label is a guess.
- **Effort.** M (schema columns on the decision, API, web form).
- **Touches.** store/migrations, `api/openapi.yaml`, web adventure page.
- **Phase fit.** Later UI phase.
- **Dawarich report.** §1.5 covers status and name lock; dates, notes,
  vehicle label and "auto stays visible" are new.
- **Licence.** Ideas only.

### P7. Freshness line on the imports page

- **What.** One sentence under "Added so far": when the newest import's
  observation range ended more than N days ago, say that the detailed part of
  a Timeline export covers only about the last month and expires at the
  source, so exporting again keeps it.
- **Inspired by.** mahlernim (`JournalFreshnessPolicy`: 14 / 24 / 29 days).
- **Problem for an Indian rider.** The high-accuracy fixes of a tour are
  lost for good if the next export comes two months later; the project's own
  notes know this, the UI never says it.
- **Effort.** S, web only, reads dates the import record already serves.
- **Touches.** `web/app/(app)/imports/page.tsx`.
- **Phase fit.** Phase 16 CP4 (zero Go, a copy change).
- **Dawarich report.** Not covered.
- **Licence.** MIT; a sentence.

### P8. Hand the plate and overlay to the share sheet

- **What.** On phones that support it, offer `navigator.share` with the
  rendered PNG as a file next to "Download", so the image goes to WhatsApp
  or Instagram in one tap; fall back to download elsewhere.
- **Inspired by.** Lashmar (`shareCard`).
- **Problem for an Indian rider.** The overlay exists to be posted; today it
  lands in Downloads first.
- **Effort.** S, web only; e2e can assert the button's presence by
  capability.
- **Touches.** `web/lib/plate-export.ts` and the adventure page.
- **Phase fit.** Phase 16 CP4 or the next web change.
- **Dawarich report.** Not covered.
- **Licence.** Browser API; nothing to copy.

### P9. Second outlier rule and a flight-survival regression test

- **What.** (a) A test built from the demo export pinning that both demo
  flights (2026-06-12 out, 2026-06-14 back) survive detection and assembly.
  (b) Optionally, an "out-and-back" rule beside the 900 km/h rule: remove a
  run of up to K points only when the track leaves and rejoins at over a
  named speed with both hops over a named length and the rejoin within a
  named distance and time; evaluate it offline against the archive's known
  impossible points before adopting.
- **Inspired by.** visualize-timeline (the measured false deletion),
  mahlernim (`filter_location_outliers`).
- **Problem for an Indian rider.** A Hyderabad–Leh flight bracketing a ride
  must never be filtered; a stuck WiFi fix on another continent should be.
- **Effort.** S for the test; S to evaluate (b), with no change unless the
  evaluation shows a gain.
- **Touches.** `internal/detect` tests; possibly one parameter block.
- **Phase fit.** The test fits any time; (b) later.
- **Dawarich report.** §1.4 covers the speed sandwich; the rejoin condition
  and the regression case are new.
- **Licence.** MIT; idea only.

### P10. A town-label policy for the plate (input to phase 17)

- **What.** When the gazetteer lands: label towns the route passed within a
  named distance in a "rode through" style, force a label at each day
  boundary and gap break, never at the first or last fix (it names home),
  cap the count, resolve overlaps by a stable score (forced, then visited,
  then population).
- **Inspired by.** Lashmar (card and video label policies).
- **Problem.** Phase 17 needs a label policy, not only names; this one was
  tuned on motorcycle tours.
- **Effort.** S–M on top of phase 17; `plate-image.ts` and the live map.
- **Dawarich report.** Not covered.
- **Licence.** Policy, not code; Natural Earth, never `cities.json`.

### P11. GPX export shape: one track per leg kind, stops as waypoints

- **What.** When the backlog GPX export is built: one `<trk>` per leg with
  the kind in `<name>`/`<desc>`, stops as `<wpt>` with the user's note, a
  per-day zip option, and (later) KMZ with photo thumbnails for Google Earth
  on a phone.
- **Inspired by.** GeoPulse (per trip/day/stay), GoogleTimelineMapper
  (waypoints with notes), GoogleMapTimelineViewer2 (KMZ thumbnails; ideas
  only, no licence declared).
- **Effort.** S once the endpoint exists.
- **Dawarich report.** §1.11 covers GPX export and the honesty rule; the
  waypoint and per-day shape is new.

### P12. Per-day weather, opt-in and cached

- **What.** At confirm time, if the operator enabled it, fetch the day's high
  and rain total for each day's midpoint from a keyless archive API, cache it
  with the journey, print it in the day narrative with attribution.
- **Inspired by.** Lashmar (which does it automatically; do not).
- **Problem.** Monsoon rides are defined by the rain.
- **Effort.** S–M; the Nominatim opt-in pattern already exists.
- **Phase fit.** Later; pairs with phase 17's batch-and-cache shape.
- **Dawarich report.** Not covered. **Licence.** Open-Meteo CC BY 4.0.

### P13. Replay: design notes for the backlog item

- **What.** When replay is built: constant-spatial-speed reveal, a follow
  camera with a dead zone and eased zoom, long hops shown zoomed out along
  the air arc, time budgeted by on-screen motion, a zoom-out ending, and
  MP4 encoded in the browser via WebCodecs; the reveal keeps the four inks
  and follows gap geometry rather than interpolating across silences.
- **Inspired by.** mahlernim (camera model, Mediabunny), Lashmar
  (MediaRecorder path), GoogleMapTimelineViewer2 (the interpolation mistake).
- **Effort.** L. **Dawarich report.** §3 lists replay as premature; this adds
  the how. **Licence.** MIT ideas; Mediabunny MPL-2.0 as a dependency is fine.

### P14. Smaller items, recorded for the backlog

- Import mismatch guard: before merging a second upload into an account,
  require overlap with stored observations, else refuse with "this looks
  like a different person's export" (mahlernim; S–M, Go import path).
- Debug bundle with a constant longitude shift for bug reports without real
  coordinates (GeoPulse; S, CLI). Fits the data-safety rule.
- Terrain basemap option for mountain rides (visualize-timeline, reitti);
  a hosting and licence decision, not code (M).
- Adventures grid filters by duration bucket and year (kurupted; S, web);
  low value at tens of adventures, note for /candidates at archive scale.
- Offset-less timestamp handling: never sort by a placeholder clock
  (mahlernim; S, parser) — add a `probe` check before assuming it occurs.
- Legacy `timelineObjects` carries place names and addresses
  (GoogleTimelineMapper); leave a source-asserted name slot in phase 17's
  schema for when the backlog legacy parser lands.

### P15. An atlas index: the figures across all confirmed adventures

- **What.** One page (or the head of `/adventures`) that states the shelf as
  a whole: per year and all-time, adventures · days away · nights · distance
  with its provenance split · countries and states · farthest destination
  (owner only) · per-mode bar · highlights · a destinations radar · an
  away-days year strip · one "against last year" sentence. The full
  inventory of what the other tools show, how, and what Roadbook can
  compute today is §7.
- **Inspired by.** reitti (statistics report), GeoPulse (Insights hero,
  Rewind digest), kurupted (yearly tiles).
- **Problem for an Indian rider.** "How much did I ride last year, and
  where" is the question every rider is asked at the next meetup; today the
  answer is spread over tens of plates.
- **Effort.** M: a pure Go aggregation beside the journey package, printed
  by the CLI (invariant 13), one endpoint, one page; mockup STOP under
  DESIGN.md; zero new data.
- **Touches.** `internal/journey` or a sibling package, `api/openapi.yaml`,
  web.
- **Phase fit.** Its own checkpoint after phase 17 (names make the
  highlights read); a first cut without names is possible earlier.
- **Dawarich report.** §2 rejected "monthly stats, insights, digests" as
  coverage analytics. §7 argues the distinction: aggregates over *confirmed
  adventures* are tens-of-records-shaped and keep provenance; per-point and
  per-day coverage figures are what the rejection was about and stay out.
- **Licence.** Ideas only throughout.

### Rejected after reading, with the reason

- Speed-coloured segments from consecutive WiFi fixes
  (GoogleMapTimelineViewer2, Lashmar's track): the instantaneous speed
  Roadbook has already rejected, and it spends the colour channel.
- Synthesising GPS points from user-drawn trips (GeoPulse "Add Missing
  Data"): breaks invariants 2 and 8; the backlog's "recalled" leg kind is the
  honest form of the same input.
- Automatic third-party calls (Lashmar's weather and road matching): every
  network call stays opt-in and disclosed.
- Google Places resolution of `placeId` (kurupted): paid API, per-visit
  disclosure to Google; phase 17's offline gazetteer is the chosen path.
- Canned narrative prose (reitti memories): the day narrative states
  measured facts only.
- Mode detection by speed bands (reitti, GeoPulse): rejected in the Dawarich
  report; both projects admit a motorcycle cannot be told from a car, which
  is why P6 proposes a user-asserted label instead.

---

## 5. Build first

1. **P1, the iOS export.** It is the only item that turns a refused user
   into a served one, it is parser-only so invariants 1–4 keep detection
   untouched, it is testable with two synthetic fixtures and a `probe` run,
   and four independent projects agree on the shape. The message fix is a
   one-line change that can land first.
2. **P2, nights.** It serves three needs with one definition: the narrative
   sentence phase 17 is about to need a place name for, a home fallback for
   sources without `INFERRED_HOME` (which P1 may add to), and the rider's
   own memory of a tour. It is pure, parameterised, and leaves every current
   pin byte-identical by construction.
3. **P3, bends on road geometry.** It is the one rider-specific figure in
   this study with a complete, small algorithm behind it, and it can be
   stated honestly because curvature of a routed leg is a property of the
   road rather than of the sparse observations. It needs a brief for the
   measurability rule and the wording, which is why it is third.

The two S-sized web items (P7 freshness line, P8 share sheet) are zero-Go
and could ride phase 16 CP4 without a brief if the maintainer wants them
there; they are not in the three because they change nothing about what
Roadbook can detect or say.

**Why the rest wait.** P4 (explanations) is strong but is a Go change in a
phase that targets none, and its sentences will read better once P2 and
phase 17 give them places and nights to name. P5 moves regression pins and
deserves its own detect brief. P6 needs a schema and UI design pass. P9's
test can land any time; its rule needs an offline evaluation first. P10–P13
depend on phase 17, the GPX endpoint, or the replay item respectively. P14
is backlog.

---

## 6. Not verified in this study

- GitHub activity (issues, stars, release cadence) for any project; all
  clones were shallow.
- Whether genuine current iOS exports always carry `semanticType` on visits
  (the parsers read it; the one sample seen omits it).
- Behaviour of kurupted's Places-dependent views, reitti and GeoPulse at
  runtime, and the two Vite apps (none were run).
- `cities.json` provenance (Lashmar) and `tz-lookup`'s licence (kurupted).
- Whether `valhalla1.openstreetmap.de` still serves `trace_route` at the
  sizes Lashmar sends.

---

## 7. Dashboards: what the others show, how they draw it, and what Roadbook can show

Added 2026-11-02 at the maintainer's request. Source: a second read of the
clones aimed only at the statistics, summary and digest views (reitti
`/statistics` and its grouped timeline; GeoPulse Dashboard, Journey
Insights, Rewind, place and classification dialogs; kurupted Summary, World
and Trips; Lashmar's tile block and profile; mahlernim's cards). File:line
citations are in the working notes; the figures below were each confirmed
in code.

### 7.1 What the dashboards show

Grouped by figure family rather than by project. "Needs" is the input the
figure is computed from in that project.

**A. Movement totals.**
Total distance; time moving; trip count; daily average; distance, time and
trip count *by mode* as three 100 % stacked bars with a one-line caption
("Driving covered the ground, walking made the trips", reitti); speed per
mode as distance over time; a single proportional mode bar under a hero
"{distance} of movement." heading (GeoPulse, twice). Needs: trips with
distance, duration and mode.

**B. Places.**
Top five places by time or visits; unique places with a "new this period"
count; countries and cities visited, and separately *slept in* (reitti);
per-place card with first and last visit, visit count, typical day and
arrival period (GeoPulse); most common route as a consecutive-stay pair; a
polar "radar" of places by bearing and log-distance from home, dot size by
time spent (reitti, inline SVG). Needs: named places (every project
depends on a geocoder or the Places API for this), home coordinate.

**C. Nights and home.**
Nights away; longest consecutive stretch away; nights per sleeping place;
nights in transit; "second base" (slept at on at least two separate
occasions, not a home); a per-day home derived from nights with a "this
looks like your home" suggestion; share of weekday time at home. Needs:
visits with dwell, a night definition (22:00 to 06:00 local in both
projects that have one), home.

**D. Day and time patterns.**
Day types (commute, wander, travel, home, untracked) as a donut and a
year calendar; weekday versus weekend table (median leave and return
time, outings, stops); 24 by 7 hourly moving-time heatmap per week;
busiest day, month and year; peak hours; routines ("every 3 days, usually
at 08:10, for 45 min"); most active day of week and hour. Needs: continuous
daily coverage. None of this survives sparse data.

**E. Per-trip statistics.**
Distance; duration; average speed from distance over duration; GPS point
count; origin and destination; time-weighted average speed and a 30-second
sustained maximum with a reliability check against the calculated average
(GeoPulse); elevation gain with hysteresis, bends left and right, twistiness
in degrees per km with named bands, best 10 km, and an elevation-plus-speed
profile (Lashmar); a "coverage" line, "~N km recorded · M points · D active
days", with a Good or Limited verdict (mahlernim). Needs: per-fix speed
and altitude for the Lashmar block, which exists only in `rawSignals`.

**F. Digest, highlights, comparison.**
A monthly or yearly "Rewind": hero distance, "{trips} trips across
{cities} cities", a pill "{p}% more than the previous period", four tiles
(trips, active days, moving time, stays), highlights (longest trip, most
visited place, busiest day), a trend bar chart, top five places, tiered
milestones, a heatmap, a PDF (GeoPulse). reitti's year period is its
recap, with "x, against y" sentences for any chosen comparison period and
a tracked-days caveat. kurupted has per-year and per-month tiles
(visits, cities, countries, distance, per-mode list) that drill down.
Needs: per-period sums only.

**G. Gamification.**
GeoPulse: 49 all-time badges with progress bars (countries 2 to 50,
cities 3 to 200, total distance 1,000 km to 1,000,000 km, single trip 42,
100 and 500 km, consecutive-day streaks, time of day, speed, flights,
trains, weather) and per-period tiered milestones (bronze to diamond for
distance, places, trips, active days, longest trip, with different
thresholds for a month and a year). reitti: the Eddington number only,
with a note explaining it. The browser tools: none.

**H. Coverage.**
H3 hexagons of every raw point with a "+3.8 % of {home area}" exploration
line (reitti); a "seen area" in km² from buffered tracks and a grid map
(GeoPulse); heatmaps of visits and path points (kurupted, GeoPulse).
Needs: continuous raw points.

**I. Explain-this-figure.**
GeoPulse's Trip Classification Details dialog: the effective versus
automatic classification, six speed statistics with a one-line hint each,
the priority order, and one card per candidate type listing every
threshold check as "name, operator, threshold, actual" with pass or fail.
reitti puts a `(?)` explainer in plain words on every statistics card and
four footnotes under the page (timezone, how home was decided, the night
definition, the untracked caveat), and omits any card it cannot compute
rather than showing zero.

### 7.2 How they are visualised

- **Number tiles dominate.** Every project leads with a row of three to
  six tiles (icon, value, label). GeoPulse's dashboard is nine cards of
  tiles and lists; reitti's KPI strip is five tiles under one generated
  sentence; Lashmar's block is ten tiles.
- **One chart type is in actual use: the bar.** GeoPulse has exactly one
  chart component, a grouped bar (chart.js via PrimeVue) used for distance
  by mode per day or week, and twelve monthly bars in the yearly Rewind.
  kurupted has two stacked bars (distance by mode, visits by category)
  whose grain switches day, month, year by range length. Nothing else in
  eight projects is a chart library call; reitti loads none at all.
- **CSS does the proportions.** Mode shares are `<span>` widths; donuts
  are `conic-gradient` strings built on the server; calendars are CSS
  grids with seven rows and a level class per cell.
- **Inline SVG for the two bespoke pictures:** reitti's polar radar of
  places and Lashmar's elevation-plus-speed profile with a hover
  crosshair.
- **Maps carry the density views:** deck.gl heatmap and H3 layers,
  leaflet.heat, Google pins.
- **Text sentences carry the comparisons and highlights:** "x, against
  y"; "Busiest month: May, 1,240 km"; "{place} is a second base, not a new
  home". These cost nothing and read well on a phone.
- **Period controls** are a month, year, all-time toggle with previous and
  next arrows, and a comparison select; the dashboard variant juxtaposes
  three fixed windows (selected, 7 days, 30 days) without computing a
  delta.

### 7.3 What Roadbook holds, and what each figure needs

Roadbook serves today, per confirmed adventure: window start and end,
legs by kind with distance (observed, routed, unknown, air), stops with
start, end, location and dwell, countries and states crossed, the summary
(span hours, civil days, stops, dwell hours, observed hours and pace), the
per-mode km and hours as Google asserted them, Google's own ground figure
and the divergence flag, point counts kept, photos placed. Per candidate:
destination distance from home, track km, days, score breakdown, repeat
count. Per run: home bases with eras. Not held: place names (phase 17),
per-point speed or altitude outside the `rawSignals` window, elevation,
weather, nights (proposal P2).

Three tiers follow.

**Tier 1, computable now from served data, no new Go logic beyond a sum.**

| figure | from | caveat |
|---|---|---|
| adventures per year and all-time | decisions | — |
| days away per year (civil days summed) | summary | truncated windows carry their flag |
| distance per year with the provenance split | journey km by kind | the aggregate must keep the four-way split; a single km total would be invariant 8 broken at the shelf |
| per-mode bar per year | mode breakdown | labelled source-asserted as it is on the plate |
| countries and states per year, new this year | journey countries and states | "visited", not "slept in", until P2 |
| stops and dwell hours per year | summary | — |
| longest adventure (km, days), most stops | journey, summary | — |
| farthest destination from home per year | candidate `dest_km` | owner-only, never on a share or export, same rule as the plate |
| repeat destinations (visited on more than one adventure) | candidate `repeat` and destination | names wait for phase 17; "a place 420 km north-east, three times" is honest without them |
| destinations radar by bearing and log-distance from home | bases and candidate destination | the one bespoke picture worth drawing; log scale puts a 150 km weekend and a 3,000 km flight on one disc |
| away-days year strip | windows | two states only, away or not; reitti's five day types need commute detection |
| air versus ground km per year | air km | — |
| photos placed per year, adventures with none | photos | reitti's "long stays without a photo" line, reshaped |
| "against last year" sentence | any of the above, two years | with the caveat when a year is partly imported |
| explain-this-score dialog | score breakdown | already served; the threshold-check presentation is the missing part |

**Tier 2, needs a small pure addition first.**

| figure | needs | proposal |
|---|---|---|
| nights away, longest stretch, nights per stop, second base | a night definition over stops | P2 |
| "slept in" versus "passed through" countries and states | nights plus per-stop region | P2 and phase 17 |
| named top destinations, breadcrumbs, "Day 2: Jispa" | gazetteer | phase 17 |
| vehicle on the plate and in the year's mode bar | user-asserted label | P6 |
| bends and twistiness per leg and per year | routed geometry, measurability rule | P3 |
| an explanation sentence per leg, stop and gap | reason codes | P4 |

**Tier 3, only honest inside the current `rawSignals` window or with a
network call; not for the atlas index.**

Max speed, moving time versus stopped time from fixes, elevation gain and
the elevation-plus-speed profile (per-fix altitude and speed exist only
for the last month before each export; a profile for historical
adventures would be drawn from 28 to 40 points and would mislead);
weather (P12, opt-in); coverage hexagons, seen-area km², and heatmaps (per
point, continuous data).

**Do not port, whatever the data.** Day types, commute and routine
detection, weekday versus weekend tables, hourly heatmaps, peak hours,
busiest hour of day, time at home, Eddington number (it measures regular
daily riding, which the product does not observe), streak and habit
badges, speed badges.

### 7.4 What an atlas index could look like

A sketch for the mockup STOP, not a design. One page, "Years", reachable
from the adventures shelf; the same figures also serve a per-year share
card later if wanted.

- **All-time line.** "{n} adventures · {days} days away · {km} km" with
  the provenance bar under the km, exactly as on a plate, and the per-mode
  bar under that with its source-asserted caption.
- **One card per year, newest first.** Adventures, days away, km with
  provenance bar, countries and states (new ones marked), longest and
  farthest (farthest owner-only), three highlights as sentences ("Longest:
  Spiti, 9 days · Farthest: 1,840 km from home · Most stops: Konkan, 14").
  Clicking a figure opens the shelf filtered to that year.
- **Two pictures.** The destinations radar (every confirmed adventure's
  destination around its era's home, log distance, dot size by days) and
  the away-days strip (52 columns by 7 rows, two inks).
- **One comparison sentence per year.** "{year}: 7 adventures, against 4
  in {year − 1}; 6,200 km, against 3,100." with the import caveat when
  the earlier year is partial.
- **Nights, named places, vehicle and bends** join their cards as P2,
  phase 17, P6 and P3 land, each in the wording its proposal fixes.
- **Presentation rules borrowed:** a plain-words `(?)` on every card;
  footnotes stating the day and night definitions and the provenance
  rule; a card that cannot be computed is omitted, never shown as zero;
  no chart library, bars and the radar are spans and SVG.
- **Not on it:** badges and milestones. They are cumulative thresholds
  and would work technically; they are left out because the product's
  stance is that the person decides what counts, and a progress bar
  towards "Road Warrior" argues with that. The maintainer can overrule
  this at the mockup STOP; the reitti-style explanatory note would be the
  form if so.

Every figure on the page is printed by one CLI command (`roadbook atlas`
or an extension of `journey`) so that invariant 13 holds for the index as
it does for the plate.
