# Phase 11 — decision log

Three lines each: what was chosen, what was rejected, what would change our mind.
Written as decisions are made, not reconstructed.

## 2026-08-24 — Charter: ingestion ahead of the front gate

- **Chosen:** photos-as-import-source as phase 11; front gate shifts to 12,
  accounts/tenancy to 13. Decided by the maintainer at the charter STOP after
  hearing the trade-offs.
- **Rejected:** building the front gate first — it would recruit strangers into a
  funnel the pilot proved closed to the iPhone majority (four of six pilot
  instances at zero imports because no Timeline data exists to export).
- **Would change our mind:** nothing retroactively; the roadmap's own
  resequencing clause pre-decided this and fired on pilot evidence stronger than
  the planned audit.

## 2026-08-24 — Binding scope input

- **Chosen:** friend-5's two report items enter the brief as named scope
  candidates (§6): bulk triage actions and per-mode distance breakdown.
- **Rejected:** treating the report as backlog-only; the maintainer's explicit
  instruction was that this brief must argue them.
- **Would change our mind:** n/a — instruction, not inference.

## 2026-08-25 — CP2: photo_records is its own table

- **Chosen:** a separate `photo_records` table (migration 00010), records
  created only for usable photos (position AND instant); the thumbnail
  directory is shared with attachment photos (both name files by content
  hash, so the same photo attached and ingested stores one file).
- **Rejected:** relaxing `photos.decision_id` to nullable — the two tables
  share a shape but not a lifecycle (photos ride decision re-matching,
  records ride imports), and a nullable FK would make every photos query
  carry both meanings; also rejected: rows for unusable photos (per-file
  verdicts already report them; retained rows would be storage with no
  product use).
- **Would change our mind:** a feature needing to remember unusable uploads
  across sessions (none is recorded).

## 2026-08-25 — CP2: record→fix join via stored fix hash

- **Chosen:** `photo_records.fix_content_hash` names the paired
  `raw_positions` row (content-hash unique) — the observation stratum gains
  no column.
- **Rejected:** a nullable `photo_record_id` on `raw_positions` — a
  photo-shaped column on the shared stratum, migration on the largest
  table, for a join the hash already provides.
- **Would change our mind:** measured join cost at real scale (the hash
  index exists; none expected at tens of thousands of rows).

## 2026-08-25 — CP2: per-file verdicts, no all-or-nothing, no request 413

- **Chosen:** the photo batch endpoint gives every file its own verdict;
  an oversized or over-count file gets a per-file verdict and the rest of
  the batch lands. Transport failure mid-stream still records nothing
  (no row exists until the stream completes).
- **Rejected:** the Timeline path's all-or-nothing rule (one bad file must
  not sink 800 good ones) and a request-level 413.
- **Would change our mind:** evidence of users misreading partial success —
  then the summary copy is the remedy, not the semantics.

## 2026-08-25 — CP2: HEIC accepted for ingestion, still rejected for
## attachment; AVIF stays rejected

- **Chosen:** the sniffer accepts the HEIC brand family as a third kind;
  ingestion extracts metadata (BMFF box walk into the existing TIFF
  parser); the phase-4 attachment endpoint rejects HEIC with its own
  message (attachment requires a thumbnail; no codec). AVIF remains
  rejected as a web re-encode.
- **Rejected:** cgo/libheif for pixel decode (excluded by the brief);
  accepting AVIF (not a camera capture format).
- **Would change our mind:** the ingested-photo attachment follow-up firing
  — that is the recorded trigger for revisiting HEIC pixels.

## 2026-08-24 — Gate 1 review input: three photo futures (maintainer)

- **Chosen:** the maintainer's three raised scenarios sorted per the standing
  buckets — proof-of-location is in-system and foundational (§6.3.1);
  road-condition evidence is the parked community direction, no design (§6.3.2);
  a points-not-routes view is mockup-gated new IA for a future phase (§6.3.3).
  One recommendation amended: ingestion keeps **photo records** (hash +
  metadata + thumbnail where decodable), not bare fixes (§4D) — all three
  futures need the photo to survive as a referenceable thing at a point.
- **Rejected:** fixes-only retention (forecloses all three futures, forcing a
  full re-upload later); full-original retention (the photo-host storage and
  privacy posture phase 4 already rejected); designing anything for the parked
  direction beyond not destroying evidence.
- **Would change our mind:** on records — storage evidence that thumbnails at
  camera-roll scale strain real instances (then metadata-only records keep the
  seam at near-zero cost); on the parked direction — only its own charter
  change, never this phase.

## 2026-08-24 — CP1: home-evidence precedence is fallback-on-failure

- **Chosen:** synthetic home evidence is consulted when INFERRED_HOME
  *derivation yields zero bases*, not merely when zero INFERRED_HOME visits
  exist. A user with a handful of Timeline home visits (below MinVisits) plus
  a camera roll still gets a home.
