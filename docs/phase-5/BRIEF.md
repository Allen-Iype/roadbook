# Phase 5 design brief — deploy and self-host

**Goal (PLAN):** someone else can run this against their own export. `docker compose
up`, point at an export, see adventures.

Phase 5 inherits a system that is complete for its maintainer: import, detection,
decisions, journeys on a map, routing from a cache, photos. What it lacks is the
ability to exist on a machine that is not this one. That gap has four parts, and each
is a real design problem, not packaging chores: containerising a three-runtime system
without breaking the dev setup; making environment the single configuration channel;
treating backup of irreplaceable user data as a product feature; and writing a README
that claims nothing the repository cannot demonstrate.

The scope is PLAN's phase 5 feature list with one Gate 1 amendment: the legacy
Timeline formats (Semantic History, Records.json) moved to the backlog behind an
evidence trigger — see §5 and DECISIONS.md. The §2 rejections in
`docs/feature-comparison-dawarich-roadbook.md` are final and none of them re-enter
through the deploy door (no multi-tenant auth, no rate limiting, no notification
center — see §5).

---

## 1. Concepts this phase introduces

### 1.1 Containerising a Go + Postgres + Next system

A container image is a frozen filesystem plus a start command; a container is one
process tree running on that filesystem with its own network identity. Docker Compose
is the declarative description of several containers, their networks, and their
volumes — the whole topology in one file, brought up with one command. Three ideas do
almost all the work here:

- **Multi-stage builds.** A Go binary needs a Go toolchain to build but only a few
  megabytes of filesystem to run. A Dockerfile can use one image to build (`FROM
  golang:…` — compile) and copy the single artifact into a minimal final image. The
  build stage is discarded; users pull only the small result. Next has the same
  split: `npm ci` and `next build` need the full node_modules (hundreds of MB); the
  runtime needs Next's *standalone output* — a self-contained `server.js` plus the
  small subset of node_modules it traced — which exists precisely for this.
- **Service names are DNS names.** Inside a compose network, the API container
  reaches Postgres at `db:5432` and the web container reaches the API at
  `api:8080`. The hostnames in configuration stop being `localhost` and become
  service names; this is why configuration must come from environment (§1.2) rather
  than being compiled in.
- **Volumes are the state.** Containers are disposable by design — `docker compose
  up` after an image rebuild replaces them. Anything that must survive lives in a
  volume: the Postgres data directory, and the photo thumbnails (the second class of
  irreplaceable data). The user's export file enters as a read-only bind mount — the
  container can read it and structurally cannot modify it, which is the `data/`
  never-modify rule enforced by the mount flag instead of by discipline.

