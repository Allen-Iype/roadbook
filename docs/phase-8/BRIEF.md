# Phase 8 design brief — friend pilot hosting: the link

**Goal:** the second half of v1. Phase 7 built everything that happens after
the click — pitch, walkthrough, upload, import, detection, life map. Phase 8
supplies the click: a link that works from anywhere, an isolated instance
behind it for each tester, access control proportionate to location data, and
an operator practice (provision, hand over, back up, reset, rotate) that a
runbook captures completely. When the phase closes, a person with a link and
their own data reaches their adventures with zero operator involvement in
their journey. "Zero operator involvement" is a per-user property — nobody
runs a CLI for anybody — and is distinct from availability, which in this
phase is deliberately the operator's laptop (below).

**Amended at Gate 1 (2026-08-12).** The maintainer's direction: send links
to several friends now, spend no money, and accept keeping the laptop open
as the availability arrangement. The original draft recommended a rented
host; that recommendation and its reasoning are preserved in §3A as the
recorded upgrade path with named triggers, and the decision log carries the
amendment. Everything below plans the laptop arrangement.

This is an ops phase by design. The product side of the loop exists and is
verified; Go and web code stay untouched except two small carried items
(§3E) whose evidence was collected at the phase 7 close. Most of what this
phase produces is not code at all: a serving posture, templates, and a
runbook.

**The evidence base.** An interim, pre-phase-8 pilot ran a dedicated compose
instance from the operator's laptop behind Tailscale Funnel. It proved the
essential shape this phase now builds on: a public HTTPS link that a phone
opens with zero client install, terminating at a loopback-published
instance, with no LAN or WAN port exposure on the host. What it did not
prove: more than one tester at a time, any authentication in front, upload
behaviour at archive scale over the funnel relay, or survival of a reboot —
each is a checkpoint item below. Separately, the phase 7 close found two
long-forgotten dev servers bound to all interfaces on the operator's LAN;
the lesson (loopback discipline must be structural, not habitual) becomes a
design principle here: nothing on the laptop listens beyond loopback except
what Tailscale carries, and every public path goes through one audited
front.

---

## 1 Concepts this phase introduces

### 1.1 Serving from the machine you live on

