# Phase 5 log — deploy and self-host

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. Public numbers in this phase come
from the committed demo dataset and reproduce by the commands the README
states; counts derived from private data are not stated here.

## What the phase does

Someone else can run this against their own export. `docker compose up`,
point at an export, see adventures — with the dev setup untouched.

- **Topology.** Three services matching the three runtimes: `db`
  (Postgres + PostGIS), `api` (one static Go binary, CLI and server both,
  migrations embedded), `web` (Next standalone output). Only the web port
  is published, and only on loopback — an authless product must not be
  internet-exposed by default; widening the binding is a deliberate edit
  behind the operator's own reverse proxy. Startup order is declared, not
  raced: `pg_isready` on db, `GET /healthz` on the API (in the OpenAPI
  contract per invariant 10; 200 only when a database round-trip
  succeeds), `service_healthy` conditions chaining db → api → web.
  Migrations run in the api service's compose command
  (`roadbook migrate && exec roadbook serve`) — goose is idempotent and
  embedded, so an upgraded image cannot forget its schema, while dev
  `roadbook serve` keeps no implicit migration side effect. State lives in
  named volumes (`pgdata`, `photos`); the export directory mounts
  read-only, making the never-modify rule a mount flag. Build contexts are
  provably free of private data: `.dockerignore` excludes `data/`, `.env`,
  and `docs/private/`, and the measured context transfer is a few hundred
  KB against a multi-GB `data/`.
- **Import visibility.** Migration 00007 adds `status`, `error`, and
  `detected_format` to `imports`; the import command writes its row before
  parsing and finalises it, so a failed import is visible in the product,
  and the sniffer's stable format label is recorded on every attempt —
  queryable evidence for the legacy-formats backlog trigger, deliberately
  separate from the reword-able message prose. `GET /imports` and a plain
  `/imports` page surface it.
- **The demo.** `testdata/demo/` holds a deterministic committed generator
  and its committed output: three months of a fictional persona over real
  public Icelandic geography, scripted to put every honesty state on
  screen — commute noise detection must ignore, an away-day with no far
  destination that is correctly not a candidate, a dense drive (mostly
  observed legs), a sparse trip (unknown gaps dominate), and a flight
  weekend (air arcs, excluded from ground validation). Detection over it —
  3 candidates, 1 home base, 0 outliers — and its parse counts are pinned
  by an ungated regression test, which is what lets the README state them
  under invariant 13. Iceland was chosen so one small Geofabrik extract
  covers the whole demo.
- **Backup and restore.** `roadbook backup` writes the irreplaceable set —
  decisions, photo rows, thumbnail bytes, a manifest with format and
  schema versions — as one tar.gz; everything else regenerates from the
  user's own exports. Rows are archived by durable identity (decision
  anchor, photo content hash; photos reference decisions by anchor tuple,
  never row id), and `roadbook restore` merges by those identities into
  any instance, skipping and reporting overlap. Restored into an empty
  instance, decisions are orphans until the next import + detection
  re-attaches them — restore rides the anchored-matching machinery rather
  than duplicating it. Migration 00008 makes the anchor structurally
  unique, which is what ON CONFLICT merges against.
- **Optional routing.** An `osrm` compose service behind
  `--profile routing`, absent from every default `up`, port unpublished,
  parameterised by `OSRM_DATA`; `scripts/osrm-setup.sh` downloads a
  Geofabrik extract (region a required argument — invariant 9 lets nothing
  guess where the operator lives) and preprocesses it with the same OSRM
  image the profile serves with, stamping the dataset name from the pbf's
  own date. Invoked by an operator typing it and by nothing else. After
  the batch, the profile comes down and serve reads only the cache,
  exactly as phase 3 left it.
- **The README** replaces a one-line stub: what the product is with the
  honesty rule first, the supported input stated precisely (legacy formats
  named as recognised-but-unsupported), the demo quickstart, the
  deployment's network posture, configuration, backup, and the routing
  walkthrough. Every number cites its reproducing command.

Cut at Gate 1, recorded in DECISIONS and PLAN's backlog: the legacy
Timeline parsers (Semantic History, `Records.json`) and the stay-point
synthesis they need — speculative work for users not yet observed, held to
the same evidence standard as HEIC. `detected_format` is what makes that
trigger a queryable count.

## What broke, and why each fix took its form

**The official PostGIS image excludes every ARM machine.**
`postgis/postgis:18-3.6` publishes no arm64 manifest — discovered when the
pull failed outright on the maintainer's own machine, which also meant the
test-database fallback in `compose.test.yaml` had been silently broken on
ARM since phase 4 (unnoticed because the usual dev machine has local
Postgres). Fix: `imresamu/postgis:18-3.6` — the multi-arch build the
official README itself points ARM users at — everywhere, one image name
across test and deploy. A two-line revert if upstream ever ships arm64.

