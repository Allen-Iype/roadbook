# Feature comparison: dawarich → Roadbook

Source: Dawarich master @ `da551a0e`, analyzed 2026-07-31 (codebase map + feature/data-flow sweep). Compared against Roadbook's three principles (curation over coverage; post-hoc assembly with a deliberate confirmation step; visible honesty about observed vs inferred route parts) and its out-of-scope list (timeline browsing, live tracking, mobile app, sharing/social, multi-user).

A framing note before the list: Dawarich and Roadbook are near-opposites in intent. Dawarich is a coverage product — capture everything, continuously, and browse it. Roadbook is a curation product. Most of Dawarich's surface area therefore does **not** transfer, and the rejections below are structural, not accidents. What does transfer is concentrated in three places: import handling, data-quality filtering, and the machinery for keeping user decisions authoritative over recomputed data.

---

## 1. Candidate features worth adopting

### 1.1 Import hardening: source detection + failure taxonomy
- **In Dawarich:** detects 15 formats by reading only the first 2 KB (magic bytes + extension), then JSON shape rules, then substring fallback. Crucially, it has a taxonomy of *user-facing* failure messages for every wrong thing people actually upload: PDFs, HEIC photos, gzip, `.DS_Store`, Snapchat/Amazon exports, Google "My Activity" (wrong Takeout product), **encrypted Google Timeline backups**, truncated JSON, `archive_browser.html`.
- **In Roadbook:** Google Timeline export is your *only* input, and it comes in at least three shapes (Semantic History `timelineObjects`, `Records.json` with `latitudeE7/longitudeE7`, and the newer on-device phone-takeout format with `semanticSegments`/`rawSignals`). Dawarich's detection rules are a field-tested map of these variants and of every failure mode your users will hit on day one. Adopting them means the difference between "unsupported file: this looks like an encrypted Timeline backup — export it this way instead" and a silent parse error.
- **Phase:** 1 (it hardens the import you already have). **Complexity: medium** (the rules are portable; the work is porting them, not inventing them).

### 1.2 Streaming Takeout parsing
- **In Dawarich:** SAX/event-based JSON and XML parsing (never full-document parse), because real Google Takeouts are multi-GB. Whole-import transaction so a partial failure doesn't leave half-imported data.
- **In Roadbook:** Go's `json.Decoder` token streaming gives you this idiomatically. Wrap each import in one transaction.
- **Phase:** 1. **Complexity: small** (if done now; painful to retrofit after users hit OOM on a 4 GB `Records.json`).

### 1.3 Idempotent ingestion via natural-key upsert
- **In Dawarich:** unique index on `(coordinates, timestamp, user)` + `INSERT ... ON CONFLICT DO NOTHING`, with the returned row count distinguishing inserts from duplicates. Every import path is safely re-runnable; dedup is free.
- **In Roadbook:** unique `(timestamp, lat, lon)` per import corpus. This matters specifically because your model says *user decisions survive re-detection* — re-importing an updated Takeout is the sibling problem, and upsert dedup makes it a non-event. One trap Dawarich hit: dedup keys must compare coordinates as *numbers*, not formatted strings, or equal points slip through and violate the constraint mid-batch.
- **Phase:** 1. **Complexity: small.**

### 1.4 GPS anomaly filter (the "speed sandwich")
- **In Dawarich:** three passes flag noise points: exact (0,0) ("Null Island"); reported accuracy worse than a threshold (default 100 m); and a spike test — a point is anomalous only if **both** its incoming and outgoing speeds exceed `max(1000 km/h floor, 3 × median)`, where the median is computed only over sub-floor speeds so outliers can't inflate it. Flagged points are excluded from every derived computation but never deleted.
- **In Roadbook:** GPS spikes will both create false adventure candidates (a teleporting point looks like a journey) and make drawn routes ugly. This is a pure function over a point sequence with three named parameters — it drops directly into your "detection is a pure function, every threshold named" architecture as a pre-filter. Flag, don't delete, so the filter is re-runnable with different parameters.
- **Phase:** 1 for candidate quality; pays again in 2–3 for route quality. **Complexity: small–medium.**

