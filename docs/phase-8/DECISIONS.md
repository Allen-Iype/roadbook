# Phase 8 decision log

Three lines each: what was chosen, what was rejected, what would change our
mind. Written at the moment of decision, not reconstructed.

## 2026-08-12 — Gate 1 passed: brief approved as amended

Chosen: BRIEF.md in its amended form (laptop + Funnel + local Caddy basic
auth, three-listener ceiling, upgrade path recorded behind named triggers) —
approved by the maintainer after review and two clarifying discussions
(single-port routing; laptop as the permanent first verification environment
for any future v2), neither of which changed the design.
Rejected: further amendment at this gate.
Would change: checkpoint evidence, logged per decision as always.

## 2026-08-12 — Phase 8 chartered: friend pilot hosting, brief before code

Chosen: phase 8 is the second half of v1 (the link with zero operator
involvement); BRIEF.md drafted with recommendations across §3A–§3E — host
(ARM VPS), front (domain + Caddy wildcard + secret subdomain + basic auth),
provisioning (project-per-tester template + script), custody (backup cron +
offsite `age` archives + restore drill), two small carried product changes
(`countries -if-empty`, `restart: unless-stopped`). All are proposals until
Gate 1.
Rejected: treating any recommendation as decided before the Gate 1 STOP.
Would change: Gate 1 amendments supersede the brief text; log them here.

## 2026-08-12 — This is an ops phase; product code stays still

Chosen: Go and web code untouched except the two carried items in BRIEF §3E,
both with evidence collected at the phase 7 close; everything else the phase
produces is host state, templates, and the runbook.
Rejected: folding in nearby product work (backup-format extension for
uploads, self-serve deletion, auth of any kind) — each has a recorded home
(backlog trigger, Direction 6) and an ops phase is the wrong place to design
product surface.
Would change: a Gate 1 scope amendment, or CP evidence that a product change
is required to meet a checkpoint's visible.

## 2026-08-12 — Gate 1 amendment: the pilot serves from the laptop, at zero cash cost

Chosen: the maintainer's direction at Gate 1 — links go to several friends
soon, no money is spent, and lid-open laptop availability is accepted
knowingly; BRIEF §3A/§3B/§5–§7 amended accordingly, with the rented-host
plan preserved as the upgrade path behind four named triggers (friend hits a
down instance; upload fails for availability reasons; a fourth concurrent
tester; willingness to spend).
Rejected: the brief's original ARM-VPS recommendation as the build target —
the assistant's recommendation, recorded here as not taken; the maintainer
decided.
Would change: any §3A trigger firing re-opens the choice with the original
recommendation on the table; everything built (Caddyfile, template, script,
runbook) is designed to transfer.

## 2026-08-12 — The front is Funnel's three ports behind local Caddy basic auth

Chosen: up to three concurrent instances on the laptop node's three funnel
listeners (443/8443/10000), each terminating at a local Caddy with
per-tester basic auth; the credential is each person's rotating secret,
because funnel hostnames are CT-logged and therefore public — the link is
an address, not a secret.
Rejected: link-only access (CT logs defeat URL secrecy for other people's
location data); Tailscale sidecar-per-tester for now (clean per-person
hostnames and >3 concurrency, but more machinery — the recorded extension
for when the ceiling binds).
Would change: basic auth failing in a phone's in-app browser (fallback
ladder in BRIEF §3B); CP1's relay-throughput test failing at archive scale
(early §3A trigger); a fourth concurrent tester (sidecars).

## 2026-08-12 — CP1 relay measurements: accept the ceiling, remove the cliff

Chosen: the funnel relay measured ~221 KB/s sustained with Node's default
5-minute request timeout producing an opaque 408 at ~65 MB — accepted for
the pilot with a runbook/handover note (the real 8-year export is 52 MB ≈
4 min, so realistic exports fit), plus a CP2 rider investigating raising
the standalone server's request timeout so oversized uploads fail honestly
slow instead of at a cliff; browsing through the relay is fine (~1.7 s/page).
Rejected: re-fronting at zero cost via DDNS + router port-forward (home IP
in the link, router surgery — recorded fallback), and calling the §3A VPS
trigger fired (maintainer holds zero-spend).
Would change: a friend's export exceeding the ceiling, or the timeout rider
failing — either promotes option C/D from fallback to plan.

## 2026-08-12 — CP1 reboot drill deferred, gated before the first friend link

Chosen: the maintainer cannot restart the machine now; the drill is deferred
with a hard rule — it must pass before any friend link goes out (CP4 gate at
the latest), and may ride the next OS-update restart; the settings half
(login items) is verified without rebooting.
Rejected: skipping the drill, or blocking CP2 on it — nothing in CP2 depends
on a reboot.
Would change: the drill failing when run — then the serving-posture checklist
gets fixed before CP4, not after.

