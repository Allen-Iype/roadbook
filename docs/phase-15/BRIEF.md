# Phase 15 — Hosting on a real platform: design brief

Status: Gate 1 draft, written 2026-10-16 at the post-phase-14 charter STOP.
No code, no purchase, no account creation before the gate passes.
**Amended 2026-10-27:** the maintainer decided provider, address, and ceiling
(§8). §3A, §3B, §3G are superseded by §8; §3C, §3D, §3E, §3F remain open, and
§8.3 lists four points where the stated hosting plan meets the built product
and needs a word before the gate closes.

Charter basis: `docs/PLAN.md`, "The road to strangers", phase 15 — "the
finished product reachable by strangers on a money-returning platform: the
front-gate proof phase 12 never reached, the live-Google sign-in walk from
phase 13's checklist, off-machine backups with a staleness alarm, and the
landing, waitlist, and entry drill re-deployed on a durable host. Its brief
opens with the provider decision that the zero-cash constraint deferred."

This is the third hosting brief. Phase 8 put the pilot on the laptop; phase
10 moved it to a rented host at zero cash; phase 12 tore that host down when
the trial ended. What phases 13 and 14 built in between changes what is being
hosted, and this brief takes that as its first input rather than re-running
phase 10 with a credit card.

**Target: zero product-code diff**, as phases 10 and 12 targeted. Two seams
found while writing this brief press on that target (§1.3, §1.4); each has a
zero-code answer recommended and a product-code fallback named. If a
product-code change turns out to be required, it is surfaced at a STOP, not
made quietly.

---

## 1 Concepts this phase introduces

### 1.1 What is being hosted has changed shape

Phase 10 hosted **one compose stack per person**: separate Postgres, separate
volumes, a secret subdomain, a basic-auth credential, all stamped by
`new-instance.sh`. Isolation was structural because nothing was shared. That
shape existed because the product had no accounts.

Phase 13 gave the product accounts (`ROADBOOK_AUTH=oidc`): one instance
holds several people's data, every user-data row carries its owner, every
store query filters by it, identity is delegated to Google (or any OIDC
issuer), and the cross-tenant test family is the standing proof. The finished
product is that mode. Hosting it means **one stack**, not ten — and the
hosted topology becomes:

```
stranger ──HTTPS──▶ Caddy ──▶ www.<domain>   static landing + waitlist form
                          ──▶ app.<domain>   ONE compose stack, oidc mode
                          ──▶ demo.<domain>  one compose stack, off mode,
                                             demo data, basic auth (optional)
operator ──tailnet──▶ sshd bound to the tailnet interface only
```

Consequences, each a real change from the phase-10 runbook: entry is no
longer "stamp an instance" (§1.3); isolation is rows, not volumes, and the
custody sentence told to users must say so honestly (§1.6); per-person
scripts (`new-instance`, `reset-instance`, `rotate-credential`,
`backup-instance`) retire or change meaning (§2); deletion is self-serve
(`DELETE /me`, phase 13 CP5), so the operator's "wipe the instance" promise
becomes "you press the button, and the backups age out."

The self-host reference deployment (`compose.yaml`, auth off) is untouched.
Hosted stays a superset, never a fork — the hosted stack is the same
`compose.yaml` with five environment variables set.

### 1.2 Why a real domain is now a precondition, not an upgrade

Phase 12 chose DuckDNS deliberately: zero cash, and links survive a later
re-point. That reasoning held while the front was basic auth. OIDC changes
it. Google's sign-in needs a registered OAuth client whose redirect URI
(`<ROADBOOK_PUBLIC_URL>/api/auth/callback`) sits on a public hostname, and
the consent screen's branding wants an **authorised domain** the operator
can verify as theirs. Whether a DuckDNS subdomain can be made to pass that
check is unverified and not worth finding out: the domain is the product's
public name, it is the cheapest line in the budget (~₹800–1,200/year for a
`.in` or `.com` at October 2026 list prices), and phase 10 §3B already
recorded it as the first purchase when money returned. It has.