**Postgres 18 images moved the data directory.** Mounting the traditional
`/var/lib/postgresql/data` would have left the database outside the volume
— data loss on the first image replacement. The volume mounts at
`/var/lib/postgresql`, the image's declared volume, with the reason
commented in compose.yaml.

**The demo's first generation produced four perfect away-spans and zero
candidates.** The detector's dwell lookup binary-searches visits in file
order — a documented assumption ("the export is chronological") that the
generator, appending trip scripts after the daily loop, violated. Fix in
the generator: sort segments chronologically before writing, honouring the
assumption real exports satisfy, rather than touching the detector.

**A flight from the city airport can never render as an arc.** Reykjavík's
airport sits inside NEAR, so the away-span — and therefore the journey
window — begins only at the far end, excluding the flight entirely. The
demo flies via Keflavík (beyond NEAR, a real airport). Separately, journey
assembly draws legs from trace points and raw positions, not visits or
activity endpoints, so an activity-only flight is a 30-hour slow silence:
each flight is bracketed by a stationary gate fix at both airports, which
is also what real phones record. Both discoveries are exactly the kind of
pipeline truth a demo exists to be honest about, and both are commented in
the generator.

**Byte-identical duplicate points made file mode and database mode
disagree.** A "minute-doubled" stationary fix hashed identically, deduped
on import, and shifted a density score by 0.2 between
`detect -src` and `detect -from-db` — the reproducibility drift invariant
13 exists to catch, caught by comparing the two modes during verification.
Fix: the generator emits distinct timestamps; a first import now inserts
every parsed row and the two modes agree exactly.

**The gitignore would have swallowed the demo.** The belt-and-braces
`*.timeline.json` pattern (a data-safety measure from the near-leak era)
matches the BRIEF's illustrative `demo.timeline.json` name. The file is
named `demo.json` instead of punching a negation hole in a safety pattern.

**Restore needed an identity the schema only enforced by convention.**
Decisions had no unique constraint on their anchor; restore's merge
semantics need one for ON CONFLICT. Migration 00008 adds the unique index
— verified duplicate-free on the live data first — turning an identity
that held by code-path discipline into one the database enforces.

**A backup must not preserve a state it would be a bug to create.** A
photo row without its thumbnail file is a permanently broken image (the
phase 4 delete-order reasoning), so backup excludes fileless rows with a
warning, and restore writes each file before its row and skips rows with
no thumbnail available anywhere. For the same class of reason,
`backup -out` refuses to overwrite an existing archive and deletes its own
partial output on failure: a backup can be the last copy.

## Verification at close

- The README quickstart executed verbatim from a clean `git clone` of the
  committed state (no `data/`, no `.env`, fresh volumes, images built from
  the clean context): compose up ordered and healthy, demo imported
  (720 observations, all new), 177 countries loaded, 3 candidates with
  scores identical between file mode and database mode, candidate list and
  imports page served from containers, loopback answering and the LAN
  address refusing.
- Cold-cache `make test` green with the DB-backed store and backup tests
  running (not skipped), both golden journey regressions, the demo
  regression, and the private-data-gated archive regression all passing;
  fixture detection still 18 candidates / 1 base / 32 outliers.
- Backup/restore demonstrated live on real data at checkpoint 3: the real
  instance archived (18 decisions, 2 photos, schema 8), restored into a
  scratch compose instance as 18 orphans, and re-attached 18/18 with 0
  orphans after import + detection, both photos serving through the API
  and the browser proxy; double-restore and restore-into-source proven
  no-ops by test.
- The routing profile run for real at checkpoint 4: Iceland extract
  prepared by the script, 13/13 gap pairs routed in-compose, the dataset
  name recorded in `route_runs`, the sparse journey's chords becoming
  road geometry, air legs never routed, and the cache serving routed legs
  after OSRM came down; a default `docker compose up` never mentions the
  profile.
- `git add -A --dry-run` clean of `data/`, `.env`, and archives at every
  commit point.

## Carried forward

- **Legacy Timeline formats** live in PLAN's backlog behind the evidence
  trigger, with the design conclusions to restart from recorded there;
  `select detected_format, count(*) from imports where status = 'failed'
  group by 1` is the trigger's query.
- **HEIC** unchanged: real blocked uploads first.
- **Photos as a primary import source** still owns its own future brief;
  the stay-point synthesis the legacy formats also need lands there,
  designed once.
- **A backup download endpoint / UI button** was deliberately not built;
  the archive format is the contract and an endpoint can come later
  without changing it.
- **PhotoFarWarnM retune** and the **orphan-file sweep** remain
  evidence-gated as phase 4 left them.
- **The unified timeline view** and the home-page reshape (map and
  adventures first, triage demoted to an entry point) are future
  presentation work; §3B's prose already notes the reshape intention.
