# Phase 1 design brief — candidates on screen

Status: draft for review. No application code exists; none is written until this brief
is agreed.

Goal, features, and the four build checkpoints are in `docs/PLAN.md` and are not
restated. This brief covers: the concepts the phase introduces, what gets built and
where, the schema — including candidate identity, the hard problem of the phase — and
the real choices with recommendations.

---

## 1. Concepts this phase introduces

### Go: the pipeline shape

**`internal/` packages.** Go treats any package under a directory named `internal/` as
importable only from within this module. The pipeline lives there — `internal/timeline`,
`internal/detect`, `internal/store`, `internal/api` — because nothing outside this
repository should ever depend on its internals; the public surface is the HTTP API and
nothing else.

**A pure core package.** `internal/detect` exports (approximately)
`Detect(obs Observations, p Params) Result` — a function whose output depends only on
its arguments. No database handle, no clock, no file reads; it does not even log. The
CLI reads files and the store writes rows; `detect` only computes. This is invariant 1
made physical: the package's import list is the proof (nothing but the standard
library's pure parts), and it is what lets the reference detector's output become a
plain unit test instead of an integration test needing Postgres.

**Table-driven tests.** The idiomatic Go pattern: a slice of case structs — name,
input, expected — iterated in one test function with `t.Run(name, …)`. Detection is
exactly the shape this fits: many small scenarios (a span with no dwell, a point
faster than 900 km/h from one neighbour but not both, a home-to-home corridor). Each
regression from the prototype era becomes one more row, and the fixture comparison is
one case whose expected value is the reference detector's full output.

**`goose` migrations.** Numbered SQL files under `migrations/` applied in order;
goose records progress in a `goose_db_version` table so applying is idempotent. The
schema is therefore plain SQL you wrote, in git, with history — not the emitted
artifact of an ORM. Rejected alternative: any tool that generates SQL from code
annotations; the schema is the most durable interface in the system and deserves to be
written, not emitted (CLAUDE.md, "do not add").

**`pgx` without an ORM.** `internal/store` is hand-written SQL through the `pgx`
driver: one Go function per query, scanning into domain types. Cost: some repetition.
Benefit: every query is visible, explainable, and tunable, and the store is a thin,
boring layer — which is what you want at a boundary you'll debug at 11pm.

### The OpenAPI contract

**What `openapi.yaml` is.** A single YAML document describing every endpoint: paths,
request and response schemas, error shapes, status codes. It is written by hand and is
the only hand-written definition of the API (invariant 10).

**How both sides are generated.** `oapi-codegen` reads the spec and emits a Go
interface — one method per operation, with typed request and response structs. Our
handlers implement that interface, so the *compiler* enforces that the server matches
the spec: change the spec, regenerate, and every handler that no longer conforms fails
to build. `openapi-typescript` reads the same spec and emits TypeScript types; a thin
typed fetch wrapper (`openapi-fetch`) uses them so a frontend call to a wrong path or
with a wrong body is a type error at build time, not a surprise at runtime. Generated
files are committed (so builds don't depend on regeneration) but never edited; a
`make generate` target regenerates both sides from the spec.

**Why this ordering matters.** The spec is written first, reviewed like code, and only
then implemented. Drift between the two languages is impossible by construction — the
entire justification for the API architecture rests on this.

### Postgres: the derived/user split

The schema divides into three strata with different lifecycles — this is the
organising idea of §3 below. **Observations** are immutable inputs: written once at
import, never updated or deleted (invariant 2). **Candidates** are derived and
disposable: every detection run regenerates them, and deleting all of them loses
nothing that a re-run cannot restore. **Decisions** are user data: never regenerated,
never deleted by any automated process. Idempotent upsert (`INSERT … ON CONFLICT DO
NOTHING`) is what makes re-import safe: rows already present are silently skipped, so
importing overlapping exports accumulates rather than duplicates.

### Next.js, React, TypeScript

**Server components and client components.** In the App Router, components are
server-only by default: they run during the request on the server, may be `async`, and
their code never ships to the browser. A file opting in with `'use client'` becomes a
client component: it ships to the browser and may hold state and handle events. The
candidate list page is a server component that calls `GET /candidates` on the Go API
and renders rows; only the decide controls (buttons, the name input) are a client
component. Why: the page is mostly static per request, and the less JavaScript sent,
the less there is to go wrong — interactivity is opted into per-island, not paid for
everywhere.

**Server actions.** A function marked `'use server'` that the client invokes like a
normal async call; Next serialises the invocation to the server, where the function
runs — here, calling `POST …/decision` on the Go API. This is how writes keep the
"browser talks only to Next" rule with no hand-rolled API route between. Rejected
alternative: Next route handlers proxying to Go — more moving parts for the same hop,
and actions are the framework's intended mechanism for mutations. (Next 16 postdates
training data: `node_modules/next/dist/docs/` is read before writing any of this, per
CLAUDE.md.)

