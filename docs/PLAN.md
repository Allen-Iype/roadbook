# Roadbook — phase plan

Vertical slices. Each phase ends with something visible. **Do not reorder into
horizontal layers** — an earlier plan front-loaded 42 hours of invisible backend work,
and that was its central flaw. A slice that produces nothing to look at cannot be
evaluated.

Architecture and invariants are in `CLAUDE.md` and are not restated here. The
reference detector is `prototype/detect_fixture.py`; run from the repository root, it
prints the expected candidates for the development fixture in `data/`, and its output
is the regression target for the Go port. Specific counts are not stated in this
document because they derive from a private dataset; the committed, reproducible
expectations live in `testdata/`.

Each phase follows the documentation requirement in `CLAUDE.md`: a design brief before
code, a decision log as decisions are made, a phase log at the end. A phase is not
complete until its log exists.

Several items below were adopted from a comparative analysis of Dawarich, a mature
coverage-oriented timeline product — see
`docs/feature-comparison-dawarich-roadbook.md` for each item's origin, the features
rejected on principle, and one parked direction that must not enter this plan without
a charter change (its §8.2).

One seam for all sources: any input format parses into domain types and joins the
identical pipeline (CLAUDE.md invariant 4). Detection, confirmation, and routing never
learn where an observation came from. This is also how a 15-format product keeps its
pipeline sane — the pattern is field-tested.

---

## Phase 1 — Candidates on screen

**Goal.** Import the fixture, detect candidates, store them, list them, decide on each
one, and have those decisions survive re-import.

**Features**

- Import a Timeline export from a local path, with a configurable date window
- Detect adventure candidates per the detection rule (reference: the prototype; the
  rule is specified in the phase 1 design brief)
- Persist observations, candidates, and decisions
- Serve candidates and accept decisions over HTTP
- List candidates ranked, showing date, duration, distance, stop count, repeat count
- Confirm a candidate as an adventure with a name, or dismiss it
- Re-import and re-detection preserve every decision already made
- Import hardening: recognise the known non-importable inputs (legacy Takeout
  variants, encrypted Timeline backups, "My Activity" exports, KML, binary formats,
  truncated JSON) and reject each with a specific, actionable message; parse as a
  stream so file size never dictates memory

**How**

Go owns the pipeline and the data: `internal/timeline` parses, `internal/detect`
implements the detection rule as a pure function, `internal/store` persists via `pgx`.
Migrations with `goose`. A CLI entry point drives import and detection; an HTTP
service exposes candidates and accepts decisions. Detection is tested table-driven
against the reference detector's output — the development fixture (single home base),
and the multi-home case from the full private archive. The fixture has one static
home, so testing only against it leaves the hardest part of the algorithm
unexercised.

`openapi.yaml` is written by hand and is the contract. Go server interfaces come from
`oapi-codegen`, the TypeScript client from `openapi-typescript`. Generated files are
never edited.

The frontend calls that API — reads in server components, writes through server
actions — and renders the list with optimistic update so a decision responds
immediately. It has no database connection.

**Decisions and their motivation**

*Candidates and decisions are separate tables.* Candidates are derived and disposable
— change a parameter and they are all regenerated. Decisions are user data and must
never be regenerated. Putting a `dismissed` flag on the candidate row would destroy
the user's triage every time a threshold moved, which is guaranteed to happen given
that parameters are meant to be tuned (CLAUDE.md invariant 3).

*Decisions need a stable candidate identity.* A row id will not survive re-detection,
and neither will an exact time span, because parameters shift boundaries. Identity
must be derived from something stable — the destination region and the approximate
start date are the obvious basis. Design this deliberately; getting it wrong means
re-triaging the whole list after every parameter change, which is the specific failure
this phase exists to prevent.

*Detection is pure and parameters are stored per run.* Both follow from CLAUDE.md
invariants 1 and 3. The consequence worth stating: two runs with different parameters
can be compared, and any candidate can be explained by the parameters that produced
it.