The pilot host is the development laptop. That choice costs three things,
each accepted knowingly at Gate 1 and each with a mitigation this phase
builds rather than hopes for. **Availability** is a human posture — lid
open, mains power, `caffeinate` — so the handover message promises honesty
instead of uptime ("hosted on my laptop; if it doesn't load, tell me or try
later"), and a serving-posture checklist plus a reboot drill (§6 CP1) make
recovery mechanical. **Colocation**: the same machine holds the operator's
own irreplaceable `data/`; the instance template's `!override` volume guard
(proven by the interim pilot) is what makes "a pilot instance has no path
to it" structural rather than careful. **Shared fate**: a laptop mishap
takes the instances and their backups down together, which is why backup
copies leave the machine encrypted (§3D). The rented-host shape (§3A)
removes all three, and is recorded as the upgrade, not the plan.

### 1.2 One front door: reverse proxy and auth termination

Each instance's web container publishes on a loopback port
(`127.0.0.1:<port>`), exactly as compose does today. None of them is
directly reachable. A single local Caddy is the only front: Tailscale
Funnel hands each public request to Caddy, Caddy enforces per-tester basic
auth and forwards to the right instance's loopback port. This upgrades the
project's loopback discipline from "nothing is exposed" to "exactly one
thing is exposed, and it is the thing that was audited." Caddy is chosen
because basic auth, per-listener routing, and header hygiene
(`X-Robots-Tag: noindex`) are a few declarative lines, and because it is
already the project's named reverse proxy in the upgrade path — the
Caddyfile written now survives a later move to a rented host nearly
unchanged.

### 1.3 Certificates leak names: CT logs, and why the URL is not the secret

Every publicly trusted TLS certificate is published to Certificate
Transparency logs — a public, searchable, append-only record. Funnel
hostnames get exactly such certificates, so every funnel URL is
discoverable by anyone watching the logs; no amount of randomness in the
name helps, because CT publishes the full hostname. Consequence, and the
central access-control fact of this phase: **the link is an address, not a
secret.** The secret is the per-tester credential that Caddy enforces
(§1.4). (The original draft's wildcard-certificate answer — which does keep
names out of CT — needs a purchased domain and is preserved in §3B's
upgrade path.)

### 1.4 The credential as capability

The pilot has no product auth (v1 definition; Direction 6 owns accounts).
Access control is **proxy basic auth**: a per-tester username/password
enforced by Caddy before any request reaches an instance. A leaked or
CT-discovered URL alone yields a 401, not someone's location history;
crawlers and link-preview fetchers are shut out wholesale. The browser
prompts once and remembers the credential, so per-visit friction on a
phone is near zero. Link and credential travel together in the handover
message and are treated as one bundle, but the credential is the rotating
part: at every handover — or on suspected leak — the outgoing credential
dies and a new one is issued, which is cheaper and more precise than
renaming hosts. This is honest security for n ≤ 5 known people, not a
substitute for accounts — which is why Direction 6's trigger is unchanged.

### 1.5 Tailscale Funnel, now load-bearing

Funnel relays public HTTPS to a tailnet node with no open ports: TLS for
`<machine>.<tailnet>.ts.net` terminates on the laptop, and the laptop's
firewall posture never changes. It is free, and the interim pilot proved
it end to end. Two properties now matter operationally. **Three ports per
node** (443, 8443, 10000): one laptop node can front at most three
concurrent listeners, so the pilot's concurrency ceiling is three active
testers — links differ by port (`https://<machine>.<tailnet>.ts.net:8443`),
which is cosmetic. The extension past three is a Tailscale sidecar
container per instance (each its own node, own hostname); recorded in §3B,
not built now. **The relay is in the upload path**: a multi-hundred-MB
Timeline export rides Tailscale's infrastructure, whose throughput this
project has never measured — a CP1 proof item, not an assumption. The
tailnet's other role from the interim period (private admin access) is
moot while operator and host are the same machine, but returns in the
upgrade path.

### 1.6 Compose project isolation

`docker compose -p roadbook-<slug>` gives each tester a fully separate
stack: own Postgres, own API, own web, own named volumes, own network.
Nothing is shared, so no cross-tester bug class exists — there is no query
to forget a user filter on, which is precisely why instance-per-user was
chosen over tenancy for n ≤ 5. The interim pilot's override proved the
pattern; this phase turns it into a committed, parameterised template plus
a stamp-out script, and keeps the per-tester facts (who, which slug, which
credential) out of the public repository.

### 1.7 Custody of other people's location data

The moment a friend uploads, the operator holds their location history —
special-category data by any regulator's definition, held informally among
friends but held nonetheless. The phase treats custody as a first-class
deliverable: informed consent in the invitation message (what is stored,
where — "my laptop", plainly — that the operator is technically able to
see it, how deletion works); encrypted backup copies that leave the
machine; reset that is structurally complete (volumes, not rows); and
honesty about what is *not* protected — whoever can use this laptop can
read its disks, and its loss or compromise exposes whatever instances
hold. The mitigation for that residual risk is scale and retention
discipline (a handful of known people, wiped at handover), not
cryptography theatre.

---

## 2 What gets built

Committed to the repository (all generic — see the public-repo rule below):

- `docs/phase-8/RUNBOOK.md` — the operator runbook (§4), with
  placeholders where the real tailnet, machine, and testers go.
- A parameterised per-instance compose override template and a stamp-out /
  reset script under `scripts/` (§3C).
- A Caddyfile template (auth front, per-listener routing, noindex) with
  placeholder hostnames and hashes (§3B).
- `restart: unless-stopped` on the base `compose.yaml` services — today a
  reboot leaves every stack down; a serving machine cannot have that, and
  self-hosters get the same fix (§3E).
- `roadbook countries -if-empty` and its fold into the compose startup
  command, retiring the last operator CLI step from a fresh instance
  (§3E).
- A README hosting section: what the multi-instance setup is, pointer to
  the runbook.
- `docs/phase-8/DECISIONS.md` (live now) and `LOG.md` at close.

Operator-side, never committed: the real Caddyfile and funnel
configuration; the per-tester ledger (slug, port, link, credential,
consent record, handover log) in `docs/private/pilot/` — gitignored, as
today; backup archives, outside the repository entirely.

**The public-repo rule.** This repository is public on GitHub. No committed
file may name the tailnet, the machine, a tester, a link, or a credential —
the committed artifacts are templates and a runbook anyone could reuse; the
instantiation lives in `docs/private/` and on the host. (Same discipline as
data safety, applied to infrastructure identity.)

**Excluded, deliberately:** product auth, accounts, OIDC (v2, Direction 6 —
trigger unchanged); multi-user tenancy (banked); self-serve deletion
(Direction 6; operator volume reset covers the pilot); the charter
amendment (drafted in `docs/private/`, pending the maintainer's re-read of
the parked doc; this phase neither needs nor touches it); a marketing site
(launch item; the front door is the pitch); routing/OSRM on pilot
instances by default (optional per instance if a tester's region warrants
— runbook notes the steps, `scripts/osrm-setup.sh` exists); Kubernetes,
queues, and orchestration beyond compose (charter); paid anything — the
phase's cash cost is zero (§5); Tailscale sidecar nodes (the >3
concurrency extension, built only when a fourth concurrent tester exists).

