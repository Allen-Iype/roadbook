# Phase 4 design brief — photos

Written before any code, per the working agreement. Goal (PLAN.md): attach
photos to an adventure, position them, and use them to check the route.

Closed decisions this brief builds on, not reopens: photos are the most
accurate positions in the project — camera GPS is metres against tens of
metres for WiFi positioning (PLAN, logged); they independently validate the
route — the only ground truth that does not come from the same source as the
error (PLAN, logged); thumbnails only, originals never hosted (PLAN, logged);
video is excluded (PLAN, logged); Google Photos Takeout JSON sidecars are a
second metadata source alongside raw EXIF (PLAN, features); the journey
pipeline is pinned by `testdata/journey-27jul2026.CONTRACT.md` and both golden
fixtures — photo placement consumes journeys and must not disturb them; the
Dawarich summary answers photo-integration mechanics only (normalize
`{lat, lon, timestamp}`, then reuse the normal pipeline; thumbnails proxied
through the backend), never architecture.

---

## 1. Concepts this phase introduces

### 1.1 EXIF, and binary metadata parsing in Go

A JPEG file is a sequence of *segments*, each opened by a two-byte marker.
Metadata rides in an application segment near the front: marker `0xFFE1`
("APP1"), a two-byte length, the ASCII signature `Exif\0\0`, and then a
complete embedded TIFF structure. TIFF is the actual container: a header
declaring byte order (`II` little-endian or `MM` big-endian — both exist in
the wild, so a parser must handle both), then linked *image file directories*
(IFDs). An IFD is a count followed by 12-byte entries: tag number, field type,
value count, and either the value itself or an offset to it elsewhere in the
blob. The zeroth IFD holds camera-level tags and, behind pointer tags, two
sub-directories this project cares about: the Exif IFD (capture time) and the
GPS IFD (position). Coordinates are stored as three *rationals* — numerator
and denominator pairs for degrees, minutes, seconds — plus a hemisphere
reference (`N`/`S`, `E`/`W`); a parser converts to signed decimal degrees.

The properties that shape the code: the format is offset-based (an entry can
point anywhere in the blob, so a malicious or truncated file can point out of
bounds — every read must be bounds-checked, and Go's slice indexing makes the
check natural rather than heroic); it is redundant in confusing ways (several
timestamp tags with different meanings — §3E); and files in the wild violate
the specification routinely. The parser therefore treats *absence* as normal
and *malformation* as absence-with-a-reason, never as a crash: the product
answer to "no GPS IFD" and to "GPS IFD offset points past end of file" is the
same — a photo without a position, stored and shown honestly as unplaced.

Where this lives: `internal/photo`, a parser package in the mould of
`internal/timeline` — it emits domain types and nothing else (invariant 4).
Nothing tag-shaped or IFD-shaped escapes it. §3C weighs hand-rolling this
walk against importing a dependency.

### 1.2 Image processing and thumbnailing

Go's standard library models a decoded image as the `image.Image` interface —
a bounds rectangle plus per-pixel colour access — with `image/jpeg` decoding
into it and encoding out of it. Scaling is not in the standard library proper
but in `golang.org/x/image/draw` (same authors, versioned separately): a
`Scaler` interpolates source pixels into a destination rectangle. The scalers
trade quality for speed — nearest-neighbour, bilinear, Catmull-Rom — and for
a batch of tens of photos the slowest, best one (Catmull-Rom, a cubic filter)
costs milliseconds per image and is the right default.

Two subtleties matter more than the scaling itself. First, **orientation**:
phones store sensor-native pixels plus an EXIF `Orientation` tag telling the
viewer to rotate or mirror. The thumbnail must *bake the transform into the
pixels*, because the pipeline strips metadata — a thumbnail with neither
pixels rotated nor a surviving tag renders sideways forever. Second,
**metadata stripping is a privacy property that falls out of the design**:
re-encoding pixel data produces a fresh JPEG with no EXIF block at all, so a
served thumbnail carries no embedded position or timestamp. The position
lives in Postgres, where the API decides what to expose; the image file the
browser receives is pixels only.

Thumbnail geometry and quality are named parameters (`ThumbMaxPx` default
512 — comfortably larger than any rendering on the page; JPEG quality 80).
Not because anyone will tune them often, but because invariant 3's discipline
is cheap here and a magic 512 in an algorithm body is exactly what it forbids.

