# Phase 8 log — friend pilot hosting

## What the phase does

Phase 7 built everything after the click; this phase built the click. A
person with a link and a credential now reaches their own Roadbook
instance from anywhere: Tailscale Funnel terminates public TLS on the
operator's machine, a local Caddy enforces per-tester basic auth on one
loopback listener per instance, and each tester gets a fully isolated
compose stack — own Postgres, own volumes, no path to the operator's
`data/`. Three funnel ports bound the pilot at three concurrent testers.

Deliverables: the Caddyfile template and four operator scripts
(`scripts/pilot/` — stamp, reset/retire, rotate, backup/restore), nightly
age-encrypted backups via a LaunchAgent with only ciphertext leaving the
machine, the operator runbook (`RUNBOOK.md`, sections 1–9), a README
hosting section, and three small product riders: `countries -if-empty`
folded into compose startup (the last operator CLI step retired),
`restart: unless-stopped` on the base compose services, and a
request-timeout preload for the standalone web server. The brief was
amended at Gate 1 from the drafted rented-host plan to the maintainer's
zero-cash laptop arrangement; the VPS remains the recorded upgrade path
behind named triggers.

The phase closed with the pilot already operating: the first friend's
instance was stamped and handed over mid-checkpoint (a logged gate
amendment), and the tester-zero walk ran the full v1 loop — WhatsApp
link, cellular, credential prompt, front door, on-phone export, 52 MB
upload through the relay, detection, confirmations, life map — with the
operator's own eight-year archive and zero CLI.

## What broke, and why each fix took its form

- **Caddy site addresses are host matchers.** `http://127.0.0.1:8100`
  matched only requests whose Host header was `127.0.0.1`; funnel traffic
  (Host: `*.ts.net`) fell through to Caddy's empty default response —
  auth "worked" locally while the public path served nothing. The fix is
  a pair: a port-only address (match any Host) plus `bind 127.0.0.1`
  (loopback socket only) — because a port-only address alone binds all
  interfaces, which is the stale-dev-server lesson from the phase 7 close
  wearing a new coat. The template carries the explanation.
- **Testing the funnel from the serving machine tests nothing.** MagicDNS
  resolves the machine's own funnel hostname locally, so the relay is
  never touched; the first "85 MB/s through the funnel" measurement was a
  loopback. Real measurements force resolution to the public ingress
  (`curl --resolve`) or use a phone on cellular. Measured honestly:
  ~220 KB/s sustained through the relay, ~1.7 s per page browse.
- **Node's request timeout made a ~65 MB upload cliff.** The standalone
  Next server inherits Node's 300 s `requestTimeout`, and Next exposes no
  knob. Setting `server.requestTimeout` after construction is silently
  ignored (measured — deadlines derive from constructor options), so the
  fix is a `node --require` preload that injects the option into
  `http.createServer` (`web/server-timeout.cjs`, default 1 h,
  `ROADBOOK_REQUEST_TIMEOUT_MS`). Proven in both directions (a 40 s cap
  408s; a deliberate 6.5-minute upload completes), then vindicated in the
  field: tester zero's real upload took **295 s — five seconds under the
  old cliff**.