---

## 3 The real choices

### 3A Where instances live — decided at Gate 1, upgrade path recorded

| option | for | against |
|---|---|---|
| **The laptop (decided)** | zero cash; proven end-to-end by the interim pilot; links can go out as soon as the phase closes | availability = a lid held open; colocated with the operator's own `data/`; shared fate of instances and their backups (§1.1 carries all three, with mitigations) |
| Small ARM VPS (the recorded upgrade) | always on; no `data/` on the host; blast-radius isolation | ~€5/month class (provider list price, checked 2026-08-12; re-verify if taken) — money the pilot deliberately does not spend |
| PaaS (Fly.io, Render, …) | managed availability | compose stops being the deployment unit — the self-host reference and the hosted pilot fork, violating "hosted is a superset, never a fork" |

The maintainer decided at Gate 1: laptop, zero spend, lid-open accepted.
The assistant's original recommendation (the VPS) stands recorded here and
in the decision log as recommendation-not-taken; its substance is
preserved because everything built this phase is designed to survive the
move — the Caddyfile, the instance template, the stamp-out script, and the
runbook all transfer, so the upgrade is a re-homing, not a redesign.
**Upgrade triggers, named now:** a friend hitting a down instance during a
real attempt; an upload failing for laptop-availability reasons; the pilot
outgrowing three concurrent testers in practice; or the maintainer
deciding to spend. Any one firing re-opens §3A with the original
recommendation on the table.

### 3B What fronts the instances

| option | for | against |
|---|---|---|
| **Funnel (3 ports) + local Caddy basic auth per tester** | zero cash; auth means a CT-discovered URL yields 401 (§1.3–1.4); rotation = credential swap, purely operator-controlled; the Caddyfile survives the upgrade path | ceiling of three concurrent testers (§1.5); non-standard ports in two of three links (cosmetic); upload rides the funnel relay (throughput proven at CP1 or the option falls) |
| Funnel link-only (no Caddy) | simplest possible | the URL is CT-public, so an unauthenticated instance's location history is one log-watch away — unacceptable for other people's data; nothing to rotate but the hostname |
| Domain + Caddy wildcard + secret subdomain + basic auth (the upgrade-path front) | subdomains never in CT; clean per-tester hostnames; no relay in the upload path | costs a domain (~€10–15/year, checked at purchase) and pairs naturally with the rented host — deferred with §3A |
| Testers join the tailnet | strongest transport privacy | each tester must install Tailscale and accept an invite — the "person with a link" v1 definition fails at the first step; DIRECTIONS' original "tailnet preferred" note predates the v1 definition and is superseded by it |