- **Rejected:** presence-based precedence — it would strand exactly the mixed
  thin-Timeline case this phase serves, for no honesty gain.
- **Would change our mind:** a real dataset where weak INFERRED_HOME evidence
  should have beaten strong synthetic evidence. Byte-identity is unaffected
  either way: without photo fixes there is no synthetic evidence to fall to.

## 2026-08-24 — CP1: HomeMinDays day-spread guard on synthetic bases

- **Chosen:** a synthetic cluster must recur across ≥ HOME_MIN_DAYS (8)
  distinct civil days, on top of the MinVisits count — recurrence alone
  cannot tell a residence from a week's hotel.
- **Rejected:** count-only qualification (a 12-stay hotel week would become a
  "home" and erase its own trip); applying the guard to INFERRED_HOME
  evidence too (byte-identity — Google already asserted home there).
- **Would change our mind:** corpus or real-data evidence of a true home
  lost to the guard (a weekend-only residence, e.g.); the parameter is the
  remedy, not code.

## 2026-08-24 — CP1 deviation from brief §4B: no on/off parameter for
## photo-fix observation inclusion

- **Chosen:** photo-sourced fixes always join the observation stream —
  inclusion is scoped by the source class itself, with no boolean parameter.
  The brief promised "behind a named parameter, default on".
- **Rejected:** the boolean — a dead knob nobody would turn off; invariant 3
  governs thresholds, and this is a structural rule made safe by scoping
  (data without PHOTO rows is byte-identical by construction, proven by the
  fixture/archive/demo regressions running green with synthesis live).
- **Would change our mind:** evidence that direct photo observations harm
  detection somewhere synthesis alone would not — then the knob earns its
  existence. Flagged for the maintainer at the CP1 STOP.

## 2026-08-24 — Gate 1 PASSED: brief accepted as amended

- **Chosen:** all §9 recommendations stand — A/B (photo records in their own
  table, fixes in `raw_positions` `Source="PHOTO"` with a back-reference to the
  record; detection fed via synthesis plus photo-scoped flatten inclusion),
  C (picker + CLI directory now, Takeout zip follow-up), D (photo records:
  hash + metadata + thumbnail where decodable, originals discarded), bulk
  triage in scope as CP4 with the auto-confirm PRODUCT amendment declined,
  per-mode breakdown in CP4, synthesis defaults as starting values, PLAN
  resequencing as prepared, §6.3 three-futures sorting as recorded.