*Resolve the boundary-truncation question here.* An import window truncates whatever
journey was in progress at its edge. Walk backwards to the previous home visit, or
surface the candidate as incomplete — a real product decision that cannot be deferred,
because it affects the schema.

**Concepts introduced**

Go: package layout, a pure core package, table-driven tests, `goose` migrations, `pgx`
without an ORM, an HTTP service with generated interfaces. API design: writing an
OpenAPI spec by hand, code generation on both sides, resource shapes, error semantics,
status codes. Postgres: schema design, idempotent upsert, why the derived/user-data
split matters. Next.js and React: the server/client component boundary, server
actions, fetching from an external service, loading and error states, keyed lists,
optimistic updates, controlled inputs. TypeScript: `strict` mode, discriminated unions
for decision state, consuming a generated client.

**Build order — four checkpoints, each one visible**

1. **Go CLI** — parse, detect, print. *Checkpoint: output matches the reference
   detector on the fixture — same candidates, same home count, same outliers dropped —
   and the multi-home case from the full archive is handled.*
2. **Postgres and the API** — schema, store, `openapi.yaml`, HTTP handlers.
   *Checkpoint: `curl` returns the same candidates, and re-running detection with
   different parameters leaves decisions intact.*
3. **Frontend, read-only** — generated client, server component, the list.
   *Checkpoint: the full candidate list rendered in a browser.*
4. **Frontend, decide** — confirm with a name, dismiss, optimistic update.
   *Checkpoint: decisions persist across a reload and a re-import.*
5. **Import hardening** — source-detection rules and the failure taxonomy, streaming
   parse. *Checkpoint: every known wrong input is rejected with a message that says
   what it is and what to do instead; the supported variant still imports
   byte-identically.*

Do not start step 3 before step 2 returns real data over HTTP. The extra layer this
architecture adds is worth paying for only if it is proven working before anything is
built on top of it.

Deliberately deferred out of this phase: candidate confidence scoring (live with the
plain ranked list first — see phase 2), PostGIS (enters with phase 2's countries
table; leg segmentation stays in pure Go), and parsing the legacy Takeout variants
(backlog, behind an evidence trigger — they serve other users' old archives and need
visit synthesis, not just a parser).

**Done when.** Every candidate the reference detector finds in the fixture appears on
a page. Each can be confirmed with a name or dismissed. Re-running import and
detection with different parameters leaves every decision intact.

---

## Phase 2 — One adventure on a map

**Goal.** Open a confirmed adventure and see the route, with gaps visibly marked as
unknown.

**Features**

- An adventure detail page
- The journey rendered as ordered legs
- Observed legs drawn distinctly from gaps
- Distance, duration, and stop list
- A provenance line stating what fraction was observed
- The leg model carries a gap `kind` (`unknown | road | air`) from the first design —
  phase 3 classifies and consumes it, but retrofitting the field after the renderer
  exists means reworking both
- Ingest `rawSignals` positions (the export's only high-accuracy data; currently
  parsed past) so recent adventures draw from dense points
- Countries crossed per adventure, from a bundled polygon table queried locally —
  PostGIS enters the stack here, for point-in-polygon, never as the leg-segmentation
  engine (that stays in the pure Go core, tested against the golden fixture)
- Candidate confidence scoring from named weighted components, the per-component
  breakdown stored and shown in the confirm UI — decision support, never
  auto-confirmation
- Name suggestion at confirm time: a one-shot geocoder lookup behind the same
  pluggable-with-offline-fallback seam as routing
- Anomaly-filter extensions beyond the speed-spike test (exact 0,0; accuracy
  thresholds once rawSignals carries them), as named parameters, flagging never
  deleting

**How**

Go assembles a journey from its activities and points into an ordered list of legs,
each tagged observed or gap (CLAUDE.md invariant 5). Straight lines only in this phase
— no routing yet. The frontend wraps MapLibre, which is imperative and wants to own
its DOM node, inside React, which wants to own rendering. That conflict is the
substance of this phase. The committed golden fixture
(`testdata/journey-27jul2026.anon.json` and its `.expected.json`) pins the leg
assembly: reproduce its expected values exactly.