### 1.3 File upload through the one API boundary

Uploads are the first *binary* traffic through the architecture, and the
architecture does not bend for them:

```
browser ──▶ Next.js server action ──▶ Go API (multipart POST) ──▶ Postgres
                                                              └─▶ photos dir
browser ◀── Next.js route handler ◀── Go API (thumbnail GET)
```

The transport is `multipart/form-data` — the HTTP encoding for "several named
parts, some of them files, in one request." The browser produces it from a
form or `FormData` object; the Next server action receives it as web-standard
`File` objects; the action forwards the parts to the Go API, which is the
only process that touches the filesystem or the database. `openapi.yaml`
declares the multipart request body, and the oapi-codegen strict server hands
the handler a typed multipart reader — the contract covers binary endpoints
the same way it covers JSON ones (invariant 10).

Coming back the other way, thumbnails are images, not JSON, and the browser
must not talk to the Go service (the one-boundary rule). So the Next layer
proxies them: a *route handler* — the App Router's mechanism for endpoints
that return arbitrary HTTP responses rather than pages — streams the bytes
through. Server actions and route handlers split the work along their grain:
actions are for mutations invoked from React (the upload), route handlers for
plain HTTP resources (`<img src="/api/photos/…">` needs a URL, not a
function call). Both run only on the server; the browser still holds no
connection string and no storage path (invariant 11). The practical
constraint to verify against the local Next docs at implementation time is
the server-action request body limit — the default is around 1 MB, far below
a photo, and the config knob for raising it must be set deliberately, not
discovered in production. Sizes and counts are named parameters
(`MaxUploadBytes` default 25 MB per file, `MaxUploadFiles` default 50 per
request) enforced in Go — the Next-side limit merely accommodates them.

### 1.4 Photos as the second class of irreplaceable user data

Migration 00001 divided the schema into strata: observations (immutable
inputs), derived data (candidates — regenerated wholesale, disposable), and
user data (decisions — never regenerated, never deleted by any automated
process). Photos join the third stratum, and they widen it: a decision is one
row, but a photo is a row *plus a file* — the thumbnail on disk — and the
original is discarded after extraction (the closed thumbnails-only decision),
so the system cannot regenerate what it stores. If the row or the file is
lost, re-upload is the only recovery, and only if the user still has the
original. Everything downstream follows from taking that seriously:

- **Durable identity.** Candidates carry no identity across runs; decisions
  survive re-detection by anchored matching. Photos must survive it too. §3A
  chooses the anchor.
- **Idempotency by content hash**, exactly as imports work: the SHA-256 of
  the original's bytes is unique in the table, so re-uploading the same photo
  is a no-op, not a duplicate — and the hash doubles as the thumbnail's
  filename, which makes row↔file association trivial and collision-free.
- **No automated deletion.** Only an explicit user action deletes a photo,
  and it deletes exactly that photo's row and file. Nothing in detection,
  re-import, or routing can touch the photos stratum.
- **The backup surface widens.** "Back up your decisions" was a
  database-only story; it now includes a directory. Phase 5's backup/restore
  feature inherits this (noted in §7, carried to PLAN).
- **Same safety class as `data/`.** Thumbnails are real location-bearing
  personal images. They live under the gitignored `data/` tree, and any
  committed fixture is synthetic (§4).

### 1.5 Optimistic UI for slow operations

The repository already has optimistic UI at millisecond scale: `decide-cell`
flips its badge with `useOptimistic` before the server action confirms.
Uploads stretch the same idea across seconds and multiple items, which
changes the shape of the problem: the user selects five photos, the round
trip (upload + EXIF extraction + thumbnailing) takes visible time, and any
one file can fail independently (unsupported format, over the size limit).

