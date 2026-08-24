# Phase 10 — hosting readiness: phase log

Written at phase close, 2026-08-24. The brief is BRIEF.md (its Gate 1
addendum is the governing record); decisions are in DECISIONS.md, written
as they were made. Private operational detail — credentials, addresses,
measurements per instance — lives in `docs/private/pilot/` (LEDGER.md,
RUNBOOK.md, ORACLE-CP1A.md), which this log deliberately does not repeat.

## What the phase produced

The pilot moved off the maintainer's laptop onto a rented host, with the
laptop reduced to a nightly pull of encrypted backups. Concretely:

- **A hardened host** (Oracle Cloud, India region): public SSH closed at
  two firewall layers, operator access over the tailnet only, ports 80/443
  the sole public surface, Docker with the loopback-publish convention
  carried over from phase 5. The host runs on **trial credits at zero
  cash** — the phase's paid plan was reverted mid-gate when spending
  became impossible (DECISIONS 2026-08-20), and the free-tier target shape
  (A1.Flex 2 OCPU/12 GB) turned out to be capacity-starved in the home
  region, so a paid shape of the same size bridges on credits while an
  automated retry (**the A1 sniper**, a systemd timer on the host itself)
  hunts for the free instance. The bridge is mortal by design: the trial
  ends in September and the exits are recorded.
- **A wildcard TLS front without buying a domain**: a free dynamic-DNS
  name, Caddy built with the DNS provider's plugin, one wildcard
  certificate via DNS-01 (Certificate Transparency sees only the
  wildcard), one host-matched `handle` block per instance with per-tester
  basic auth, unknown subdomains answering a bare 404. Links survive
  re-homing: the name re-points, the certificate follows.
- **The pilot scripts, re-shaped for the host topology**
  (`scripts/pilot/host/`): stamp, reset/retire, rotate, backup, restore,
  plus the nightly backup units and the laptop-side pull script. The
  laptop flavor of the scripts was deleted at the end of the phase — a
  dated two-flavor transition, not a fork, exactly as decided at CP2.
- **A measured capacity statement** (CP2): ten scratch instances stamped,
  loaded, and measured on the 12 GB host — roughly a hundred megabytes a
  stack, memory not the binding constraint at ten; the verdict that the
  free 2/12 shape holds the pilot is recorded with the numbers in the
  private ledger.
- **Fresh instances instead of migration** (CP3, reshaped at the CP2
  STOP): nothing on the laptop was worth moving — a handful of redoable
  confirmations — so every tester got a fresh stamp on the host and the
  proven backup/restore pipe was kept for what it is: the re-homing
  mechanism, not a ceremony.
- **A durability chain that runs without anyone** (CP4): nightly
  encryption on the host to a public key whose secret half has never
  touched it; a nightly pull to the maintainer's machine into the
  existing iCloud chain; a restore drill executed from the *pulled* copy;
  a reboot drill with every stack returning unattended; the laptop front
  retired by checklist.

## What broke, and why each fix took the form it did

**The free tier itself.** Oracle halved the Always Free ARM allowance
unannounced mid-2026; the maintainer's challenge caught the assistant
working from stale knowledge, and the corrected arithmetic (2 OCPU/12 GB
continuous, not 4/24) reshaped the target. Capacity for that shape was
then simply unavailable for days. The fix is an automated 15-minute retry
on the host rather than manual console attempts, because two days of
manual attempts had already failed and the laptop could not be the retry
vehicle without reintroducing the lid-open dependency this phase exists
to remove.

**Caddy could not read the pilot tree — and validation said it could.**
The systemd Caddy runs as its own user, which cannot traverse
`/home/ubuntu`; a glob `import` over unreadable files matches *nothing*,
silently, so every subdomain fell through to the 404 fallback while
root-run `caddy validate` reported a valid configuration. The external
curl check caught what validation could not. The fix moved pilot state to
`/srv/pilot` with explicit per-file permissions rather than loosening the
home directory, and the lesson is procedural as much as technical:
validate as the service user, and trust only the outside view.

**Docker Desktop had been hiding a uid mismatch.** The api container runs
as a nonroot user; on macOS, Docker Desktop's file sharing papered over
ownership, but Linux bind mounts are literal, so the backup staging
directory (0700, host user) was unwritable from inside the container. The
scripts now set explicit modes on their throwaway staging directories —
a two-line fix once seen, invisible until the first Linux run.

**Records disagreed with machines, twice.** A VCN rename recorded as done
had never stuck (three identically-named networks; the keeper is now
identified by OCID, never by name), and a scratch stack recorded as torn
down was found running on the laptop days later — most likely a restart
policy resurrecting it after a reboot. Both fixes were small; the lesson
is standing: a record of an action is not the action, and cleanup obeys
identifiers, not names.

**The archive's outlier count moved.** The maintainer's full archive,
uploaded through the browser path, detected the reference 66 candidates
but reported fewer outliers than the file-mode reference. Same file,
byte-identical parse counts — the difference is import dedupe collapsing
a few thousand byte-identical rows before detection, which shifts the
neighbor pairs the speed-based outlier rule sees. Laptop and host agree
exactly; the file-mode regression still pins the reference number. This
is the phase-5 file-vs-DB dedupe trap wearing a new face, now recorded so
the next reader does not re-derive it.

**Unattended tailnet operations meet the re-auth check.** Tailscale SSH
periodically requires the operator to re-authenticate in a browser; a
long-running command or an unattended pull hits it and waits. Accepted
with logging rather than engineered around: archives accumulate on the
host between successful pulls, and any interactive session clears the
check.

## What the pilot taught while the phase ran

The hosted front produced the project's first real user evidence, all
recorded privately and already sorted into the plan: the first
non-operator completed the entire v1 loop unassisted; one-by-one
confirmation measurably stalls triage at full-archive scale (the friction
is now a named backlog item with in-system remedies, and the reported
auto-confirm alternative is recorded as the product-level decision it
would be); and none of the pilot's iPhone users had Timeline data at all
— which fires the roadmap's pre-decided resequencing clause in favor of
the ingestion phase, and makes the audit message that was to establish
this fact unnecessary.

## Close state

- All checkpoints done: CP1a/CP1b (bridge host + zero-cash front), CP2
  (scripts + ten-stack measurement), CP3 (reshaped to fresh instances),
  CP4 (durability + laptop retirement).
- Zero product-code diff across the phase: `git diff` over `cmd/`,
  `internal/`, `web/`, `go.mod`, `go.sum`, `api/`, `compose.yaml`,
  `Dockerfile` from the charter amendment to close is empty, as targeted.
- Cold `make test` green at close (Go suites including DB-backed store
  and backup tests, both goldens, demo and archive regressions, web
  vitest).
- Standing threads that outlive the phase: the A1 sniper and the trial
  migrate-by date; the PAYG conversion watch; one tester's return
  awaiting a fresh stamp. The next chartered decision — ingestion before
  front gate, per the evidence — belongs to its own session and brief.