**Recommendation: Funnel's three ports, each terminating at a local Caddy
listener with per-tester basic auth.** The deciding fact is §1.3: with
funnel certificates CT-logged, link-only access is not a secret-URL scheme
at all, so the free path is only honest with an auth layer — and Caddy
provides one for the cost of a config file. Three concurrent testers is
accepted as the ceiling; five friends are unlikely to be simultaneously
active during a pilot, handovers stagger naturally (§3C), and the sidecar
extension is specified in §1.5 for the day the ceiling binds.

What would change our mind: basic auth failing inside the WhatsApp in-app
browser on either platform (§7 risk; the fallback ladder is "open in
browser" guidance in the handover message, then a Caddy cookie/form-auth
page, then link-only with aggressive rotation as the last resort); CP1's
relay-throughput test failing at archive scale (then uploads are the
trigger that re-opens §3A early); a fourth concurrent tester (sidecars).

### 3C Provisioning and reset

| option | for | against |
|---|---|---|
| **Compose project per tester: committed override template + stamp-out script** | proven by the interim override; isolation is structural (§1.6); reset is one command; the committed artifact is generic and reusable | per-instance resource overhead on a machine that is also the dev machine (§5 measures it) |
| One shared stack, one Postgres, DB-per-tester | fewer processes | a shared API/web would need tenant routing — product code the charter forbids and Direction 6 owns; sharing only Postgres still couples failure and upgrade domains for zero code saved |
| Hand-edited per-friend overrides (interim status quo) | nothing new to write | the interim file was written by hand once; five testers × handovers means repetition, and repetition without a script is where mistakes live |

**Recommendation: project-per-tester, stamped from a template.** The
stamp-out script takes a slug, allocates a loopback web port and one of
the three funnel listeners, generates the instance `.env` (including a
random per-instance Postgres password — the compose default of `roadbook`
is tolerable for a single private stack, not for a serving posture),
writes the override from the template, brings the stack up, writes the
Caddy listener block (funnel port → basic auth → loopback port) with a
freshly hashed credential, reloads Caddy, and prints the link + credential
for the handover message plus the ledger line. The committed template
keeps the interim override's `!override` volume guard — the laptop is
exactly the machine where "no path to `data/`" must be structural (§1.1).

**Reset and rotation** (the handover rules, now runbook procedure): reset
is `docker compose -p roadbook-<slug> down -v` — volume-level, so DB,
photo thumbnails, and phase 7's retained uploads all go structurally,
never by row deletes that can miss a file. One credential per person,
rotated at every handover: the outgoing tester is told first and reminded
to keep their own export file (the instance's retained copy may be their
only one); then volumes are wiped and the credential is replaced, so the
old link+credential pair dies (401) even though the hostname persists.
The drill for both is a CP2 acceptance item, not prose.

What would change our mind: nothing foreseeable at n ≤ 5; growth beyond
it re-opens the shared-stack question under Direction 6, where it
belongs.

### 3D Custody: backups and consent

Per-instance backup on a schedule: `roadbook backup` via
`docker compose run` (decisions, photos, thumbnails — the durable
identities), plus a plain copy of the uploads volume alongside it —
phase 7's retained exports are a third class of hard-to-replace data that
`roadbook backup` does not cover (carried tension, phase 7 LOG). For the
pilot the uploads copy is a runbook-level volume tar next to the backup
archive; extending the backup *format* to include uploads stays in the
backlog — it deserves manifest and hash thought, and an ops phase is the
wrong place to design an archive format. Because instances and backups
now share a machine (§1.1 shared fate), archives are encrypted with `age`
and a copy leaves the laptop to whatever offsite location the operator
already trusts — encrypted before departure, so the offsite service holds
ciphertext only. Backup destinations live outside the repository and
outside `data/` (which holds the operator's own source data and is
never added to on this machine). A restore drill onto a scratch instance
is a CP3 acceptance item, because a backup that has never been restored
is a hope, not a backup. Consent is part of the handover message template
(runbook): what is stored and where — the operator's laptop, stated
plainly — that the operator can technically access it, that
reset-on-request means volume deletion, and that encrypted backup copies
exist and where they live.