The React pieces, since this territory is new: a client component owns a file
input; on selection it builds the pending state *locally* — the browser can
render a preview of a not-yet-uploaded file via `URL.createObjectURL`, which
mints a temporary in-memory URL for a `File` object, no server involved. The
component calls the server action inside a *transition* (`useTransition`),
which is how React marks "this update is in flight" without freezing the
input, and `useOptimistic` overlays the pending photos onto the
server-confirmed list. When the action completes it calls `revalidatePath`,
the server re-renders with the photos now in Postgres, and React swaps the
optimistic overlay for the real rows — same reconciliation `decide-cell`
relies on. Failures must be per-file: one HEIC in a batch of five means four
thumbnails and one inline error naming the file and the reason, not a failed
batch. The Go API's per-file result list (§3F) exists precisely so the
frontend can render that truthfully.

---

## 2. What gets built

- `internal/photo` (new): the EXIF parser (JPEG APP1 → TIFF walk → the tags
  this product needs and no more), the Takeout sidecar JSON parser, format
  sniffing with actionable rejections, and the thumbnailer. Pure functions
  over bytes; emits domain types only; no I/O.
- Migration `00006_photos.sql`: the photos table (§3B).
- `internal/store`: photo insert (content-hash idempotent), list by decision,
  delete; thumbnail file write/read/delete under the photos directory.
- `api/openapi.yaml` + `make generate`: multipart upload, photo list with
  placement, thumbnail bytes, delete (§3F).
- `roadbook photo -inspect FILE`: prints extracted metadata and the
  sniff verdict for any file — the parser made visible, and the debugging
  tool for every future "why didn't my photo place?" question.
- `web`: upload island on the adventure detail page (file input, optimistic
  pending previews, per-file errors), photo strip, thumbnail proxy route
  handler, map markers and timeline placement, the far-from-route flag
  rendering.
- `docs/phase-4/`: DECISIONS.md as decisions are made; LOG.md at close.

---

## 3. The real choices

**A. What a photo attaches to — durable identity.** Recommendation: a photo
belongs to a *decision* — `decision_id` FK, uploads accepted only for
candidates whose attached decision is `confirmed`. Decisions are the one
durable representation of "an adventure" in the system: they are never
regenerated, they already survive re-detection by anchored matching, and
they are user data attaching to user data — when re-detection renumbers
candidates (today's ids 79–96 were 61–78 a run ago), the photo rides with the
decision, which re-matches by anchor. Upload to an unconfirmed candidate is
rejected with a message saying to confirm first: PLAN's feature is "upload
photos for an adventure," and an adventure is a confirmed candidate — the
product; photos on dismissible rows would attach user data to disposable
rows. *Placement*, by contrast, is not stored at all: where a photo sits on
the map and timeline is derived at read time from the photo's own timestamp
against the assembled journey (§3G), so re-detection that shifts a span
re-places photos automatically. Rejected: FK to candidates (disposable by
design — every run would orphan every photo); anchor-copying span+dest into
the photo row as decisions do (photos are not independent judgments about a
span; they belong to an adventure that already owns an anchor — duplicating
it re-solves a solved problem and lets the two copies disagree); no
attachment at all, association purely by timestamp overlap (loses upload
intent — a photo from the journey's departure morning taken at home sits
outside the span and would attach to nothing, though the user chose the
adventure to give it to).

**B. What is stored, and where.** Recommendation:

- *Files:* thumbnails only, named by content hash, in a flat directory —
  `data/photos/<sha256>.jpg` by default, configurable via
  `ROADBOOK_PHOTOS_DIR` (environment, no coordinates in config — invariant
  9). Flat because the working set is tens to low hundreds of files
  (photos-per-adventure × tens of adventures) and fan-out subdirectories
  are scale theatre. Under `data/` because that is the tree the safety
  rules already guard: gitignored wholesale, dry-run-checked, never
  committed. The originals' bytes exist only in memory during the request —
  parsed, hashed, thumbnailed, discarded; they are never written to disk,
  which is both the closed thumbnails-only decision and the strongest
  possible form of "scripts must never write real coordinates outside
  `data/`."
- *Schema:* one row per photo — `id`, `decision_id` (FK, `ON DELETE
  RESTRICT`; decisions are never deleted, and the constraint documents it),
  `content_hash` unique, `original_name` (display and sidecar pairing),
  `taken_at timestamptz` + `taken_offset_sec` (nullable pair — the schema's
  existing convention for instants that must round-trip with their civil
  offset), `time_source`, `lat`/`lon` (nullable), `pos_source`,
  `thumb_w`/`thumb_h` (render without decoding), `uploaded_at`. The two
  `*_source` columns are provenance in the spirit of invariant 3: every
  position and instant states which reading produced it (§3D, §3E), so a
  wrongly-placed photo is explainable a year later.
- *Not stored:* distance-from-route, the far flag, leg association — all
  derived at read (§3G), the same reasoning that keeps journeys, countries,
  and scores derived: stored copies of pure functions over data already in
  Postgres are only a chance to disagree with their inputs. Camera
  make/model: not stored — nothing in the product reads it, and a column
  that exists "in case" is how schemas rot.
- *Deletion:* `DELETE /photos/{id}` removes the row and the file, in that
  order (a file with no row is unreachable garbage; a row with no file is a
  broken image on the page — if the two-step can fail in the middle, fail
  toward garbage). User-initiated only.

Rejected: keeping originals in a private directory "since we have them
anyway" (re-opens a closed decision; turns the photos directory into an
unbounded archive with a backup bill, exactly what PLAN declined); storing
thumbnails as bytea in Postgres (couples database size to image data,
makes every backup heavier, and buys nothing — the filesystem is already
inside the trust boundary that `data/` defines); object storage (a
dependency the deploy story doesn't have, for a problem measured in
megabytes).