**Decisions and their motivation**

*Straight lines first, deliberately.* Rendering the raw truth before adding inference
means the improvement in phase 3 is measurable rather than assumed, and it forces the
leg model to be correct before routing can paper over its flaws.

*Gaps are rendered differently from the first commit*, not added later as polish.
Retrofitting honesty into a renderer that already draws one confident line means
rewriting it.

**Concepts introduced**

React: refs, effect lifecycle and cleanup, and the general pattern for wrapping an
imperative library — the hardest React in this project. MapLibre: GeoJSON sources,
layers, paint properties, synchronising component state to map state without fighting
the render cycle.

**Done when.** A recognisable route appears, and the parts the system does not know
are visibly marked as unknown.

---

## Phase 3 — Routing fills the gaps

**Goal.** Gaps become road-following inference, clearly labelled as inference.

**Features**

- A `Router` interface with self-hosted OSRM, public-endpoint, and null
  implementations
- Gap legs routed in batch, results cached
- Routed totals validated against Google's own distance figure, with divergence
  flagged
- Routed gaps rendered distinctly from both observed legs and unrouted gaps
- Air-leg classification: a gap whose implied speed exceeds a named threshold gets
  `kind = air`, renders as a great-circle arc (a third visual class), and is excluded
  from road-distance validation — without this, any adventure containing a flight
  fails validation systematically
- A stated expectation, not an error path: OSM coverage is patchy exactly where
  adventures happen (rural and mountain roads). A gap the router cannot fill stays
  `unknown` and renders as visibly unknown — that is the designed behaviour for an
  incomplete road network, and a failed distance validation there is the expected
  case, not a detection bug

**How**

Routing runs offline, in batch, keyed and cached by rounded coordinate pair. The
deployed application gains no runtime dependency on a routing service. Only gaps are
routed (CLAUDE.md invariant 6).

**Decisions and their motivation**

*One interface, three implementations.* A self-hoster who will not build a regional
OSM extract must still see something. Straight lines that admit what they are beat an
error page, and the boundary exists for that reason rather than for architectural
neatness.

*Validate against Google's distance.* An independent check on inference costs almost
nothing and catches the case where routing produces a plausible road that is not the
road taken.

**Concepts introduced**

Go: interface-based pluggability and how to choose the seam, batch processing with a
cache, graceful degradation as a design property rather than error handling.

**Done when.** The phase 2 screen is visibly transformed, and the routed distance is
reported alongside the source's own figure.

---

## Phase 4 — Photos

**Goal.** Attach photos to an adventure, position them, and use them to check the
route.

**Features**

- Upload photos for an adventure
- Extract EXIF position and timestamp
- Place them on the map and on the journey timeline
- Store a thumbnail, not the original
- Flag photos that sit far from the inferred route
- Accept Google Photos Takeout JSON sidecars (`geoData` + `creationTime`) alongside
  raw EXIF: photo auto-backup is near-universal for the target audience, which makes
  a decade of geotagged photos a primary location source — for users with no
  Timeline data at all, the primary source

**How**

Go extracts EXIF and generates thumbnails. Positions and timestamps are stored;
originals are not hosted.

**Decisions and their motivation**

*Photos are the most accurate positions in the project* — camera GPS is metres,
against tens of metres for typical WiFi positioning in the source data. On a journey
they land exactly where something mattered.

*They independently validate the route.* A photo timestamped mid-journey but sitting
kilometres off the inferred road proves the inference wrong. This is the only ground
truth available that does not come from the same source as the error.

*Thumbnails only.* Hosting originals turns a small deployment into a storage problem
with an ongoing bill, and buys nothing the journey page needs.

*Video is excluded for now.* MP4 location metadata is far less standardised than photo
EXIF; revisit if photos prove their worth.

