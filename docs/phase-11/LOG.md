# Phase 11 log — photos as an import source

Written at phase close. What the phase built, what the real-data measurement
showed, what broke while building, and why each fix took the form it did.
The committed regressions — fixture, archive, both journey goldens, the demo
dataset, and the new photo corpus — closed byte-identical; the detection and
journey cores gained code but no existing output changed anywhere photo rows
are absent. Public numbers here come from committed test data unless
explicitly marked as measured on private data.

## What the phase does

A person whose phone never had Timeline on — the pilot's iPhone majority —
can now import geotagged photos and get detection from them. A person who
has both gets photos as enrichment: each usable photo becomes one more
observed point on the adventures it falls inside. Five checkpoints.

- **The synthesis core (CP1).** Photos yield positions, not visits — and
  the detector's away-span and dwell logic runs on visits. So
  `internal/detect` gained a pure stay-point pass (`synthesizeStays`): a
  running-mean cluster walk over photo-sourced fixes that emits synthetic
  visits where the camera dwelt (`STAY_RADIUS_M` 200, `STAY_MIN_MIN` 30,
  `STAY_MAX_GAP_MIN` 240 — all named in `Params.Synth` and recorded per
  run, invariant 3). The pass is scoped to fixes with `Source == "PHOTO"`;
  that scoping *is* the byte-identity guarantee — a dataset with no photo
  rows takes exactly the pre-phase-11 path, proven by the fixture, archive,
  and demo regressions running green with synthesis live. Home derivation
  gained a second evidence source: when `INFERRED_HOME` clustering yields
  zero bases, synthetic stay clusters can qualify as home — but only when
  they recur across ≥ `HOME_MIN_DAYS` (8) distinct civil days, because
  recurrence alone cannot tell a residence from a week in one hotel. The
  base-derivation literals (grid, merge radius, minimum visits, era pad)
  were lifted into `Params.Bases` on the way — an invariant-3 gap that had
  survived since phase 1. `internal/photosource` is the parser package
  (sibling of `internal/timeline`, invariant 4): EXIF/HEIF scan to domain
  fixes, Takeout sidecar pairing, and a time ladder that refuses to guess —
  a photo with only wall-clock time and no offset evidence is honestly
  unplaced rather than placed in the wrong hour. The committed corpus
  (58 fabricated files, generator committed, double-generation
  checksum-identical) pins all of it: 57 photos, 55 fixes, 2 unpositioned,
  1 sidecar-restored, one home base, and two candidates detected from
  photos alone — with the Höfn candidate's destination correctly the
  farthest place *dwelt* (Stokksnes), the bug-3 rule holding on synthetic
  visits too.
- **Intake (CP2).** HEIC metadata extraction is a bounds-checked BMFF box
  walk (`meta` → `iinf` → `iloc`, construction method 0 only) that hands
  the embedded Exif block to the *same* TIFF parser the JPEG path uses —
  pinned by a fixture asserting the HEIC and JPEG forms of the same photo
  parse identically, plus truncation and per-byte corruption sweeps.
  No pixel decoding: a HEIC photo contributes its position but no
  thumbnail, and every surface says so rather than pretending. Migration
  00010 adds `photo_records` — its own table, deliberately not a nullable
  variant of the phase-4 `photos` table, because the two share a shape but
  not a lifecycle: attached photos ride decision re-matching, records ride
  imports. Records exist only for usable photos (position and instant);
  the paired fix in `raw_positions` is named by content hash, so the
  shared observation stratum gained no photo-shaped column.
  `POST /imports/photos` streams parts one at a time — scan, cut the
  thumbnail while the bytes are in hand, discard the bytes; originals
  never touch disk. Every file gets its own verdict (fix / no position /
  no time / sidecar paired / unpaired / unsupported); there is no
  all-or-nothing rule here, unlike the Timeline path, because one
  stripped photo must not sink a camera roll. Import is synchronous;
  auto-detect runs behind the same goroutine-and-lock machinery phase 7
  built. The browser island accepts JPEG and HEIC (so an iOS picker that
  converts can't hurt), shows per-file verdicts in plain language, and
  carries the retention sentence: positions, times, thumbnails — never
  the photos.
- **Real-data measurement, and the answer it forced (CP3).** The brief's
  open question was "is photo evidence enough?" — answered on the
  maintainer's own camera roll through the real path (all figures in this
  paragraph measured on private data, recorded in
  `docs/private/PHASE-11-CP3-MEASUREMENTS.md`; they do not reproduce from
  the repository). Where photo and Timeline fixes sit sub-minute apart,
  they agree to a median of ~114 m — photo GPS is real evidence, not
  decoration. Enrichment works and is exactly bounded: the full archive
  still detects the same 66 candidates with photos added, and the one
  candidate the photos fall inside gained precisely its 13 usable photos
  as observations. But a single real month, photos-only, detects zero
  candidates — *correctly*: a camera roll is trip-heavy (38 photos at the
  destination, 7 near home in that month), so home never qualifies. The
  relaxed-parameter sweep made the failure mode vivid: loosening the
  guards until something detected turned the *destination* into "home" —
  the exact inversion `HOME_MIN_DAYS` exists to prevent, vindicated by
  measurement. The maintainer closed photo gathering on that evidence:
  photos are read for where/when and plotted; with a Timeline they
  enrich; without one, the honest product answer is the points-not-routes
  view, which stays mockup-gated at its own STOP with this measurement as
  its evidence base. The synthesis core stays — corpus-pinned, shared
  with the legacy-formats future, and its honest-zero path costs nothing.
  `/welcome` gained the photos door: a section that opens the photo
  library directly, the "never had Timeline on?" box now ends at it
  instead of at a dead end, and an image dropped into the Timeline upload
  redirects there by typed anchor.