The Go side is the easy case, deliberately so: one static binary that is CLI and
server both, with the migrations embedded (`migrations/embed.go`) so `roadbook
migrate` inside a container needs nothing but the binary and a database URL. The Next
side is the fiddly case: standalone output must be enabled, and the MapLibre worker
copy (the `copy:maplibre` script that phase 2's silent-blank-map trap made necessary)
must run before `next build` so the worker files land inside the image.

The architecture already fits containers unusually well: the browser talks only to
Next, Next talks to Go, Go talks to Postgres. That chain becomes the compose network
literally — only the web port is published to the host, and only on loopback (§3A);
the API and the database need not be reachable from outside at all. Invariant 11
(frontend never reaches the database) is now enforced by network topology as well as
by code.

Two operational properties complete the picture. **Startup order is a declared
dependency, not a race:** each service carries a healthcheck (a command Docker runs
inside the container to answer "ready?"), and `depends_on` with
`condition: service_healthy` holds a service back until its dependency answers yes —
the database is genuinely accepting connections before the API migrates, and the API
is genuinely serving before web starts. **Exposure is opt-in:** publishing a port
binds it to an interface, and binding to `127.0.0.1` means only the machine itself
can connect — the safe default for a service with no auth (§3A).

### 1.2 Configuration via environment — the twelve-factor seam

The twelve-factor rule ("config in the environment") exists because the same image
must run on any machine: whatever differs between machines — addresses, paths,
providers — must arrive at run time, not build time. Roadbook is already most of the
way there, and this phase finishes and documents the seam rather than inventing it.
The full set today:

| variable | consumer | meaning |
|---|---|---|
| `DATABASE_URL` | Go | the one Postgres connection |
| `ROADBOOK_PHOTOS_DIR` | Go serve | thumbnail directory |
| `ROADBOOK_GEOCODER`, `ROADBOOK_NOMINATIM_URL` | Go serve | opt-in name suggestion |
| `ROADBOOK_ROUTER`, `ROADBOOK_ROUTER_URL` | Go route (batch) | routing backend |
| `ROADBOOK_API_URL` | Next server | where the Go API is |
| `ROADBOOK_MAP_STYLE` | Next server | map style URL — the configurable tile provider |

Compose reads a `.env` file beside `compose.yaml` and interpolates it; a committed
`.env.example` documents every knob with its default, and the real `.env` is
gitignored. Invariant 9 holds by inspection: nothing in this table is a coordinate,
a region, or anything about a specific user. Detection parameters remain CLI flags
recorded per run (invariant 3) — they are per-invocation experiment inputs, not
per-machine deployment config, and conflating the two channels would blur which runs
produced what.

### 1.3 Backup as a product feature

Two classes of data in this system cannot be regenerated: decisions (the user's
triage and names) and photos (a row plus a thumbnail file whose original was
deliberately discarded at upload — the thumbnail *is* the copy). Everything else —
candidates, journeys, routes, countries — recomputes from source exports and cached
inference, and the source exports are files the user already holds outside Roadbook.

`pg_dump` is the operator's disaster-recovery tool and remains available, but it is
not the product answer: it couples the backup to a schema version, restores
regenerable bulk alongside the irreplaceable rows, and knows nothing about the
thumbnail files, which live outside Postgres by design (phase 4). A product backup
is small, versioned, and semantic: the decisions, the photo rows, the thumbnail
bytes, and a manifest — restorable into a *different* instance at a *later* schema.

Restore has a story the system is already built for. Decisions attach to candidates
by anchored identity, and a decision whose candidate does not yet exist is an
*orphan* — a state the API reports and the code has handled since phase 1. Restoring
into an empty instance therefore needs no new machinery: the decisions land as
orphans, the user imports their export and runs detection, and re-attachment — the
most battle-tested path in the project, proven across seven detection runs — claims
them. Photos ride their decisions. §3D chooses the format.

### 1.4 A README under invariant 13

The README is the repository's public claim about itself, and invariant 13 applies
in full: no number that a command cannot reproduce. The maintainer's own dataset is
private, so none of its numbers can appear; the committed fixtures and the demo
dataset (§3C) are the only sources of public figures, and every figure cites its
command. The same discipline covers prose: no claim the repository cannot
demonstrate — the supported input is the current phone export, stated plainly, with
the legacy formats named as recognised-but-unsupported (§5). The honest-degradation
states — null router, unknown gaps, unplaced photos — are described as designed
behaviour, because that honesty is the product.

---

## 2. What gets built

- `Dockerfile` (Go, multi-stage → small runtime image), `web/Dockerfile` (Next,
  standalone output), `.dockerignore` files, `compose.yaml` with services `db`
  (postgis image), `api`, `web` — web published on loopback only, healthchecks and
  `service_healthy` ordering throughout — and an optional `osrm` profile;
  `.env.example`; `GET /healthz` added to `openapi.yaml`.
- Import runnable inside the topology (`docker compose run api roadbook import …`)
  with the existing `-from`/`-to` date window documented as the import-range
  parameter.
- Migration 00007: `status`, `error`, and `detected_format` columns on `imports`;
  `GET /imports` in `openapi.yaml`; an imports view in the web UI.
- A committed demo dataset (generator + output, fictional persona over public
  geography) and the README quickstart that uses it.
- `roadbook backup` / `roadbook restore` (tar.gz archive: manifest, decisions,
  photo rows, thumbnail files; idempotent restore by anchor identity and content
  hash).
- `scripts/osrm-setup.sh` + the `osrm` compose profile, implementing
  `docs/phase-3/OSRM.md`, operator-invoked only.
- The rewritten root `README.md` (currently a one-line stub).

---

## 3. The real choices

### 3A. Compose topology

**The choice: how many services, which images, where migrations run, what is a
volume, what is exposed, and how dev stays untouched.**

*Services.* Three, matching the three runtimes: `db` from `postgis/postgis:18-3.6`
(the image family `compose.test.yaml` already validated — migration 00003 needs the
extension), `api` from our Go Dockerfile, `web` from our Next Dockerfile. Only
`web`'s port is published to the host, and only on loopback:
`127.0.0.1:3000:3000`, never `3000:3000`. A bare port publishes on all interfaces,
which on a VPS means an authless product is internet-exposed the second it starts;
binding to loopback makes exposure a deliberate edit rather than a default. The
README states this beside the reverse-proxy sentence: anyone serving Roadbook
beyond the machine itself puts a reverse proxy with auth in front and widens the
binding knowingly. `api` and `db` stay internal to the compose network entirely.
Rejected: a single mega-image running all three under a process supervisor — it
discards the isolation compose exists to provide, and makes the Postgres upgrade
story ours instead of the image's. Rejected: exposing `api` on the host by default —
nothing outside the network needs it; a self-hoster who wants to curl the API can
publish the port in an override file.

*Images.* Two Dockerfiles, one per build context (`Dockerfile` at the root for Go,
`web/Dockerfile` for Next), rather than one multi-target file: each context's
`.dockerignore` is then honest and small, and the web build never receives the Go
tree or vice versa. The Go final stage is a minimal base with a shell (alpine-class,
tens of MB) rather than scratch — the migrate-then-serve start command (below), the
container healthcheck, and operator debugging all want a shell and busybox `wget`,
and the size difference is noise at this scale. The Next image enables
`output: "standalone"` in `next.config.ts` (additive; dev server behaviour
unchanged) and runs `copy:maplibre` before `next build` so the phase-2 worker-copy
requirement holds inside the image.

*Migrations.* Run by the `api` service's start command: `roadbook migrate && exec
roadbook serve`. The alternatives — an operator-run `docker compose run api roadbook
migrate` step, or a separate one-shot migration service — both make the quickstart
longer and create a failure mode ("forgot to migrate after pulling a new version")
that goose's idempotence exists to remove. Migrations are embedded in the binary, so
the binary and its schema expectations cannot drift; running them at start is safe
for a single-instance deployment (no replica race — and we never run replicas;
"multi-user" is out of scope). Critically this lives in the *compose command*, not
in `roadbook serve` itself: dev behaviour is unchanged, and serve never grows an
implicit migration side effect.

*Healthchecks and startup order.* A racing first `docker compose up` is the worst
first impression a deploy phase can make, so readiness is declared, not hoped for.
`db` healthchecks with `pg_isready` (the `compose.test.yaml` pattern, proven);
`api` gains `GET /healthz` — in `openapi.yaml`, because invariant 10 permits no
hand-written route outside the contract — which answers 200 only when a trivial
database round-trip succeeds, and the container healthcheck calls it via busybox
`wget`. Ordering: `api` `depends_on` `db` with `condition: service_healthy` (so
migrate meets a ready database), `web` `depends_on` `api` the same way (so the
first page render meets a serving API). `web` itself carries no healthcheck —
nothing depends on it. Rejected: a bare-TCP healthcheck on `api` (a listening
socket with a dead database behind it is precisely the not-ready state the check
exists to catch).

*Volumes.* Three mounts carry all state. A named volume `pgdata` for Postgres — the
database's lifecycle belongs to Docker, not to the repo checkout. A named volume
`photos` mounted at `ROADBOOK_PHOTOS_DIR` — irreplaceable files, decoupled from the
checkout, covered by `roadbook backup` rather than by filesystem archaeology. And a
**read-only** bind mount of a host directory (default `./data`) at `/data` for
export files — the container reads the user's export where they put it and the `:ro`
flag makes the never-modify-`data/` rule a mount property. Rejected: bind-mounting
photos into `./data/photos` — it couples irreplaceable state to a git checkout
directory and reopens the leak surface that `data/` hygiene rules exist to guard.

*Dev unchanged.* `compose.yaml` is a new file; nothing existing moves. The brew
Postgres + `./bin/roadbook serve` + `npm run dev` flow, `compose.test.yaml`, and
`scripts/test.sh` are all untouched. The two stacks share nothing but the code.

### 3B. Import in a containerised world, and import bookkeeping

**The choice: how a self-hoster runs import, and what the UI shows about it.**

Import stays a CLI batch step, run inside the topology: `docker compose run --rm api
roadbook import -src /data/Timeline.json`, then `… roadbook detect`. The date window
is the existing `-from`/`-to` — PLAN's "import date range as a parameter" has
existed since phase 1; this phase's work is documenting it as part of the deploy
story, not building it. Rejected: an upload-a-file web UI for imports — it turns a
few-times-a-year batch operation into a request/response problem (multi-GB uploads,
progress, timeouts) and the CLI already owns the taxonomy of actionable rejections.

Bookkeeping (PLAN: "two additive columns", grown to three at Gate 1): migration
00007 adds `status` (`running | completed | failed`), `error` (text), and
`detected_format` (nullable text — the sniff's stable format label, populated on
every import, successful or failed) to `imports`. The format label is what makes
the legacy-formats backlog trigger *queryable* rather than anecdotal
(`select detected_format, count(*) from imports where status = 'failed' group by
1`), and it is deliberately separate from `error`: the user-facing message is being
reworded in this very phase, so evidence must not be stored inside prose. The
import command today writes its row only after success; it reorders to write the
row first (`running`) and finalise it (`completed` with counters, or `failed` with
the sniff taxonomy's message) so a failed import is *visible in the product* and
not only in a terminal scrollback that a self-hoster has closed. `GET /imports`
joins `openapi.yaml`; the UI gets a plain imports view (label, window, date,
counters, status, format, error) — recommended as a small `/imports` page linked
from the home page, keeping the home page the triage surface (today's shape —
phase 6 intends to promote the map and adventures to the home page with triage
demoted to an entry point; the point here is only that imports do not belong on
it). Rejected: surfacing import errors
as a notification mechanism — the comparison doc §2 rejected notification machinery;
a list page is the inline substitute it prescribed.

### 3C. The demo dataset

**The choice: how to demonstrate the product without real location data, and what
the README may claim from it.**

Recommended: a committed generator (`testdata/demo/gen`) and its committed output —
a Timeline export in the current phone format for a fictional persona over real
public geography. Real roads, invented person: coordinates trace public highways and
towns, which carries no one's history and is safe to commit (the data-safety rule
guards *real* location history; a scripted itinerary over public geography is
authored fiction). The current phone format only — it is the one supported format
(§5), and the demo demonstrates the product.

The persona's script is chosen to exercise every honesty state on screen: a home
with commute noise (detection must ignore it), a dense car adventure (mostly
observed legs), a sparse adventure (routed/unknown gaps dominate), and one flight
(air arc, excluded from ground validation). Without OSRM the sparse gaps render as
honest chords — the demo *demonstrates* invariant 7's degraded state rather than
hiding it; with the optional §3E profile, the same demo routes fully, which is the
before/after the README can show.

One deliberate coordination: the demo's geography sits inside a single small
Geofabrik extract (candidates: Iceland or Estonia — extracts in the low hundreds of
MB, versus 1.5 GB for India), so the optional routing walkthrough is cheap for
anyone who tries it. Exact region chosen at implementation; the property that
matters is "one small extract covers the whole demo."

README numbers come only from the demo and `testdata/`, each with its reproducing
command ("the demo contains N adventure candidates: `roadbook detect -src
testdata/demo/demo.timeline.json`"). Rejected: an anonymised slice of real data as
the demo — anonymisation of a *whole life pattern* is much harder than of one
journey window (the golden fixtures shift one trip's frame; a demo timeline exposes
home/work rhythm, and the 27jul CONTRACT already showed how much a documented frame
leaks). Rejected: generating the demo at runtime instead of committing it —
invariant 13 wants the README's numbers to trace to checked-in bytes, and a
generator drift would silently invalidate every stated figure.

### 3D. Backup and restore

**The choice: format, scope, and the restore-into-another-instance story.**

Recommended format: one `tar.gz` written by `roadbook backup -out FILE`, containing
`manifest.json` (format version, schema version at write time, created-at, counts),
`decisions.json` (every decision with its anchor identity, action, name,
timestamps), `photos.json` (every photo row — content hash, original name, captured
metadata, provenance fields — referencing its decision by anchor identity, not by
row id), and `thumbnails/<content-hash>.jpg` (the bytes). Scope is exactly the
irreplaceable set (§1.3). Explicitly out: observations, candidates, route cache,
countries — all regenerable from the user's own export files, and the README states
plainly that source exports are the user's to keep (with the standing advice that
exporting roughly monthly banks each expiring rawSignals window).

Restore: `roadbook restore -src FILE` into any instance, empty or not. Decisions
insert by anchor identity, `ON CONFLICT` skip — restoring twice, or restoring into
an instance that already holds some of the same decisions, is a reported no-op for
the overlap (the imports/photos idempotency precedent). Photos insert by content
hash, files written to the photos dir if absent. Into an empty instance the
decisions are orphans by construction; the API already reports orphans, the UI
already shows them, and the user's next import + detect re-attaches — restore rides
the anchored-matching machinery rather than duplicating it. Row ids are never in the
archive, so id drift between instances cannot corrupt anything.

Rejected: `pg_dump` as the product mechanism (§1.3 — schema-version coupling, no
files, restores regenerable bulk; it stays documented as the operator's
whole-instance option). Rejected: SQL `COPY` formats (couples to columns; JSON plus
a format version is inspectable and migratable). Rejected: an API/UI backup button
this phase — backup is an operator action beside import and route; a download
endpoint can come later without changing the archive format. Rejected: including
source export files — multi-GB archives to protect files the user already owns.

### 3E. The OSRM optional profile and operator script

**The choice is mostly made — `docs/phase-3/OSRM.md` is the spec and the phase-3
decision log already commits to this shape. What remains is the boundary's exact
expression in compose.**

The `osrm` service sits behind a compose *profile* (`--profile routing`): absent
from `docker compose up` unless explicitly requested, image
`ghcr.io/project-osrm/osrm-backend`, serving MLD from a bind mount of
`data/osrm/`, port not published to the host (the API container reaches it as
`http://osrm:5000`; AirPlay's port-5000 squat is a host concern that vanishes
inside the network). `scripts/osrm-setup.sh` automates OSRM.md §§1–2: takes the
region explicitly as an argument (no default region — invariant 9 allows no
regional assumption), downloads the extract to `data/osrm/`, runs
extract/partition/customize via the OSRM image, and prints the dataset name
(`region-YYYYMMDD`) to pass to `roadbook route -dataset`. It is invoked by an
operator typing it, and by nothing else: no Dockerfile, compose hook, Makefile
target, or install path calls it. The batch run is then `docker compose run --rm
api roadbook route -router osrm -router-url http://osrm:5000 -interval 0 -dataset
…`, after which the profile comes down and serve continues reading only the cache —
serve gains no router, exactly as phase 3 left it.

