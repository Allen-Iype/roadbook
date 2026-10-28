# Phase 15 — decision log

Three lines each: what was chosen, what was rejected, what would change our
mind. Written as decisions are made, not reconstructed.

## 2026-10-16 — Charter: phase 15 hosting confirmed at the post-phase-14 STOP

- **Chosen:** hosting on a money-returning platform as the next phase, per
  the roadmap's sequence (PLAN.md "The road to strangers"); the brief opens
  with the provider decision the zero-cash constraint deferred, and takes
  the phase-13 topology change (one `oidc` stack, not one stack per person)
  as its first input. Brief before any code or purchase.
- **Rejected:** re-running phase 10's instance-per-person topology with a
  paid host (the finished product is the accounts mode; stamping ten
  stacks would host the product phase 13 replaced); pulling any product
  feature into the phase.
- **Would change our mind:** the brief's two seams (§1.3 cap, §1.4 CLI
  owner) proving unsolvable without product code — then the zero-diff
  target is amended openly at the gate, not eroded at a checkpoint.

## 2026-10-16 — Gate 1: brief written, awaiting review

- Seven questions in BRIEF §7. Recommendations: Hetzner CX23 EU; buy the
  domain now (OIDC makes it a precondition); Cloudflare DNS-only; stock
  Caddy with HTTP-01 (the wildcard is no longer load-bearing); cap via the
  OAuth client's Testing status; whole-instance archive pushed to B2 with
  a write-only key plus a dead-man alarm; keep a demo stack; accept
  unrouted gaps for hosted users this phase.

## 2026-10-27 — Gate 1 §3A/§3B/§3G: provider, address, and ceiling (maintainer's decision)

- **Chosen:** Contabo Cloud VPS 10, Asia (India / Navi Mumbai), one-month
  term, Ubuntu 24.04, ~$7.79/month incl. 18 % GST (~₹740). Address deferred:
  free `<ip-with-dashes>.sslip.io` with Caddy automatic HTTPS now;
  `getroadbook.app` pointed at the same server later. Ceiling ₹2,000/month.
  Plan B: GigaNodes Cloud Nano / Cloud XS if Contabo's slowness shows.
  Known trade-off accepted: the cheapest plan benchmarks poorly on CPU and
  disk; fine for ~10 testers; Docker makes a move cheap. Full list of
  rejected providers and reasons in BRIEF §8.1.
- **Rejected:** the brief's Hetzner EU recommendation (no India region, cheap
  plans not orderable); the Oracle retry (§7 Q7 — sign-up and capacity both
  failed); DO/Vultr/Linode India at ~₹1,100–1,350 for 2 GB; PaaS free tiers
  (sleep, upload limits, three services) and paid (usage billing); the
  laptop funnel; Hostinger's 24-month prepay; the brief's proposed ₹700
  host ceiling in favour of ₹2,000 all-in.
- **Would change our mind:** CP1's measured build and import timings, or
  testers' reports, showing Contabo's slowness in use → plan B; Google
  refusing an sslip.io redirect URI or authorised domain at CP2 → the
  domain purchase becomes a CP2 precondition instead of "later".

## 2026-10-27 — Operator path: public SSH, key-only (maintainer's plan point 1)

- **Chosen:** ufw allowing 22/80/443, non-root sudo user, key-only sshd on
  the public interface; provider console as break-glass.
- **Rejected:** BRIEF §1.7's tailnet-bound sshd with public 22 closed. The
  re-auth trap §1.7 existed to avoid is avoided equally by plain key-only
  sshd, so the backup chain does not depend on this.
- **Would change our mind:** brute-force noise in the auth log at CP1 or a
  provider firewall that makes closing 22 free → bind sshd to the tailnet
  again. The CP1 external scan expects 22/80/443 open and nothing else.

## 2026-10-27 — Four plan points flagged against the built product, not resolved

- Recorded in BRIEF §8.3 for the maintainer's word before the gate closes:
  (a) `/api/*` to Go and uploads bypassing Next would break sign-in and
  uploads and expose the Go service to the browser, against CLAUDE.md's
  architecture — the upload-timeout concern was met in phase 8; (c) nightly
  `pg_dump` alone omits the photos and uploads volumes and the charter's
  staleness alarm; (d) Caddy belongs in a hosting overlay, not in the
  reference `compose.yaml`; (e) sslip.io turns §1.2's domain precondition
  into a risk tested at CP2. No decision taken here; the entries above
  record only what the maintainer decided.

## 2026-10-27 — Domain bought: getroadbook.app; Gate 1 closed