**C. The EXIF parser: hand-rolled minimal walk, not a dependency.**
Recommendation: write the parser — JPEG segment scan to APP1, TIFF header,
IFD walk, and exactly the seven tags the product consumes: GPS latitude,
longitude and their hemisphere refs, GPS date and time stamps,
`DateTimeOriginal`, `OffsetTimeOriginal`, `Orientation`. Reasoning: the
needed surface is a few hundred lines against a format whose relevant subset
is stable since 1998; the candidate dependencies are a poor trade —
`rwcarlsen/goexif` is unmaintained since 2019 with known panics on malformed
input (a parser of untrusted uploads that can panic is a denial-of-service
invitation), and the maintained alternatives pull in full read-write EXIF
manipulation for what is, here, seven read-only tags; and PLAN names binary
metadata parsing as a concept this phase introduces — the learning goal is
the walk itself, in this project's defensive-parser idiom, not the wrapping
of a black box. The existing sniff taxonomy (phase 1 checkpoint 5) extends
to uploads with actionable messages: HEIC ("iPhone default format — convert
to JPEG or export via Google Photos, which serves JPEG"), PNG ("no camera
position metadata — screenshots and edited exports typically lose it"),
video formats (excluded by PLAN), and the existing gzip/zip/pdf/html
rejections reused verbatim. Rejected: `goexif` (above);
`dsoprea/go-exif` (maintained but ~10× the surface this needs, and the
project's precedent — countries, fonts, map workers — is to own small
things rather than depend for them); decoding via `image.DecodeConfig`
alone (yields dimensions, no metadata).

**D. Position: EXIF first, sidecar fills the gaps.** A Takeout sidecar is a
small JSON file Google exports beside each photo (`IMG_1234.jpg.json`, or
`….jpg.supplemental-metadata.json` in newer exports) carrying `geoData`
(latitude/longitude/altitude), `photoTakenTime` (epoch seconds, UTC), and
`title` (the image filename). Recommendation: when both sources speak,
**EXIF GPS wins for position**; sidecar `geoData` is used when EXIF has no
GPS block. The EXIF position is the camera's own sensor reading embedded in
the artifact; `geoData` is whatever Google holds — sometimes the same EXIF
value round-tripped, sometimes inferred from Location History, sometimes
hand-placed in the Photos UI. Preferring the measurement over the mirror is
this project's standing bias, and `pos_source` records which one answered
(`exif` | `sidecar` | `none`). A `geoData` of exactly (0, 0) means absent,
not Null Island — the same rejection the anomaly filters already apply to
raw positions. Sidecars pair with their image within one upload request by
filename (strip the `.json` / `.supplemental-metadata.json` suffix, fall
back to the `title` field); an unpaired sidecar is reported as such in the
per-file results, not silently dropped. One correction to PLAN, recorded
here openly: PLAN names `creationTime`, but that field is the upload-to-
Google time; capture time is `photoTakenTime`, which is what the parser
reads, falling back to `creationTime` only when `photoTakenTime` is absent.
Rejected: sidecar-first (prefers the copy to the measurement); merging
per-axis (a position assembled from two sources is a position nobody ever
measured).

**E. Time: resolving the EXIF timezone problem.** `DateTimeOriginal` — the
main EXIF capture-time tag — is *civil wall-clock time with no UTC offset*:
`2026:07:27 21:15:03` says nothing about where on Earth that was. Placement
needs an instant; guessing wrong by hours puts a photo on the wrong leg.
Recommendation, a precedence ladder recorded per-photo in `time_source`:

1. **GPS date + time stamps** (`gps`): the GPS receiver's own clock, defined
   UTC, present whenever a full fix was recorded. Strongest.
2. **`DateTimeOriginal` + `OffsetTimeOriginal`** (`exif_offset`): the
   explicit-offset tag added in EXIF 2.31 (2016); newer phones write it.
3. **Sidecar `photoTakenTime`** (`sidecar`): epoch seconds, unambiguous.
4. **`DateTimeOriginal` interpreted in the adventure's own offset**
   (`exif_local`): the span's `*_offset_sec` columns preserve the journey's
   civil offset precisely so instants can round-trip; a photo uploaded to
   that adventure almost certainly shares its timezone. Weakest, stated as
   such, and correct in the overwhelming case — a single-timezone journey.
5. **No usable time** (`none`): stored, shown in the strip, unplaced on map
   and timeline.

The known blind spot, stated rather than hidden: rung 4 misresolves a photo
taken across a timezone boundary *within* a multi-zone adventure by the zone
difference (for this data, typically 30 minutes — IST against Nepal or
Myanmar time). The photo still lands on the journey, at worst one leg off;
`time_source` says exactly how much to trust it; and rungs 1–3 cover every
photo with any explicit evidence. Rejected: treating wall time as UTC
(hours wrong, guaranteed misplacement — the worst option dressed as the
simplest); requiring explicit-offset evidence and refusing rung 4 (drops
placement for most older Android JPEGs, the likeliest actual uploads);
per-photo timezone lookup from the photo's *position* (circular — position
is the thing being validated, and a wrong position would then corrupt the
timestamp too).

