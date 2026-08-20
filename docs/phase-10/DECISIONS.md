# Phase 10 — decision log

Three lines per decision: chosen, rejected, what would change our mind.
Written as decisions are made, not reconstructed.

---

**2026-08-18 — Phase chartered.** Chosen: phase 10 = hosting readiness (the
VPS re-homing), first phase of the roadmap "The road to strangers"
(docs/PLAN.md), under the dual-mode charter amendment committed the same day.
Rejected: starting with the front gate (needs the durable host first) or the
ingestion phase (still gated on audit returns). Would change our mind: pilot
evidence of the v1 loop failing for product reasons — a fix-the-loop pass
would precede this phase's close.

**2026-08-18 — Brief drafted, Gate 1 pending.** BRIEF.md presents four real
choices: §3A provider/region (recommendation: India-region first), §3B
registrar/DNS (recommendation: Cloudflare DNS-only for DNS-01), §3C deployment
drive (recommendation: repo on host, no IaC), §3D backup destination
(recommendation: laptop-pull baseline), and §3E the cost ceiling (maintainer
sets the number — this phase explicitly retires phase 8's zero-cash stance).
No purchase or provisioning before the gate passes.

**2026-08-18 — Topology challenged at Gate 1 review; instance-per-user
reaffirmed.** Chosen: ten users = ten isolated compose stacks (own Postgres,
API, web) behind one Caddy — the pilot vehicle; shared-database multi-tenancy
is affirmed as the destination, scheduled as phase 12 shape (a) behind its
trigger (cap filled + waitlist demand). Rejected: building tenancy now — it is
the project's largest and riskiest code change (auth, a user filter on every
query where one miss leaks location history, GDPR lifecycle), and building it
before any stranger has asked to sign up buys nothing the pilot needs; the
per-stack overhead at n=10 is a few hundred MB each, linear, and priced within
§3E's ceiling. Would change our mind: credible evidence of dozens of committed
users arriving within a couple of months — then the waitlist is just a queue
for a known rewrite and phase 12 starts earlier; the maintainer raised and
accepted this reasoning at the review.

**2026-08-18 — Gate 1 PASSED; §3 decided (full record in BRIEF Gate 1
addendum).** Chosen: Oracle Always Free (India home region, pay-as-you-go
conversion) as primary host with Hetzner (~€5–9/mo, prices checked
2026-08-18) as pre-decided fallback; working-name domain deferred until the
free host proves out (CP1a/CP1b split — maintainer's sequencing); Cloudflare
DNS-only for DNS-01; laptop-pull backups; ceiling zero recurring + domain
~€10–15/year, >€10/month returns to a STOP. Rejected: paid India-region VPS
(Vultr Mumbai $20–40/mo, DO Bangalore $24–48/mo — 4–5× the EU class for the
same RAM; prices verified same-day), and the draft's blanket free-tier
dismissal (the phase's own backups + scripted re-homing reduce reclamation
to downtime, never data loss). Would change our mind: free-tier ARM capacity
unobtainable, or a real tester-facing outage attributable to the free host —
either fires the Hetzner fallback without a new gate.

**2026-08-20 — Zero-cash reversion; trial-credits bridge instead of the paid
fallback.** Chosen: the maintainer cannot spend on infrastructure right now,
so the Hetzner fallback (which the capacity failure would otherwise have
fired) is DEFERRED to when money returns; CP1a proceeds today on a paid AMD
shape (E5.Flex 2 OCPU/12 GB — mirrors the A1 target so sizing measurements
transfer) covered by the ~30-day trial credits, ₹0 out of pocket; trials do
not auto-convert — at trial end paid resources are stopped, never billed.
Bridge exit before the trial ends: an automated A1 capacity-retry (2/12,
free) plus PAYG conversion when Oracle's stuck billing setup completes;
re-homing is the proven cheap operation. Front without buying a domain:
free DuckDNS subdomain + Caddy wildcard via its DNS-01 plugin + per-tester
basic auth (real TLS, direct traffic, no relay ceiling) — bare-IP links
REJECTED (no TLS for IPs in practice: plaintext credentials + location data,
the line phase 8 refused to cross; links also rot at re-homing). Learned en
route, verified (Oracle docs + press): Always Free A1 was halved unannounced
on 2026-06-15 to 1,500 OCPU-hrs + 9,000 GB-hrs/month = 2 OCPU/12 GB
continuous; both 4/24 and 2/12 create attempts failed on Hyderabad capacity
across two days with the PAYG lever unavailable. Would change our mind:
money returning (Hetzner per the addendum) or the A1 sniper landing the free
instance (the intended exit).