- **Chosen:** `getroadbook.app` purchased the same day the sslip.io deferral
  was recorded; §1.2's precondition is met and §8.3(e) retires. Hostnames
  `www` (landing, apex redirects there), `app`, `demo`; stock Caddy with
  HTTP-01; DNS-only if Cloudflare. Gate 1 closed on the maintainer's "go
  ahead": the brief's standing recommendations are taken for every point
  not decided otherwise (§3C/§3D/§3E/§3F, §8.3 a/c/d), per the phase-14
  precedent, each reversible by a word before its checkpoint.
- **Rejected:** sslip.io (never used); Caddy inside `compose.yaml` (the
  reference file stays zero-diff — host-level Caddy via apt is the
  overlay, and the phase-12 scripts already assume it); `/api/*` routed to
  Go (breaks sign-in and uploads; exposes Go — BRIEF §8.3a).
- **Would change our mind:** a hostname the phase-12 scripts cannot express
  (they write `www.$ROADBOOK_DOMAIN`) → edit the scripts, not the
  topology; Google's consent screen rejecting `getroadbook.app` as an
  authorised domain at CP2 — unlikely for a bought domain, verified then.

## 2026-10-27 — Order form: EU region, SSD, no provider backup

- **Chosen:** Contabo VPS 10 ordered in the EU region at $7.79/month; the
  plan ships 100 GB SSD only (the NVMe variant third-party listings
  described is not on the form); provider auto-backup and every add-on
  declined; the included snapshot is the correlated second copy §3D
  allowed for.
- **Rejected:** the India region at +$3/month — the extra ~150 ms affects
  only server round trips (tiles come from OpenFreeMap), pages render once
  and idle, uploads are bound by the phone's uplink; no residency
  requirement at ten friends. Contabo's auto-backup — same provider and
  account, unencrypted location data under their control; CP3's encrypted
  off-provider chain is the backup.
- **Would change our mind:** testers reporting slowness that the CP1/CP2
  measurements attribute to round trips rather than disk → India region by
  new order + CP3 restore + DNS re-point (one evening, TTL 300 set from
  the start); CP1 build/import timings showing the SSD as the bottleneck →
  plan B provider.

## 2026-10-27 — CP1 built: host hardened, front live, demo stack up

- **Chosen:** hardening order user-first (operator user proven from
  outside before root and password login were closed); sshd override in a
  `00-` file because sshd keeps the FIRST value and cloud-init's `50-`
  file had re-enabled passwords; ufw 22 LIMIT/80/443; fail2ban; swap 2 GB;
  Docker CE from Docker's repository. Front: stock Caddy from the Caddy
  apt repository, one explicit-hostname site block importing the
  committed snippets unchanged (landing, instances), apex redirecting to
  `www`, ACME contact at the domain mailbox; `app.<domain>` joins at
  CP2. Demo stamped with `new-instance.sh` and loaded through the
  browser path; its first `up` hit the known initdb race and the aborted
  stamp's unprinted credential was rotated (LEDGER has the traps).
- **Rejected:** the wildcard certificate and the DNS-plugin Caddy build
  (no secret subdomains remain); Caddy inside `compose.yaml`; cloud-init's
  sshd defaults; auto-reboot from unattended-upgrades before CP3's drill.
- **Would change our mind:** an instance snippet needing something the
  explicit-hostname block cannot express → revisit the Caddyfile shape at
  CP2 when `app` joins; the 143 s on-host build growing with the image →
  a registry-built image (BRIEF §2 noted swap/registry as the fallback).

## 2026-10-28 — The cap moves into the product (BRIEF §9); the zero-diff target gains its one named exception

- **Chosen:** open Google sign-in with the cap enforced by the product:
  `ROADBOOK_USER_CAP` (default 0 = unlimited, `oidc` mode only), a
  counting insert under a named advisory lock in `UpsertOIDCUser`,
  returning identities never refused, a refused identity sent to the
  sign-in page's full state with the waitlist link and no cookie, a
  deleted account freeing its seat; the OAuth client published rather
  than held in Testing; the entry-email and admit-from-the-list process
  retired. Decided at a STOP on the maintainer's restatement of the
  August model — the process-shaped cap was phase 12's answer for a
  product without accounts and had been carried forward unexamined.
