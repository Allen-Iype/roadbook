# Phase 4 log — photos

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. Counts derived from private data are
not stated here; the synthetic fixtures in `testdata/photos/` carry the
public numbers, and every extraction verdict reproduces via
`roadbook photo -inspect`.

## What the phase does

Photos attach to confirmed adventures, position themselves from their own
metadata, and check the route from outside it — the only ground truth in
the project that does not come from the same source as the error.

- `internal/photo` is a parser package in the `internal/timeline` mould
  (invariant 4): pure functions over bytes, nothing EXIF-shaped escaping.
  The EXIF walk is hand-rolled — JPEG segments to APP1, the embedded TIFF
  in either byte order, and exactly the seven tags the product consumes.
  Every read is bounds-checked; malformation is absence, never a crash,
  and a test feeds the parser every truncation and ~1,400 single-byte
  corruptions of a valid file to hold that line. Takeout sidecar JSON is
  the second source through the same seam: EXIF wins for position, the
  sidecar fills gaps, `photoTakenTime` (capture) over `creationTime`
  (upload-to-Google) — a correction to PLAN's field name, recorded in the
  brief. Capture time resolves down an explicit five-rung ladder (GPS UTC
  clock → OffsetTimeOriginal → sidecar epoch → wall clock in the
  adventure's own offset → unplaced), recorded per photo in
  `time_source`; position provenance in `pos_source`. Thumbnails bake the
  EXIF orientation into pixels and re-encode, which strips every metadata
  block by construction — the served file is pixels only.
- Photos are the second class of irreplaceable user data (migration
  00006): a row plus a hash-named thumbnail file under gitignored
  `data/photos/`, originals discarded in-request and never written to
  disk. Durable identity comes from attaching to the *decision* — the one
  row that already survives re-detection by anchored matching — proven
  live at close: a re-detection renumbered every candidate and both real
  photos served under the new id untouched. Content-hash uniqueness makes
  re-upload idempotent, the imports precedent.
- The upload path is the architecture unbent: browser → server action
  (multipart FormData) → Go API → Postgres and disk; thumbnails come back
  through a Next route handler proxy, so the browser never talks to Go
  even for images. Results are per-file — accepted, duplicate, rejected
  with an actionable reason (the phase 1 sniff taxonomy extended to HEIC,
  video, PNG, WebP, RAW), sidecar paired or unpaired — so one bad file
  never fails a batch, and the upload island renders each verdict inline
  under optimistic previews (`useOptimistic` + object URLs at
  seconds-scale, the decide-cell pattern stretched).
- Placement is derived at read time, never stored: `journey.PlacePhoto`
  finds the stop or leg whose span holds the photo's instant and measures
  point-to-polyline against exactly the geometry the map draws — observed
  points, routed polyline, or gap chord; stops before legs because the
  dwell is the more specific claim; air legs place but never flag
  (the arc is presentation, not a claimed path). `PhotoFarWarnM`
  (default 1000 m) is named, echoed in every list response, and
  deliberately not part of `journey.Params` so the golden fixtures keep
  pinning assembly alone. On the page: thumbnail markers with an amber
  disagreement ring when flagged, the distance statement phrased
  identically in popup and strip, photos slotted into leg and stop rows,
  legend entries in words.
- The store gained its test harness mid-phase (maintainer's call):
  scratch-database integration tests running by default — `make test`
  resolves a database itself (env → local Postgres → Docker compose →
  visible skip) — covering decision re-attachment across re-detection,
  import idempotency, the photo round-trip with its conflict path, and
  delete ordering proven by injected failure. No mocks: the store's risk
  is the SQL, so the SQL is what is tested.

## What broke, and why each fix took its form

**The maintainer's own phone violates the EXIF specification.** Xiaomi
writes the Orientation tag as LONG where the spec says SHORT; the
spec-strict walk ignored it. Harmless on the actual file (value 0, unset)
but a latent sideways-thumbnail bug. Fix: accept both integer types for
the tag — because the parser's job is the field, not the spec — with the
LONG variant written into the big-endian fixture so the shape has
regression coverage. Found within hours of checkpoint 1 by running
`-inspect` on a real photo; the CLI window earned its place immediately.

**The fixture generator mispointed the GPS IFD.** With no Exif IFD
present, `gpsBase` still added an empty directory's size and the pointer
overshot by six bytes — caught the moment the big-endian test ran, because
the committed fixtures are parsed as bytes, not trusted as intentions.
Fix in the generator's offset arithmetic; the discipline of committing
generator *and* output is what made the bug loud.

**WhatsApp strips everything.** The first real upload
(`IMG-…-WA0009.jpg`) carried no EXIF at all — verified against the raw
bytes, not just the parser. Not a bug: the designed degenerate case
(stored, shown, honestly unplaced) exercised by real data on day one. The
data reality — messenger-saved photos never place themselves; camera
originals and Takeout sidecars do — is now stated in the empty-strip help
text.

**`INSERT … ON CONFLICT DO NOTHING RETURNING` returns no row on
conflict.** pgx surfaces that as `ErrNoRows`, which reads like an error
but means "duplicate": the fix is the explicit refetch path returning the
untouched original row, and the store harness pins it — including that
the original's name and adventure survive a duplicate upload claiming
different ones.

**The Next server-action body limit is 1 MB by default** — below a single
camera JPEG, and the failure would have surfaced as an opaque transport
error. Raised deliberately to 64 MB in `next.config.ts` (the local Next
docs state the knob and its multipart overhead caveat); the real limits
stay in Go as named parameters, per-file and per-request, and a
too-large batch fails with a message saying to send fewer at once.

**Placement parameters must not touch `journey.Params`.** The first
instinct — add `PhotoFarWarnM` beside the assembly thresholds — would
have changed the params object both golden fixtures round-trip. Placement
runs after assembly, so its parameter lives with `PlacePhoto` and is
echoed by the photos endpoint instead. The goldens stayed byte-identical
through the whole phase, which was the point of drawing the line there.

## Verification at close

Cold-cache `make test` green with the DB-backed store tests running (not
skipped) and both golden journey regressions byte-unchanged; fixture
detection still 18 candidates / 1 base / 32 outliers; re-detection (run 7)
on top of migration 00006 re-attached all 18 decisions with 0 orphans and
both photos followed their decision to the renumbered candidate, the stale
id answering 404; upload, duplicate, per-file rejection, thumbnail
serving, and delete each demonstrated live by curl and again in the
browser; the far flag demonstrated end to end with a deliberately
mis-placed synthetic photo (placed on the golden bus journey's first leg,
distance stated, ring drawn, then deleted); `git add -A --dry-run` clean
of `data/` and of anything near 1 MB throughout.

## Carried forward

- **HEIC** stays excluded with an actionable rejection. Trigger to
  revisit: real uploads blocked by it in practice; the decode-path cost
  decision (CGO libheif vs required pre-conversion) is made then, in
  DECISIONS.md.
- **Photos as a primary location source** (PLAN: users with no Timeline
  data) is deferred to the phase that owns imports; `internal/photo` is
  deliberately the seam it will reuse. It needs its own brief: stay-point
  synthesis over photo positions, home derivation without visits.
- **Backup** now spans the database *and* `data/photos/` — phase 5's
  backup/restore feature covers both (PLAN updated).
- **`PhotoFarWarnM` retuning**: 1000 m is reasoned, not measured. Living
  with real flagged photos decides whether it moves; a retune is a
  recorded parameter change. Viewpoints reached on foot from a parked
  vehicle are the expected false-positive class.
- **Orphaned thumbnail files**: the row-first delete ordering fails
  toward sweepable garbage by design. If garbage ever accumulates, a
  sweep command (files in the photos directory whose hash no row
  references) is trivial; not built without evidence.
- **The unified timeline view** (photos, stops, and legs as one vertical
  narrative rather than table-plus-lists) is a presentation idea for a
  later phase; placement data already supports it.