**Concepts introduced**

Go: binary metadata parsing, image processing. Next.js and React: file inputs, upload
handling, optimistic UI for slow operations.

---

## Phase 5 — Deploy and self-host

**Goal.** Someone else can run this against their own export.

**Features**

- `docker compose up`, point at an export, see adventures
- Import date range as a parameter
- Configuration via environment, no coordinates anywhere in it
- A README that states only what the repository can demonstrate
- Import bookkeeping in the UI: per-import status, error message, and counters
  surfaced (two additive columns on `imports`)
- A demo dataset, so Roadbook can be evaluated without handing over a real export
- Configurable map tile provider — the viewport request is the one thing that leaks
  to a third party; self-hosters can point at their own tiles
- Backup/restore of what users cannot regenerate: their decisions, and — since
  phase 4 — their photos (the `photos` table and the thumbnail directory
  together; originals are discarded at upload, so the thumbnail is the copy)
- Optional routing setup for self-hosters who want it on their server: an OSRM
  compose profile plus an operator-run script automating the extract download and
  preprocessing (specification: `docs/phase-3/OSRM.md`). Explicitly invoked only —
  never at build or install time — and strictly optional: the deployed service
  reads only the routing cache, and the null-router state is fully usable

**Decisions and their motivation**

*Self-hostability is tested by doing it*, not by intending it. Any assumption about a
specific user or region surfaces the first time someone else's export is imported.

---

## After phase 5 — backlog, unordered

Adopted into the backlog from the Dawarich comparison or moved here from a phase;
none is scheduled:

- Legacy Timeline variants — `Records.json` (E7 integer coordinates, raw samples;
  no visit segments, so it needs stay-point visit synthesis and a home-derivation
  pass) and Semantic History `timelineObjects` (monthly files). Cut from phase 5 at
  its Gate 1 as speculative work for users not yet observed — the same evidence
  standard as HEIC. Trigger: a real export that fails to import. The population is
  real but invisible until then: the current phone export carries full history, so
  only users whose past survives solely in old Takeout archives (the 2024 on-device
  migration purged server-side data for those who missed it) need these parsers,
  and the sniffer already names both formats in its rejection; phase 5 records the
  sniff's format label per import (`imports.detected_format`), so the trigger is a
  queryable count of failed imports by format, not an anecdote. Design conclusions
  to restart from (phase 5 Gate 1 review):
  both variants parse inside the existing sniff seam to existing domain types
  (invariant 4; Records samples are raw positions, not trace points); synthesis is
  a pure derived pass at detection time, parameterised and never persisted
  (invariants 2 and 3); home derivation generalises to "home evidence" with strict
  semantic precedence, so fixture and archive outputs stay byte-identical
- GPX ingestion (Garmin/Strava/Komoot/OsmAnd) — the enthusiast tier. GPX tracks are
  dense, so these adventures arrive nearly gap-free: Timeline is the sparse case,
  GPX the dense case, one observed/inferred model covers both
- Photos as an import source — geotagged photos become timestamped position fixes:
  a new parser emitting existing domain types (invariant 4), plus stay-point visit
  synthesis and home derivation without visits (machinery shared with the legacy
  Timeline variants above — one brief should cover the synthesis pass for both).
  Phase 4 already established the principle: a positioned photo is a measurement.
  Serves anyone with a camera roll but no Timeline — the common iPhone case.
  Carried from phase 5; chartered 2026-08-24 as phase 11
  (`docs/phase-11/BRIEF.md`)
- Manual trip creation — a user with no data draws the trip they took: waypoints
  and dates, road geometry routed between asserted points, geotagged photos
  slotting in as real measured fixes. Honesty-compatible only as a third
  provenance class ("recalled": own leg kind, own ink, legend wording, provenance
  bar stating measured vs recalled) — recalled geometry must never be mistakable
  for, or used to correct, measured geometry (invariants 2, 5, 8). The largest
  backlog item: a route editor, a creation path bypassing detection, and a
  PRODUCT.md amendment — a product-identity decision to take deliberately.
  Trigger: real users that photo and GPX ingestion still leave stranded