What the domain also *removes*: the wildcard certificate and the custom
Caddy DNS-plugin build. Phase 10 needed a wildcard so Certificate
Transparency would not list per-person secret subdomains. With one app
behind OIDC there are no secret subdomains — `www`, `app`, `demo` are
all meant to be found. Stock Caddy with per-hostname HTTP-01 certificates
is enough, and one less custom build is one less thing to maintain. (The
phase-10 reasoning is recorded in `docs/phase-10/BRIEF.md` §1.2; it was
right then and does not apply now.)

### 1.3 The cap: who may sign in

Phase 12 §1.3 set the ten-person cap as **process, not code**: the LEDGER
roster was the counter and stamping an instance was the gate. In `oidc`
mode there is no stamp. The sign-in callback creates a user row for any
Google identity that completes the dance (`internal/store/sessions.go`) —
phase 13's brief states it: "no invite system yet". A hosted `app.<domain>`
with no gate is open to the world, which is exactly what the charter's
waitlist exists to prevent.

Three ways to hold the cap, in §3C. The recommended one keeps it process,
not code: a Google OAuth client left in **Testing** publishing status
admits only the identities listed as test users (Google's cap is 100). The
operator adding an address to that list *is* the entry act — one console
step replaces `new-instance.sh`. Google refuses everyone else before our
callback ever runs.

### 1.4 Two CLI commands that do not know about accounts

`roadbook backup`, `roadbook restore`, and `roadbook route` pass
`store.SelfUser` — the one implicit owner of auth-off mode (phase 13 CP1:
"CLI passes store.SelfUser everywhere"). On a hosted `oidc` stack:

- **Backup.** `roadbook backup` archives the `self` user's decisions and
  photos — on the hosted stack, nobody's. Phase 10's `backup-instance.sh`
  (which calls it) would produce empty archives nightly and report success.
  §3D's answer is a whole-instance backup (database dump plus the two
  volumes), which is the right unit for one shared stack anyway: it carries
  users, sessions, share-link hashes, photo records — everything the
  per-person format deliberately leaves out.
- **Route.** `roadbook route` would route the `self` user's confirmed
  adventures — again nobody's. Hosted users' gaps stay **unknown**: drawn
  dashed, labelled, honest (invariant 7's null-router degradation,
  exercised for real). §3F asks whether that is accepted for this phase or
  whether a small `-all-users` flag on the batch commands is pre-approved
  as the one product-code change. The pure core is untouched either way.

### 1.5 Off-machine backups with a staleness alarm

Phase 10's chain was host → laptop pull → iCloud. The phase-12 teardown
found the laptop pull had been **silently stalled for about three weeks**
behind Tailscale SSH's periodic re-authentication; the host-side timer
never missed, so nothing complained. Two lessons, both taken here: the
copy must leave the host **without the laptop in the loop**, and
*something* must notice when a night is missed.

The shape: the host encrypts the nightly archive to the backup public key
(the secret half still never touches the host — phase 10 §1.5 unchanged)
and pushes it to a free-tier object store (§3D) using a credential that can
**write but not delete** (so a compromised host can add noise, never erase
history). After a successful push the job pings a dead-man's-switch
service; if the ping is late, the operator gets an email. The alarm service
sees a timestamp and nothing else. A missed night is tested on purpose at
CP3 by disabling the timer once and waiting for the mail.

### 1.6 Custodianship, restated for a shared instance

