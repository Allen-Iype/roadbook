# Phase 10 — Hosting readiness: design brief

Status: Gate 1 draft, awaiting maintainer review. No code or purchase before the
gate passes.

Charter basis: the dual-mode amendment (2026-08) and the roadmap in
`docs/PLAN.md` ("The road to strangers"). Phase 8 recorded this exact upgrade
path (§3A: small ARM VPS; §3B: domain + wildcard TLS + secret subdomains +
basic auth) with named triggers; two have fired — the maintainer deciding to
spend, and the planned capacity (ten instances) exceeding the three-port funnel
front. This phase is a re-homing, not a redesign: the instance template, the
stamp-out scripts, the Caddyfile structure, and the runbook all transfer.

**This phase retires phase 8's zero-cash stance.** That was an explicit Gate-1
decision then; retiring it is an explicit Gate-1 decision now, with a named
monthly ceiling (§3E).

**Target: zero product-code diff.** Go and web are untouched. Everything this
phase produces is operations: a host, a domain, configuration, scripts, drills,
and documentation. If a product-code change turns out to be required, that is a
finding to surface at a STOP, not a change to make quietly.

---

## 1 Concepts this phase introduces

### 1.1 A rented host, and why not a PaaS

A VPS (virtual private server) is a rented Linux machine: root access, a public
IPv4/IPv6 address, and nothing pre-decided. Everything proven on the laptop —
Docker, compose-per-instance, Caddy, the scripts — moves verbatim, which is the
point. The PaaS alternative (Fly.io, Render, …) was rejected at phase 8 §3A and
the reasoning still holds: compose stops being the deployment unit, so the
self-host reference and the hosted pilot fork — exactly what the amendment
forbids ("hosted is a superset, never a fork").

### 1.2 Why wildcard TLS is load-bearing, and what it demands

Phase 8 established that Certificate Transparency (CT) makes every issued TLS
certificate's hostname a public, watchable record — that is why funnel
hostnames needed basic auth in front. The upgrade front keeps per-instance
*secret subdomains* out of CT by using one **wildcard certificate**
(`*.example.com`): CT logs the wildcard, never the individual names, so a
tester's subdomain is not discoverable from certificate logs. Basic auth stays
in front regardless — the subdomain is unlisted, not a secret that can bear the
whole load (it appears in DNS queries, browser history, and chat logs).

The demand this makes: a wildcard certificate cannot be issued via the ordinary
HTTP-01 ACME challenge (prove control of one hostname by serving a file). It
requires **DNS-01**: prove control of the whole zone by creating a DNS TXT
record. Caddy automates that only if it can call the DNS provider's API — via a
provider-specific Caddy DNS plugin, which means running a Caddy build that
includes the plugin (a one-line Dockerfile or `xcaddy` build, not an ongoing
burden). Consequence: the DNS provider must be chosen for its API and Caddy
plugin support, not just as a place to buy the name (§3B).

### 1.3 Two front doors: users through 443, operator through the tailnet

The public internet sees exactly two open ports: 80 (redirect only) and 443
(Caddy). SSH is **not** exposed publicly: Tailscale goes on the VPS and the
operator path (SSH, admin, emergency access) rides the tailnet. Funnel retires;
Tailscale does not — its role changes from "front for users" to "path for the
operator." This removes the entire class of public-SSH attacks without
password/fail2ban tuning, and the fallback (provider web console) covers the
case where the tailnet itself is broken.

### 1.4 The Docker/firewall trap

Docker publishes ports by writing its own iptables rules, **bypassing** ufw —
`-p 8080:8080` is world-reachable even with a firewall that claims otherwise.
Our discipline already avoids the trap: every compose port binding is
`127.0.0.1:` (loopback-only publish, the pattern since phase 5), so nothing a
stack publishes is reachable from outside the host, firewall or no firewall.
The host firewall then only has to allow 80/443 and the tailnet interface.
This must be verified at CP1 from an external machine, not assumed from config
(the phase-8 lesson: loopback discipline holds only where it is proven).