## 2026-08-12 — The upload-timeout rider: a preload, injecting constructor options

Chosen: `web/server-timeout.cjs`, loaded via `node --require` in the web
Dockerfile CMD — wraps `http.createServer` and injects `requestTimeout`
(default 3,600,000 ms; `ROADBOOK_REQUEST_TIMEOUT_MS`, 0 = uncapped) into the
constructor options; proven both directions (a 40 s cap 408s a slow upload;
the deployed default carried a 6.5-minute upload to its proper sniffer 400,
where Node's 300 s default had produced an opaque 408).
Rejected: forking the generated standalone server.js (Next exposes no knob —
KEEP_ALIVE_TIMEOUT governs idle keep-alive, not in-flight requests);
setting `server.requestTimeout` after construction (measured: silently not
enforced — Node derives connection deadlines from constructor options);
routing uploads around Next straight to Go (the browser talks only to
Next.js — architecture, not preference).
Would change: Next gaining a supported request-timeout setting — then the
preload is deleted and the setting used; headersTimeout (60 s) deliberately
untouched as the slow-loris guard.

## 2026-08-12 — Stamp-out: slot-addressed scripts, private per-instance state

Chosen: `scripts/pilot/` — compose.instance.yaml (env-parameterised
override, `!override` drops the ./data mount), new-instance.sh (slug + slot
1..3 → web port, Caddy listener with fresh bcrypt credential, funnel port;
refuses an existing slug or occupied slot), reset-instance.sh (volume-level
wipe; `--retire` also frees slot/credential/funnel), rotate-credential.sh
(in-place hash swap); per-instance state lives in
docs/private/pilot/instances/<slug>/ (gitignored), globbed into the private
Caddyfile by an `import`. All drills executed: fresh instances auto-load
177 countries; upload → 3 candidates through the authed front with zero
CLI; cross-credential 401; reset removes all three volumes leaving the
sibling untouched; rotation kills the old password on reload. Three
concurrent stacks measured ~100 MiB each.
Rejected: automatic slot allocation (three slots, one operator — a picked
slot beats hidden state); storing instance state under scripts/ or a new
top-level dir (docs/private/ is already the gitignored home with the
public-repo rule attached).
Would change: sidecar-per-tester arrival (slots stop mapping to funnel
ports); the pilot outgrowing one operator (then allocation earns
automation).

## 2026-08-13 — Backups ride launchd, encrypted with age to an operator keypair

Chosen: a LaunchAgent (daily 03:30, `launchctl kickstart` for on-demand
runs, missed nights run on wake) executing backup-all.sh — per instance:
`roadbook backup` + uploads-volume tar, bundled and age-encrypted to the
operator public key, written to `$ROADBOOK_BACKUP_DIR` (default an iCloud
Drive folder; only ciphertext leaves the machine); restore-instance.sh is
the drilled inverse (CP3: 3/3 decisions re-attached on a fresh instance
through the browser path). The secret key must also live off-machine
(password manager) — recorded in RUNBOOK §8.
Rejected: cron (`crontab -` is blocked by macOS from this context, and
launchd tests on demand and catches missed schedules — strictly better
here); passphrase encryption (needs a tty nightly); plaintext offsite
(location data through a cloud account).
Would change: moving to the rented host (systemd timer replaces the
LaunchAgent; the scripts survive).

## 2026-08-16 — Gate amendment: first friend instance stamped before the reboot drill and tester zero

Chosen: the maintainer's call at a STOP — a friend instance was needed now;
friend-1 stamped into slot 2 and the link handed over with the reboot drill
and the full tester-zero walk still outstanding, replaced for this handover
by a compressed check (maintainer opens the friend link from WhatsApp on
cellular before forwarding). Risk stated openly: an unplanned reboot has
never been proven to bring the serving posture back, and the friend may hit
a dead link until the operator notices.
Rejected: refusing the handover until both gates passed (the gates are the
maintainer's own; amending them is theirs to do), and silently skipping
them (they remain CP4 items — the drill at the next restart, the full walk
with the maintainer's own export before the phase closes).
Would change: nothing — this entry records sequencing, not design; the
drill and walk still gate PHASE CLOSE, just no longer the first link.

## 2026-08-12 — Committed hosting artifacts carry no infrastructure identity

Chosen: the repository is public, so committed templates, runbook, and README
text use placeholders only — never the pilot domain, VPS host, a tester, a
link, or a credential; the instantiated versions live in `docs/private/pilot/`
(gitignored) and on the host. Effective immediately, from this brief onward.
Rejected: committing the real Caddyfile or ledger for convenience — the same
class of mistake as committing `data/`, applied to infrastructure.
Would change: nothing foreseeable; the repository going private would relax
the pressure but not the rule.