### 1.5 Decision durability: status enum + lock timestamps + derived-never-authoritative
- **In Dawarich:** three cooperating patterns. (a) Visits carry `status ∈ {suggested, confirmed, declined}` — detection only ever creates `suggested`; re-detection never touches confirmed or declined rows. (b) Places have `name_locked_at` — once a user renames something, automated naming never overwrites it. (c) Aggregates (trips, stats) are derived queries, recomputed at will, never authoritative; user-entered fields live apart from computed ones.
- **In Roadbook:** this is your "user decisions survive re-detection" requirement, already production-proven. Concretely: adventures carry `{candidate, confirmed, dismissed}`; re-detection matches new candidates against existing decisions (Dawarich groups by rounded cluster center — you'd match by time-range overlap) and skips anything already decided; a `name_locked_at`-style field protects user-given adventure names; computed fields (distance, stops, route) are always safe to regenerate.
- **Phase:** 1 — you are building exactly this right now; worth checking your schema against these three patterns before phase 1 closes. **Complexity: small.**

### 1.6 Stay-point detection algorithm (for stops and destination dwell)
- **In Dawarich:** a single-pass online sweep, not DBSCAN: maintain an open "stay" with a running-mean anchor; a point joins iff it's within `radius` (default 100 m) of the anchor **and** within `radius × 1.5` of the stay's first point (the drift cap stops a slow walk from smearing the mean into one giant blob). Break on a time gap > 60 min; emit only if dwell ≥ 5 min and ≥ 3 points; merge stays separated by brief absences. All five parameters user-tunable with documented clamp ranges. A DBSCAN fallback exists but the sweep is the primary for a reason — it's O(n) and explainable.
- **In Roadbook:** "reports … stops" is a phase-2 deliverable and "a real destination" implies dwell detection in phase-1 candidate scoring. This algorithm is simple, fast, pure, and fully parameterized — exactly your detection style. The overlap trick matters too: Dawarich processes long ranges in month batches with 1-hour overlap so a stay straddling a boundary isn't cut in half.
- **Phase:** 2 (stops), with the dwell test usable in 1. **Complexity: medium.**

### 1.7 Candidate confidence scoring with stored breakdown
- **In Dawarich:** visits get a 0–100 score from named weighted components (dwell 0.30, spatial tightness 0.25, place match 0.20, point density 0.15, GPS accuracy 0.10); missing components have their weight *redistributed*, not zeroed; the per-component breakdown is stored on the row and shown in the UI; bands at ≥70/≥40.
- **In Roadbook:** score adventure candidates from named components (distance from home, destination dwell, point density, span) and show the breakdown in the confirmation UI. This serves principle 2 directly — the human decides better when shown *why* the machine proposed — and the stored breakdown makes threshold tuning empirical rather than guesswork. Note it's decision *support*, not auto-confirmation; auto-confirming above a threshold would violate principle 2.
- **Phase:** 1–2. **Complexity: small–medium.**

### 1.8 Flight-leg classification (the one slice of mode detection you need)
- **In Dawarich:** full transportation-mode detection (11 modes), but the load-bearing part for you is one threshold: points/gaps moving faster than ~500 km/h are treated as flyovers and never contribute to ground statistics; flights render as great-circle arcs, visually distinct from roads.
- **In Roadbook:** phase 3 validates routed distance against the source's own figures. An adventure containing a flight will break this: OSRM will either fail to route the gap or produce a nonsense 4,000 km road detour, and validation fails. Classifying a gap as "air" when implied speed exceeds a named threshold, rendering it as an arc (a third visual class alongside observed and road-inferred), and excluding it from road-distance validation protects phase 3's core promise. Skip the other 10 modes — leg-level mode annotation is timeline-browser territory.
- **Phase:** 3, but design the gap model in phase 2 so a gap can carry a kind (`unknown | road-inferred | air`). **Complexity: small** once phase 3's routing exists.

### 1.9 PostGIS `geography` column + SQL window-function segmentation
- **In Dawarich:** one `geography(Point, 4326)` column; distances come out in meters with no projection math; route segmentation and total distance are a single SQL window query (`LAG(...) OVER (PARTITION BY device ORDER BY timestamp)` → break flags → running sum → group), with distance pre-summed in SQL so application code never recomputes it.
- **In Roadbook:** you're on Postgres already; adding PostGIS and doing observed-segment splitting + distance in one query ports this verbatim into pgx. Dawarich's history also carries a hard-won lesson here: at one point its JS and SQL paths implemented the distance threshold inconsistently, and reconciling them without breaking existing users' numbers took deliberate care. The lesson for a pure-function pipeline: exactly one segmentation implementation, exercised by every caller.
- **Phase:** 2. **Complexity: small–medium** (small if PostGIS is adopted before phase 2 code exists).

### 1.10 Local countries table for offline attribution
- **In Dawarich:** a bundled table of country MultiPolygons (GiST-indexed); country attribution is a local point-in-polygon lookup — no network call, ever.
- **In Roadbook:** "journey to France" in a candidate description, or countries-crossed in an adventure report, with zero runtime service dependency — the same property you demanded of routing. Ship the polygons with the app (Natural Earth data is adequate at this precision).
- **Phase:** 2. **Complexity: small.**

### 1.11 Per-adventure GPX/GeoJSON export
- **In Dawarich:** background-generated GPX/GeoJSON exports; exports deliberately bypass all visibility filtering — users can always take their data out.
- **In Roadbook:** a confirmed adventure exported as GPX/GeoJSON honors the self-hosting ethos and makes routes portable to other tools. Tens of adventures means no background-job machinery needed — synchronous generation is fine. Interesting wrinkle: the observed/inferred distinction should survive export (e.g., separate track segments per confidence class), or the export silently violates principle 3.
- **Phase:** after 3 (exporting an unrouted adventure is a scatter of points). **Complexity: small.**

---

## 2. Rejected features, with the principle they fail

| Dawarich feature | Verdict |
|---|---|
| Live tracker APIs (OwnTracks/Overland/Traccar), WebSocket live map, realtime debounced pipelines | **Out of scope: live tracking + mobile.** Also structurally different: Roadbook's pipeline is batch-by-design, so it never needs the substantial realtime machinery that Dawarich's live-tracking mission legitimately requires. |
| Timeline panel, month calendar, points browser, day replay of arbitrary days | **Fails principle 1.** These exist to make browsing thousands of records pleasant — "it is not a timeline browser" rejects the entire category. |
| Sharing system (public links, live location shares, magic-phrase gates, shared stats/digests) | **Out of scope: sharing/social.** |
| Family features, plan tiers, subscriptions, rate limiting, multi-tenant auth | **Out of scope: multi-user.** Single-user self-hosted needs none of it. |
| Monthly stats, insights, digests, activity heatmaps | **Fails principle 1.** Coverage analytics — value scales with data volume, and Roadbook deliberately holds tens of records. Per-adventure reporting (already planned) is the curation-shaped substitute. |
| Fog of war, scratch map, H3 hexagon density grids | **Fails principle 1 most directly** — fog of war literally gamifies coverage ("go everywhere to clear the map"). The H3 precompute machinery solves an O(millions-of-points) problem Roadbook will never have. |
| Reverse-geocoding every point (4 pluggable providers, Redis dedup, nightly sweeps) | **Scale/dependency mismatch.** Per-point geocoding assumes millions of points and accepts a runtime service dependency — against your no-runtime-dependency stance. A one-shot lookup to *suggest a name at confirmation time*, behind the same pluggable-with-offline-fallback pattern as routing, is the acceptable remnant. |
| Speed-colored routes | **Conflicts with principle 3's visual channel.** Route color is the one channel that must encode observed-vs-inferred; spending it on speed muddies the honesty guarantee. Reject at least until the confidence encoding is established and clearly dominant. |
| Slim-array responses, ETag conditional GETs, concurrent page fan-out, raw-data archival | **Scale mismatch, not principle failure.** Built for 100k+ points per viewport; tens of adventures with a few thousand points each doesn't need it. Adopting it now is speculative complexity. |
| Dual map implementations (Leaflet + MapLibre in parallel) | **A migration-period trade-off, not a pattern to copy.** Running two map frontends in parallel obliged Dawarich to keep every behaviour consistent across both — understandable mid-migration, but costly. One map library, forever. |
| Notification center | **Unneeded for single-user local.** Surface import/pipeline errors inline in the UI instead. |

---

## 3. Good but premature

- **Poster studio** (printable map poster of a route). Of everything in Dawarich this is the most *Roadbook-shaped* idea — a curated set of tens of adventures is exactly what someone prints. But it consumes finished routes, so it needs phases 2–3 (and honesty question to settle first: do inferred segments render distinctly on paper too?). Revisit after phase 3.
- **Adventure replay animation** (Dawarich's day-replay, retargeted to an adventure). Strong fit for "the satisfaction is seeing the route," but it animates a *route*, so phase 3 first. Medium effort.
- **Elevation profile per adventure** (Dawarich stores gain/loss/max/min per track). Nice adventure-report addition; needs routed geometry with elevation data, so post-phase-3.
- **Photo display on the route** (Dawarich's EXIF extraction and thumbnail proxying). Phase 4 already plans photos-as-validation; Dawarich's pattern of normalizing photo EXIF into ordinary pipeline input is the right shape to borrow *when phase 4 starts* — its Google Photos JSON handling is directly relevant. Nothing to do earlier.
- **Watched-folder auto-import and full-archive backup/restore.** Self-hosting conveniences; natural phase-5 items, pointless before docker-compose exists.

---

## 4. Top 3 recommendations, ranked by value-to-effort

1. **Import hardening (1.1 + 1.2 + 1.3).** Every Roadbook user's first session begins with a Google Takeout upload, the format has at least three live variants plus an encrypted-backup trap, and a failed first import is a lost user. Dawarich has already enumerated the real-world failure modes and detection rules, so this is porting knowledge, not research. Streaming + upsert-dedup added now cost little; retrofitted later they cost a rewrite.

2. **Decision durability patterns (1.5).** "User decisions survive re-detection" is stated as a Roadbook invariant, and Dawarich's status-enum / lock-timestamp / derived-never-authoritative trio is a tested implementation of precisely that invariant. It's a schema-shape decision, cheapest to verify now while phase 1 is still open, and nearly impossible to bolt on cleanly after real user decisions exist.

3. **GPS anomaly filter (1.4).** Noise points inflate candidate detection and disfigure drawn routes, so the filter improves phase 1 immediately and phases 2–3 again later. The algorithm is fully specified (three passes, three named thresholds, both-directions spike test against a robust median) and fits Roadbook's pure-function detection architecture without any new infrastructure.

Near-miss: **flight-leg classification (1.8)** would rank third on value alone — it protects phase 3's distance-validation promise from its most likely systematic failure — but its payoff arrives only when routing exists. Do the cheap part now: let phase 2's gap model carry a `kind` field.

---

## 5. Phase-1 adoption checklist

Phase 1 is nearly done, so these are framed as things to **verify or harden before closing the phase**, ordered by cheapest-while-the-code-is-still-open:

**Schema-shape decisions (nearly free now, expensive later)**

- [ ] **Decision durability trio (1.5):** adventures carry `status ∈ {candidate, confirmed, dismissed}`; detection only ever *creates* candidates and never touches decided rows; a `name_locked_at`-style timestamp protects user-given names; every computed field (distance, stops, route) is regenerable, every user-entered field is not. Decide *now* how re-detection matches new candidates to existing decisions — time-range overlap is the natural key.
- [ ] **Idempotent import (1.3):** unique `(timestamp, lat, lon)` + `ON CONFLICT DO NOTHING` so re-importing an updated Takeout is a non-event. Dedup on numeric values, not formatted strings.
- [ ] **PostGIS `geography(Point,4326)` (1.9):** a one-line migration now; retrofitting after phase 2's map code exists means rewriting queries. Distance-in-meters comes free and phase 2's segmentation SQL ports verbatim.

**Import pipeline (the riskiest surface)**

- [ ] **Source detection + failure taxonomy (1.1):** handle the three Google Timeline variants (`timelineObjects`, `Records.json` E7 coords, `semanticSegments`/`rawSignals`) and give friendly rejections for encrypted Timeline backups, "My Activity" exports, HEIC/PDF, truncated JSON. Dawarich's rules live in `app/services/imports/source_detector.rb` and read as a spec.
- [ ] **Streaming parse + whole-import transaction (1.2):** `json.Decoder` tokens, never `ReadAll`; real Takeouts hit multiple GB.
- [ ] **Import bookkeeping UX:** per-import status enum, error-message field, counters (points parsed / created / duplicates), and an explicit "all points already exist" outcome.

**Detection quality**

- [ ] **Anomaly pre-filter (1.4):** Null Island, accuracy > threshold, both-directions speed-spike test (`max(1000 km/h, 3×median)`). Pure function, three named parameters. Flag, don't delete.
- [ ] **Confidence scoring with stored breakdown (1.7):** named components (distance-from-home, destination dwell, density, span), weight redistribution for missing components, breakdown shown in the confirm UI.
- [ ] **Dwell test (from 1.6):** the running-mean-anchor + 1.5× drift-cap stay test as the "real destination" check, even if full stop detection waits for phase 2.

---

## 6. Proposed roadmap additions

Items *not* in the current phases 1–5 that survive the principles filter:

| Addition | Attach to | Why |
|---|---|---|
| **Gap `kind` field** (`unknown \| road \| air`) in the route model | Phase 2 (design), phase 3 (use) | Cheap now; protects phase 3's distance validation from flights — its most likely systematic failure |
| **Local countries polygon table** → "countries crossed" per adventure | Phase 2–3 | Offline point-in-polygon, zero runtime dependency — matches the routing principle |
| **Name suggestion at confirm time** (one-shot geocoder lookup, pluggable with offline fallback, same pattern as routing) | Phase 2 | "Journey to Munich?" makes the confirm step faster without removing it |
| **Per-adventure GPX/GeoJSON export** — observed/inferred distinction preserved in the output | Post-phase 3 | Data ownership; synchronous generation is fine at this scale |
| **Poster / print view** of an adventure | Post-phase 3 | The most Roadbook-shaped idea in all of Dawarich — tens of curated adventures is exactly what someone prints |
| **Adventure replay animation** | Post-phase 3 | Serves "the satisfaction is seeing the route"; needs routes first |
| **Elevation profile** | Post-phase 3 | Nice report addition once routed geometry exists |
| **Demo/sample dataset** (Dawarich seeds demo data for onboarding) | Phase 5 | Lets people evaluate Roadbook without handing over their real Takeout — strong fit for a privacy-minded audience |
| **Configurable tile provider** | Phase 5 | Tile requests are the one thing that leaks (viewport) to a third party; self-hosters can point at their own tiles |
| **Full backup/restore archive** | Phase 5 | Small, and adventure decisions are the one thing users can't regenerate |

Deliberate omissions: watched-folder auto-import (marginal for a tool imported into a few times a year) and anything realtime (WebSockets, live channels) — the batch pipeline never needs them, and they account for much of the complexity a batch design avoids.

---

## 7. Data sources: what Roadbook should ingest (India-adjusted)

Dawarich ingests 15 file formats plus live tracker APIs. Filtered for Roadbook's audience — Indian users, ~95% Android, Google Maps near-universal — the realistic source landscape collapses to three tiers:

### Tier 1 — Google Timeline family (mainstream; effectively the only mass-market source)
For most Indian users, if the Timeline import fails there is no plan B, which upgrades import hardening (§1.1) from "top recommendation" to load-bearing. Three live JSON variants, all worth supporting:

| Variant | How the user obtains it | Status |
|---|---|---|
| Phone-device export (`Timeline.json`: `semanticSegments`/`rawSignals`/`timelinePath`) | Android: Settings → Location → Location Services → Timeline → Export Timeline data | **Current path** — Google moved Timeline on-device in 2024 |
| `Records.json` (raw samples, E7 integer coords) | Old Google Takeout archives | Deprecated but valuable — dense raw points, and *past* adventures only exist in old archives |
| Semantic History (`timelineObjects`, monthly files) | Old Google Takeout archives | Same — retrospective product, old archives are gold |

Also detect-and-reject with a helpful message: the **encrypted Timeline backup** (a different file the phone produces), "My Activity" exports, and KML from old Timeline UIs.

### Tier 2 — Google Photos EXIF (near-universal, usually forgotten)
Google Photos auto-backup is the Android default and arguably more universally enabled in India than Timeline itself. A decade of geotagged trip photos is a location history most users already have without knowing it. Two entry points: Photos Takeout JSON sidecars (`geoData` + `creationTime`), or EXIF from photo files directly. This both strengthens phase 4 (photos as route validation) and justifies promoting photos to a *primary source* for users with no Timeline data.

### Tier 3 — GPX (enthusiast tier, later phase)
The Indian motorcycle-touring crowd (Ladakh/Northeast circuit) and the Strava cycling community record rides deliberately — plausibly the most passionate early users. Product property worth noting: GPX tracks are dense, so those adventures arrive almost entirely *observed* — the honesty rendering shows them nearly gap-free with no routing inference needed. Timeline is the sparse case, GPX the dense case; the same observed/inferred model handles both. Obtainable from Garmin Connect, Strava (per-activity or bulk archive), Komoot, OsmAnd.

### Explicitly not worth building (for this audience)
- **Polarsteps, KML parsers** — European/Western backpacker tooling; negligible Indian usage. KML's one edge case (old Timeline exports) is handled by the failure taxonomy, not a parser.
- **OwnTracks `.rec`, Immich/PhotoPrism API pulls, AirTrail flights** — presuppose live-tracking or self-hosted-photo setups (Dawarich's audience, not Roadbook's).
- **CSV, FIT/TCX** — defer until someone asks; FIT/TCX users can export GPX.
- **Apple** — no practical export of iPhone Significant Locations exists (Dawarich's answer is live tracking, which Roadbook has ruled out); with India's small iPhone share this gap is acceptable. Apple users' path is photos and GPX.

### India-specific flag for phase 3 (routing)
OSRM runs on OpenStreetMap data; OSM coverage in India is good for highways and cities but patchy for rural and mountain roads — exactly where adventures happen. A failed distance validation on a Spiti Valley route is more likely "OSM doesn't know this road" than a detection bug. The honest-gap rendering (`unknown` stays visibly unknown rather than force-routed) is the *designed* behavior for an incomplete road network — worth stating in the phase 3 plan so it's an expected case, not a surprise.

### Architectural seam to preserve
All of Dawarich's 15 formats funnel through one seam: detect format → parse to a normalized point stream → identical pipeline afterward. Keep that seam in Go (a `PointSource` interface yielding normalized `{lat, lon, timestamp, accuracy?}` points) and every new format is an isolated parser; detection, confirmation, and routing never know the difference.

---

## 8. Future vision (parked — do not build yet)

Recorded 2026-08-01 so it isn't lost. These are *not* on the current roadmap; the second one requires an explicit charter change before any work starts.

### 8.1 Roadmap item: OSM amenity overlay along a route (post-phase 3)

**Problem:** on Indian road trips, finding restrooms, restaurants, fuel, and hospitals along a stretch is a real, personally-experienced pain point. Google Maps solves it poorly for route-level planning.

**Scope that survives the principles:** overlay OSM amenity data (toilets, restaurants/dhabas, fuel, hospitals) along an adventure's route as a read-only layer, from bundled OSM extracts — no accounts, no community, no runtime service dependency (same offline pattern as the countries table and pluggable routing). Solves the *facts* half of the problem ("what exists on this stretch"); deliberately leaves the *opinions* half ("is it clean") unsolved, because opinions need the community below.

**Effort:** medium. **Attach:** post-phase 3 (needs routes to overlay onto).

### 8.2 Parked direction (recorded privately)

A further product direction is parked outside this document, in the project's
private notes. It contradicts the current charter on several counts and must not be
absorbed into the roadmap silently; the private record includes explicit trigger
conditions, all of which must hold before it is even reconsidered.
