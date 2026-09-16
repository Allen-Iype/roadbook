# Phase 12 — decision log

Three lines each: what was chosen, what was rejected, what would change our mind.
Written as decisions are made, not reconstructed.

## 2026-08-27 — Charter: phase 12 front gate confirmed at the post-phase-11 STOP

- **Chosen:** the front gate as chartered in PLAN.md, with the host cliff
  (bridge-host migrate-by 2026-09-12, A1 unresolved) taken as a named opening
  condition — build the gate now, admit no stranger until hosting is durable —
  rather than as a blocker on starting. Sign-in and shared multi-user were
  asked about at the STOP and confirmed as phase-13 scope, not this phase.
- **Rejected:** resolving the hosting question as its own phase first (the
  gate's product work is host-agnostic and the links survive re-homing; serial
  waiting buys nothing); pulling accounts/tenancy forward (demand unproven,
  the largest risk surface in the codebase, groundwork already banked).
- **Would change our mind:** the host being reclaimed with no successor before
  CP3 — then ops-first becomes forced, per the brief's §0.

## 2026-08-27 — Gate 1 PASSED: brief accepted, all recommendations taken

- **Chosen:** §4A static Caddy-served landing; §4B stated email alias (the
  specific address is the maintainer's to pick at CP2 — publishing an email
  is his act); §4C DuckDNS this phase, domain purchase recorded as the first
  zero-cash-lift upgrade; §4D AGPL-3.0, landing in CP1; §0 opening condition
  agreed (no real entry until hosting is durable, even past the migrate-by);
  cap ten, operator-sent entry email, no waitlist verification.
- **Rejected:** everything each §4 option list rejects, as written; deferring
  the licence (the landing's open-source claim would be legally false).
- **Would change our mind:** per-choice triggers as recorded in the brief.

## 2026-08-27 — CP1: landing shape details

- **Chosen:** one static file, zero JavaScript (`landing/index.html`) — the
  no-scripts fact is itself stated on the page as part of the custody story;
  hero = the app's own /welcome pitch plate SVG converted to plain HTML
  (geometry shared, inks can't drift); Atlas tokens copied verbatim with a
  same-commit sync rule recorded in landing/README.md; fonts and demo
  screenshots staged at deploy from web/app/fonts and docs/screens, never
  duplicated in git; hostname www.$ROADBOOK_DOMAIN (rides the wildcard cert
  — the bare apex would need its own certificate entry); the Caddyfile
  gains one operator-added import line, the deploy script refuses to edit
  /etc/caddy itself; single-rule frame on the way-in box (the double rule
  stays reserved for drawn plates).
- **Rejected:** a build step or shared-token pipeline for one page
  (machinery for a file that changes a few times a year); committing font/
  screenshot copies (duplication with drift risk); apex hostname (cert
  addition, CT-visible, for no reader benefit).
- **Would change our mind:** the landing growing real interactivity — which
  §2's zero-product-code rule already routes to a future brief instead.

## 2026-08-27 — CP2: Gate-1 §4B REVERSED — the waitlist is a form, not a stated address

- **Chosen (maintainer's challenge, assistant agreed on the evidence):** a
  plain HTML form on the landing (zero-JS preserved — no scripts needed for
  a POST), handled by a tiny standalone intake service on the host
  (stdlib-only, loopback, systemd, reverse-proxied by Caddy for exactly one
  route), appending timestamp+email to one 0600 file under
  /srv/pilot/waitlist/. Spam: CSS-hidden honeypot + in-handler per-IP rate
  limit (Caddy's rate-limit plugin is not in our build). IPs are used for
  the limit in memory and never stored — held minimally. Copy is an
  invitation ("ten slots, join the list"), never a date promise.
- **Rejected:** the Gate-1 stated-address choice — two things the gate
  underweighted: mobile is the primary funnel and mailto: on phones is
  unreliable, high-friction (nobody composes "I would like to join"); and a
  published Gmail address is both worse custody optics and an impression
  cost on a trust-sensitive page. Also still rejected: third-party form
  services (PII to another party), and putting the handler in product code
  (PRODUCT.md boundary — it is operator machinery).
- **Would change our mind:** abuse that outgrows honeypot+rate-limit (then
  the endpoint earns real defenses or retires back to an address); the
  entries file is deliberately NOT in nightly backups so the deletion
  promise stays absolute — if losing pending entries to a host failure ever
  stings, that trade-off gets revisited openly.

## 2026-09-09 — hosting torn down; phase 12 closes early; product-first re-charter

- **Chosen (maintainer's call at a STOP):** deliberate teardown of the
  Oracle bridge host three days ahead of trial reclaim (migrate-by
  2026-09-12), after a final verified backup of every instance (all
  archives decrypt on the laptop; decisions, photos, and retained uploads
  all present). The A1 Always Free escape never materialised — three weeks
  of 15-minute attempts, zero capacity in the single-AD home region, PAYG
  upgrade button never activated. New sequence: build the remaining
  product (per-user accounts, share links, the planned backlog) with the
  laptop as the only runtime, then return to hosting a finished product
  on a platform chosen when money returns. Phase 12 closes with CP1–CP3
  delivered and proven (landing, waitlist, entry drill) and CP4 (a real
  stranger enters) explicitly not reached — its §0 opening condition
  (durable hosting) was never satisfiable on trial credits.
- **Rejected:** paying to keep the host (zero-cash constraint stands);
  laptop re-hosting of the pilot (reviving the funnel's known problems to
  serve six mostly-idle instances is upkeep without product progress);
  letting the trial reclaim the host with services still nominally up
  (an unmanaged end to a custodial promise).
- **Would change our mind:** nothing to reverse — the landing, waitlist,
  and entry machinery are all committed and redeploy onto any future host
  in minutes (deploy-landing.sh, setup-waitlist.sh, new-instance.sh);
  DuckDNS re-points; testers' data restores by durable identity. The
  accounts/share-links work pulled forward from phase 13 replaces its
  demand gate ("cap filled + waitlist") with the maintainer's
  product-first judgement — recorded openly, decided at this STOP.