What would change our mind on uploads-in-backup-format: a self-hoster
(not the pilot operator) losing retained uploads a backup should have
carried — that promotes the backlog item to a product phase.

### 3E Two small carried product changes, in scope

- **`roadbook countries -if-empty`** (new flag) folded into the compose
  startup command (`migrate && countries -if-empty && serve`): the
  embedded Natural Earth data loads on first boot of a fresh instance,
  never fetching (the data is `go:embed`ed — that decision was made in
  phase 2 exactly so this fold would be safe), never re-running on
  restarts, and `roadbook countries` without the flag still replaces
  wholesale for operators with higher-res data. This retires the last
  operator CLI step from the browser-only path — carried from phase 7
  with evidence in hand, and directly in service of this phase's goal:
  a freshly stamped instance must be browser-complete.
- **`restart: unless-stopped`** on the base compose services: a serving
  machine must survive a reboot without a human replaying `up` commands
  per instance. Self-hosters inherit the same behaviour; the phase 7
  known issue where the db healthcheck can pass during a fresh volume's
  initdb restart (api exits once) likely becomes self-healing under a
  restart policy — verified, not assumed, at CP1.

Both change files the goldens never touch; `make test` and byte-identical
goldens remain the regression gate as always.

---

## 4 The operator runbook

`docs/phase-8/RUNBOOK.md`, generic with placeholders. Sections it must
contain (drafted across CP1–CP3, complete at CP4):

1. **Serving posture** — the laptop checklist: mains, lid, `caffeinate`,
   Docker Desktop / Tailscale / Caddy set to start at login, funnel
   listeners re-enabled, and the reboot drill that proves the whole
   posture returns unaided.
2. **Stamp a new instance** — the script, what it emits, ledger entry.
3. **Hand over a link** — message template: consent (§3D, "my laptop"
   stated plainly), what to expect (from phase 7's front door: export
   size, upload, "you can close the tab" — now with the honest
   availability caveat), the link + credential, "open in browser" note.
4. **Watch** — a scheduled job curling each instance's `/healthz` through
   the front, logging failures; `docker compose ls` as the one-glance
   state; disk watermark check (uploads are large).
5. **Update** — `git pull`, rebuild, per-instance `up -d`, in what order,
   and how to verify (healthz + a page load per instance).
6. **Handover** — notify, remind about the export copy, `down -v`,
   rotate the credential, new message (§3C verbatim).
7. **Incident** — suspected credential leak → rotate immediately (same
   procedure as handover minus the wipe); laptop loss or compromise →
   assume all instance data exposed, inform testers; funnel or Tailscale
   outage → honest message to testers, nothing to fix locally.
8. **Backup and restore** — the schedule, the encrypted offsite copy, the
   drill and its cadence.
9. **Decommission** — end of pilot: testers notified, per-tester backup
   handed to each tester on request, volumes wiped, funnel and Caddy
   listeners removed.

---

## 5 Cost envelope

**Cash: €0.** Funnel, Caddy, and the Tailscale free tier cost nothing; the
laptop is already owned. The costs are in kind, named in §1.1: lid-open
availability, home upload bandwidth shared with the household, and
colocation/shared-fate risk. The upgrade path's prices are recorded for
the day a §3A trigger fires — ARM VPS under €5/month and a domain at
€10–15/year, provider list prices as checked 2026-08-12, re-verified at
purchase — and appear here attributed and dated because the repository
cannot reproduce them (invariant 13; the README carries no prices).

Sizing is an estimate to be **measured, not asserted**: three concurrent
stacks × (one Postgres + one 42 MB-image Go binary + one Next standalone
server) alongside the operator's own dev workload is expected to fit this
machine, and CP1 records the actual RSS of one stack idle and during an
import+detect run before CP2 stamps more. If the measurement disagrees,
the concurrency ceiling drops before the laptop swaps — an honest ceiling
beats a thrashing host.

---

## 6 Checkpoints

Each ends at a STOP with named visibles, per the working agreement.