**F. The API surface.** Recommendation — four operations, additive, via
`openapi.yaml` and `make generate` as always:

- `POST /candidates/{id}/photos` — multipart; images and optional sidecar
  JSONs in one request. Resolves the candidate's attached decision (409 if
  none or not confirmed — mirrors the existing decide semantics). Response:
  **a per-file result list** — accepted (with the stored photo resource),
  duplicate (content hash already present; the existing photo returned),
  or rejected (the sniff taxonomy's actionable message). Per-file because
  §1.5 needs it: one bad file must not fail a batch, and the UI must say
  which file and why.
- `GET /candidates/{id}/photos` — the photo list with derived placement
  (§3G): each photo's resolved instant, position, sources, thumbnail
  dimensions, and — when placeable — leg index, distance-from-route, and
  the far flag, with the parameters echoed.
- `GET /photos/{id}/thumbnail` — `image/jpeg` bytes, streamed through the
  Next route handler proxy (§1.3).
- `DELETE /photos/{id}` — row and file (§3B).

Rejected: folding photos into the journey response (the journey shape is
pinned by CONTRACT.md and two golden fixtures; photos are a separate
concern with a separate lifecycle, and a second consumer of the journey
endpoint should not pay for photo bytes); a separate upload service or
presigned-URL pattern (a second network boundary for a problem the one
boundary handles — scale theatre).

**G. Placement, and the far-from-route flag.** The flag's meaning, from
PLAN: a photo timestamped mid-journey but sitting kilometres off the
inferred road proves the inference wrong. That meaning dictates the
geometry: the comparison is *time-aware* — recommendation:

- Find where the journey claims to have been: the leg (or stop) whose time
  span contains the photo's resolved instant.
- Measure the photo's distance to *that* element's drawn geometry: an
  observed leg's measured points; a road gap's routed polyline
  (`RoutedPoints`); an unknown gap's endpoint chord — in every case,
  point-to-polyline distance against exactly what the map draws, so the
  flag and the picture can never disagree. A photo during a stop measures
  to the stop's location.
- Flag when the distance exceeds `PhotoFarWarnM`, default **1000 m**: an
  order of magnitude above the source data's positioning error (~100 m
  WiFi p90), so measurement noise cannot fire it, and well below the
  kilometres-scale divergence of a genuinely wrong route. A named
  parameter, echoed in the response; if real journeys prove it noisy —
  viewpoints reached on foot a kilometre from the parked car are a
  plausible false positive — it retunes as a recorded change, not a code
  edit (the `DivergenceWarnPct` precedent).