**Optimistic update.** React's `useOptimistic` renders the expected post-action state
immediately (row shows "confirmed" the moment the button is pressed) while the action
runs; if the action fails, React reverts to the server state automatically. The row
responds instantly without pretending the network doesn't exist.

**Controlled inputs.** The confirm-name field holds its text in React state
(`value` + `onChange`) rather than in the DOM, so the component — not the browser — is
the single source of truth for what's typed. That is what makes "disable confirm until
a name is non-empty" a one-line derivation instead of DOM spelunking.

**Discriminated unions for decision state.** In strict TypeScript:

```ts
type DecisionState =
  | { status: 'undecided' }
  | { status: 'confirmed'; name: string }
  | { status: 'dismissed' };
```

Checking `state.status === 'confirmed'` *narrows* the type — `state.name` exists in
that branch and is a compile error in the others. Illegal states (a dismissed
candidate with a name) are unrepresentable. This is the pattern strict mode exists to
enable, and it appears wherever the frontend models decision state.

---

## 2. What gets built, and where

```
api/openapi.yaml            the contract (hand-written)
cmd/roadbook/               single binary: import · detect · serve · probe
internal/timeline/          parser: Timeline JSON → domain types (invariant 4)
internal/detect/            pure detection + candidate/decision matching
internal/store/             pgx persistence, hand-written SQL
internal/api/               HTTP handlers implementing the generated interface
migrations/                 goose SQL files
web/                        Next.js app (list page, decide island, server actions)
```

One binary with subcommands rather than several binaries: one artifact to build, ship,
and version, and `roadbook serve` / `roadbook import` reads naturally in a compose
file. `probe` reports observed JSON key paths and their frequencies — it exists
permanently because Google changes the export schema without announcement.

Build order and checkpoints are fixed in `docs/PLAN.md`: CLI → Postgres + API proven
with `curl` → read-only list → decide actions. The frontend does not start until the
API returns real data over HTTP.

---

## 3. Schema

Three strata, per the lifecycle split above. All timestamps UTC (`timestamptz`), all
coordinates `double precision` degrees.

```
imports          id · source_label · window_start · window_end · imported_at · counts
visits           id · start_ts · end_ts · lat · lon · semantic_type · content_hash (unique)
activities       id · start_ts · end_ts · start/end lat,lon · distance_m · mode · content_hash (unique)
path_points      id · ts · lat · lon · content_hash (unique)

detection_runs   id · ran_at · params (jsonb) · window_start · window_end
candidates       id · run_id → detection_runs · seq · span_start · span_end
                 · start_truncated · end_truncated · dest_lat · dest_lon
                 · dest_km · track_km · stop_count · repeat_count · obs_count

decisions        id · user_id · action ('confirmed' | 'dismissed') · name
                 · anchor_span_start · anchor_span_end · anchor_dest_lat · anchor_dest_lon
                 · created_at · updated_at
```

Notes, briefly, before the two hard parts:

- **Observations are three typed tables**, not one polymorphic table: the three kinds
  share almost no columns, and typed tables let the store scan directly into the
  domain types the parser emits. Rejected: a `kind` column with nullable
  everything-else — every query would re-discriminate what the schema already knew.
