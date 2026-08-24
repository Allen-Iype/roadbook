# Phase 10 — decision log

Three lines per decision: chosen, rejected, what would change our mind.
Written as decisions are made, not reconstructed.

---

**2026-08-18 — Phase chartered.** Chosen: phase 10 = hosting readiness (the
VPS re-homing), first phase of the roadmap "The road to strangers"
(docs/PLAN.md), under the dual-mode charter amendment committed the same day.
Rejected: starting with the front gate (needs the durable host first) or the
ingestion phase (still gated on audit returns). Would change our mind: pilot
evidence of the v1 loop failing for product reasons — a fix-the-loop pass
would precede this phase's close.

**2026-08-18 — Brief drafted, Gate 1 pending.** BRIEF.md presents four real
choices: §3A provider/region (recommendation: India-region first), §3B
registrar/DNS (recommendation: Cloudflare DNS-only for DNS-01), §3C deployment
drive (recommendation: repo on host, no IaC), §3D backup destination
(recommendation: laptop-pull baseline), and §3E the cost ceiling (maintainer
sets the number — this phase explicitly retires phase 8's zero-cash stance).
No purchase or provisioning before the gate passes.

**2026-08-18 — Topology challenged at Gate 1 review; instance-per-user
reaffirmed.** Chosen: ten users = ten isolated compose stacks (own Postgres,
API, web) behind one Caddy — the pilot vehicle; shared-database multi-tenancy
is affirmed as the destination, scheduled as phase 12 shape (a) behind its
trigger (cap filled + waitlist demand). Rejected: building tenancy now — it is
the project's largest and riskiest code change (auth, a user filter on every
query where one miss leaks location history, GDPR lifecycle), and building it
before any stranger has asked to sign up buys nothing the pilot needs; the
per-stack overhead at n=10 is a few hundred MB each, linear, and priced within
§3E's ceiling. Would change our mind: credible evidence of dozens of committed
users arriving within a couple of months — then the waitlist is just a queue
for a known rewrite and phase 12 starts earlier; the maintainer raised and
accepted this reasoning at the review.

