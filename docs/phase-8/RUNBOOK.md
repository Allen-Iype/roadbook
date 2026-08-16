# Pilot hosting runbook

Operating the friend pilot: per-tester Roadbook instances on the operator's
machine, fronted by Tailscale Funnel and a local Caddy enforcing per-tester
basic auth (design: BRIEF.md; running decisions: DECISIONS.md). Generic by
design — real hostnames, testers, links, and credentials live only in the
private ledger (`docs/private/pilot/LEDGER.md`, gitignored) and never in
this file (the public-repo rule, BRIEF §2).


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

**Reboot drill** (run at every restart until first passed, then after any
posture change): restart the machine, log in, touch nothing, then confirm
from a phone on cellular that each active link answers with its credential
prompt and serves after login. This proves the whole chain — Docker
autostart, container restart policy, Caddy service, the caffeinate agent,
funnel persistence — with no human replaying commands. Status: the
ingredients are all in place, but the end-to-end drill has not yet run
(deferred 2026-08-16, DECISIONS); treat the next restart, planned or not,
as the drill and record its result.

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

## 6 Handover

Moving a slot from one tester to the next, in order — the order is the
point (data is wiped only after its owner has been told):

1. **Tell the outgoing tester first.** Remind them to keep their own
   export file — the instance's retained copy may be their only one — and
   offer their backup archive (it is their data; `restore-instance.sh`
   into a fresh stamp can resurrect it later if they return).
2. `scripts/pilot/reset-instance.sh <slug> --retire` — volumes wiped,
   Caddy block gone, funnel listener off, slot free. The old
   link+credential pair is dead.
3. Update the ledger: outgoing holder closed out with the date.
4. Stamp fresh for the next person: `new-instance.sh <new-slug> <slot>`.
   Never reuse a slug or a credential — old chats keep old messages
   forever, and a reused credential would let the previous holder into
   the new holder's data.
5. Ledger again (new holder, consent date), then the section-3 message.

Sequential testers on one slot are the normal case; three slots only cap
*concurrent* testers.

## 7 Incident

- **Suspected credential leak** (forwarded message, shoulder-surfed,
  device lost): `scripts/pilot/rotate-credential.sh <slug>` immediately —
  the old password dies on Caddy reload — then resend the new pair over
  the private channel. Rotation is cheap; rotate on suspicion, not proof.
- **Login loop reported by a tester** ("it keeps asking again"): check
  the listener's access log first. `auth:none` on every attempt means the
  browser never sent credentials; `auth:sent` + 401 means a string
  mismatch — keyboard artifacts are the measured cause (Android appends a
  space, iOS capitalizes; the stamp script's username variants absorb
  those). If it persists, rotate to a fresh password and resend with
  copy-don't-type emphasis.
- **Machine loss or compromise:** assume every instance's data exposed.
  Tell every tester plainly what was on the machine and what it means;
  offsite archives (age-encrypted) are unreadable without the key, so
  state that too. Rebuild, restore per instance, rotate everything.
- **Funnel or Tailscale outage:** nothing to fix locally — links answer
  again when the service does; send an honest "not on your side" message
  to anyone mid-onboarding.
- **A tester asks for deletion:** that is section 6 steps 1–3 without a
  successor, plus deleting their offsite archives; confirm to them when
  done. On an authless pilot the operator IS the deletion mechanism.

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

## 9 Decommission

End of pilot, in order: tell every tester and offer each their final
backup archive; `reset-instance.sh <slug> --retire` per instance; demo
off the funnel (`tailscale funnel --https=443 off`) or the whole funnel
down; `launchctl bootout gui/$(id -u)/com.roadbook.pilot.backup` to stop
the nightly job; decide the fate of offsite archives per tester (keep by
request, else delete); close the ledger with dates. The repo, scripts,
and runbook remain — the pilot can restart any time with one stamp.