- Air legs are excluded from flagging: a photo out an aircraft window can
  sit anywhere along (or off) the great-circle arc; the arc is presentation
  between two endpoints, not a claimed path (phase 3 §3F), so "far from
  it" asserts nothing. Shown on the map at its position, unflagged.
- Photos with no position, or no placeable instant, or an instant outside
  every leg and stop: in the strip, marked unplaced — absence rendered as
  absence.

Rendering, under invariants 5 and 8: a photo marker *is a measurement* — the
most accurate in the project — so it renders as a solid, confident marker
(the thumbnail itself, small, at its coordinates), never dashed and never
downgraded by what it sits near. The *flag* is a disagreement between two
data classes, and it renders as exactly that: a visible warning ring on the
marker plus a plain statement in the photo's popup and the strip — "2.4 km
from the routed road at this time" — next to the leg-kind language the page
already speaks. The legend gains the photo marker and its flagged variant.
On the timeline, photos slot between stops and legs by instant, thumbnails
inline. Rejected: nearest-distance-to-any-leg (a photo near yesterday's leg
scores well while contradicting today's — destroys the validation meaning);
interpolating an expected position along the leg at the photo's instant and
measuring point-to-point (manufactures a position on an inferred line and
then treats it as ground — fabrication compounding inference; and gap
endpoints' timestamps make interpolation inside routed geometry undefined
anyway); hiding flagged photos or gating uploads on proximity (the flag is
the product working — a conversation starter, never a gate, exactly as
divergence is).

**H. Photos never feed the pipeline they validate.** Photo positions do not
enter assembly, detection, routing, scoring, or countries attribution — this
phase or by accident later. The reason is the validator's independence, the
same property PLAN bought by validating against Google's figure: the moment
photo positions merge into the observation stream, a route inferred partly
*from* a photo can no longer be checked *against* it, and the far flag would
be testing the pipeline's agreement with itself. Structurally, photos also
sit in the wrong stratum to be inputs: they are user data arriving after
detection, not immutable observations (invariant 2's tables are written only
by import). Countries stays measurement-only exactly as phase 3 §3H decided
— a photo position adding a country the track never showed would be the
same inference-stacking that decision rejected. The PLAN feature "for users
with no Timeline data at all, photos are the primary source" is real and is
*not* this phase: that is photos-as-an-import-source — a bulk ingestion
through the observation strata via the same domain-type seam, a different
feature with its own brief (deferred to the import phase that owns it, noted
in §7). Rejected: "photos are the most accurate data, so let them correct
the track" — accuracy is exactly why they must stay outside; the check is
worth more than the correction.

---

## 4. Data safety, extended to photos

The standing rules apply unchanged; photos add three specifics:

- Real photos and their thumbnails exist only under `data/` (default
  `data/photos/`) or a maintainer-chosen `ROADBOOK_PHOTOS_DIR`. Original
  bytes are never written to disk at all (§3B). Nothing photo-shaped is ever
  written outside the gitignored tree.
- Committed fixtures in `testdata/photos/` are **synthetic**: tiny
  program-generated JPEGs with hand-assembled EXIF blocks at fabricated
  coordinates (the anonymised-frame discipline of the golden fixtures), plus
  hand-written sidecar JSONs to match. They test the parser's walk, the
  precedence ladders, and the malformation taxonomy — degenerate files
  (truncated APP1, out-of-bounds IFD offset, zero-denominator rational) are
  committed as fixtures too, because the defensive paths deserve regression
  coverage as much as the happy path.
- The `git add -A --dry-run` review gains a habit, not a rule change:
  nothing from `data/`, nothing over 1 MB — a real thumbnail would trip
  both checks if it ever escaped, which is the defence-in-depth the rule
  was written for.

---

## 5. Checkpoint order — four slices, each visible, each a STOP

1. **The parser and thumbnailer, pure, with a CLI window.**
   `internal/photo` complete: sniffing, EXIF walk, sidecar parsing, the
   position and time precedence ladders, orientation-aware thumbnailing —
   all pure, all tested against the synthetic fixtures. `roadbook photo
   -inspect FILE` prints the verdict: format, resolved position and its
   source, resolved instant and its source, orientation, thumbnail
   dimensions — or the actionable rejection. *Visible: the maintainer runs
   it against real photos from a real adventure and the terminal shows
   correct metadata; against a HEIC and sees the actionable message.*
   Parser first for the same reason the Go CLI preceded everything in
   phase 1: every later checkpoint consumes its output, and its
   correctness is checkable in the terminal with zero infrastructure.

2. **Schema, store, and the API, proven by curl.** Migration 00006; store
   functions; the four endpoints live. *Visible: `curl -F` uploads real
   photos to a confirmed adventure — rows appear with sources recorded,
   thumbnails appear under `data/photos/`; re-uploading reports duplicates
   and adds nothing (idempotency demonstrated); a HEIC in the batch comes
   back rejected with its message while the JPEGs land (per-file results
   demonstrated); `GET …/thumbnail` renders in a browser; `DELETE` removes
   row and file.* The frontend does not start until curl proves the API —
   the phase 1 build-order rule, unchanged.

3. **Upload UI and the photo strip.** The upload island (file input,
   `useTransition` + `useOptimistic` pending previews via object URLs,
   per-file inline errors), the server action, the thumbnail proxy route
   handler, the strip on the adventure page — placed and unplaced photos
   shown, sources stated. *Visible: the maintainer drags five photos onto
   an adventure in the browser, watches previews resolve into thumbnails,
   sees one bad file fail inline while four succeed, reloads and everything
   persists.* Next docs re-read before this checkpoint (body limit, route
   handler streaming, `revalidatePath` interaction with the proxy).

4. **Placement, the flag, and validation.** Derived placement in the photo
   list response; markers on the map (thumbnail markers, warning ring when
   flagged, legend updated); timeline slotting; the distance statement in
   popup and strip. *Visible: an adventure's photos sit on its route at the
   right places; a photo deliberately mis-timed or mis-placed (a synthetic
   test upload) shows the ring and the distance; the far flag catches it.*
   Then the phase log, and the phase is complete when it exists.

---

## 6. Verification at phase close

`make test` green from a cold cache, both golden journey regressions among
them and byte-unchanged (photo placement consumed journeys without touching
assembly); fixture detection still 18/1/32; every decision still attached
after migration 00006 and a re-detection, photos still attached to their
decisions after the same (durable identity demonstrated the way decisions'
was); CLI `-inspect`, API, and page agree on every photo's position, instant,
and sources; re-upload idempotency and per-file rejection shown live; the
serve path demonstrably free of any code path that writes outside the photos
directory; `git add -A --dry-run` clean of `data/` and of anything near
1 MB throughout, with real photos and thumbnails living only under
`data/photos/`.

## 7. Outstanding questions and exclusions

**Excluded and staying excluded** (closed in PLAN, restated so no checkpoint
re-opens them): hosting originals — the request handler discards original
bytes after extraction, and no flag revives them; video — MP4/MOV uploads
get a sniff rejection naming the exclusion.

**HEIC.** Excluded this phase with an actionable rejection (§3C). The
revisit trigger is concrete: real uploads blocked by it in practice. The
decode path (a CGO libheif binding, or requiring pre-conversion) is a cost
decision to make against evidence, in DECISIONS.md, not pre-emptively.

**Photos as a primary location source.** PLAN's bulk-ingestion direction
(§3H) — photos entering the observation strata as an import source for
users with no Timeline data. Deferred to the phase that owns imports and
self-hosting; it needs its own brief (stay-point synthesis over photo
positions, home derivation without visits). The `internal/photo` parser
built here is deliberately the seam it will reuse.

**Backup tooling.** Phase 5's backup/restore of decisions now also covers
the photos table and directory (§1.4). Carried to PLAN's phase 5 notes.

**`PhotoFarWarnM` retuning.** Default 1000 m is reasoned, not measured
(§3G). Living with real flagged photos decides whether it moves; a retune
is a recorded parameter change.
