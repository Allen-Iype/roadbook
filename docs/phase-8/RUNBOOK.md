# Pilot hosting runbook

Operating the friend pilot: per-tester Roadbook instances on the operator's
machine, fronted by Tailscale Funnel and a local Caddy enforcing per-tester
basic auth (design: BRIEF.md; running decisions: DECISIONS.md). Generic by
design — real hostnames, testers, links, and credentials live only in the
private ledger (`docs/private/pilot/LEDGER.md`, gitignored) and never in
this file (the public-repo rule, BRIEF §2).

Sections 6, 7 and 9 are completed at CP4; the rest is live procedure.

## 1 Serving posture

The serving host is a laptop (BRIEF §3A — decided with eyes open). The
posture checklist; all of it must hold for the pilot to be up:

- Mains power connected, lid open, `caffeinate -imsu` running.
- Docker Desktop running, with **Start at login** enabled in its settings.
- Tailscale running (menu-bar app; starts at login), funnel listeners on —
  `tailscale funnel status` lists one line per active slot.
- Caddy running as a brew service (`brew services list`); the system
  Caddyfile (`/opt/homebrew/etc/Caddyfile`) imports the private pilot
  Caddyfile, which imports each instance's block.
- Instance stacks up: `docker compose ls` shows the demo project and one
  project per active tester; all containers carry `restart:
  unless-stopped`, so a reboot revives them once Docker starts.

**Reboot drill** (must pass before the first tester link goes out, and
after any change to the posture): restart the machine, touch nothing, then
confirm from a phone on cellular that each active link answers with its
credential prompt and serves after login. This proves the whole chain —
Docker autostart, container restart policy, Caddy service, funnel
persistence — with no human replaying commands.

## 2 Stamp a new instance

```
scripts/pilot/new-instance.sh <slug> <slot>
```

Slug: short lowercase handle (never a full real name — it shows in
`docker ps`). Slot 1/2/3 maps to web port 3010/3011/3012, Caddy listener
8100/8101/8102, funnel port 443/8443/10000; slot 1's link has no visible
port. The script creates the stack (own Postgres with a random password,
own volumes, no path to the operator's `data/`), writes the Caddy block
with a fresh bcrypt credential, reloads Caddy, enables the funnel
listener, and prints link + username + password once.

Then: record the trio in the ledger with the holder's name and date, and
send the handover message (section 3). A fresh instance is
browser-complete — migrations and the countries table load at startup;
there is no CLI step.

## 3 Hand over a link

Send the link and credential over the same private channel (they are one
secret bundle — BRIEF §1.4). The message template; keep every element:

> [What Roadbook is — one sentence, the front door says the rest.]
>
> Your link: `<link>`
> Username: `<slug>` — type it exactly; **no space at the end** (phone
> keyboards add one, and it will just re-ask you to log in).
> Password: `<password>` — **copy it from this message**, don't type it.
>
> It runs on my laptop, so if it ever doesn't load, tell me and I'll fix
> it — nothing is wrong on your side. Your data stays on that machine:
> I can technically access it, encrypted backups exist, and if you want
> everything deleted I wipe it completely — just ask.
>
> The upload can take a few minutes on mobile data — Wi-Fi is faster,
> and you can leave the page open in the background.

Consent is that paragraph: storage location, operator access, deletion on
request — stated before any upload, in writing, in the chat that holds
the link.

## 4 Watch

- `docker compose ls` — every expected project running.
- `tailscale funnel status` — every expected listener on.
- Per-listener access logs: `/opt/homebrew/var/log/roadbook-pilot-*.log`
  (status codes and user agents; no credentials).
- Disk: uploads are retained per instance (content-hashed); check
  `docker system df` and the instance uploads volumes when a tester
  imports a large archive. A server killed mid-upload can leave an
  `upload-*.tmp` in an uploads volume — stale `.tmp` files are safe to
  delete.
- The backup log (section 8) — one line per instance per night; a
  "skipped (stack not running)" line for a stack that should be up is a
  finding, not noise.

## 5 Update

From the repo root, after a `git pull` or local commit:

```
docker compose build
docker compose up -d                          # demo project
```

Then for each stamped instance (the ledger lists them):

```
docker compose -p roadbook-<slug> --env-file docs/private/pilot/instances/<slug>/.env \
  -f compose.yaml -f scripts/pilot/compose.instance.yaml up -d --build
```

Then verify: each link loads through its front, and `docker compose ls`
shows everything running. Migrations run automatically in the compose
command; `countries -if-empty` never overwrites an existing load. Do not
update while a tester is mid-upload — a rebuilt api container kills the
stream and leaves a stale `.tmp` (measured at CP2).

## 6 Handover (completed at CP4)

Outline, binding since the brief: tell the outgoing tester first; remind
them to keep their own export file; `scripts/pilot/reset-instance.sh
<slug> --retire`; stamp fresh for the next person (never reuse a
credential); update the ledger both directions.

## 7 Incident (completed at CP4)

Outline: suspected credential leak → `scripts/pilot/rotate-credential.sh
<slug>` immediately, resend; machine loss/compromise → assume all instance
data exposed, tell every tester plainly; funnel/Tailscale outage → honest
message, nothing to fix locally.

## 8 Backup and restore

Nightly at 03:30 a LaunchAgent (`com.roadbook.pilot.backup`, installed in
`~/Library/LaunchAgents`) runs `scripts/pilot/backup-all.sh`: for the demo
project and every stamped instance whose stack is running, it takes
`roadbook backup` (decisions, photos, thumbnails) plus a tar of the
uploads volume, bundles them, encrypts with `age` to the operator key, and
writes `<name>-<timestamp>.tar.age` to the offsite directory
(`$ROADBOOK_BACKUP_DIR`, default an iCloud Drive folder — only ciphertext
leaves the machine). launchd runs a missed night on the next wake. Fire it
manually any time:

```
launchctl kickstart gui/$(id -u)/com.roadbook.pilot.backup
tail docs/private/pilot/backup.log
```

**The key** is `docs/private/pilot/keys/backup.key`. A copy must live off
this machine (password manager) — encrypted backups die with the laptop
otherwise, which is the exact fate they exist to survive.

**Restore** (drill this; a backup that has never been restored is a hope):

```
scripts/pilot/restore-instance.sh <slug|demo> <archive.tar.age>
```

Restored decisions are orphans until the tester's export is imported and
detection runs — then they re-attach by anchored identity (proven at CP3:
3/3 on a fresh instance, through the browser upload path). The uploads
tar restores the retained export alongside.

## 9 Decommission (completed at CP4)

Outline: testers notified; per-tester backup offered (their archive, their
data); `reset-instance.sh --retire` for each; funnel off; LaunchAgent
unloaded; ledger closed out.