- **Pilot response (CP4).** Three items, one of them summoned by review.
  (1) *Imported photos visible on their adventures.* Found at the CP3
  STOP: the maintainer uploaded real photos, confirmed the adventure they
  enriched, and could not see a single photo anywhere — records had no UI
  by CP2 design. The fix is a read-time span join: an adventure's photo
  records are the ones whose capture instant falls inside its span,
  computed per request and never stored, so re-detection cannot orphan
  what is never anchored. Thumbnails place against the drawn geometry via
  the existing pure placement; HEIC records list honestly without a
  thumbnail. (2) *Per-mode distance breakdown* (friend-5's ask): a pure
  `journey.ModeBreakdown` computed **outside** `Assemble` — the golden
  contract pins assembly byte-for-byte, and a display-layer figure must
  not churn it. Modes are Google's guesses, so the line is labelled
  source-asserted; an activity overlapping the window counts in full,
  never pro-rated (slicing a guessed distance pretends precision); and a
  photo-sourced journey has no activities, so the cover says "no mode
  record" — absent is never rendered as zeros. `roadbook journey` prints
  the same line: invariant 13's reproduction command. (3) *Bulk triage*
  (friend-5 stopped 10 decisions into 79 candidates; the operator was at
  5 of 66): one atomic `POST /candidates/decisions` (all-or-nothing; a
  stale id anywhere applies nothing), per-card checkboxes and a sticky
  bar over server-rendered cards, a select-all scoped to undecided
  candidates only (a bulk action must never silently re-decide curated
  rows), `?sort=score` for a confidence-ordered sweep, and focus
  advancing to the next undecided card after each decision.
- **Close (CP5).** README gains the two-sources statement (what photos
  are read for, what is kept, HEIC position-only) and the browser-path
  section acknowledges the photos door; cold-cache `make test`; this log.

## The confirm-all evolution, honestly

The one product-boundary argument of the phase, in three steps across
three STOPs — recorded because the record is the point.

1. **Gate 1:** friend-5's literal ask was auto-confirm-all with edit-later.
   That inverts the product's curation stance — the detector over-produces
   deliberately (rank, don't filter), so confirming everything puts known
   noise on the life map. Declined as a PRODUCT.md amendment; bulk triage
   chartered instead.
2. **CP4 review:** the maintainer raised confirm-all again against the
   built selection UI. Resolution: select-all as a control, scoped to
   undecided rows, feeding the existing bulk actions — "confirm all" as a
   two-gesture composition rather than a standing default. The Gate-1
   decline reaffirmed in discussion.
3. **Same review, the naming question:** what should Confirm-selected do
   with candidates that have no name suggestion? The assistant recommended
   nameless confirms (NULL name, display-side date fallback, a rename
   nudge) so machine names would stay forever distinguishable from chosen
   ones. The maintainer weighed that extra contract machinery against its
   value and chose the stored date fallback: one pass confirms the whole
   selection, suggestion-less candidates are named "Journey of
   <span-start date>" as a real stored name, and the response reports the
   suggested/date-named split. The recorded cost, accepted knowingly:
   which plates were human-named is not recoverable later. Maintainer's
   call at a STOP; DECISIONS.md carries all three entries.

## What broke, and why each fix took its form

- **Imported photos were invisible as photos.** CP2 built records as a
  foundation (§4D) with no display surface, and the gap only became
  visible when the maintainer, at the CP3 STOP, looked for his own photos
  on the adventure they had provably enriched. The fix became a named CP4
  item rather than a CP3 patch — CP3 was a measurement checkpoint, and
  the STOP should close it, not grow it. The span-join shape (computed at
  read, never stored) was chosen so records keep exactly one anchor:
  their import.
- **The corpus generator had to emit chronologically sorted shots.** The
  detector's dwell lookup bisects visit starts in file order — the same
  trap the demo generator hit in phase 5, rediscovered when synthetic
  visits entered through a second door. The merge sorts; the generator
  emits sorted anyway; the note lives in both places.
- **`GRID_PER_DEG` is a multiplication, not a division.** Lifting the
  base-derivation literals into parameters changed `/0.05` into a
  cells-per-degree multiplier — dividing by the float constant drifts,
  and byte-identity of the archive regression is what caught the naive
  lift.
- **HEIC's `iloc` has four versions and two Exif prefixes.** The box walk
  handles iloc v0/1/2 but only construction method 0 (file-offset
  extents), and accepts both `Exif\0\0` block prefixes seen in the wild.
  Anything else is treated as metadata absence — the phase-4 rule
  (malformation = absence, never an error) extended to a second container,
  and the corruption sweep pins that no input can panic the walk.
- **oapi-codegen renamed constants out from under the attach path.** The
  new photo-import status enum collided with the phase-4 upload result's
  short constant names, so the generator prefixed both sets; the attach
  handler needed a mechanical rename. Accepted as the cost of invariant
  10 — the generator owns naming, and hand-aliasing it back would be
  editing generated output in spirit.
- **The suggestion prefetch burst-fired at the public geocoder.** The
  CP4 triage page fetched name suggestions for visible candidates in
  parallel — caught frame-by-frame in the maintainer's screen recording
  of a real sweep, against Nominatim's ~1 req/s public policy. Fixes made
  the fetches sequential. The politeness rule had been honoured by the
  batch CLI since phase 2; the browser surface had to relearn it.
- **The e2e suite fought hydration twice more.** Checkbox clicks before
  island hydration silently no-op (the phase-9 `clickUntil` idiom covers
  it), and a score assertion via `textContent` matched digits run
  together across elements — the regex now targets the element whose
  `title` carries the figure. Both are instances of the standing rule:
  measure the DOM atomically, don't race React.
- **Port 8080 is two different servers depending on where you stand.**
  The local dev `roadbook serve` (real data) and the scratch compose
  api (not published to the host) both answer on 8080 — one probe during
  CP2 verification read real-data counts and briefly looked like a
  corpus regression. Operational note, recorded: probe compose services
  with `docker compose exec`, never through the host port.

## Verification at close

- Cold-cache `make test` green (test cache cleared first): all Go suites
  including the DB-backed store/api/backup harnesses against scratch
  databases; fixture regression 18 candidates / 1 base / 32 outliers;
  archive regression; both journey goldens byte-identical; demo 3/1/0;
  photo corpus pins (57 photos / 55 fixes / 2 unpositioned / 1 sidecar
  pair; 1 base; 2 candidates with their figures). Web: 63 vitest tests.
  The Playwright layout suite stood at 79 passing at the CP4 walk against
  a live scratch stack.
- The corpus end-to-end proof (CP2): 58 files through the Next proxy on a
  fresh scratch instance → 55 fixes → the two pinned candidates from
  photos alone; the confirmed one renders as observed stay-legs joined by
  honestly unknown gaps, unrouted, countries attributed.
- The real-data proofs (CP3, private): enrichment bounded and exact;
  photo-only month honestly zero; guard-relaxation inversion demonstrated;
  all recorded in `docs/private/PHASE-11-CP3-MEASUREMENTS.md`.
- Data safety: the maintainer's photos and camera-roll copies live only
  under `data/`; the corpus is fabricated at committed ocean/Reykjavík
  coordinates by a committed generator; `git add -A --dry-run` reviewed
  at every commit.

## Carried forward

- **Device walks.** The Android walk of the photos door (and the
  `/welcome` stamp upgrade) waits until the maintainer is home with a
  phone that can reach a stack; the iPhone walk (real HEIC through a real
  picker — the §1.3 conversion fact) waits on a friend. The page states
  its verification honestly meanwhile.
- **Points-not-routes view.** Mockup-gated at its own STOP, now carrying
  the CP3 measurement as evidence; the curation boundary is decided
  there, not silently.
- **Ingested-photo attachment / HEIC pixels.** The recorded trigger for
  revisiting HEIC decoding is a real want of thumbnails for ingested
  iPhone photos.
- **Takeout zip transport.** The picker and CLI directory ship; feeding a
  Takeout archive without unpacking is the recorded follow-up.
- **Bulk-confirm name provenance.** Under the date-fallback choice,
  human-named vs machine-named is not recoverable; if it starts to
  matter, the distinction needs a migration-time backfill guess —
  recorded at the decision.
- **Standing items untouched by this phase:** front gate (phase 12) and
  accounts (13) per the resequenced roadmap; the A1 capacity hunt and
  the trial-host migrate-by date live in the pilot ledger.