**CP1 — the front, on the laptop.** Caddy installed and templated; one
funnel listener live, fronting the demo-persona instance behind basic
auth; `restart: unless-stopped` committed; the reboot drill passed
(restart the laptop → Docker, Tailscale, Caddy, funnel, and the stack
all return unaided, including the initdb-race check); a large
**synthetic** file (never real data — infrastructure tests use noise)
uploaded through the full funnel+Caddy path to measure relay throughput
and prove timeout/buffering behaviour at archive scale; one-stack
resource measurement recorded (§5). Visible: the demo instance reachable
from a phone on cellular via the link, credential prompt and all —
including from inside a WhatsApp-opened browser (§7's auth risk checked
here, early, on Android at least).

**CP2 — stamp-out, reset, rotation.** Template + script committed; two
instances up side by side from the script on two funnel listeners;
isolation checked (distinct volumes, no `./data` mount anywhere, one
instance's wipe leaves the other untouched); the reset drill (volumes
provably gone) and the rotation drill (old credential 401s, new one
works, other instance unaffected) both executed. `countries -if-empty`
lands here — a freshly stamped instance must come up browser-complete
with country attribution.

**CP3 — custody.** Backup schedule live for both instances; encrypted
copies leaving the laptop to the operator's offsite location; restore
drill: a scratch instance restored from an offsite archive re-attaches
decisions after import + re-detection (the phase 5 proof, now as an
operator procedure); uploads-volume copy verified in the same pass.
README hosting section drafted. RUNBOOK sections 1–5 and 8 complete.

**CP4 — tester zero, and close.** The full v1 walk with the operator as
tester zero: a WhatsApp link opened on a phone on **cellular** (off the
home LAN), away from the laptop, which nobody touches for the duration —
link → credential prompt → pitch → walkthrough → upload (a real export,
uploaded by its owner) → candidates → confirm → life map, zero operator
involvement. RUNBOOK complete (6, 7, 9); the handover messages for the
first real friends drafted from the template; LOG.md written; cold
`make test` green; goldens byte-identical. **The phase closes on tester
zero, not on the first friend** — sending the real links is day-one
operations after the close, scheduled by the friends' availability, not
the phase's (the iPhone-stamp lesson: a phase gate must not wait on
someone else's calendar). The standing side-item rides along: when the
iPhone friend's walkthrough report arrives, the `/welcome` iPhone branch
is fixed or stamped as a small follow-up commit, independent of this
phase's state.

---

## 7 Risks and open questions

- **Basic auth in WhatsApp's in-app browsers.** Android opens links in a
  Chrome custom tab (shared Chrome state; basic auth expected to work);
  the iPhone in-app browser is less predictable. Checked at CP1 on the
  operator's device and again in the CP4 walk; the fallback ladder is in
  §3B, and the handover message carries "open in browser" guidance
  regardless.
- **Relay throughput at archive scale.** The funnel relay is now on the
  upload critical path and this project has never measured it; CP1's
  synthetic large-file test is the proof. A failure here is a named §3A
  upgrade trigger, found before any friend meets the path.
- **Availability honesty.** The laptop sleeps, travels, and reboots.
  Mitigations are posture (runbook §1), `restart: unless-stopped`, and —
  above all — the handover message saying so plainly. A friend who hits
  a down instance is a §3A trigger, not an embarrassment to hide.
- **Colocation with `data/`.** The serving machine holds the operator's
  irreplaceable source data. The template's `!override` guard is the
  structural answer; the stamp-out script never touches `data/`; nothing
  in this phase writes there.
- **Shared fate of instances and backups.** Both live on one machine
  until the encrypted offsite copy exists — which is why CP3 is a gate,
  not a nice-to-have, and why the first real friend link waits for the
  phase close.
- **The public repository.** Committed templates and runbook must never
  name the tailnet, machine, testers, links, or credentials — reviewed
  at every commit like data safety (the `git add -A --dry-run` habit
  extends to reading the diff for infrastructure identity).
- **The three-listener ceiling.** Accepted; the sidecar-per-instance
  extension (own node and hostname per tester) is specified in §1.5 and
  built only when a fourth concurrent tester actually exists.
