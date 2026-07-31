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

**Expected regression values live in `data/`, produced by the prototype itself**
(`fixture-candidates.json`; `archive-candidates.json` from a scratch copy pointed at
the archive). Committed tests skip when `data/` is absent.
Rejected: embedding expected values in committed test code — they derive from
private data (CLAUDE.md invariant 14 beats test self-containment).
Reconsider if: an anonymised fixture can be constructed that exercises multi-home
detection; then a committed expected file becomes possible.
