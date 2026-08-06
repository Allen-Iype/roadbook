# Phase 5 decisions

Three lines each: what was chosen, what was rejected, what would change our
mind. Written as decisions are made. The brief's §3 recommendations were
approved at Gate 1 (2026-08-06) with three amendments and one scope cut,
recorded below; entries after them record decisions made during
implementation.

## 2026-08-06 — Gate 1: web publishes on loopback only

Chosen: `compose.yaml` publishes the web port as `127.0.0.1:3000:3000`, so a
fresh install answers only on the machine itself; exposing it further is a
deliberate edit, stated in the README beside the reverse-proxy sentence.
Rejected: `3000:3000` with a README warning — a bare port binds all
interfaces, which on a VPS internet-exposes an authless product the second it
starts; a warning is not a default.
Would change our mind: the product growing an auth boundary of its own —
out of charter (comparison §2), so effectively nothing.

## 2026-08-06 — Gate 1: healthchecks and declared startup order

Chosen: `pg_isready` on db, `GET /healthz` on the API (in `openapi.yaml`,
per invariant 10; answers 200 only when a database round-trip succeeds),
and `depends_on: condition: service_healthy` chaining db → api → web, so
the first `docker compose up` comes up ordered, never racing.
Rejected: no healthchecks (a racing first start is the worst first
impression for a deploy phase), and a bare-TCP check on the API (a
listening socket with a dead database behind it is exactly the not-ready
state the check exists to catch).
Would change our mind: nothing foreseeable — the cost is a few lines of
compose and one trivial endpoint.

## 2026-08-06 — Gate 1: legacy Timeline formats cut to the backlog on an evidence trigger

Chosen: Semantic History and `Records.json` parsers — and with them
stay-point synthesis and home-derivation-without-visits — leave phase 5 for
the backlog, behind the trigger "a real export that fails to import"; the
phase supports the current phone export only and the README says so; the
sniff rejections drop "planned for a later release" for plain "not
supported". PLAN updated.
Rejected: building both parsers speculatively — the same evidence standard
that defers HEIC applies (no observed blocked user), and the current phone
export carries full history, narrowing the affected population to those
whose past survives solely in old Takeout archives.
Would change our mind: the trigger firing — a real user with a
legacy-only archive, visible because the sniffer names both formats in its
rejection message; restart then from the comparison doc §1.6 (the
stay-point sweep), the phase-4 carried-forward note on synthesis, and the
design conclusions preserved in PLAN's backlog entry (synthesis as a pure
derived pass at detection time, never persisted; home evidence generalised
with strict semantic precedence so existing outputs stay byte-identical).

## 2026-08-06 — Gate 1: import bookkeeping stays, folded into checkpoint 1

Chosen: migration 00007 (`status`, `error` on `imports`), `GET /imports`,
and the `/imports` page remain in the phase — they shared a checkpoint with
the cut parsers but are a separate PLAN feature the scope challenge did not
touch — and land in checkpoint 1, where import-in-compose is demonstrated
and a failed import must be visible in the product.
Rejected: cutting them with the checkpoint they happened to share, and
making them their own checkpoint (four checkpoints was the agreed shape;
bookkeeping is small and belongs where import is proven).
Would change our mind: checkpoint 1 growing unwieldy in practice — the
bookkeeping slice moves to checkpoint 2 without redesign.

## 2026-08-06 — Gate 1: `detected_format` on imports is what makes the legacy trigger observable

Chosen: migration 00007 also adds `detected_format` (nullable text), the
sniff's stable format label, written on every import, successful or failed —
so the legacy-formats backlog trigger ("a real export that fails to import")
is a query (`select detected_format, count(*) from imports where
status = 'failed' group by 1`), a count rather than an anecdote.
Rejected: relying on the `error` message as the evidence store — the
user-facing wording is being reworded in this very phase, and evidence
kept inside prose breaks the moment the prose improves; also rejected:
recording the format only on failure (the label costs nothing on success
and dates when each format was last seen working).
Would change our mind: nothing foreseeable — it is one nullable column
whose absence would make the phase's own scope cut unmeasurable.

## 2026-08-06 — checkpoint 1: the Postgres image is imresamu/postgis:18-3.6, everywhere

Chosen: `imresamu/postgis:18-3.6` for both compose.yaml's db and
compose.test.yaml's testdb — the multi-arch (amd64+arm64) build of the
official postgis image that the postgis/postgis README itself points ARM
users at; same Postgres 18 + PostGIS 3.6 as the rest of the project, one
image name across test and deploy.
Rejected: `postgis/postgis:18-3.6` (publishes no arm64 — discovered when
the pull failed on this Apple Silicon machine; it would exclude every ARM
self-host and had silently broken the test-database fallback on ARM,
unnoticed because the usual dev machine has local Postgres), and
`ghcr.io/baosystems/postgis` (also multi-arch, but imresamu is the
official README's own pointer and tracks the upstream Dockerfiles).
Would change our mind: postgis/postgis publishing arm64 manifests — then
the official image wins and this is a two-line revert.

## 2026-08-06 — checkpoint 2: the demo file is testdata/demo/demo.json, not *.timeline.json

Chosen: the committed demo output is named `demo.json`, diverging from the
BRIEF's illustrative `demo.timeline.json`, because the root `.gitignore`
carries a belt-and-braces `*.timeline.json` pattern that would swallow the
committed file.
Rejected: a `!testdata/demo/demo.timeline.json` negation — it punches a
hole in the pattern that exists because anchored-pattern subtleties caused
one of the two data near-leaks; the safety pattern stays intact and the
file is named around it.
Would change our mind: nothing — the name communicates less than the
pattern protects.

## 2026-08-06 — checkpoint 2: demo composition is pinned by what the pipeline actually consumes

Chosen: Iceland (one small Geofabrik extract covers everything, per the
brief); the flights route through Keflavík, not the Reykjavík city
airport — the city airport sits inside NEAR, so its flights fall outside
the away-span and the journey window, and no arc would ever render; each
flight is bracketed by a stationary trace-point pair at the airports,
because journey assembly draws legs from trace points and raw positions,
not visits or activity endpoints — without a gate fix the flight is a
30-hour slow silence, not an air-speed gap. All of it is pinned by the
ungated `internal/detect/demo_test.go` (3 candidates / 1 base / 0
outliers, parse counts, per-candidate destinations), which is what makes
the README's numbers regression-held rather than aspirational.
Rejected: relocating the persona's home to keep the city airport (breaks
commute realism for one flight), and stating README numbers without a
pinning test (invariant 13 by promise instead of by test).
Would change our mind on the gate fixes: journey assembly learning to use
activity endpoints as observations — a core change no demo should force;
the demo then simplifies to match.