- **Rejected (the maintainer's one raised concern, resolved in discussion):**
  a separate table for photo *fixes* — it would fork every pipeline consumer
  into a permanent two-table UNION while adding no queryable fact the photo
  record doesn't hold; invariant 4's one-seam rule is the deciding reason.
  The separate-table instinct is satisfied by the §4D photo-record table.
- **Would change our mind:** a real consumer that needs fix-level photo
  attributes the record link cannot serve — none is known or foreseen.

## 2026-08-26 — CP3: home-photos copy softened; points-not-routes recorded, not built

- **Chosen:** the /welcome photos walkthrough carries one honest line for
  photo-only users — adventures need a spread of photos including ordinary
  days near home, plus a plain statement of what the server keeps (positions,
  times, thumbnails; never the photos) — no prominent callout. The
  maintainer's "just show points on a map" direction for no-Timeline users is
  recorded as strengthened input to the §6.3 points-not-routes future, which
  keeps its own mockup STOP (after this phase's close or at the next charter);
  nothing of it is built in CP3.
- **Rejected:** silence on home photos (a trip-only uploader would get a
  silent zero-candidate result with no explanation — measured on real data
  2026-08-26: 13 trip photos, all usable fixes, 0 candidates, correctly);
  full-emphasis instruction (the real user cost is privacy, not effort —
  select-all is one gesture); pulling the points view into phase 11 (new IA
  re-opening the curation boundary belongs behind mockups at its own STOP,
  per the Gate-1 three-futures sorting).
- **Would change our mind:** pilot evidence that the soft mention is missed
  and photo-only users still hit silent zeros — then the copy hardens; or the
  points-view STOP resolving the curation boundary — then the walkthrough can
  promise something for trip-only uploads.

## 2026-08-26 — CP3: real-data measurement closed; photo-only detection is not
## the photo story

- **Chosen:** stop real-photo gathering at one month — the brief's "is photo
  evidence enough" question is answered structurally, not left half-measured:
  a real camera roll is trip-heavy (38 destination photos vs 7 near-home in
  the same month), photo-only detection needs 2–3 months of home recurrence
  to derive a base, and a *selective* upload — what a real user will actually
  provide — is more trip-heavy still. The maintainer's direction: photos are
  read for where/when, then plotted; with a Timeline they enrich adventures
  (measured working, +13 obs on the Oct-2 candidate); without one, the
  answer is the §6.3 points-not-routes view at its own mockup STOP, now
  carrying this measurement as its evidence base. The synthesis core stays
  (corpus-pinned; shared with the legacy-formats future; its guards proved
  right — the relaxed-parameter sweep turned the trip destination into
  "home", the exact inversion HOME_MIN_DAYS exists to prevent).
- **Rejected:** gathering Jun+Jul to force a photo-only success (confirms an
  extrapolation, changes no decision); loosening synthesis defaults to make
  one month "work" (measured: it inverts home and destination); removing the
  synthesis core (it serves the multi-month and legacy-format cases and the
  honest-zero path costs nothing).
- **Would change our mind:** a pilot user whose roll genuinely carries home
  recurrence (photo-only bases then derive at defaults and the positive path
  deserves a real-data pin); or the points-view STOP deciding against a
  points surface — then the photo-only story needs a different answer.

## 2026-08-26 — CP3 review finding: imported photos are invisible as photos;
## records→adventures display added to CP4

- **Chosen:** a new named CP4 item, beside bulk triage and the per-mode
  breakdown: photo records whose fix falls inside a confirmed adventure's
  span appear on that adventure's page and map like attached photos —
  thumbnails placed against the drawn geometry via the existing pure
  placement, HEIC records listed honestly without a thumbnail. Found at the
  CP3 STOP: the maintainer uploaded real photos, confirmed the adventure
  they enriched, and could not see a single photo anywhere — records had no
  endpoint and no UI by CP2 design (§4D foundation), and the fixes are
  visible only as route geometry. This is the §6.3 proof-of-location future
  arriving as in-system scope, on the evidence the brief said would summon it.
- **Rejected:** building it immediately as a CP3.5 (CP3 is a measurement
  checkpoint; the STOP should close it, not grow it); backlogging to the
  points-view STOP (that STOP owns the no-Timeline points *surface* — this
  is display of records on adventures that already exist, blocked on nothing).
- **Would change our mind:** the CP4 design hitting a real privacy or
  identity question (records ride imports, not decisions — deletion and
  re-detection semantics must be settled in the CP4 work, and if they turn
  out deep, the item gets its own brief section rather than a quiet corner).

## 2026-08-26 — CP4 item 1: records join adventures at read time, by span

- **Chosen:** an adventure's photo-record list is computed per request — the
  records whose capture instant falls inside the candidate's span — and never
  stored against the candidate. Records keep riding imports (00010's shape);
  re-detection cannot orphan what is never anchored; deletion semantics stay
  the import's problem; no migration.
- **Rejected:** anchoring records to decisions like attached photos (double
  bookkeeping and a re-attachment machine for rows whose identity is already
  durable via content hash + import); folding records into the attached
  PhotoList response (attached photos are deletable and decision-anchored —
  one schema for two capability sets confuses the contract; the UI can merge
  visually without the API lying).
- **Would change our mind:** a real need to *exclude* one imported photo from
  one adventure (span-join has no per-adventure hide) — that would force a
  stored relation, and it should arrive as its own decision.

## 2026-08-26 — CP4 items 2+3 shapes: mode line placement; bulk triage anatomy

- **Chosen:** (2) mode breakdown computed OUTSIDE Assemble (golden contract
  pins assembly byte-for-byte) as a pure `journey.ModeBreakdown`; overlap
  counts in full, never pro-rated (slicing a guessed distance pretends
  precision); absent ≠ empty — a photo-sourced journey has no activities and
  the cover says "no mode record", never zeros; `roadbook journey` prints the
  same line (invariant 13's reproduction command). (3) one atomic bulk
  endpoint POST /candidates/decisions (all-or-nothing, per-item semantics =
  the single endpoint's); selection = module-level external store bridging
  per-card checkbox islands and a sticky bottom bar (cards stay server HTML —
  66 cards of leg geometry must not ship as client props); Dismiss-selected
  is the primary bulk gesture; Confirm-selected rides name suggestions and
  reports the unnamed rest for individual naming — a name is never invented;
  sweep order = ?sort=score search param (URL state, server-sorted);
  keyboard pace = focus advances to the next undecided card after each
  decision.
- **Rejected:** mode figures inside the pure assembly (golden churn for a
  display-layer claim); pro-rating activity distance by window overlap;
  client-side gallery (geometry payload); auto-naming bulk confirms ("Journey
  of <date>" on an atlas cover the user never chose — the quick-confirm
  stance, applied at selection scale); a modal sweep mode (sort + focus
  advance reach keyboard pace without new UI state machinery).
- **Would change our mind:** pilot evidence that suggestion-less instances
  (no geocoder — the default) make Confirm-selected useless in practice —
  the fallback shapes are batch-naming UI or a geocoder-on default, each its
  own decision; or a sweep session showing focus advance is not enough pace
  and a true modal flow earns its complexity.
