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

Do not start step 3 before step 2 returns real data over HTTP. The extra layer this
architecture adds is worth paying for only if it is proven working before anything is
built on top of it.

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

**Decisions and their motivation**

*Self-hostability is tested by doing it*, not by intending it. Any assumption about a
specific user or region surfaces the first time someone else's export is imported.
