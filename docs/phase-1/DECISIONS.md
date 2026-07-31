# Phase 1 decision log

Three lines per decision: chosen, rejected, what would change our mind. Written as
decisions are made, not reconstructed.

**Candidate identity is anchored matching, recomputed per run** (BRIEF §3.1).
Rejected: any computed stable key (grid cell + date, content hash of span).
Reconsider if: decided candidates churn through split/merge in normal use, which
would argue for explicit user-visible re-linking instead.

**Boundary truncation is marked, never repaired** (BRIEF §3.2): `start_truncated` /
`end_truncated` on the candidate.
Rejected: walking backwards past the window cut to a home visit.
Reconsider if: user-narrowed windows become the common case and re-import proves too
slow as the remedy.

**Import idempotency via `content_hash` unique index + `ON CONFLICT DO NOTHING`.**
Rejected: natural composite keys (precision-fragile); wipe-and-reload (destroys the
accumulated monthly exports that bank expiring `rawSignals`).
Reconsider if: Google re-emits the same record with revised values — that needs
last-write-wins on a natural key.

**Go module path is bare `roadbook`.**
Rejected: guessing a `github.com/...` path before the repo has a public home.
Revisit when: the repository is published; the rename is mechanical.

**Domain types live in `internal/domain`, not in the parser or detector package**
(one addition to the BRIEF §2 layout).
Rejected: types in `internal/timeline` (couples detection to one source format);
types in `internal/detect` (store and API need them too, and a parser importing the
detector reads backwards).
Reconsider if: the domain package starts accumulating behaviour — it must stay types.

**The detector port replicates Python semantics, not Go idiom** (`pyport.go`):
banker's decimal rounding via `strconv`, stable sorts, civil dates in each
timestamp's own UTC offset, bisect's exact midpoint arithmetic.
Rejected: idiomatic Go equivalents (`math.Round`, `sort.Slice`, UTC dates) — each
diverges from the prototype in edge cases and breaks bit-for-bit regression parity.
Reconsider if: the prototype is ever retired as the reference; then Go becomes the
definition and the shims can go.

**A truncated span is one that touches the window edge**: `start_truncated` ⇔ the
span begins at the first surviving observation, symmetrically for the end.
Rejected: scanning for a preceding home visit — equivalent on real data, more code.
Reconsider if: a real export produces a window whose first observation is away yet
the journey demonstrably began at home inside the window.

**No derivable home base returns an empty result, not an error.** The reference
implementation crashes; deliberate divergence, since "away from home" is undefined
without a home.
Rejected: panicking or returning an error from the pure core.
Reconsider if: the CLI needs to distinguish "no data" from "no home" — surface it in
Result then.

**Timestamps are stored as `timestamptz` plus a `*_offset_sec` column.**
Rejected: `timestamptz` alone (forgets the writer's UTC offset, and home-base eras
take civil dates in each timestamp's own offset — a DB round-trip would shift
late-evening dates and break file/DB detection parity); `timestamp without time
zone` (loses the instant instead).
Reconsider if: detection ever stops using offset-local civil dates.

**Migrations are embedded in the binary; `roadbook migrate` applies them.**
Rejected: requiring the goose CLI as an external tool — a self-hoster's deployment
could then drift from the schema its binary expects.
Reconsider if: migrations need to run against a database the binary shouldn't reach.

**Observations load in `(start_ts, id)` order.**
Rejected: storing an explicit source-file sequence column. Insertion order breaks
timestamp ties identically to file order within one import, and the regression
tests hold through the DB round-trip, which is the actual requirement.
Reconsider if: a DB-path regression run ever differs from the file path — then ties
across overlapping imports are the suspect and a sequence column is the fix.

**Re-deciding updates the matched decision in place and refreshes its anchor.**
Rejected: append-only decision history — nothing needs it yet, and the update is
the user overwriting their own user-data, which invariant 2 does not protect.
Reconsider if: an undo feature is wanted; then an audit table, not a flag.

**A stale candidate id (from a superseded run) is a 404, not a best-effort match.**
Rejected: silently resolving to the nearest current candidate — deciding the wrong
span quietly is exactly the class of failure this phase exists to prevent.
Reconsider if: it never fires outside tests; the reload prompt may then soften.

**The candidates page is `force-dynamic`.**
Rejected: Next's default (`auto`), which prerenders the page at build time and
freezes the candidate list into the build output; also rejected (for now) the new
opt-in Cache Components model — one caching model at a time, and the simplest one
that is correct.
Reconsider if: the app grows pages that genuinely benefit from partial
prerendering; adopt `cacheComponents` deliberately then.

**Journey dates render by slicing the ISO string, never `new Date()`.**
Rejected: Date parsing, which shifts the civil date into the viewer's timezone —
an evening departure viewed from the west shows the wrong day. The API string
carries the journey's own offset end to end; the substring is the truth.
Reconsider if: the UI ever needs date arithmetic, then a TZ-aware library, not Date.

**The API client module imports `server-only`.**
Rejected: convention alone. The package makes any client-component import of the
API client a build failure, which turns the architecture rule "the browser never
talks to the Go API" into a compile-time fact (the frontend twin of invariant 11).
Reconsider if: never; this guard is nearly free.

**The scaffold's Google-hosted font was removed.**
Rejected: keeping `next/font/google`, which made every production build depend on
an external fetch — a build-time network dependency nothing needs, and this
build's first npm install already failed once on this network.
Reconsider if: typography ever matters enough to self-host a font file.

**Expected regression values live in `data/`, produced by the prototype itself**
(`fixture-candidates.json`; `archive-candidates.json` from a scratch copy pointed at
the archive). Committed tests skip when `data/` is absent.
Rejected: embedding expected values in committed test code — they derive from
private data (CLAUDE.md invariant 14 beats test self-containment).
Reconsider if: an anonymised fixture can be constructed that exercises multi-home
detection; then a committed expected file becomes possible.