- **Phone keyboards corrupt basic-auth usernames.** Android appended a
  trailing space (operator's walk), iOS auto-capitalized (the first
  friend's) — each producing a silent re-prompt loop indistinguishable
  from a wrong password. Diagnosed from Caddy access logs (auth header
  present + 401 = mismatch; briefly enabling credential logging showed
  `"demo "` verbatim). The fix reframes the username as an address, not a
  secret: the stamp script writes four accepted spellings (exact,
  capitalized, trailing-space variants) against the same hash, and the
  handover message says copy-don't-type for the password, which remains
  the only real secret.
- **`crontab` is blocked on this macOS.** The backup schedule moved to a
  LaunchAgent — strictly better here: `launchctl kickstart` tests the
  job on demand in its real execution context, and launchd runs a missed
  night on wake, which cron silently skips.
- **The age key was nearly generated into a committable path.** A
  working-directory drift put `web/docs/private/pilot/keys/` on disk —
  not covered by the root `.gitignore` (mid-slash patterns anchor; the
  phase 2 lesson). Caught by `git status` before staging; the key moved,
  the stray tree deleted. Third occurrence of the cwd-relative-write
  hazard class in project history; the dry-run rule remains the net.
- **Updating a stack mid-upload kills the stream.** A container rebuild
  during a background test 500'd the client and left an orphaned
  `upload-*.tmp` (no row — the all-or-nothing rule held). Runbook §5 now
  says never update mid-upload; the orphan-sweep backlog item has its
  first evidence.
- **A locked phone kills an in-flight upload.** Tester zero's first
  attempt died at 36% when the screen locked; the instance recorded
  nothing (0 rows, 0 files — the phase 7 failure design passing its first
  field test) and the immediate retry correctly hit the one-import guard
  (409) while cleanup finished. This fired the recorded trigger for the
  **resumable uploads** backlog item; the immediate mitigation (a
  keep-the-screen-on line in the upload copy) is held for a follow-up
  commit alongside the iPhone stamp.
- **The reboot drill found its failures without rebooting.** Docker
  Desktop autostart was off (guaranteed dead links after any restart) and
  the wake posture was a transient `caffeinate` someone else had started.
  Fixed: the checkbox, plus a KeepAlive LaunchAgent owning
  `caffeinate -ims`. The end-to-end drill itself is deferred to the next
  natural restart (maintainer's call, logged) — the ingredients are all
  in place; the proof rides the next reboot, planned or not.

## Verification at close

- Cold-cache `make test` green with the DB-backed suites running; both
  goldens byte-identical; demo regression 3/1/0; web vitest 53/53.
- CP2 drills: stamped instances came up browser-complete (177 countries,
  9 migrations, zero CLI); demo upload through the authed front produced
  the expected 3 candidates; cross-credential requests 401; volume-level
  reset removed all three volumes leaving the sibling untouched; rotation
  killed the old password on reload. Three concurrent stacks ≈ 100 MiB
  each.
- CP3 drill: a demo backup archive restored into a fresh instance left 3
  orphaned decisions that re-attached 3/3 after a browser upload and
  re-detection, with the retained upload restored alongside. Nightly
  backups have run unattended since (age ciphertext in the offsite
  directory).
- CP4, tester zero: the operator walked link → credential → front door →
  on-phone export → upload (52 MB in 295 s through the relay) → detect →
  66 candidates — **the reference detector's exact count for the
  eight-year archive, reproduced on a pilot instance from a phone upload**
  — → 3 confirmed → life map. Laptop untouched throughout.
- Live at close: demo on slot 1; the first friend's instance on slot 2
  (login-loop fix verified server-side, awaiting her return); tester
  zero's instance on slot 3 holding real data under the pilot's own
  custody terms.

## Carried forward

- **Reboot drill** — the next restart is the drill; result gets logged
  either way. Until it passes, recovery is designed but unproven.
- **Resumable uploads** — backlog trigger has fired (a real pilot upload
  killed by a locking phone); needs its own design pass.
- **Follow-up commit pending the iPhone friend's report:** the `/welcome`
  iPhone walkthrough stamp (or fix), plus the keep-the-screen-on line in
  the upload expectation copy.
- **VPS upgrade path** — §3A triggers standing: a friend hits a down
  instance, an availability-caused upload failure, a fourth concurrent
  tester, or willingness to spend.
- **Sidecar-per-tester funnel extension** — specified in BRIEF §1.5,
  built only when the three-slot ceiling binds.
- **Slot-1 handover question** — whether the demo leaves the funnel so a
  friend gets the portless link; decided at the next handover.
- **Charter amendment** — still pending the maintainer's re-read of the
  parked community doc; unchanged by this phase.