### 1.5 Off-machine backups with asymmetric age

`age` encrypts to a recipient's *public* key. The VPS holds only the public
key: nightly archives are encrypted on the host, and nothing on the host can
decrypt them. The private key stays where it already is (the maintainer's
password manager, and the laptop for restore drills). A compromised VPS
therefore leaks at most the current live data, never the backup history, and a
stolen backup archive is unreadable. The near-miss history (a key once
generated into a non-gitignored path) is the standing caution: keys are
generated in a gitignored location, verified by `git status`, and never
transit the repository.

### 1.6 Custodianship and residency

The host will hold other people's location history — the most sensitive data
this project touches. Three consequences: the provider/region choice is a
judgement about jurisdiction and trust, not only price and latency (§3A); disk
encryption at rest is worth having but is *not* the main line of defence (the
host runs decrypted while up — the main defences are the small attack surface
of §1.3/1.4 and the per-instance isolation); and the honest statement to
testers ("your data lives on a rented server in <place>, isolated per person,
encrypted backups") should be written once, in the runbook, and reused by
phase 11's landing copy.

---

## 2 What gets built

- A provisioned, hardened VPS: non-root user, key-only SSH over tailnet,
  firewall (80/443 + tailnet), unattended security upgrades, Docker.
- A purchased domain with DNS at an API-capable provider; Caddy (with the DNS
  plugin) terminating TLS with a wildcard certificate; per-instance secret
  subdomains with per-instance basic auth — the phase-8 Caddyfile structure,
  re-addressed.
- `scripts/pilot/*` ported and exercised against the host (new-instance,
  reset, rotate-credential, backup-instance, backup-all, restore-instance).
- The three existing instances (demo + two testers) migrated via the proven
  backup/restore path, links rotated at the move.
- A nightly backup unit on the VPS (systemd timer) producing age-encrypted
  archives, shipped off the host (§3D), plus a restore drill from the
  off-machine copy.
- Runbook v2 (`docs/private/pilot/`): the new topology, drills, and the
  laptop-retirement checklist (funnel off, LaunchAgents removed, local Caddy
  stopped).

## 3 The real choices

### 3A Provider and region

| option | for | against |
|---|---|---|
| EU ARM VPS (Hetzner class) | best price/performance on record (phase 8 noted the ~€5/month ARM class, checked 2026-08-12; re-verify); mature provider; EU jurisdiction is a well-understood regime | ~120–150 ms from India — noticeable on first paint for every user, all of whom are in India today; no India region |
| India-region VPS (DigitalOcean Bangalore / Vultr Mumbai class) | lowest latency for every current user; data stays in-country | roughly 2–4× the price for equivalent RAM (list-price impression, must be verified at purchase); x86 small instances |
| Oracle/other free tiers | free | capacity and continuity are not contractual; building the pilot's home on a revocable free tier repeats the laptop's availability problem in a different shape |

**Recommendation: decide by residency-and-latency first, price second — which
points at an India region** for a user base that is entirely in India, with the
EU option as the fallback if verified prices are disproportionate. Sizing is
**not** decided here: CP2 measures ten stacks' real memory/disk before the
final instance size is fixed; start with the smallest plausible size and resize
upward (a supported, minutes-long operation at this class of provider). A
rough planning envelope, to be replaced by CP2's measurement: each stack runs
Postgres + Go API + Next standalone, a few hundred MB together, so ten stacks
plus OS plausibly want 4–8 GB.

### 3B Registrar and DNS

| option | for | against |
|---|---|---|
| Registrar + Cloudflare DNS (free tier, DNS-only mode) | first-class Caddy DNS plugin; instant API; registrar can be anyone | adds a second account; must be run **DNS-only** (grey cloud) — proxied mode would terminate TLS at Cloudflare and put a third party inside the connection to location data, which is unacceptable here |
| Registrar whose own DNS has a Caddy plugin (Porkbun class) | one account for name + DNS | plugin ecosystems vary in maintenance; verify the specific plugin before purchase |

**Recommendation: whichever registrar wins the naming decision, with
Cloudflare in DNS-only mode as the default DNS API** — the plugin is the most
battle-tested, and DNS-only mode keeps TLS end-to-end at our Caddy. The domain
name itself is a naming decision (it becomes the product's public name and
phase 11 inherits it); it deserves its own short exploration at the gate, not a
default to whatever is unregistered.

### 3C How deployment is driven

| option | for | against |
|---|---|---|
| Repo cloned on the host; scripts run there over SSH (chosen shape of the laptop pilot) | zero new machinery; the runbook stays true; scripts/pilot already work this way | manual; host state is described by the runbook, not enforced by a tool |
| `docker context` from the laptop | no repo on host | splits truth between laptop and host; scripts assume local execution |
| IaC (Terraform/Ansible) | reproducible host | a new toolchain for exactly one host — the same "building for thousands is theatre" judgement the charter applies elsewhere |

**Recommendation: repo on the host, scripts run there.** One host does not
earn an IaC layer; the runbook plus a hardening checklist is the reproduction
path. What must be honoured: the host's checkout never contains `data/` (there
is no real-data mount on the VPS at all — instances hold only what users
upload to them; the operator's own archive lives on an instance like anyone
else's, not as a host-path mount).

### 3D Backup destination

| option | for | against |
|---|---|---|
| Laptop pulls nightly from the VPS, then the existing laptop→iCloud chain | zero new accounts; reuses the proven age+iCloud chain and launchd wake behaviour; pull (not push) means the VPS holds no credential to reach the maintainer's machines | copies exist only when the laptop wakes; two-hop chain to the durable copy |
| Object storage (B2/S3 class), pushed from the VPS | durable immediately; no laptop in the loop | a new account and credential ON the host; at gigabyte scale cost is trivial but nonzero; a push credential on the VPS can overwrite history unless write-only/versioning is configured correctly |
| Provider's own storage box | cheap, adjacent | correlated failure and correlated jurisdiction with the primary — weakest off-machine property |

**Recommendation: laptop-pull as the baseline (it reuses everything proven),
with object storage as the upgrade** if CP4's drill shows the laptop-pull
cadence leaves unacceptable gaps. Either way the VPS-side archive is
age-encrypted with the existing key's public half before it goes anywhere; the
private key never touches the host (§1.5).

### 3E The cost ceiling

To be set by the maintainer at this gate. Proposed: **domain (~€10–15/year
class) + VPS at whatever CP2's measurement justifies, capped at €15/month;
anything above the cap returns to a STOP.** The number is a leash, not a
budget to spend up to.

---

## 4 Checkpoints

*(CP1 amended at Gate 1: split into a free half and a paid half, with the
domain purchase gated on the free half proving the host — see the Gate 1
addendum after §7.)*

1. **CP1a — the host (zero spend).** Free-tier account established
   (pay-as-you-go conversion for reclaim protection; expected bill zero),
   India home region, ARM instance obtained, hardened (§1.3, §1.4), Docker +
   Tailscale up, demo stack stamped and verified by the operator over the
   tailnet. No public exposure of anything. *Visible: the demo instance
   browsable via the tailnet; an external port scan shows nothing but the
   provider's defaults closed.* If capacity or signup fails, fall back to the
   paid option (§3A addendum) having spent nothing.
   **CP1b — the front (the domain, the one purchase).** Working-name domain
   bought; DNS at the API-capable provider (§3B); wildcard certificate via
   DNS-01; the demo instance behind basic auth on a subdomain. *Visible: the
   demo link answers from a phone on cellular with the laptop closed; the
   certificate in CT logs shows only the wildcard.*
2. **CP2 — topology at ten.** All pilot scripts exercised on the host; a
   scratch tenth-instance fill (stamp ten, load the demo dataset into several)
   with memory/disk measured and recorded; the final instance size chosen from
   the measurement; scratch instances torn down. *Visible: one command stamps
   a fresh instance; a measured capacity statement in the runbook.*
3. **CP3 — migration.** The three existing instances move via backup/restore
   with zero decision loss (counts verified before/after); links rotated;
   outgoing links dead. *Visible: existing testers' instances answer at new
   URLs with their data intact; the drill recorded in DECISIONS.*
4. **CP4 — durability.** Nightly encrypted backup unit live on the VPS;
   off-machine copy landing (§3D); a restore performed from the off-machine
   copy into a scratch instance; a reboot drill on the VPS (everything returns
   without hands); laptop front retired via checklist. *Visible: the restore;
   the drill log; the laptop no longer serving anything.*

STOP at each checkpoint per the working agreement.

## 5 Explicitly excluded

Any product code (Go or web); the public landing page; waitlist; entry emails;
auth beyond per-instance basic auth; tenancy; new ingestion; the phase-9 shell
seam and header slot stay unused. Also excluded: serving anyone *new* this
phase — capacity is built and measured, not filled; phase 11 owns entry.

## 6 Risks and what would change our mind

- **The v1 loop failing for product reasons** (pilot reports pending) — a
  fix-the-loop pass would precede this phase's close; infrastructure under a
  loop nobody completes is waste (the roadmap's resequencing clause).
- **CP2's measurement not fitting the price envelope** — reopens §3A/§3E at a
  STOP rather than silently upsizing.
- **DNS-01/plugin friction with the chosen provider** — fallback is switching
  DNS to the other §3B option; per-subdomain HTTP-01 is *not* a fallback (it
  would put every secret subdomain into CT, §1.2).
- **Migration surprising us** — backup/restore is proven between local
  environments; CP3 is its first cross-machine, cross-architecture run
  (ARM→x86 or reverse for the database image). The restore drill into a
  scratch instance happens *before* any live instance is touched.

## 7 Gate 1 questions for the maintainer

1. §3A — provider/region: India-region first (recommended) or EU-price first?
2. §3B — the domain name itself: explore names now, or buy under a working
   name and treat naming as a phase 11 concern? (Recommendation: decide the
   real name now; it is the product's public identity and re-homing twice is
   waste.)
3. §3D — laptop-pull baseline accepted, or object storage from day one?
4. §3E — the ceiling number.

---

## Gate 1 addendum — decisions taken 2026-08-18 (gate PASSED)

The maintainer reviewed with prices verified same-day (list prices; re-verify
at purchase):

- **§3A — Oracle Cloud Always Free tier, India home region (Mumbai or
  Hyderabad), as the primary host.** Up to 4 ARM cores / 24 GB RAM free;
  account converted to pay-as-you-go (card on file, expected bill zero) to
  remove the idle-reclaim policy. This overrides the draft's free-tier
  dismissal *because the phase's own machinery changes the risk*: nightly
  off-machine encrypted backups plus scripted re-homing mean a reclaimed
  free instance costs downtime, never data. **Named fallback, pre-decided:**
  Hetzner EU ARM/x86 class (~€5–9/month at 2026-08-18 list prices) — taken
  without a new gate if free-tier capacity cannot be obtained, or if the
  free host causes a real tester-facing outage (the phase-8 trigger logic,
  reapplied).
- **§3B — working name now, real name later.** The rename cost (link + cert
  rotation) is accepted; naming becomes a phase 11 concern. Registrar chosen
  at purchase; DNS at Cloudflare free tier in DNS-only mode (recommendation
  stands).
- **§3D — laptop-pull baseline.** As recommended.
- **§3E — ceiling: zero recurring; domain ~€10–15/year is the one planned
  spend.** Any month wanting more than €10 returns to a STOP. The paid
  fallback, if triggered, re-opens §3E with the Hetzner figure on the table.
- **CP1 split (§4):** CP1a free host proof over the tailnet first; the
  domain purchase (CP1b) happens only after CP1a proves the host — the
  maintainer's sequencing, adopted because it moves the only spend behind
  the riskiest unknown (free-tier capacity).