- **`content_hash`** is a SHA-256 over the parsed record's canonical form (UTC
  RFC 3339 times, fixed-precision coordinates) with a unique index; import is
  `ON CONFLICT DO NOTHING`. See choice 3 in §4.
- **`params` is `jsonb`**, serialised from the Go `Params` struct, not one column per
  threshold: parameters will be added over time (invariant 3 guarantees tuning), and a
  run's record must hold exactly the set that produced it without a migration per new
  knob.
- **Candidates keep their run**; old runs' candidates are retained (tens of rows), so
  two runs can be diffed. The UI reads the latest run.
- **`user_id` on `decisions`** always holds the same value — the multi-user
  accommodation from `docs/PRODUCT.md`, and nothing more.
- **No `adventures` table.** A confirmed adventure *is* a decision with
  `action = 'confirmed'` and a name. A separate table would hold nothing that isn't
  already on the decision or derivable from the matched candidate.

### 3.1 Candidate identity — the hard problem

**The requirement.** A decision made on a candidate must still apply after
re-detection with different parameters, when every candidate row has been regenerated.
Getting this wrong means the user re-triages the entire list after every parameter
change — the specific failure this phase exists to prevent.

**Why the obvious schemes fail.** Any *computed key* — round the destination to a
grid cell, take the start date, hash the pair — has quantisation cliffs: a parameter
change that moves the destination a few hundred metres across a cell edge, or the
span start across a date boundary, silently changes the key, and the decision is lost
without any signal that it was lost. The failure mode of a computed key is exactly the
failure it exists to prevent, just rarer.

The plan's suggested basis — destination region plus approximate start date — has a
second, structural problem: **the start date is precisely what boundary truncation
changes.** The fixture's own first candidate demonstrates it (see 3.2): re-importing
with an earlier window start extends the span backwards by days. Any key that
incorporates the start would break on the very case bug 4 describes. This brief
therefore departs from the plan's suggestion deliberately.

**Proposal: decisions are anchored, and identity is matching, not a key.**

A decision stores its own *anchor* — the span and destination of the candidate as the
user saw it when deciding (`anchor_span_start/end`, `anchor_dest_lat/lon`). Candidates
carry no identity beyond their row. The association between the current run's
candidates and the decisions is *recomputed* by a pure, deterministic matching
function (`internal/detect`, tested table-driven like everything else there):

1. A candidate and a decision are *compatible* if their time spans overlap at all and
   their destinations are within `MATCH_KM` (default 50, a named parameter).
2. Score every compatible pair by temporal overlap duration.
3. Assign greedily, highest score first; each candidate and each decision is used at
   most once. Ties break by earlier span start, then earlier `created_at` —
   deterministic, so identical inputs give identical matchings (invariant 1 applies).

Time overlap is the load-bearing signal: away-spans are disjoint in time by
construction, and parameter changes move boundaries by hours against spans lasting
days. Destination distance is the guard against a reshaped span being claimed by the
wrong trip. Repeated trips to the same destination — common in the real data — cannot
collide, because they occupy different time ranges; a destination-only key would
conflate them.

The API serves the matched pair: `GET /candidates` returns each candidate with its
decision state attached, plus any *orphaned* decisions — decisions no current
candidate matched. A new decision (`POST /candidates/{id}/decision`) snapshots the
candidate's current span and destination as the anchor; re-deciding updates the
decision and refreshes its anchor to the current candidate.

**What breaks it — stated honestly:**

- **Orphaning.** Raise `FAR_KM` enough and a decided candidate stops being detected
  at all; its decision matches nothing. The decision is *not lost* — it is returned as
  orphaned, and lowering the parameter re-matches it — but phase 1's UI must at
  minimum count orphans rather than hide them.
- **Splits.** A parameter change splits one span in two. The old decision attaches to
  the better-overlapping half; the other half appears undecided and asks for triage
  again. Defensible (it is genuinely a span the user never separately judged), but it
  is re-work.
- **Merges.** Two decided spans merge into one candidate. One decision attaches; the
  other goes orphaned. If the two decisions disagreed (one confirm, one dismiss),
  which one attaches is decided by overlap score, not by user intent.