Rejected: auto-detecting a region from the imported data to save the operator an
argument — it is a regional assumption derived from location data leaking into an
operational surface, and the operator knows where they live. Rejected: bundling
preprocessing into the osrm service's start command — preprocessing is a
tens-of-minutes, ~10 GB-RAM batch job; conflating it with "serve queries" makes
`--profile routing` dangerous instead of cheap.

### 3F. Configurable tile provider

Already built — `ROADBOOK_MAP_STYLE` has existed since phase 2 checkpoint 3. This
phase's work: surface it in `.env.example` and compose, and state the privacy
property in the README — tile fetches are the one browser request that leaves the
self-hosted boundary (the viewport leaks to the style's tile host), and pointing
the variable at self-hosted tiles closes even that. No new code expected; if
implementation finds the style URL insufficiently pluggable (attribution, glyphs),
that surfaces as a checkpoint 1 finding, not a redesign.

---

## 4. Data safety in a compose world

The hazards this phase adds are image and archive shaped:

- **Images must be provably free of `data/`.** Both `.dockerignore` files exclude
  `data/`, `.env`, and `docs/private/` from the build context — nothing private can
  be COPY'd even by a future careless Dockerfile edit. An image is a distributable
  artifact; treat its contents with commit-level suspicion.
- **Backup archives are real location-adjacent data** (decision names, photo
  metadata, thumbnails). They must never land in the repo: default output path
  outside the tree or under `data/`, README says to store them like the exports.
- **The export mount is read-only** (`:ro`), making container-side modification of
  `data/` structurally impossible (§3A).
- **All new fixtures are synthetic** — generated fiction with committed generators
  (§3C), never anonymised-by-hand real data.
- The standing rules continue unchanged: `git add -A --dry-run` before every
  commit, nothing from `data/`, nothing over 1 MB (the demo dataset must respect
  this too).

---

## 5. Excluded, and stays excluded

- **Legacy Timeline formats — cut at Gate 1, to the backlog, behind an evidence
  trigger.** Semantic History (`timelineObjects`) and `Records.json` were in PLAN's
  phase 5 list; the Gate 1 review cut them as speculative work for users not yet
  observed — the same reasoning that defers HEIC ("real blocked uploads first"),
  applied consistently. The current phone export carries the user's full history
  (the maintainer's own eight-year archive is one phone export), so the population
  needing legacy parsers is only those whose history survives *solely* in old
  Takeout archives — real (the 2024 on-device migration purged server-side data
  for users who missed it) but invisible until one appears. The sniffer already
  identifies both formats by name with an actionable message, and migration 00007
  records the sniff's format label per import (`detected_format`, §3B), so a
  blocked user is a queryable count, not an anecdote; that count is the trigger.
  Stay-point synthesis and
  home-derivation-without-visits go with the parsers — nothing remaining in this
  phase needs them. The sniff rejection wording drops "planned for a later
  release" for plain "not supported", and the README states the supported input
  precisely.
- Multi-user, auth, tenancy, rate limiting — out of charter (comparison §2). The
  compose deployment is single-user and published on loopback only; anyone exposing
  it beyond localhost puts their own reverse proxy with auth in front and widens
  the binding deliberately (§3A), and the README says exactly that.
- TLS termination — the reverse proxy's job, not this repo's.
- A hosted/SaaS offering, telemetry, update checks, or any phone-home.
- Watched-folder auto-import (comparison doc's deliberate omission: marginal for a
  few-times-a-year operation).
- HEIC decoding — carried trigger unchanged: real blocked uploads first.
- Photos as a primary import source — keeps its phase-4 carried-forward status:
  its own brief, in the phase that takes it up, where the synthesis machinery the
  legacy formats also need will be designed once, on evidence.
- GPX and the rest of the post-phase-5 backlog — unordered backlog, untouched.
- Kubernetes, queues, Redis, image registries, CI/CD pipelines — the deliverable is
  `docker compose up` on one machine, and CLAUDE.md's do-not-add list stands.
- The §8.2 parked direction — parked, trigger conditions unmet, not reopened here.

---

## 6. Checkpoint order — four slices, each visible, each a STOP

1. **The containers, and import visibility.** Both Dockerfiles, `.dockerignore`s,
   `compose.yaml` (db, api, web; loopback publish; healthchecks and
   `service_healthy` ordering; migrate-then-serve; volumes per §3A),
   `.env.example`, standalone Next output, `GET /healthz`; migration 00007,
   `GET /imports`, and the imports page. *Visible: from a clean checkout, `docker
   compose up` comes up ordered and healthy; import + detect of the private
   fixture via `docker compose run`; adventures browsable at `127.0.0.1:3000` —
   every byte served from containers; the imports page shows the import row, and a
   deliberately failed import shows `failed` with its actionable message.* STOP.
2. **Demo + README.** Demo generator and committed dataset; the root README
   rewritten with the quickstart against the demo. *Visible: a person following
   README top-to-bottom on a clean machine reaches adventures on a map without
   touching real data; every number in the README reproduces by its stated
   command.* STOP.
3. **Backup and restore.** `roadbook backup` / `roadbook restore` per §3D.
   *Visible: backup of the real instance; restore into a scratch compose instance;
   import + detect there; decisions re-attach and photos serve — the full
   restore-into-empty story demonstrated live.* STOP.
4. **The OSRM profile and operator script.** Compose profile + `osrm-setup.sh` per
   §3E, run once for real against the demo's small region. *Visible: the demo's
   sparse adventure before and after — chords become routed roads; the script's
   dataset name recorded in `route_runs`; `docker compose up` still clean without
   the profile.* STOP.

Phase close: the §7 verification pass, then LOG.md — the phase is not complete
until the log exists.

## 7. Verification at phase close

- Cold-cache `make test` green, DB-backed store tests running, both goldens
  running ungated.
- Fixture detection still 18/1/32; archive expectations unchanged.
- The README quickstart executed end-to-end from a clean checkout (fresh volumes),
  demo numbers reproduced by their stated commands.
- Backup/restore round-trip demonstrated on real data (restore target: scratch
  instance, then discarded).
- `docker compose config` clean; images build from a clean context; a `data/`-file
  canary confirms the build context excludes it; the web port answers on
  `127.0.0.1` and not on the host's LAN address.
- `git add -A --dry-run` clean of `data/`, of `.env`, of backup archives, and of
  anything near 1 MB.