**2026-08-18 — Gate 1 PASSED; §3 decided (full record in BRIEF Gate 1
addendum).** Chosen: Oracle Always Free (India home region, pay-as-you-go
conversion) as primary host with Hetzner (~€5–9/mo, prices checked
2026-08-18) as pre-decided fallback; working-name domain deferred until the
free host proves out (CP1a/CP1b split — maintainer's sequencing); Cloudflare
DNS-only for DNS-01; laptop-pull backups; ceiling zero recurring + domain
~€10–15/year, >€10/month returns to a STOP. Rejected: paid India-region VPS
(Vultr Mumbai $20–40/mo, DO Bangalore $24–48/mo — 4–5× the EU class for the
same RAM; prices verified same-day), and the draft's blanket free-tier
dismissal (the phase's own backups + scripted re-homing reduce reclamation
to downtime, never data loss). Would change our mind: free-tier ARM capacity
unobtainable, or a real tester-facing outage attributable to the free host —
either fires the Hetzner fallback without a new gate.

**2026-08-20 — Zero-cash reversion; trial-credits bridge instead of the paid
fallback.** Chosen: the maintainer cannot spend on infrastructure right now,
so the Hetzner fallback (which the capacity failure would otherwise have
fired) is DEFERRED to when money returns; CP1a proceeds today on a paid AMD
shape (E5.Flex 2 OCPU/12 GB — mirrors the A1 target so sizing measurements
transfer) covered by the ~30-day trial credits, ₹0 out of pocket; trials do
not auto-convert — at trial end paid resources are stopped, never billed.
Bridge exit before the trial ends: an automated A1 capacity-retry (2/12,
free) plus PAYG conversion when Oracle's stuck billing setup completes;
re-homing is the proven cheap operation. Front without buying a domain:
free DuckDNS subdomain + Caddy wildcard via its DNS-01 plugin + per-tester
basic auth (real TLS, direct traffic, no relay ceiling) — bare-IP links
REJECTED (no TLS for IPs in practice: plaintext credentials + location data,
the line phase 8 refused to cross; links also rot at re-homing). Learned en
route, verified (Oracle docs + press): Always Free A1 was halved unannounced
on 2026-06-15 to 1,500 OCPU-hrs + 9,000 GB-hrs/month = 2 OCPU/12 GB
continuous; both 4/24 and 2/12 create attempts failed on Hyderabad capacity
across two days with the PAYG lever unavailable. Would change our mind:
money returning (Hetzner per the addendum) or the A1 sniper landing the free
instance (the intended exit).

**2026-08-20 — A1 sniper runs on the bridge host as a systemd timer.**
Chosen: the capacity retry lives on the host itself (OCI CLI + a 15-minute
systemd timer attempting the free A1.Flex 2/12 launch; stops itself and
writes a success marker on the first landing; double-launch guarded); the
API private key is generated on the host and never leaves it. Rejected:
running the retry from the laptop (reintroduces the lid-open dependency the
phase exists to remove) and manual console retries (two days of them
already failed). Would change our mind: the trial ending before a landing —
the sniper dies with the host, and the exit becomes laptop-return or
Hetzner-when-money-returns per the addendum.

**2026-08-20 — CP2: host scripts land beside the laptop scripts, not in
place of them.** Chosen: the host-topology pilot scripts go to
`scripts/pilot/host/`; the laptop originals stay untouched until CP3
closes, then are deleted in the same change that retires the laptop front.
Rejected: replacing in place (the laptop's nightly backup LaunchAgent runs
`scripts/pilot/backup-instance.sh` from the working tree while the laptop
still serves two real testers — a silent break weeks before retirement)
and dual-mode scripts (two topologies branching inside one script is
untestable). Would change our mind: nothing — the two-flavor state is a
dated transition ending at CP3, not a fork.

**2026-08-20 — CP2: routing by hostname; slots retired; the domain is a
parameter.** Chosen: one wildcard TLS site whose per-instance config is an
imported host-matched `handle` block (`import .../instances/*/caddy.conf`
before the 404 fallback — Caddy evaluates `handle` blocks in source order);
web ports allocated sequentially from 3010 by scanning instance `.env`
files; the base domain read from `~/pilot/front.env` as `ROADBOOK_DOMAIN`,
its real value recorded only in the private ledger (the repo is public).
Rejected: per-instance Caddy listener ports (an artifact of the funnel's
three fixed ports) and hardcoding the DuckDNS domain in committed scripts.
Would change our mind: the phase-11 naming decision buying a real domain —
the same parameter absorbs it with zero script changes.

**2026-08-20 — CP2: pilot state lives at `/srv/pilot`, not under
`/home/ubuntu`.** Chosen: the instance tree moves to `/srv/pilot`
(directories 755, `.env` 600, snippets 644; `~/pilot` stays as a symlink)
because the systemd Caddy runs as user `caddy`, which cannot traverse
`/home/ubuntu` (750) — and a glob `import` matching unreadable files
silently matches NOTHING, turning every subdomain into the 404 fallback
with a "Valid configuration" from a root-run validate (found live: the
external check caught it; validation must run as the caddy user).
Rejected: loosening `/home/ubuntu` to 755 (a hardening regression) and
splitting snippets into `/etc/caddy` away from their instance dirs (two
places for one instance's state). Would change our mind: nothing — this
is filesystem hygiene, not policy.

**2026-08-20 — CP2: the age secret never touches the host.** Chosen:
host-side backup encrypts with a recipient string only
(`~/pilot/keys/backup.pub`); host-side restore consumes plaintext tar from
a file or stdin, with decryption done on the laptop and piped over tailnet
SSH. Rejected: copying `backup.key` to the host for convenience — a
compromised host must never be able to read backup history (BRIEF §1.5).
Would change our mind: nothing foreseeable; this is the asymmetry the
design exists for.

**2026-08-20 — CP3 reshaped: fresh instances, no migration.** Chosen: the
laptop pilot instances are not migrated — every tester gets a fresh stamp
on the host (the maintainer's call at the CP2 STOP): the host demo already
exists; allen-zero is re-stamped and the maintainer re-uploads his archive
(doubling as the real-scale rehearsal CP2's demo-sized load could not
provide); friend-1 — who never got past login and almost certainly holds
no data — is stamped fresh when she returns. Rejected: the BRIEF's
backup/restore migration of all three (its mechanism is proven and stays
the re-homing path for E5→A1, but at this pilot's size there is nothing
worth moving: three confirmations on allen-zero the maintainer can redo
in a minute, and nightly encrypted backups already hold the laptop state
regardless). Would change our mind: an instance accumulating decisions or
photos that are expensive to recreate — then the proven restore pipe is
the path, unchanged. Laptop stacks stay up until each holder has their
new link (handover rules); the laptop front retires via the CP4
checklist.

**2026-08-21 — CP4: the backup chain is host-timer + Mac pull agent, both
nightly.** Chosen: systemd `roadbook-backup.timer` at 22:00 UTC (03:30
IST) produces age ciphertext on the host (public key only); a Mac
LaunchAgent (`com.roadbook.pilot.pull`, 04:30 IST) rsyncs it into the
proven iCloud directory; no pruning on either side yet — archives are
kilobytes-to-megabytes and deletion is the only irreversible act in the
chain. The Tailscale-SSH periodic re-auth can block an unattended pull;
accepted with logging, because archives accumulate on the host between
successful pulls and any interactive ssh clears the check. Rejected:
object-storage push (a credential on the host, per BRIEF §3D) and
host-side retention (could destroy not-yet-pulled archives). Would change
our mind: silent pull failures in backup.log (→ dedicated pull keypair)
or archive growth past tens of MB/night (→ retention policy).

**2026-08-21 — CP4: laptop retired to pull-only; its volumes kept.**
Chosen: laptop pilot stacks down with volumes KEPT (final deletion
deferred until the host has run clean for a while), brew Caddy stopped,
backup + caffeinate LaunchAgents removed (only the pull agent remains —
the laptop no longer hosts and may sleep), laptop-flavor pilot scripts
deleted from the repo (closing CP2's dated two-flavor transition; git
history keeps them), runbook v2 written to docs/private/pilot/. Rejected:
`down -v` today — the laptop volumes are the last pre-host state and disk
is cheaper than irreversibility. Would change our mind: nothing urgent;
a month of clean host operation retires the volumes too.
