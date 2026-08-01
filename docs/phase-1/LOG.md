# Phase 1 log — candidates on screen

Written at phase close. What the phase built, what broke while building it, and
why each fix took the form it did. Counts derived from the private dataset are
not stated here (see `docs/PLAN.md` on reproducibility); "the reference output"
below always means: what `prototype/detect_fixture.py` prints for the fixture in
`data/`.

## What the phase does

A Timeline export goes in; a decidable candidate list comes out; decisions
outlive everything derived.

- `roadbook import` parses an export as a stream (`internal/timeline`) and
  stores visits, activities, and path points idempotently
  (`internal/store`): each row carries a content hash, re-import inserts only
  what is new, and nothing is ever updated or deleted.
- `roadbook detect -from-db` runs detection (`internal/detect`, a pure
  function: no I/O, clock, or randomness) and persists a run — parameters,
  outlier count, derived home bases — plus its candidates, which are
  disposable by design and regenerated wholesale every run.
- `roadbook serve` exposes the latest run over HTTP per `api/openapi.yaml`;
  the Go server interface and the TypeScript client are both generated from
  that file and never edited.
- The web app (`web/`) renders the list in a server component and decides in
  a per-row client island: confirm-with-name or dismiss, optimistically
  applied, persisted through a server action that calls the Go API. The
  frontend has no database access and no hand-written API types.
- Decisions carry an anchor (span + destination as decided); every read
  re-associates them with current candidates by pure matching
  (`detect.Match`). Re-detection with different parameters therefore cannot
  destroy triage: a decision either re-attaches or surfaces as an orphan,
  never disappears.
- Unknown or wrong import files are rejected with a message naming what the
  file is and how to produce the supported export; truncated candidates at
  the window edge are marked, not repaired.

Verification at close: `make test` (pure-core tests and taxonomy tests run
anywhere; regression suites compare Go against the reference output when
`data/` is present and skip otherwise); the CLI reproduces the reference
output for both the fixture and the full archive; decisions were shown to
survive reload, re-import, and re-detection through the full UI stack.

## What broke, and why each fix took its form

**1. Go and Python disagree on four defaults, and each would have silently
broken parity with the reference detector.** Python's sort is stable, its
`round()` is banker's and correctly rounded, `.date()` uses the timestamp's
own UTC offset, and `bisect` has fixed probe arithmetic; Go's idiomatic
equivalents differ on all four. The fix is `internal/detect/pyport.go`:
deliberate ports of Python's semantics, commented against "simplification."
Form: parity shims rather than accepting drift, because the reference output
is the only regression anchor the port has — divergence would have made every
future refactor unverifiable.

**2. A regression test dropped an observation the algorithm was entitled to
drop.** The outlier rule discards a point impossibly fast from *all* its
neighbours; a point at the very edge of the window has only one neighbour, so
one impossible speed suffices there. Surfaced by a synthetic test whose
scenario ended mid-sequence. Form: fix the test, keep the rule — the edge
behaviour matches the reference implementation, and the test now documents it
instead of contradicting it.

**3. Postgres `timestamptz` forgets the writer's UTC offset, and detection
depends on it.** Home-base eras take civil dates in each timestamp's own
offset; a round-trip through the database returned UTC and would have shifted
early-morning home visits to the previous day, moving era boundaries.
Detection from the database would then disagree with detection from the file.
Form: an `*_offset_sec` column beside every timestamp, restored on load —
chosen over `timestamp without time zone` (which loses the instant instead)
and over accepting drift (which breaks the file/DB parity the regression
tests enforce).

**4. The frontend scaffold failed to install, then shipped a hidden network
dependency.** The first `npm install` died on a network timeout mid-scaffold;
retried with longer timeouts. The scaffold also fetched a Google-hosted font
at build time — a second network dependency in the build path of a
self-hostable project. Form: remove the font and use the system stack, rather
than retry harder; a build that needs the network for typography fails for
someone, eventually, for no benefit.

**5. Next's default caching would have frozen the candidate list into the
build.** Under the scaffold's caching model, a page whose fetch has no cache
options is prerendered once at build time; decisions would never have
appeared without a rebuild. Caught by reading `node_modules/next/dist/docs`
before writing Next code — the framework postdates the assistant's training
data, and its two caching models are exactly the kind of thing intuition gets
wrong. Form: `export const dynamic = "force-dynamic"` on the list page, and
the read-the-bundled-docs rule kept.

**6. The repository's ignore rules would not have covered the frontend.**
`.gitignore` had root-anchored `/node_modules` and `/.next`; the app lives in
`web/`, so neither would have matched — the same anchored-pattern class that
caused a location-data near-leak during project setup. Caught before
scaffolding, fixed by unanchoring. Form: fix-before-creating rather than
after, because the standing dry-run check only helps if the patterns are
right first.

**7. `go test ./...` started testing a dependency's vendored Go code.** A
JavaScript package inside `web/node_modules` ships a stray Go package, and
`./...` found it. Form: Makefile targets scoped to `./cmd/... ./internal/...
./migrations/...` — narrowing the build surface beats special-casing other
ecosystems' directory trees.

## What changed about the plan mid-phase

A comparative analysis of Dawarich (`docs/feature-comparison-dawarich-roadbook.md`)
landed between checkpoints 4 and 5 and was audited against the then-open
phase. Three of its schema-level recommendations were already satisfied by
the shipped design (decision durability via anchored matching; hash-based
idempotent import; dwell-based destinations). Two became checkpoint 5 —
the import failure taxonomy and streaming parse. Three were deferred with
logged reasoning: PostGIS (phase 2, countries table only — leg segmentation
stays in pure Go), confidence scoring (phase 2, after living with the plain
list), and legacy Takeout variants (phase 5, where their real users are).
The phase grew one checkpoint; nothing was reordered.

## Carried forward

- The gap `kind` field (`unknown | road | air`) must be designed into phase
  2's leg model from the start; phase 3 consumes it.
- `rawSignals` — the export's only high-accuracy positions — are recognised
  and skipped by the parser but not yet ingested; phase 2 route drawing wants
  them.
- The decide UI updates decisions in place; if re-deciding ever needs an
  undo, it needs a history table, not a flag (logged in DECISIONS.md).
