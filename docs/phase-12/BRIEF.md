# Phase 12 — The front gate (design brief)

Chartered 2026-08-27 at the post-phase-11 STOP, per the roadmap ("The road to
strangers", PLAN.md, resequenced 2026-08-24). Goal, restated from the charter:
a stranger can find Roadbook, understand it, and either receive one of the
capped instance slots or join a waitlist; an accepted person gets an entry
email with a private link and completes the existing loop — upload, import,
detection, life map — with no ad-hoc operator work. This phase is the finish
line for "strangers can be brought in."

Two facts have changed since the charter was written, and this brief takes
both as inputs rather than silently inheriting stale text:

- **The supported-data statement improved.** The charter's checkpoint 1 says
  "Google Timeline only today." Phase 11 made photos a second import source —
  the gate no longer turns away the measured iPhone majority, and the landing
  inherits the README's two-sources statement.
- **The host has an expiry date.** The current server is a bridge host on
  trial credits, migrate-by 2026-09-12, with the free-tier capacity hunt
  (the A1 sniper) still unresolved. This phase will straddle that date.

## 0. The standing constraint: build now, open later

The gate's product work — landing, waitlist, entry process — is
host-agnostic: the public links are DuckDNS names that survive re-homing, and
nothing in this phase depends on which machine answers them. The host
question resolves on its own clock (sniper success, the PAYG upgrade
activating, or the exit decision due ~2026-09-08).

**The opening condition, stated once and binding:** no stranger receives an
entry link until the hosting under it is durable — a reclaimed-host email to
a person who just entrusted us their location history is the exact failure
this phase exists to prevent. Checkpoints 1–2 proceed regardless; checkpoint
3's *drill* runs on a scratch slot regardless; checkpoint 4's real entry
waits for the condition. If the host must be rebuilt or replaced mid-phase,
that is ops work under the phase-10 runbook, not new scope here.

A rider that lands with this phase either way: the host still runs
pre-phase-11 images. The rebuild to current (migration 00010, the photos
door, bulk triage) is the first host action of this phase — checkpoint 4's
proof needs it, and allen-zero's deferred decisions restore rides it.

## 1. Concepts this phase introduces

### 1.1 A public, unauthenticated surface — the first one

Everything Roadbook serves today sits behind something: loopback for
self-hosters, per-instance basic auth for the pilot. The landing page is the
first thing strangers reach with no credential, which makes its *shape* the
security decision. A static page — files served by Caddy, no runtime, no
database, no write path — has almost no attack surface: there is nothing to
inject into, no state to corrupt, no endpoint to abuse. Every dynamic
alternative (a public app instance, a form backend) imports a threat model
this project has never had to hold. That is why the recommendation in §4A is
static-first, and why the waitlist mechanism decision (§4B) is really a
decision about whether the public surface acquires its first write endpoint.

### 1.2 Contact data — the first stranger PII

A waitlist is email addresses of people who are not friends. PRODUCT.md's
amendment sentence governs: held minimally, stated plainly to the person,
deletable on request. Concretely that means: the page says exactly what the
address will be used for (an entry email, nothing else), where it lives, and
how to get it removed; and the storage answer must make "deletable" true —
one place, not copies scattered across a provider's logs. This is a smaller
version of the custody question the product already answers for location
data, and it should be answered with the same posture: the least data, in
the fewest places, stated honestly.

### 1.3 The cap as process, not code

Ten instances is a number an operator can hold in their head, and the
LEDGER already holds the slot roster. Enforcing the cap in code (a counter,
a gate service) would be machinery whose only job is to say "no" — and at
this scale, saying no is a human sending a "you're on the list" reply. The
charter leans this way explicitly ("by process, not code, at this scale,
unless the brief argues otherwise"); this brief does not argue otherwise.
What *is* systematised is the entry: one command from accepted-person to
stamped instance (the phase-10 scripts already are that command), plus a
written entry-email template so every entrant gets the same honest handover.

### 1.4 The public name, and what a certificate reveals

The front is `*.roadbook-host.duckdns.org` with a wildcard certificate —
chosen in phase 10 precisely so Certificate Transparency logs see the
wildcard, not per-person subdomains. Two consequences for this phase. First,
a wildcard does **not** cover the apex name itself: the landing needs its
own hostname decision (a subdomain like `www.` or `hello.` rides the
existing cert; the bare apex needs a certificate addition). Second, the
landing's URL is the product's public face, and a DuckDNS name reads as what
it is — a free dynamic-DNS subdomain. §4C treats that as a real choice
rather than pretending it is neutral.

### 1.5 What "open source" claims when a stranger checks

The landing will say self-hostable and open source, and strangers in this
audience click through to the repository. The repository is public with no
LICENSE file — which, under default copyright, means it is not legally open
source at all (all rights reserved). This was already the carried backlog
item from phase 10; the landing turns it from a gap into a contradiction.
The licence decision is the maintainer's, made at this gate or in CP1; the
recorded analysis (private launch notes, §3.1) recommends AGPL-3.0 — it
encodes "hosted is a superset, never a fork" in law — with MIT rejected for
permitting exactly the closed hosted fork the charter says is unwanted.

## 2. What gets built

1. **The landing** — a static pitch surface at a public hostname: what
   Roadbook is (the honesty rule as the differentiator), screenshots from
   the committed demo record (`docs/screens/`), the two-sources supported-
   data statement with the never-had-Timeline honesty kept intact, the data
   posture in plain words (your own instance, isolated, credentialed,
   backed up, deletable), and the way in: slots and the waitlist. Built in
   the Atlas design language; lives in the repository outside the product
   app (PRODUCT.md: hosted-operator machinery stays out of product code);
   deployed by a small operator script; served by the existing Caddy front.
2. **The waitlist** — the mechanism §4B decides, its copy stating purpose,
   storage, and deletion; entries landing somewhere durable the operator
   actually checks.
3. **Cap and entry** — the entry-email template (link + credential +
   what-to-expect + the retention/deletion facts); the LEDGER slot roster as
   the cap; `new-instance.sh` as the one command; the handover/rotation
   rules from phase 8 applying unchanged. The host rebuilt to phase-11
   images as the first host action (rider, §0).
4. **The proof and the close** — one real person from landing to life map
   with zero ad-hoc operator work (subject to §0's opening condition);
   README/docs updated; LOG.md.

Go and the web app are expected untouched: a zero product-code diff is the
target, same as phase 10. Anything that seems to need product code (a
read-only demo mode, a waitlist endpoint in the API) is out of scope by that
fact alone and returns to a future brief.

## 3. What this phase does not decide

Accounts, sign-in (Google or otherwise), tenancy, self-serve signup — all
phase 13, gated on the cap filling with real demand beyond it. Share links —
staged at 13 with access control. The public self-host launch (the Show-HN /
r/selfhosted post) — a separate deliberate act with its own private
checklist; this phase builds several things that launch needs (landing,
LICENSE, screenshots already committed), but the post itself is not a
checkpoint here. A public hosted demo — argued against in the launch notes
(an authless upload endpoint on the open internet is a custody problem);
the landing's screenshots and the one-command local demo are the substitute,
and the landing should say why, because in this audience stated judgement
reads better than a missing feature.

## 4. The real choices

### A. The landing's shape

- **(1) Static page served by Caddy — recommended.** No runtime, no state,
  no new attack surface (§1.1); survives host moves as files; zero product
  code, honouring the PRODUCT.md boundary. Cost: it must be kept honest by
  hand as the product evolves (the README discipline already practiced).
- **(2) A credential-free demo instance as the landing.** Maximum "see the
  real thing" — and rejected: an authless app with a public upload endpoint
  is the custody problem the launch notes already argued against, and
  read-only-mode product code to fix that is out of scope by §2's rule.
- **(3) Evolving `/welcome` into a dual public/instance page.** Rejected:
  `/welcome` is an instance's front door behind that instance's credential;
  making one page serve a stranger with no instance and a user with one
  conflates two audiences the phase-9 shell seam deliberately separated.

### B. The waitlist mechanism

- **(1) A stated email address (mailto plus visible address) — recommended.**
  A dedicated alias, stated on the page with its purpose ("mail this address;
  you'll get one entry email when a slot opens; ask and the mail is
  deleted"). Zero machinery, zero new public write surface; entries live in
  one inbox the operator already checks; deletion is deleting an email.
  Cost: exposed address (spam is an alias-rotation away from solved) and a
  human sending replies — which §1.3 already accepts as the cap's nature.
- **(2) A form posting to a small endpoint on the host.** Honest storage
  under our control, nicer on phones — but it is the public surface's first
  write endpoint: spam handling, abuse handling, storage, backup of a new
  PII file, all for ten entries. Machinery disproportionate to the scale;
  becomes right at phase-13 scale, where it belongs.
- **(3) A third-party form service.** Rejected outright: it hands stranger
  PII to a party the page's own privacy sentence would then have to
  disclose, contradicting "held minimally, in the fewest places."

### C. The public name

- **(1) Stay on DuckDNS for this phase — recommended.** The zero-cash
  stance stands until money returns; links survive a later re-point, so
  nothing done now is wasted. The landing rides the existing wildcard as a
  subdomain (e.g. `www.roadbook-host.duckdns.org`), avoiding the apex-cert
  addition entirely. Cost: the URL looks like what it is; for a
  ten-person waitlist reached mostly by shared link, that cost is small.
- **(2) Buy the domain now.** The naming decision the roadmap notes call
  real (the name *is* the product's public face), ~€10–15/year — blocked by
  the zero-cash constraint today, and the moment it lifts this becomes the
  first purchase. The brief records it as the standing upgrade, decided
  deliberately (name chosen, not defaulted) when it fires.

### D. Decisions the brief makes without options (each reversible at review)

- **Cap: ten.** Allen's number, confirmed against the phase-10 measurement
  (ten stacks ≈ 2 GB on the 12 GB shape).
- **Entry email: operator-sent, from a template.** No sending machinery at
  ten users; the template is committed so the handover is consistent and
  reviewable.
- **Waitlist verification: none.** A mistyped or fake address simply never
  receives entry; self-correcting at this scale.
- **LICENSE: decided at this gate** (maintainer's call; AGPL-3.0
  recommended, §1.5). Ships in CP1 alongside the landing that makes the
  claim.

## 5. Verification plan

- **The cold-visitor test (CP1).** Someone who knows nothing about the
  project reads the landing and can answer: what is this, what data does it
  need, where does my data live, how do I get in. No operator contact
  required to reach "I want this" or "not for me."
- **The waitlist round trip (CP2).** A test entry from a phone lands where
  the operator reads; the deletion promise is exercised once, literally.
- **The entry drill (CP3).** A scratch slot: waitlist entry → one command →
  stamped instance → entry email (to the operator's own address) → the link
  and credential work from a phone on cellular. Repeatable; recorded in the
  runbook.
- **The proof (CP4).** One real person, zero ad-hoc operator work, landing
  through life map — subject to §0's opening condition. Their instance
  enters the LEDGER like any pilot instance.
- Throughout: zero Go/web diff (`git diff --stat` on those trees at close),
  README honest, no real name/credential/PII in anything committed —
  waitlist details and the roster stay in `docs/private/pilot/`.

## 6. Checkpoints

1. **The landing.** Static page built (Atlas language, frontend-design skill
   loaded before designing), LICENSE landed, served by Caddy at the chosen
   hostname, cold-visitor test passed. *Visible: a stranger's phone on
   cellular renders the pitch at a public URL.*
2. **The waitlist.** Mechanism live per §4B, copy states purpose/storage/
   deletion, round trip proven. *Visible: an entry submitted from a phone
   sits where the operator reads it.*
3. **Cap and entry.** Host rebuilt to phase-11 images (allen-zero restore
   rides it); entry-email template committed to the private pilot docs;
   entry drill green on a scratch slot. *Visible: one command turns an
   address into a welcomed instance.*
4. **The proof and the close.** Opening condition met (§0); one real person
   walks the whole path; README/docs updated; LOG.md written. *A phase is
   not complete until its log exists.*

## 7. Review questions for Gate 1

1. §4A landing shape — static page as recommended?
2. §4B waitlist — the stated-address mechanism, and which alias?
3. §4C — DuckDNS for this phase, domain purchase as the recorded upgrade?
4. §4D LICENSE — AGPL-3.0, MIT, or defer (defer keeps the landing's
   open-source claim legally false; recorded as the cost)?
5. §0 opening condition — agreed that no real entry happens until the host
   is durable, even if that pushes CP4 past the migrate-by date?
6. Anything the landing must promise or must not promise that this brief
   missed?