- **Rejected:** the Testing-status roster (admission by console); a Go
  email allowlist (a second roster); auto-enrolling a refused identity's
  address (breaks the landing's one-file custody sentence); any new
  status code or contract type (the existing 302 carries the refusal).
- **Would change our mind:** Google demanding verification for a
  published client with `openid email` → Testing roster as the interim
  gate, recorded; ten seats filling in days → the number revisited at a
  STOP.

## 2026-10-28 — CP2 built and walked: the accounts stack is live behind Google sign-in with the cap

- **Chosen:** the cap landed as BRIEF §9 specified (store, API, CLI,
  compose, web, tests, landing); the OAuth client was published — the
  console's own wording recorded in §9.3's spirit: "available to any user
  with a Google Account"; verification only for more than 10 domains, a
  logo, or sensitive scopes; the consent screen itself showed no
  unverified-app warning on the walk. The live walk ran at cap 1 then 10: a second
  identity was refused to the full page and reached the waitlist form; it
  was admitted after the raise and saw nothing of the first; it deleted
  itself and the seat count fell. Two findings fixed during the walk: the
  landing gained a header sign-in link and a hero call to action (the
  only link had been at the foot of the page), and the delete control is
  enabled for an empty account in accounts mode (an account holds a seat
  even with no data). The hosted imports page no longer shows the CLI hint.
- **Rejected:** keeping the delete control data-gated in accounts mode (a
  person who never uploads could not leave); a logo on the consent screen
  (it alone triggers Google's verification process).
- **Would change our mind:** Google's unverified-app lifetime cap (100
  users, counted over the app's life) → when the product's cap approaches
  it, verification becomes a phase-16 task; a stranger stalling at the
  consent screen → revisit the branding fields.

## 2026-10-28 — CP3 gate: the durability chain's shape (BRIEF §10)

- **Chosen:** whole-stack archive (one `pg_dump` inside the db container
  with `countries`/`states` data excluded as regenerable, plus both file
  volumes), checked every night and pushed only when a content
  fingerprint — SHA-256 of the plain dump plus a volume listing — changed;
  age-encrypted on the host to the reused phase-10 public key; pushed by
  rclone to **Cloudflare R2** on the existing account, the bucket under a
  90-day bucket lock with a lifecycle rule deleting at 90 days; one
  healthchecks.io check pinged every night with a per-stack body
  (pushed / unchanged, skipped), `/fail` on any error; automatic reboots
  enabled after the reboot drill; provider snapshot once per checkpoint
  close; the phase-10 backup trio retired in the CP3 commit; a private
  laptop continuity archive (docs/private, CLAUDE.local.md, memory) to
  iCloud at each checkpoint close.
- **Rejected:** Backblaze B2 (the first draft's choice for its add-only
  keys — the maintainer asked for the existing Cloudflare account, and
  R2's bucket lock gives equivalent protection with one account fewer);
  laptop-side pruning (the laptop as a moving part is the phase-12
  lesson); "newest row timestamp" as the fingerprint (the dump hash is
  complete over every table and costs seconds); one alarm check per
  stack; a weekly manual snapshot (a chore that would stall); a new key
  pair for the VPS era.
- **Would change our mind:** the bucket lock refusing a delete the
  lifecycle rule needs (the documentation says the lock is satisfied
  first; CP3 verifies on a real object) → lifecycle set beyond the lock
  age; archives growing past the free 10 GB inside 90 days → shorter
  retention or a paid tier at a STOP; a night-time reboot catching a real
  upload more than once → reboot time moved or disabled.

## 2026-10-28 — CP3 built and drilled: the chain is live and the archive travels in two parts

- **Chosen:** the backup splits into a database part and a files part,
  each with its own fingerprint and its own object in the bucket, after
  the maintainer's cost caution at the R2 activation screen: the
  database is small and moves on every sign-in, the files are large and
  move only on an upload, so a quiet night costs megabytes against the
  free 10 GB (measured: app 8.3 MB + 56 MB on the first push, then
  "unchanged" nights). The fingerprint ignores `pg_dump`'s random
  `\restrict` guard lines (PostgreSQL 17.6+/18 writes a fresh token into
  every plain dump — two dumps of identical data differed by exactly
  those two lines). Every `docker compose exec/run` in the host scripts
  reads from `/dev/null`: compose forwards stdin, and the first restore
  lost the bundle on stdin to the population-check query. The
  missed-night test ran with a 2-minute period so it fit the session;
  the first attempt produced no mail because the hello@ method was
  switched on for the check *after* it had already gone down — alerts
  are sent only at the moment of a status change — so the test was
  repeated from an "up" state and both mails arrived. Automatic reboots
  enabled at 23:30 UTC once the reboot drill passed (36 s unattended).
- **Rejected:** one archive per stack (the §10 draft — pushes the whole
  uploads volume on every sign-in night); a laptop rclone token for the
  drill (the host's rclone fetched the object; the documented no-host
  path is the dashboard download); keeping the `/start` ping out (it
  makes a hung run visible as a run that never closed).
- **Would change our mind:** a bucket listing nearing 8 GB → shorter
  lifecycle at a STOP; the initdb race on a fresh stamp (hit for the
  third time this phase) → a retry loop inside `new-instance.sh`; a
  night-time reboot catching a real upload → reboot time moved.