- Self-serve data deletion — a user-facing "delete my data" removing
  observations, imports, decisions, photos (rows and files), and retained
  uploads in one act. Deferred deliberately: on an authless instance a
  destructive control is a hazard, and for hand-provisioned instances the
  operator's volume-level reset is structurally complete where row-chasing
  can silently miss a file. Becomes user-facing — and, for location data,
  legally required — when accounts arrive. Trigger: the multi-user tenancy
  phase (phase 13 in the roadmap below)
- Per-adventure GPX/GeoJSON export, with the observed/inferred distinction preserved
- Bulk triage actions — first non-operator pilot report (2026-08-24): confirming
  candidates one at a time is a real workload at full-archive scale (the
  triage-friction-at-scale evidence phase 9 anticipated). In-system answers to
  design from: multi-select confirm/dismiss, a quick-confirm that never opens the
  card, a score-ordered sweep flow. The reported *solution* — auto-confirm
  everything, edit later — is recorded but is not a UI fix: the detector
  over-produces deliberately (rank, don't filter), so auto-confirmation would put
  known noise on the life map and invert the curation stance that defines the
  product; adopting it would be a PRODUCT.md amendment taken deliberately at a
  gate. Trigger: already fired; schedule the friction work in the next UI/product
  refinement phase
- Per-mode distance breakdown (air / road / rail / water) — pilot request
  (2026-08-24). Activities carry mode and distance, so a per-journey breakdown is
  computable from existing data. Constraint: source modes are guesses with
  recorded failures at the extremes, and the pipeline already trusts speed over
  mode for flight detection — so any breakdown must be labelled as
  source-asserted, never presented with the confidence of measured geometry
  (invariant 8's spirit applied to labels). Fits the day-narrative/stats seam
  alongside leg-average speed
  in the output (separate track segments per confidence class) — an export that
  flattens it silently violates the honesty principle
- Poster / print view of an adventure
- Adventure replay animation
- Elevation profile per adventure
- OSM amenity overlay along a route (fuel, food, hospitals, restrooms) from bundled
  extracts — read-only, offline, no community layer; see the comparison doc §8.1

**Parked, not planned:** one further direction is recorded in the project's private
notes (comparison doc §8.2 points there). It contradicts the current charter and
must not be absorbed into this plan silently; the private record includes explicit
trigger conditions, all of which must hold before it is even reconsidered.

---

## The road to strangers — phases 10–13 (roadmap, 2026-08-18; resequenced 2026-08-24)

Phases 6–9 (life-map UI and design system, browser import as the front door, pilot
hosting, UI refinement) are recorded in their own `docs/phase-N/` artifacts and are
not restated here.

This roadmap is written under the dual-mode charter amendment of 2026-08 (CLAUDE.md
do-not-add list; PRODUCT.md "Hostable as a service"). It sequences the phases from
the current state — a working self-serve loop on hand-provisioned pilot instances —
to the point where strangers can be brought in. It does not replace phase briefs:
every phase below opens with its own design brief, and a brief may overturn a
leaning recorded here with better evidence.

**The finish line.** A stranger who finds the public landing page understands what
Roadbook is, and either receives one of the capped instance slots or joins a
waitlist; an accepted person gets an entry email with a private link and completes
the existing loop — upload, import, detection, life map — with no ad-hoc operator
work (scripted provisioning counts as none). That is the close of the front-gate
phase (now phase 12). The cap is ten users to start, a chosen number, revisited
in its brief.

### Phase 10 — Hosting readiness

**Goal.** The hosted offering survives without the operator's laptop: instances on
a rented host with a real domain and TLS, sized for ten, with off-machine backups.
No product code — the target is a zero Go/web diff for the whole phase.

**Charter basis.** Phase 8 recorded the upgrade path (§3A: small ARM VPS; §3B:
domain + wildcard TLS + per-instance secret subdomain + basic auth) with named
triggers. Two now fire: the maintainer deciding to spend — the public-hosting
decision is that decision — and the pilot outgrowing three concurrent testers,
since ten exceeds the three-port funnel front. The move retires the funnel relay
from the upload path.

**Checkpoints.**

1. **The host and the front.** Server provisioned and hardened, domain bought,
   wildcard TLS, the demo instance live on a subdomain. *Visible: the demo link
   answers from a phone on cellular with the laptop closed.*
2. **Topology at ten.** The per-instance compose template and operator scripts
   (stamp, reset, rotate, backup) ported and run against the new host; capacity
   for ten stacks measured under real instances, not assumed. *Visible: one
   command stamps a fresh instance; a measured capacity statement.*
3. **Migration.** The existing pilot instances move via the proven backup/restore
   path with zero decision loss; links rotate at the move, per the handover rule.
   *Visible: existing testers' instances answer at new URLs with their data
   intact; the restore drill recorded.*
4. **Durability.** Nightly encrypted backups leave the host for a destination the
   brief decides; a reboot/failure drill runs on the new host; the runbook is
   rewritten; the laptop front is retired. *Visible: a restore from the
   off-machine copy; a documented drill.*

**Excludes.** Any product code; the landing page; waitlist; auth; tenancy; new
ingestion sources.

**Open questions for the brief.** Provider and region — other people's location
data makes residency a real consideration; the domain name, which becomes the
product's public name; the monthly cost ceiling — this phase explicitly retires
phase 8's zero-cash stance; whether the demo instance goes credential-free.

**What would resequence it.** Pilot reports showing the self-serve loop failing
for product reasons — then a fix-the-loop pass precedes this phase's close,
because infrastructure under a loop nobody completes is waste.

### Phase 11 — Ingestion: photos as an import source (chartered 2026-08-24)

**Goal.** A person with no Timeline data — the measured iPhone majority — reaches
their adventures from their camera roll: geotagged photos become position fixes,
a pure stay-point synthesis pass derives the visits detection needs, and home
derivation generalises to home evidence with strict semantic precedence so all
existing outputs stay byte-identical. Includes the pilot-response scope
candidates (bulk triage actions, per-mode distance breakdown) argued in the
brief. Design brief: `docs/phase-11/BRIEF.md`.

**Charter basis.** The resequencing clause below fired 2026-08-24: none of the
iPhone-holding pilot users has Timeline data at all (four of six instances at
zero imports), so the pilot itself returned the audit's answer and the unsent
audit message was retired. A front gate built first would recruit into a funnel
the pilot proved closed to the iPhone majority.

### Phase 12 — Front gate

**Goal.** A stranger can find Roadbook, understand it, and either get a slot or
join the waitlist; entry is an email carrying a private link into the existing
loop.

**Checkpoints.**

1. **The public landing.** The pitch surface on the apex domain. The brief
   decides whether the existing front door evolves or a distinct landing hands
   off to per-instance front doors. The supported-data statement is honest and
   up front: Google Timeline only today, per-platform expectations, "Timeline
   never enabled means nothing to export" stated before anyone is asked to want
   in. *Visible: a cold visitor reaches "I want this" or "not for me" without
   contacting the operator.*
2. **The waitlist.** Email capture, stored minimally, purpose stated plainly,
   deletable on request — the PRODUCT.md contact-data sentence governs. The
   brief decides the simplest honest mechanism and where entries live.
   *Visible: a submitted address lands somewhere durable; the page says exactly
   what will happen with it.*
3. **Cap and entry.** Scripted provisioning turns an accepted person into a
   stamped instance; the entry email carries their private link and credential;
   the cap is enforced — by process, not code, at this scale, unless the brief
   argues otherwise. *Visible: one command turns a waitlist entry into a
   welcomed user.*
4. **The proof.** One real person walks the whole path — landing, waitlist,
   entry email, upload, adventures — with zero ad-hoc operator work. README and
   docs updated. *Visible: the full loop, witnessed.*

**Excludes.** Product auth and accounts — instances stay single-user behind
per-instance credentials, consistent with the amendment, because entrants are
hand-provisioned; self-serve signup; share links; tenancy machinery.

**Open questions for the brief.** How the entry email is sent — at ten users,
operator-sent mail with no sending machinery is the lean answer; whether waitlist
entries need verification; the cap's exact number; what the landing promises
about data handling — instance isolation should be stated plainly.

**What would resequence it.** Audit returns showing most interested people have
no Timeline data — then the ingestion phase moves ahead of this one, because a
gate that turns away most entrants converts the waitlist into a disappointment
list. (This fired 2026-08-24 — hence this phase's move from 11 to 12.)

### Phase 13 — Accounts and tenancy (gated)

**Goal.** Strangers sign themselves up and the waitlist drains without per-person
operator work. Charterable under the amendment; briefed only when its trigger
binds — the cap filled and real demand beyond it. If the waitlist stays short,
this phase never runs, and nothing above depends on it.

**Shape — the brief decides between two forms that both satisfy "scale".**
True tenancy: delegated auth (OIDC, never hand-rolled), user scoping on every
query — the store test harness was built for that day — and the data-lifecycle
work that location data makes legally real, including self-serve deletion (the
backlog entry lands here). Or automated instance-per-user: provisioning becomes
software and isolation stays structural, preserving the property that no
cross-user bug class exists. The recorded direction is the first; the second is
to be argued against, not skipped.

**Also staged here:** read-only share links — the "shared view" consumer the
architecture rationale has always anticipated, and a privacy surface that only
makes sense once access control exists. Own checkpoint or own phase; the brief
decides. The UI seam is already banked: the route-group shell split and the
reserved header slot mean an auth gate drops in without moving pages.

**Excludes.** Social features, live sharing, community content — still banned
outright.

### Gated alongside — not in the sequence until their evidence arrives

- **Ingestion (photos as a source, GPX).** Photos-as-source was chartered
  2026-08-24 as phase 11 above — the pilot's iPhone finding replaced the
  planned data audit. Its brief designs the stay-point synthesis pass for all
  the visit-less sources (photos, the legacy Timeline formats, continuous
  GPX) and implements photos; GPX — the enthusiast tier — stays gated on its
  own evidence and joins the already-built synthesis pass when it arrives.
- **Manual "recalled" adventures.** Stays behind its recorded trigger — real
  users that photos and GPX still leave stranded. Requires its own PRODUCT.md
  amendment (the third provenance class), decided deliberately at its own
  gate; deliberately not bundled into the dual-mode amendment. Sequenced after
  ingestion.
- **Refinements (non-gating).** GPX/GeoJSON export preserving confidence
  classes, poster/print, a reproducible stats panel, leg-average speed in the
  day narrative (instantaneous speed stays rejected — the observation density
  cannot support it honestly), dark theme, unified timeline. These ride along
  when evidence or appetite says so; none blocks strangers.

### The sequence, and what would change it

Phase 10 (complete), then 11 (ingestion), then 12 (front gate), then 13 when
its trigger binds; recalled adventures after ingestion on its own trigger;
refinements ride along.

- Audit returns mostly without Timeline data → ingestion moves ahead of the
  front gate. **Fired 2026-08-24**: the pilot itself returned the answer —
  no iPhone-holding pilot user has Timeline data at all — and ingestion was
  chartered as phase 11 at that day's charter STOP.
- Pilot evidence of a broken loop → a fix-the-loop pass precedes phase 10's
  close. (Phase 10 closed with the loop's first non-operator completion on
  record; the reported friction items are scope candidates in the phase 11
  brief, not loop breaks.)
- The waitlist never fills → phase 13 never runs; a capped hosted pilot plus
  self-host is a complete product.
- The maintainer deciding against public hosting → phases 12 and 13 close,
  the backlog continues.