Phase 10 §1.6 wrote the custody sentence for isolated instances ("your own
instance, separate database"). It must be rewritten, because the honest
version is different: one database, your rows, filtered on every query,
proven by a standing test family; share links are the only way anything of
yours becomes visible, and you mint and revoke them; deletion is one button
and removes everything structurally; backups are encrypted with a key the
server does not hold and are whole-instance, so your data sits inside them
until they age out (state the retention window). The landing (`landing/
index.html`, §470–523 today) and the entry email say "your own instance";
both are rewritten in CP2.

### 1.7 The operator path without the re-auth trap

Phase 10 used Tailscale SSH (the tailnet both as network path and as
authenticator). Its periodic browser re-authentication is what stalled the
backups. This phase keeps the tailnet as the **network** path (public 22
stays closed at both the host and the provider firewall) but runs plain
`sshd`, key-only, bound to the tailnet interface — no interactive re-auth
check can block an unattended job, and the break-glass path is still the
provider's web console.

---

## 2 What gets built

- **A provisioned, hardened host** (§3A): non-root user, key-only sshd on
  the tailnet interface, firewall allowing 80/443 and the tailnet,
  unattended security upgrades, Docker, swap sized for on-host image
  builds (measured at CP1 — a Next.js production build is the heaviest
  thing the host ever does).
- **A purchased domain** with DNS at an API-capable provider (§3B), stock
  Caddy with HTTP-01 certificates for `www`, `app`, `demo`.
- **The landing and waitlist redeployed**: `deploy-landing.sh` and
  `setup-waitlist.sh` as committed, re-addressed to the new domain; landing
  copy updated from "your own instance" to the account model (§1.6).
- **The app stack** in `oidc` mode: a Google OAuth client in Testing
  status, `ROADBOOK_PUBLIC_URL=https://app.<domain>`, the live-Google
  sign-in walk phase 13 carried, two real identities proving isolation on
  the hosted stack, the operator's own archive uploaded from a phone.
- **The demo stack** (optional, §3E) in `off` mode behind basic auth with
  the demo dataset — the e2e target and the "see it" link for the entry
  mail.
- **Entry v2**: the entry-email template rewritten (no credential — "sign
  in with this Google address"), the runbook's entry process as: oldest
  waitlist line → add as OAuth test user → send mail → delete line.
- **Durability** (§3D): nightly whole-instance archive, encrypted on host,
  pushed off-host with a write-only credential, dead-man alarm, restore
  drill from the off-host copy into a scratch stack, reboot drill.
- **Runbook v3** in `docs/private/pilot/`, LEDGER continued, the
  phase-10 per-person scripts retired by a dated commit (not kept "just in
  case" — git history keeps them, as phase 10 did with the laptop flavour).
- **The proof**: one real person, landing through life map, zero ad-hoc
  operator work — phase 12's CP4, finally reachable.

## 3 The real choices

Prices below were checked on 2026-10-16 from secondary sources and must be
re-verified on the provider's own page at purchase; several conflicted.

### 3A Provider and region

*Superseded 2026-10-27 by the maintainer's decision — see §8. Kept as the
record of what was weighed on 2026-10-16.*

| option | for | against |
|---|---|---|
| **Hetzner CX23 (EU: Falkenstein/Nuremberg/Helsinki)** — 2 vCPU, 4 GB, 40 GB; €5.49–5.99/month after the April 2026 increase (~₹550–600) | mature provider; 4 GB makes on-host image builds comfortable; snapshots, firewall, and a backup add-on exist; the phase-10 fallback on record | ~150 ms from India; several sources report the CX line out of stock in October 2026 — check on the day; Singapore region starts at €15.49, out of budget; EUR billing needs an internationally usable card |
| **India-region VPS, 2 GB** — small vendors (Vyom, HeavenCloud, BigRock class) at ₹330–450/month; DigitalOcean Bangalore / Linode Mumbai are $5–6 for 1 GB, ~$12 for 2 GB | lowest latency for an audience that is still mostly in India; data in-country; INR billing | small-vendor quality and continuity unknown; 2 GB is tight for on-host builds (needs swap or a registry); the known providers cost roughly twice Hetzner at 2 GB |
| **Oracle Always Free, retried with a card** — ₹0/month if an A1 2/12 ever launches; PAYG tenancies are reported (unverified) to keep a larger free allowance | free; the phase-10 host scripts targeted it | three weeks of 15-minute attempts found zero capacity in Hyderabad; the PAYG upgrade never activated; building a stranger-facing product on capacity that may never arrive is the phase-12 §0 problem again |
| PaaS (Fly, Railway, Render, Koyeb) — $5–10/month realistically | managed | fragments the compose stack into three managed services and discards the self-host-equals-hosted property; Fly's free credit is gone for new organisations, Render's free tier sleeps; rejected at phase 8 and again here |
| Home hardware behind Cloudflare Tunnel — ₹0 recurring | no port forwarding | Cloudflare's free plan caps request bodies at 100 MB (the product accepts 2 GiB exports; the operator's own is 52 MB); reintroduces the lid-open and home-ISP dependence phase 8 existed to escape; rejected |

**Recommendation: Hetzner CX23 if in stock on the day; otherwise
DigitalOcean Bangalore at 2 GB with swap.** The 150 ms is a first-paint
cost on a product whose pages are server-rendered once and then idle; the
reliability and tooling difference is worth more than the latency to a
ten-person pilot. Residency: EU is a well-understood regime and the custody
sentence can say "Germany" or "Finland" as plainly as it said "Hyderabad."
If the maintainer weighs in-country residency above price, the India option
is the recorded alternative, taken without a new gate. The Oracle retry is
not recommended as a plan; if the maintainer wants it, it runs in parallel
with a fixed deadline and nothing else waits on it (links re-point).

### 3B Registrar and DNS

*Superseded 2026-10-27 by the maintainer's decision — see §8. Kept as the
record of what was weighed on 2026-10-16.*

| option | for | against |
|---|---|---|
| **Any registrar + Cloudflare DNS, free tier, DNS-only (grey cloud)** | phase 10's recommendation, unchanged; instant API if DNS-01 is ever wanted again; registrar-independent | a second account; must stay DNS-only — proxied mode would terminate TLS at Cloudflare, inside the connection to location data |
| Registrar's own DNS (Porkbun/Namecheap class) | one account | fine for HTTP-01, which is all this brief needs; weaker API story if the wildcard ever returns |

**Recommendation: registrar of the maintainer's choice; Cloudflare DNS-only
as the default.** Since HTTP-01 suffices (§1.2), either works; Cloudflare
is chosen for the option value. The domain name itself is the maintainer's
decision at this gate (§7) — it is the product's public name, and phase 10
already recorded that re-homing twice is waste.

### 3C How the cap is held

| option | for | against |
|---|---|---|
| **Google OAuth client in Testing status; test-user list is the roster** | zero product code; Google refuses non-listed identities before our callback; up to 100 names; adding a name is the entry act | Google-specific (a self-hoster with Keycloak has their own allowlist already); Testing status may show a consent-screen warning for non-sensitive scopes — verified at CP2's live walk, not assumed; a client in Testing status cannot be "published" without Google's verification process when the cap lifts (phase 16's problem) |
| An allowlist in Go (`ROADBOOK_OIDC_ALLOWED_EMAILS`, checked in the callback) | provider-neutral; one place, one test | product code in a zero-diff phase; a second roster to keep in step with the first |
| Basic auth in front of the whole app stack | phase-8 machinery, known | a shared secret for ten strangers is a credential nobody can rotate without telling everyone; also breaks the OIDC redirect on some in-app browsers |

**Recommendation: Testing status, as described.** If CP2's live walk shows
the consent-screen experience is unacceptable for strangers, the Go
allowlist becomes a named finding at that STOP (small, store-free,
callback-only — but product code, and decided openly).

### 3D Backup unit and destination

| option | for | against |
|---|---|---|
| **Whole-instance: `pg_dump` of the one database + tar of `photos` and `uploads` volumes, encrypted on host with `age`** | the right unit for one shared stack: users, sessions, share hashes, photo records, decisions, retained exports — all of it, restorable as a whole; zero product code; `pg_dump` runs inside the db container, nothing new installed | the per-person durable-identity format (`roadbook backup`) is not what is restored — fine, because one stack restores as one stack; a per-user export remains the carried user-facing item from phase 13 |
| Extend `roadbook backup` with `-all-users` | keeps the phase-5 format | product code; still omits users/sessions/share links; a solution looking for a reason |

| destination | for | against |
|---|---|---|
| **Object storage, pushed from the host** — Backblaze B2 (10 GB free) or Cloudflare R2 (10 GB free), with a write-only application key and bucket versioning | durable the moment the push completes; no laptop in the loop (the phase-12 lesson); a write-only key means a compromised host cannot delete history | a new account and a credential on the host; ~1 GB per nightly archive at the operator's archive size means the free tier holds a week or two — a retention script prunes *from the laptop*, where a delete-capable key may live |
| Laptop pull (phase 10's chain) | proven, no new account | proven to stall silently; rejected as the primary |
| Provider snapshots / backup add-on | one click | same provider, same jurisdiction, same account — correlated failure; fine as a second copy, never the only one |

**Staleness alarm:** a dead-man's switch (healthchecks.io free tier or
equivalent: the host pings a URL after each successful push; a missed ping
emails the operator). It sees a timestamp, never data.

**Recommendation: whole-instance archive, pushed to B2 with a write-only
key, dead-man alarm, provider snapshot weekly as the correlated second
copy.** The age secret stays where it is (password manager + laptop); a
restore is "laptop decrypts, pipes over ssh" as the phase-10 runbook §5
already does.

### 3E The demo stack

| option | for | against |
|---|---|---|
| **Keep a demo stack in `off` mode behind basic auth** | the e2e suite's target (`ROADBOOK_E2E_URL`) on the real host; a "see it populated" link for the entry mail (phase-10 evidence: a populated instance gives no introduction, and the landing fronts screenshots — a live link beside them is cheap); ~100 MB | one more stack, one more credential in the LEDGER |
| Screenshots only | nothing to run | loses the e2e target on the host and the live look |

**Recommendation: keep it.** It costs a hundred megabytes and the stamp is
already scripted.

### 3F Routing for hosted users

| option | for | against |
|---|---|---|
| **Accept unrouted gaps on the hosted stack this phase** | zero product code; the product degrades exactly as invariant 7 designed — dashed, labelled "unknown"; the first real exercise of that state on strangers' data | hosted users see less than the operator's local instance does |
| Pre-approve `-all-users` on `roadbook route` (and the same flag on `backup`/`restore` for symmetry) as the phase's one product-code change | hosted users get road geometry from the public OSRM endpoint batch (1 s politeness, as built); small, CLI-only, pure core untouched | breaks the zero-diff target; needs its own tests in the cross-tenant family; the public endpoint's terms at ten users' worth of pairs should be checked first |

**Recommendation: accept unrouted for this phase**, record the flag as the
first item of the hosted backlog, and let the first stranger's reaction to
dashed gaps decide its priority. Honesty is the product's claim; this is
the claim exercised.

### 3G The cost ceiling

*Superseded 2026-10-27 by the maintainer's decision — see §8. Kept as the
record of what was weighed on 2026-10-16.*

To be set by the maintainer. Proposed: **domain ≤ ₹1,200/year; host ≤
₹700/month all-in; storage and alarm on free tiers at ₹0.** Any month
wanting more returns to a STOP. As in phase 10, the number is a leash.

---

## 4 Verification plan

- **External view only.** Every "it is closed" or "it answers" claim is
  checked from outside the host (a phone on cellular, or `curl` from the
  laptop with the tailnet off) — the phase-10 lesson that validation lied
  and the outside view did not.
- **The live-Google walk (CP2).** Two real Google identities on the hosted
  stack: each sees only their own data; a non-test-user identity is refused
  by Google; sign-out revokes; `DELETE /me` leaves zero rows and empty
  volumes for that user and nothing else changed. The phase-13 carried item
  closes here.
- **Scale on the host (CP2).** The operator's real archive uploaded from a
  phone through the hosted stack; the reference candidate count reproduced;
  build and detection peak memory recorded in the LEDGER.
- **The missed night (CP3).** The backup timer disabled once; the alarm
  mail arrives; the timer re-enabled; the next archive lands.
- **Restore from the off-host copy (CP3)**, into a scratch stack on the
  host, counts verified before and after.
- **The proof (CP4).** One real person, zero ad-hoc operator work, from the
  landing to a life map. Recorded in the LEDGER like any pilot entry.
- Throughout: `git diff --stat` over `cmd/`, `internal/`, `web/`, `api/`,
  `compose.yaml`, `Dockerfile` empty at close (unless §3F's fallback was
  taken at a STOP); no real name, address, or credential in anything
  committed.

## 5 Checkpoints

1. **CP1 — host and front.** Provider account, host provisioned and
   hardened (§1.7), domain bought, DNS set, stock Caddy serving the landing
   and waitlist at `www.<domain>` with the form round trip repeated; demo
   stack up at `demo.<domain>` behind basic auth; on-host image build timed
   and measured; external port scan clean. *Visible: the landing from a
   phone on cellular; nothing but 80/443 answers from outside.* The spend
   begins here and nowhere earlier.
2. **CP2 — the app.** OAuth client in Testing status; app stack in `oidc`
   mode at `app.<domain>`; the live-Google walk; the operator's archive
   through the phone; landing copy and entry mail rewritten for accounts;
   entry process v2 drilled on the operator's own second identity. *Visible:
   two accounts on one stack, each blind to the other; a refused stranger.*
3. **CP3 — durability.** Nightly archive, push, alarm, missed-night test,
   restore drill, reboot drill, unattended upgrades, weekly snapshot.
   *Visible: an archive in the bucket, an alarm mail, a restored scratch
   stack with the right counts.*
4. **CP4 — the proof and the close.** One real person enters; README's
   hosted statement; runbook v3; the retired scripts removed in a dated
   commit; LOG.md. *A phase is not complete until its log exists.*

STOP at each checkpoint per the working agreement.

## 6 Explicitly excluded

Product features of any kind; lifting the cap or publishing the OAuth
client (phase 16 territory, needs Google verification and the charter's
"strangers sign themselves up" decision); per-user export; life-map
sharing; an allowlist in Go unless §3C's fallback fires; OSRM on the host
(the public endpoint batch is the hosted routing story if §3F's flag is
ever taken); IaC — one host still does not earn Terraform; a status page;
monitoring beyond the backup alarm and the provider's own.

## 7 Gate 1 questions for the maintainer

1. §3A — Hetzner EU (recommended) or India-region? Is an internationally
   billable card available, or does INR billing decide it?
2. §3B — the domain name. This is the product's public name; which name,
   and `.in` or `.com`?
3. §3C — Testing-status cap as recommended?
4. §3D — B2 with a write-only key plus a dead-man alarm, as recommended?
5. §3F — accept unrouted gaps for hosted users this phase, or pre-approve
   the `-all-users` CLI flag as the one product-code change?
6. §3G — the ceiling numbers.
7. Should the Oracle retry run in parallel with a deadline, or be dropped?

---

## 8 Gate 1 answers recorded 2026-10-27

The maintainer's hosting decision, taken 2026-10-27 and handed to this
brief as a statement to record, not to work around. Prices are the
maintainer's own figures at decision time; the standing rule holds — re-check
on the order page at purchase (setup fee on a one-month term, any region
surcharge).

### 8.1 Decided

| question | answer |
|---|---|
| §3A provider and region | **Contabo Cloud VPS 10, Asia (India / Navi Mumbai)**, one-month term, Ubuntu 24.04, ~$7.79/month incl. 18 % GST (~₹740). Known trade-off, accepted: the cheapest plan benchmarks poorly on CPU and disk; acceptable for about ten testers; everything runs in Docker so moving is cheap. CP1 records the measured image-build and import timings so the trade-off is a number, not a feeling. |
| §3B address | **`getroadbook.app`**, bought 2026-10-27 (the decision of the same morning had deferred it behind a free sslip.io hostname; the purchase came the same day, so sslip.io is never used). Hostnames: `www` landing + waitlist, apex redirects to `www`; `app` the accounts stack; `demo`. Stock Caddy, HTTP-01 per hostname (§1.2 holds again). DNS: four A records at the registrar's DNS or Cloudflare in DNS-only mode — never proxied (TLS would terminate inside the path to location data, and the free plan's 100 MB body cap would break uploads). |
| §3G ceiling | **₹2,000/month** all-in. Any month wanting more returns to a STOP. |
| §7 Q7 Oracle retry | **Dropped** (sign-up and capacity both failed for the maintainer). |
| Plan B | GigaNodes Cloud Nano (~₹708 incl. GST, 2 GB / 20 GB) or Cloud XS (~₹1,180) if Contabo's slowness bothers testers. |

Rejected, with the maintainer's reasons: Vercel + Render + Neon free tiers
(Render sleeps with ~1 min cold starts; large Timeline uploads hit size and
time limits; three services to run); Railway/Render paid (usage billing;
Postgres pushes cost to $10–40/month); Tailscale Funnel from the laptop (down
whenever the laptop sleeps — phase 8's lesson); Oracle Always Free (sign-up
and capacity failed); Hostinger ₹599 (24 months prepaid, renews at ₹999);
DigitalOcean/Vultr/Linode Bangalore or Mumbai (reliable, ~₹1,100–1,350 incl.
GST for 2 GB); WebEyeSoft (renewal price unclear, manual KYC); Hetzner (no
India region; the cheap plans were not orderable).

### 8.2 The stated plan

1. Server hardening: non-root sudo user, SSH key only, ufw allowing 22/80/443.
2. Docker and one compose file: postgres (named volume, no public port), api
   (Go, runs goose migrations on start), web (Next.js), caddy (the only
   public service).
3. Caddyfile: `/api/*` to api, everything else to web; uploads go straight
   to the Go API, not through Next.js.
4. Nightly `pg_dump` on the server plus a weekly copy off the server.
5. Full end-to-end test on the maintainer's phone, then two friends.

Repo rule restated: the server IP and every secret live in an uncommitted
`.env`, never in docs or compose files.

Points 2 (postgres with no public port; api migrates then serves) already
describe `compose.yaml` as committed. Point 5 is CP2 and CP4 of §5.

### 8.3 Where the stated plan meets the built product — needs a word

Reported, not resolved. Each is a gate question in its own right.

**(a) Plan point 3 — `/api/*` to Go, uploads bypassing Next.** Three facts
from the repository:

- The Go contract (`api/openapi.yaml`) has no `/api` prefix. Its paths are
  `/imports`, `/candidates/…`, `/auth/…`, `/shared/…`, `/me`. A Caddy rule
  sending `/api/*` to the api container reaches nothing there.
- `/api/*` is the **Next app's** namespace: `web/app/api/auth/{signin,
  callback,signout}` (the sign-in dance, phase 13), `web/app/api/imports`
  (the streaming upload proxy, phase 7), the thumbnail proxies, and the
  share-token thumbnail routes. Stealing that prefix breaks sign-in and
  uploads in one line.
- The architecture in `CLAUDE.md` is that the Go service is **not exposed
  to the browser**: one network boundary, no CORS or auth questions at the
  edge. Phase 13's cookie trust is built on that one hop. Exposing Go
  publicly is a product-architecture change, not a hosting choice.

The concern behind the point — large uploads timing out in a Node proxy —
was met in phase 8: the Next route handler streams the body
(`duplex: "half"`, no spooling), and `web/server-timeout.cjs` lifts Node's
request timeout (`ROADBOOK_REQUEST_TIMEOUT_MS`, default one hour); a 52 MB
archive over a 220 KB/s relay landed in 295 s. What the host needs instead
is a Caddy site that sends **everything to web** with no body limit (Caddy's
default) and no proxy read/response timeout shorter than an upload.
Recommendation: plan point 3 becomes "everything to web"; zero product code.

**(b) Plan point 1 — public port 22, key-only.** This supersedes §1.7's
tailnet-bound sshd. The reason §1.7 avoided Tailscale SSH — its periodic
browser re-authentication stalling unattended jobs — is equally avoided by
plain key-only sshd, so nothing in the backup chain depends on this. The
delta is exposure: 22 answers from the internet. Recommendation if taken:
`PasswordAuthentication no`, `PermitRootLogin no`, and the provider's
console as break-glass; the external port scan at CP1 then expects three
open ports, not two. Recorded as the maintainer's choice.

**(c) Plan point 4 — `pg_dump` nightly, weekly off-server copy.** Two gaps
against what is being hosted and against the charter line:

- `pg_dump` covers the database only. The `photos` volume (thumbnails —
  originals are discarded at upload) and the `uploads` volume (retained
  exports, possibly a user's only copy of an expired `rawSignals` window)
  are irreplaceable and sit outside the database. §3D's unit — dump plus
  both volumes, encrypted on the host with the backup public key — is the
  whole-instance answer; the age secret still never touches the host.
- PLAN.md charters this phase with "off-machine backups **with a staleness
  alarm**" (the phase-12 lesson: a silently stalled copy for three weeks).
  The weekly off-server copy has no destination named yet and no alarm.
  §3D's recommendation (push from the host to free-tier object storage with
  a write-only key; dead-man ping after each push) stands as the proposal;
  if the maintainer wants pg_dump-plus-weekly-copy instead, the charter
  line is amended openly at this gate, not left unmet.

**(d) Plan point 2 — Caddy inside the compose file.** `compose.yaml` is the
self-host reference and the phase's zero-diff target covers it. Caddy, the
static landing, and `waitlistd.py` belong to a **hosting overlay** — either
a host-level Caddy as phase 10 ran, or a `compose.hosted.yaml` layered over
`compose.yaml` the way `scripts/pilot/compose.instance.yaml` was — so a
self-hoster's `docker compose up` still mentions Caddy zero times. The
landing and waitlist still need a hostname: sslip.io resolves any prefix
(`www.<ip-dashes>.sslip.io`, `app.…`, `demo.…`), so §1.1's three-hostname
topology survives the deferred domain. Verified at CP1, not assumed.

**(e) sslip.io against §1.2 — RETIRED 2026-10-27 by the domain purchase; kept as the record of a risk that lasted one morning.** §1.2 argued a real domain is a precondition
because Google's OAuth client needs a redirect URI on a public hostname and
the consent screen wants an authorised domain. The maintainer's decision
defers the domain; the argument therefore becomes a **risk tested at CP2**:
if Google refuses an sslip.io redirect URI or authorised domain, the
`getroadbook.app` purchase moves forward from "later" to a CP2
precondition. When the domain arrives: `ROADBOOK_PUBLIC_URL` changes, the
OAuth redirect URI is re-registered, every session cookie is per-host so
everyone signs in again; share-link URLs are built from the page's origin
at display time, so minted tokens survive the move.

### 8.4 Gate closed 2026-10-27

The maintainer said "go ahead" with the domain bought and the host chosen.
By the phase-14 Gate-1 precedent, the brief's recommendations stand for
everything not decided otherwise: §3C Testing-status client, §3D
whole-instance archive pushed off-host with a dead-man alarm, §3E keep a
demo stack, §3F unrouted gaps this phase, §8.3(a) everything to web,
§8.3(c) dump plus both volumes, §8.3(d) host-level stock Caddy as the
hosting overlay (reuses `deploy-landing.sh` and `setup-waitlist.sh` as
committed). Any of these is one word from the maintainer to reverse before
its checkpoint. The original list, for the record: §3C (how the cap is held — Testing-status client recommended), §3D
(destination and alarm, see (c)), §3E (keep a demo stack — recommended),
§3F (unrouted gaps for hosted users this phase — recommended). The §7
questions 3, 4, 5 are these; questions 1, 2, 6, 7 are answered above.
