# Phase 11 — Photos as an import source (design brief)

Status: draft for Gate 1 review. No code before this brief is agreed.

---

## 0. Charter record

Decided 2026-08-24 at the charter STOP, ingestion over the front gate. The roadmap's
resequencing clause ("audit returns mostly without Timeline data → ingestion moves
ahead of the front gate") fired on stronger evidence than the audit would have
produced: none of the iPhone-holding pilot users has Timeline data at all — four of
six pilot instances sit at zero imports for exactly that reason (pilot ledger,
2026-08-24). These are people who wanted in and physically could not enter. A front
gate built first would recruit strangers into a funnel the pilot proved closed to
the iPhone majority.

Consequences recorded in `docs/PLAN.md` the same day: this phase is **phase 11**;
the front gate becomes phase 12, accounts/tenancy phase 13. The finish line — a
stranger reaches their adventures with zero operator involvement — is unchanged;
only the order moved.

Binding input from the maintainer: the first non-operator pilot report (friend-5,
2026-08-24) must be considered as named scope candidates in this brief. It is — §6.

## 1. Concepts this phase introduces

### 1.1 Stay-point synthesis (visit synthesis from bare positions)

**What.** Deriving visit-shaped dwells from a stream of timestamped positions that
carries no visit segments. A *stay-point* is a maximal run of consecutive fixes
that remain within a distance radius for at least a minimum duration; it collapses
to one synthetic visit (centroid position, first and last fix time).

**Why it is the phase's core.** The detector depends on visit segments in exactly
two places, both measured in code: home derivation consumes only visits with
`SemanticType == "INFERRED_HOME"` (`internal/detect/bases.go:25`), and destination
dwell qualification consumes only parsed visits with a location and a span ≥
`MIN_DWELL` (`internal/detect/detect.go:138`). A source with no visit segments
therefore produces no home bases (detection returns empty early) and no
destinations (every away-span is discarded). Photos are such a source — and so are
`Records.json`, Semantic History `timelineObjects`, and continuous-logger GPX. One
pure pass unblocks all four; this brief designs the pass for all of them and
implements only the photo source (the sharing conclusion is already recorded in
`docs/PLAN.md`, backlog "Legacy Timeline variants" and "Photos as an import
source", and in the private launch notes).

**How.** A single O(n) forward pass holding an open stay with a running-mean
centroid: a fix within `StayRadiusM` of the centroid extends the stay; a fix
outside it closes the stay if its duration ≥ `StayMinMinutes`, else discards it
and opens a new one. A time gap above `StayMaxGapMinutes` also closes the open
stay — a burst of photos at dinner and another at breakfast must not fuse into one
sixteen-hour dwell unless the evidence supports it. This is the algorithm class
sketched in `docs/phase-2/BRIEF.md` (the Dawarich stay-point sweep) and rejected
there for phase-2 stops — that rejection was about stops-from-activities being
cheaper *for Timeline data*, not a judgment against the algorithm; for a source
with no activities it is the only option.

**Where, and under which invariants.** Inside `internal/detect`, as a pure derived
pass at detection time — parameterised (invariant 3), never persisted (invariant 2:
synthetic visits are derived output, recomputed free on every run; persisting them
would make detection corrupt its own input class). Identical inputs and parameters
yield identical synthetic visits (invariant 1). The pre-recorded design conclusions
for the legacy formats (`docs/PLAN.md`) bind here verbatim: synthesis is pure,
parameterised, never persisted; parsers emit existing domain types.

### 1.2 Home evidence with semantic precedence

**What.** Generalising `deriveBases` input from "INFERRED_HOME visits" to "home
evidence", ranked: Google's own INFERRED_HOME assertions when present, synthetic
recurrence evidence otherwise. Strict precedence — when any INFERRED_HOME visits
exist, synthetic evidence is not consulted — is what keeps fixture and archive
outputs byte-identical (the recorded design conclusion).

**The hard case, stated honestly.** People photograph trips, not home. A camera
roll may contain months of travel bursts and only scattered home photos. Synthetic
home derivation (the dominant recurring stay-point cluster across the data's span)
can fail to clear the evidence bar. That failure must be a *designed state* with
its own message — "no home could be derived from this data; here is what home
derivation needs" — not a silent empty candidate list. Whether a user-supplied
home hint should exist is deliberately deferred (§7): a hint is user-asserted
data, not derivation, and it opens an honesty question (invariant 9's spirit)
that this phase does not need to answer to ship.

**A found invariant-3 gap, fixed in passing.** The current base-derivation
thresholds — 5 km grid, 10 km cluster merge, ≥8 visits, ±45-day era — are literals
in `bases.go`, not named parameters in `detect.Params`. Generalising this code is
the moment to lift them into `Params` with defaults that reproduce today's outputs
byte-identically (the regression suite is the proof).

### 1.3 HEIC/HEIF: metadata extraction is not pixel decoding

**What.** HEIC is the iPhone default photo format (2017→). It is an ISO Base Media
File Format container (the "box" structure MP4 uses): metadata items — including
the EXIF payload — are declared in `meta`-box tables (`iinf`/`iloc`) that point at
byte ranges; the image pixels are HEVC-encoded, a codec Go does not speak.

**The distinction that shapes scope.** Extracting the EXIF payload means walking
the box structure to the Exif item and handing its bytes to the TIFF walker that
already exists (`internal/photo/exif.go`, `findEXIFTIFF`) — pure Go, no codec, the
same "malformation = absence" posture as phase 4. Decoding pixels (for
thumbnails) means HEVC — cgo against libheif or nothing. Ingestion needs
positions and timestamps, not pixels, so this phase builds metadata extraction
only and leaves pixels excluded (§7). The phase-4 HEIC trigger ("real blocked
uploads") is satisfied by this phase's own premise — iPhone camera rolls are
HEIC — which is why the work lands now and not before.

**A device fact to verify, not assume.** iOS Safari often transparently converts
HEIC to JPEG at the file-input boundary, depending on the picker path and the
`accept` attribute. If that holds on real devices, the picker path may deliver
JPEG and HEIC parsing matters mostly for Takeout archives and AirDropped
originals. CP2's design freezes only after a real-device check (§5) — the iPhone
friends can finally participate, since testing this needs no Timeline data.

### 1.4 The photo picker as transport

On a phone, `<input type="file" multiple accept="image/*">` opens the OS photo
picker directly — no export step, no Takeout wait, no 52 MB JSON. For the
zero-Timeline user this is the entire first-run story: open the link, pick trip
photos, watch candidates appear. It is also a different upload shape from
everything built so far: many files (hundreds to thousands) of moderate size,
against phase 7's one large file. The existing machinery on each side of the gap:
`POST /imports` is strictly single-file (`internal/api/imports.go`, one `file`
part); `POST /candidates/{id}/photos` is already multi-file multipart
(`readParts`, 25 MB × 50 files) but attaches to a confirmed decision rather than
importing. The transport choice is §4C.

## 2. What gets built

- **HEIC metadata extraction** in `internal/photo`: a BMFF box walk that locates
  the Exif item and reuses the existing TIFF walker; the photo sniffer's HEIC
  branch flips from rejection to acceptance; the taxonomy keeps rejecting video,
  PNG, WebP, RAW with the existing actionable messages.
- **A photo-batch parser** emitting existing domain types (invariant 4): each
  usable photo becomes one timestamped position fix (landing spot: §4A); the
  phase-4 time ladder (`ResolveTime`) and sidecar merge are reused as-is. Photos
  with no resolvable time or position are counted and reported, never guessed.
- **The synthesis pass** in the pure detect core: stay-point visit synthesis and
  home-evidence generalisation per §1.1–1.2, all thresholds named parameters,
  recorded per run, byte-identity regressions green.
- **Transport and import surface**: multi-file photo upload from the browser into
  the existing import bookkeeping (imports row, status, counts, auto-detect),
  with per-file verdicts in the taxonomy's spirit.
- **Front-door integration**: `/welcome` gains a photos walkthrough branch (the
  first-run path for zero-Timeline users), with rejection-anchor mapping extended.
- **Pilot-response items** (§6, the binding input): bulk triage actions and the
  per-mode distance breakdown, if accepted into scope at this gate.
- **Fixtures**: a committed synthetic photo-corpus generator (a trip's worth of
  geotagged photos plus home-recurrence photos) pinning the synthesis pass, plus
  table tests for the pass itself.

## 3. What this phase does *not* decide

GPX and the legacy Timeline formats stay unimplemented — the synthesis pass is
designed so their parsers can join it later with zero changes to detection
(invariant 4 makes that a structural claim, tested by the photo source being the
first consumer). Recalled adventures, the front gate, and auth are untouched.

## 4. The real choices

### A. Where photo fixes land in the domain

| option | consequence |
|---|---|
| `path_points` | Detector sees them for free (flatten already consumes points), but provenance is lost: the journey assembler would tag them `SourceTrace`, and nothing downstream could ever distinguish a photo fix from a Timeline trace point. |
| **`raw_positions` with `Source = "PHOTO"`** | The table already carries `source` and `accuracy_m`; the journey assembler already consumes raw positions and applies `MaxAccuracyM` to them; provenance survives. Requires one deliberate detector change (choice B). |
| a new table/type | New schema and a new domain type for data that is exactly "a timestamped position with a source" — invariant 4 exists to prevent this. |

**Recommendation: `raw_positions`, `Source = "PHOTO"`.** A photo fix is the same
stratum as a rawSignals fix — a high-accuracy timestamped position — and the
existing column set describes it without addition.

**Two artifacts, not one — the distinction this choice rests on** (clarified at
Gate 1 review after the maintainer raised the separate-table question). Each
photo yields (1) a **photo record** — hash, metadata, sources, thumbnail,
import — which lives in its own photo table per §4D and is what every photo
future queries; and (2) a **fix** — the bare position-at-a-time projection —
which is how the photo participates in the pipeline. Only the fix lands in
`raw_positions`, because invariant 4 demands one seam for all sources: the
detector and assembler consume positions and must never learn about photos. A
separate *fixes* table would fork every consumer (journey inputs, detection,
and each future reader) into a permanent two-table UNION while adding no
queryable fact the record does not already hold. WIFI/CELL/GPS already coexist
in this stratum discriminated by `source`; PHOTO is a fourth value, not a new
shape. Each photo-sourced row carries a reference to its record (column versus
hash mapping is CP2 schema design), so any point on any map leads back to its
photo evidence.

### B. How photos reach detection

The detector's observation stream (`flatten`) consumes visits, activity endpoints,
and path points — never raw positions (`detect.go:232`). Two channels are
possible, and they are not exclusive:

1. **Via synthesis only**: the synthesis pass consumes photo-sourced raw
   positions, emits synthetic visits, and those visits join the stream like any
   visit. Zero change to `flatten`. But in-transit photos — taken moving, never
   part of a stay — would be invisible to span detection, and away-span
   boundaries would be only as precise as the stays.
2. **Also as observations**: `flatten` additionally includes raw positions whose
   source is `PHOTO`, behind a named parameter.

**Recommendation: both, with the flatten inclusion scoped to the photo source
class and default on.** The scoping is what makes the default safe: existing
data has no `Source = "PHOTO"` rows, so every existing run is byte-identical by
construction, while Timeline `rawSignals` (WIFI/CELL/GPS) stay out of detection
exactly as today. The parameter names an evidence class, not a user (invariant 9
holds), and is recorded per run (invariant 3).

### C. Transport shape, and how much of it now

Three candidate intake paths:

1. **Picker multi-file** — `<input multiple>` → streaming multipart → per-file
   metadata extraction. The iPhone first-run path; no export step. The all-or-
   nothing rule of phase 7 cannot hold at 800 files — per-file verdicts with an
   import-level summary is the honest shape, and the phase-4 photo endpoint
   already established that pattern.
2. **Google Photos Takeout zip** — carries the JSON sidecars, which are the only
   position source for photos whose EXIF was stripped (the WhatsApp/messenger
   case phase 4 documented) and the only path for Google Photos users whose
   originals live server-side. Zip walking is new machinery.
3. **A server-side directory** (operator drops files, CLI imports) — trivial to
   build, useful for self-hosters with big archives, no browser limits.

**Recommendation: 1 and 3 in this phase; 2 as a named follow-up unless the
checkpoint budget proves roomy.** The picker is the path the evidence demands
(the iPhone friends), and the CLI directory path costs almost nothing since the
parser is the same. Takeout zip adds container-walking work that serves users who
mostly *also* have Timeline data — real, but not this phase's blocked cohort.
What survives ingestion — originals, records, or fixes alone — is its own
choice: §4D.

### D. What survives ingestion (amended at Gate 1 review, 2026-08-24)

The first draft recommended extracting fixes and discarding everything else.
The maintainer's review input (§6.3) changed the recommendation: three named
photo futures all require the photo to survive as a referenceable thing at a
point, and "fixes only" forecloses every one of them — a user would have to
re-upload their entire roll when any of those futures lands.

| option | consequence |
|---|---|
| fixes only | Cheapest and most private, but the photo's identity is destroyed at ingestion; proof-of-location, any future photo-at-a-point view, and the attachment follow-up all require a full re-upload. |
| **photo records** | Per usable photo: content hash, extracted metadata (position, resolved time, sources), and a thumbnail where decodable — JPEG through the existing phase-4 pipeline; HEIC gets a metadata-only record until a codec exists, stated honestly. Originals still discarded. ~50–80 KB per thumbnail ⇒ a 3,000-photo roll costs roughly 150–250 MB — acceptable on the host, and stated in the README. |
| full originals | Turns the instance into a photo host — the storage problem phase 4 rejected, and the worst privacy posture. |

**Recommendation: photo records.** The fix each photo yields carries the record's
identity, so every point on a map can lead back to its photo evidence — the seam
scenarios 1–3 in §6.3 share — without hosting originals.

**Named schema question for CP2 design** (not decided here): today's `photos`
table is decision-scoped (`FK decisions ON DELETE RESTRICT`, photos ride decision
re-matching). Ingested photo records are import-scoped and exist before any
candidate does. A separate table versus a relaxed FK is a real design choice with
lifecycle consequences either way; CP2's design settles it, in the decision log.

### E. What "defaults" mean for auto-detect

Upload-triggered auto-detect runs `detect.DefaultParams` today. Synthesis
parameters join `Params` with defaults that (a) leave Timeline-only data
byte-identical (guaranteed by B's source scoping) and (b) make photo-only data
work without operator tuning. The defaults are the brief's proposed starting
values, expected to be tuned against the synthetic corpus and Allen's real
camera roll, with every run recording what produced it:

| parameter | proposed default | rationale |
|---|---|---|
| `StayRadiusM` | 200 | above photo-GPS scatter at a venue, below "different place" |
| `StayMinMinutes` | 30 | shorter than MIN_DWELL: a synthetic visit must still clear `MinDwellMin` to be a destination — synthesis finds stays, detection judges them |
| `StayMaxGapMinutes` | 240 | photos are bursty; a gap longer than this closes the stay rather than bridging an unknown |
| base-derivation set (§1.2) | today's literals | lifted into `Params`, byte-identical defaults |

## 5. Verification plan

- **Committed synthetic corpus**: extend `testdata/photos/gen` with a corpus
  generator — a fictional persona's camera roll (home-recurrence photos across
  months, one dense trip burst, one sparse trip, in-transit shots, a no-GPS
  batch, a HEIC subset) with all coordinates fabricated. A pinned expectation
  file (N candidates, home base, per-photo verdicts) becomes the ungated
  regression, the same pattern as the demo dataset.
- **Table tests** for the synthesis pass: radius/duration/gap boundaries, burst
  fusing and splitting, offset handling (synthetic visits get civil offsets via
  the phase-4 time ladder's resolved offsets), the home-precedence rule.
- **Byte-identity regressions**: fixture 18/1/32, archive counts, both journey
  goldens, demo 3/1/0 — all must pass unchanged with synthesis parameters at
  defaults. This is the "strict precedence" claim made testable.
- **Real data** (private, never in git): Allen's own camera roll / Takeout into
  a scratch instance; the interesting question is what photo-only detection
  finds versus the Timeline-derived truth for the same period — the first
  measured answer to "is photo evidence enough".
- **Device verification**: the picker path walked on a real iPhone (a pilot
  friend — no Timeline needed, so they can finally participate) and on Allen's
  Android. The iOS HEIC-conversion fact (§1.3) is checked here, before CP2's
  transport design freezes.

## 6. Named scope inputs

### 6.1 Bulk triage actions

**The problem, quantified.** Friend-5 — the loop's first non-operator completion —
stopped deciding at 10 of 79 candidates and called one-by-one confirmation "a
great hassle"; the operator's own instance sits at 5 of 66. Triage friction at
real-archive scale is no longer anticipated, it is measured. Photo ingestion
makes it worse by design: this phase mints candidate lists for exactly the users
it admits.

**The recorded solution proposal, argued against.** Friend-5's ask —
auto-confirm everything, edit later — is a PRODUCT.md amendment, not a UI fix:
the detector over-produces deliberately (rank, don't filter — the detection
reference's rule 6), so auto-confirmation would place known noise on the life
map and invert the curation stance that defines the product. The recommendation
is to decline the amendment and treat the friction as a workflow problem.
The maintainer decides at this gate.

**Proposed in-system scope** (from the PLAN backlog entry, now concrete):
multi-select with confirm/dismiss on the selection; a quick-confirm affordance
on the gallery card that accepts the suggested name without opening anything;
and a score-ordered sweep flow — the triage list ordered by score with
keyboard-pace decisions, built for "clear 60 candidates in one sitting".
Recommended: **in scope, as its own checkpoint** — it is the product's answer to
the first real user report, and this phase's new users hit the wall immediately.

### 6.2 Per-mode distance breakdown

Air / road / rail / water distance per journey, computable today from activities
(`DistanceM` + `Mode`). Two honesty constraints are binding: source modes are
guesses with recorded failures (the 1,023 km "motorcycling" relocation;
probability 0.00), and the pipeline itself trusts speed over mode for flight
classification — so the breakdown is labelled **source-asserted**, visually
subordinate to measured geometry, and never summed against routed/observed
figures as if commensurable. Photo-sourced journeys have no activities and
therefore no breakdown — the display states that rather than showing zeros.
Recommended: **in scope, inside the same pilot-response checkpoint** — it is
small, fits the day-narrative/stats seam, and answers the report's second item.

### 6.3 Maintainer input at Gate 1 review (2026-08-24): three photo futures

Raised by the maintainer while reviewing this brief, sorted per the standing
buckets (in-system / mockup-gated new IA / charter-level). One of them changed a
recommendation (§4D); none of them adds implementation scope to this phase.

**(1) Photos as proof of location — in-system; it is this phase's foundation.**
Phase 4 established the principle ("a positioned photo is a measurement") and
built route validation on it. The ingestion design preserves it twice: every fix
carries `Source = "PHOTO"` (§4A), and — after the §4D amendment — every fix
leads back to a photo record, so "prove I was here" evidence survives ingestion
rather than dissolving into anonymous coordinates.

**(2) Road-condition evidence pinned to the map — charter-parked; no design
here.** Sharing road-condition experience so other travellers can check a route
is the parked community direction verbatim (`docs/private/
parked-road-community.md`), re-read and kept parked by the maintainer
2026-08-18; its own trigger conditions include an explicit charter change, and
it must not be absorbed silently. This phase builds nothing for it. The one
honest connection: that document's own sequencing note says verified journeys
and their evidence are the trust layer such a community would someday need —
and the §4D photo record preserves exactly that evidence class at zero
additional cost. Preserving evidence is not planning the feature.

**(3) A points-not-routes view — honest, and mockup-gated new IA.** Showing
only observed points is *more* conservative than drawing routes, so invariants
5 and 8 are untroubled. Per-adventure, photos-on-the-map already exists (phase
4). A dedicated view of "the points I have been, with their photos" is a new IA
element, which per the phase-9 discipline requires Stage-B mockup treatment at
a STOP before any code — and it brushes the curation boundary (all photo points
versus confirmed-adventures-only is coverage-adjacent territory, decided
deliberately, never drifted into). Future phase; this phase's contribution is
the §4D record that makes such a view possible without re-upload.

## 7. Exclusions, each with its reason

- **HEIC pixel decoding / thumbnails** — needs a codec Go lacks; ingestion needs
  positions, not pixels. Trigger to revisit: ingested-photo attachment (below).
- **Auto-attaching ingested photos to adventures** — users who upload a camera
  roll may expect their photos to appear on the resulting adventures. That is a
  real product question (attachment today is per-confirmed-decision, with
  thumbnails), and answering it well needs the attachment flow to meet the
  ingestion flow deliberately, not as a side effect. Named follow-up; the §4D
  photo record (fix → record identity) is the seam kept clean for it. Expected
  to fire.
- **A points-not-routes photo view** — §6.3(3); new IA, Stage-B mockup
  treatment at its own STOP, curation boundary decided there. The §4D record
  keeps it buildable without re-upload.
- **Road-condition anything** — §6.3(2); the parked community direction, behind
  its own charter change. Nothing here designs for it beyond not destroying
  photo evidence, which §4D does for this phase's own reasons.
- **Takeout zip intake** — §4C; follow-up unless the phase runs ahead of budget.
- **GPX, legacy Timeline parsers** — the synthesis pass is designed for them;
  their parsers are separate later work with the trigger already recorded.
- **User-supplied home hint** — §1.2; designed failure state ships first,
  evidence decides whether a hint is ever needed.
- **Video** — unchanged from phase 4.
- **Front gate, waitlist, auth** — phases 12/13.

## 8. Checkpoints

1. **The synthesis core.** Stay-point pass + home evidence + parameter lift in
   pure `internal/detect`; corpus generator and pinned expectations; all
   byte-identity regressions green. *Visible: `roadbook detect` over the
   committed photo corpus prints the pinned candidates; the full existing test
   suite passes untouched.*
2. **Intake.** HEIC metadata extraction; photo-batch parser to domain types;
   photo records per §4D (the schema choice made and logged here); multi-file
   transport with per-file verdicts; imports surface shows a photo import with
   honest counts; CLI directory path. *Visible: a folder of mixed
   photos (JPEG, HEIC, one PNG, one no-GPS) imports through the browser with
   per-file verdicts, and auto-detect produces candidates from photos alone on
   a scratch instance.*
3. **End to end, real data.** Allen's camera roll through the real path on a
   scratch instance — candidates, confirmation, life map; the photo-vs-Timeline
   comparison measured; `/welcome` photos branch; iPhone + Android device walk.
   *Visible: an adventure detected purely from photos, on the life map, on a
   phone.*
4. **Pilot response.** Bulk triage actions and per-mode breakdown per §6, if
   accepted at this gate. *Visible: 60 candidates cleared in one sitting on a
   real instance; a source-labelled mode breakdown on an adventure page.*
5. **Close.** README supported-sources statement, docs, cold `make test`,
   LOG.md. *A phase is not complete until its log exists.*

Order note: CP4 depends on nothing in CP1–3 and can be pulled earlier if the
pilot needs it; the brief sequences it after intake so the triage work can be
exercised against a photo-minted candidate list.

## 9. Review questions for Gate 1

1. Choice A/B (domain landing + detector feed) — accept the recommendations?
2. Choice C — picker + CLI directory now, Takeout zip follow-up?
3. Choice D — photo records (hash + metadata + thumbnail where decodable,
   originals discarded), amended per the maintainer's three-futures input?
4. §6.1 — bulk triage in scope as CP4, and the auto-confirm PRODUCT amendment
   declined?
5. §6.2 — per-mode breakdown in CP4, or deferred?
6. The proposed synthesis defaults (§4E) — starting values acceptable, tuned
   against the corpus during CP1?
7. Phase numbering and the PLAN.md resequencing edits — as prepared?
8. §6.3 sorting of the three photo futures (in-system foundation /
   charter-parked / mockup-gated) — agreed as recorded?