- **Drastic reshaping.** A span whose destination moves beyond `MATCH_KM` (e.g. a
  changed `MIN_DWELL_MIN` disqualifies the old farthest-dwelt visit) orphans its
  decision even though the human would say it's the same trip.
- **`MATCH_KM` is itself a parameter.** Changing it changes associations. No scheme
  escapes this; this one at least records it.

Every failure mode degrades to *visible extra triage*; none silently destroys a
decision. That asymmetry — computed keys fail silently, matching fails loudly — is
the reason for the recommendation.

### 3.2 Boundary truncation — resolving bug 4

**The problem.** An import window cuts through whatever journey was in progress at
its edge. The detector then reports a span that starts (or ends) at the data boundary,
understating the real journey — the development fixture's first candidate is exactly
this case. Two candidate resolutions were left open: walk backwards past the cut until
a home visit is found, or mark the candidate incomplete.

**Recommendation: mark, don't walk.** Detection receives the window bounds as part of
its input and sets `start_truncated` on a candidate whose away-span begins with no
home-visit observation before it in the window (symmetrically `end_truncated`). The
UI renders it honestly: "began before the imported window."

Why walking backwards is rejected: at the export's own edge there is nothing to walk
into — the data before the boundary does not exist, and no algorithm can recover it.
Where the truncation is caused by a user-chosen import window narrower than the file,
the remedy already exists: widen the window and re-import — and the matching scheme in
3.1 is what makes that safe, because the extended span still overlaps the anchor and
the decision follows automatically. Walking backwards would thus be extra machinery
for a case re-import already handles, and no machinery at all for the case it can't.

This is also the schema impact that made bug 4 undeferrable: two booleans on
`candidates`, set by pure detection — and the identity scheme had to be chosen knowing
span starts are unstable, which 3.1 is.

---

## 4. The real choices

**1. Candidate identity: anchored matching vs computed stable key.** Recommendation:
matching, per 3.1. A computed key is simpler (one column, one join) and its failures
are silent decision loss; matching costs a scoring pass over tens of rows at read time
and its failures are visible triage. Chosen for the failure mode, not the elegance.
Would change my mind: if decided candidates routinely churned through split/merge in
normal use, the anchor model would be adding ambiguity rather than absorbing parameter
noise — that would argue for user-visible explicit re-linking instead.

**2. Truncation: mark incomplete vs walk backwards.** Recommendation: mark, per 3.2.
Would change my mind: evidence that user-narrowed windows (rather than the export
edge) are the common case *and* that re-import is too slow to be the remedy.

**3. Import idempotency: content hash vs natural keys vs wipe-and-reload.**
Recommendation: `content_hash` unique + `ON CONFLICT DO NOTHING`. Natural composite
keys (timestamps + coordinates) are fragile against precision and schema drift in the
export; wipe-and-reload would make each import destroy the accumulation of previous
exports — fatal, because banking each export's expiring high-accuracy window *is* the
archival strategy. The hash makes "already imported" a property of the record itself.
Would change my mind: evidence that Google re-emits the same semantic record with
changed field values (same visit, revised coordinates) — that would need
last-write-wins on a natural key instead, a real trade-off to bring back here.

**Defaults taken without full argument** (flag any for discussion): single binary with
subcommands · three typed observation tables · `params` as `jsonb` · decisions
updated in place (re-deciding overwrites; no undo history in phase 1) · list ordered
by span start, newest first, with all ranking metrics displayed but no composite score
yet — inventing score weights before living with the list would be tuning blind ·
API is read + decide only (`GET /candidates`, `POST /candidates/{id}/decision`,
`GET /healthz`); import and detection stay CLI-only in phase 1.

---

## 5. Done when

Per `docs/PLAN.md`: every candidate the reference detector finds in the fixture
appears on a page; each can be confirmed with a name or dismissed; re-running import
and detection with different parameters leaves every decision intact — now given
teeth by 3.1: *intact* means matched or visibly orphaned, never silently gone.
