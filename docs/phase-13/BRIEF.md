# Phase 13 — Accounts, tenancy, and share links: design brief

Status: DRAFT for Gate 1 review.

Charter position: PLAN.md staged this phase behind a demand gate (cap filled,
waitlist pressure). The 2026-09-09 re-charter (docs/phase-12/DECISIONS.md)
replaced that gate with the maintainer's product-first judgement: build the
complete product on the laptop, host a finished thing later. This brief is
that phase's front gate. The dual-mode charter amendment (PRODUCT.md
"Hostable as a service") binds everything here: **self-host is the reference
deployment; every feature must work single-user and authless on a plain
`docker compose up` before anything hosted builds on it.**

## 1. Concepts this phase introduces

**Authentication vs authorization.** Authentication answers "who is making
this request"; authorization answers "what may they touch." Today Roadbook
has neither: one instance is one person, and isolation is structural (a
stack per tester). This phase adds both — identity (who) and row scoping
(whose rows) — inside the one place allowed to hold business logic: the Go
API. The frontend learns nothing new about data; it learns only "signed in
as X" and "401 means show the sign-in page."

**Delegated authentication (OIDC).** We never store or verify passwords —
PLAN.md's charter line ("never hand-rolled") exists because password auth is
a liability engine: storage, hashing, resets, breach disclosure, all for
zero product value. OpenID Connect is the standard that outsources it: the
user proves their identity to a provider they already trust (Google, or a
self-hoster's own Keycloak/Authelia — any OIDC issuer), and the provider
hands our server a signed statement of who they are. The flow, concretely:
browser is redirected to the provider with our client id → user consents →
provider redirects back to our callback with a one-time code → the Go server
exchanges the code (server-to-server, with the client secret) for an ID
token → the token's verified claims (`sub`, a stable opaque subject id, plus
email for display) become the user row. Go implementation:
`github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` — the standard pair,
small, audited, no framework.

**Sessions.** After the OIDC dance the browser needs a way to stay signed
in. Two schools: stateless JWTs (the server signs a token and trusts it
until expiry) and server-side sessions (an opaque random token that is a
key into a sessions table). We take sessions-in-Postgres: revocation is
`DELETE`, sign-out is real, there is no signing-key lifecycle, and at our
scale a session lookup joins a query that was already happening. The token
rides an HTTP-only, `Secure`, `SameSite=Lax` cookie — HTTP-only means page
JavaScript can never read it, which kills the XSS-steals-token class.

**Where auth lives, given our topology.** The browser talks only to
Next.js; only Go reaches Postgres. So: Go owns the whole identity dance
(OIDC endpoints, session issue/verify, user rows) because identity IS
business logic; Next relays. The auth endpoints are proxied through Next
route handlers exactly like photo thumbnails are today; `Set-Cookie` passes
through, so the cookie lives on the site origin. Server components and
server actions forward the incoming session cookie to Go on every call (a
small addition to the generated-client wrapper). Next renders auth *UX*
(sign-in page, redirect on 401) but makes no trust decisions — a request
Go rejects is rejected no matter what the frontend thinks. The phase-9
shell seam was built for this: the auth gate drops onto the (app) route
group; the (public) group and the reserved header slot take the sign-in
page and the account control without moving any page.

**Row-scoped tenancy.** Multi-user in one database means every user-data
row carries its owner and every query filters by it. Phase 4's store test
harness was built explicitly "for the day every query gains a user filter"
— that day is this phase. The failure class this creates (a missed filter
shows user A user B's rows) is why the harness grows a dedicated
cross-tenant test family: for every store read path, prove user B sees
none of user A's data.

**Capability URLs (share links).** A share link is an unguessable token
that IS the permission — like a doc link set to "anyone with the link."
No account needed to view; revocation deletes the token; the URL carries
128 random bits so guessing is not a path. It is the "shared view"
consumer the architecture rationale anticipated from day one, and it works
identically for a self-hoster sharing with family — which is what lets it
pass the single-user-authless test.

**Data lifecycle.** Hosting strangers' location data makes deletion a
product feature, not an operator favour. Self-serve deletion (the backlog
entry) lands here: a signed-in user can delete everything of theirs,
structurally, without the operator.

## 2. What gets built

- A `users` table and a `user_id` owner column on every user-data table;
  the store sweep that scopes every query; the harness's cross-tenant test
  family. The single-user default `'self'` (already on `decisions` since
  00001) extends everywhere, so an authless instance is user `'self'`
  throughout and behaves byte-identically to today.
- Auth in Go behind a mode switch: `ROADBOOK_AUTH=off` (default — the
  self-host reference, no sign-in surfaces anywhere) or `oidc` (issuer URL,
  client id/secret via env). OIDC endpoints + sessions table + 401
  semantics in openapi.yaml; generated on both sides as ever.
- Web integration on the banked seam: sign-in page in the (public) shell,
  auth gate on the (app) shell, the reserved header slot becomes the
  account control (email + sign out) in multi mode and stays the instance
  label otherwise.
- Read-only share links per confirmed adventure: create/revoke from the
  adventure page, public `/shared/[token]` view (map + narrative, Atlas
  plate, noindex), living in the (public) shell.
- Self-serve deletion of a user's own data, both modes.

**Not in this phase:** hosting anything (laptop-only; the landing stays
dark until the hosting phase returns); the waitlist (already built, off
with the host); sign-up beyond OIDC's own consent (no invite system yet —
that is the future hosting phase's gate to design); life-map sharing
(bigger privacy surface, deferred deliberately); any bundled backlog
feature — recalled adventures, GPX, poster — each has its own trigger and
brief. Scope discipline is what makes this phase finishable.

## 3. The real choices

### A. Tenancy form: row scoping vs automated instance-per-user

PLAN.md records the leaning (true tenancy) and instructs that the
alternative be argued against, not skipped. Argued:

- **(2) Automated instance-per-user.** Its virtue is real: isolation stays
  structural, no cross-user bug class exists, zero query changes. But at
  signup time, "create a user" becomes "orchestrate a compose stack" —
  provisioning software as the product's core path, the most operational
  code we would ever ship. Every idle user costs ~100 MB and a migration
  at every upgrade ×N; backups are ×N; share links need per-instance
  public hostnames; and the pilot already showed the model's ceiling —
  it is exactly what we just tore down. It scales operators, not products.
- **(1) Row-scoped tenancy in one instance — recommended.** One schema,
  one migration path, one backup; signup is an INSERT; share links are
  rows. The cost is honest: the missed-filter bug class now exists, and
  it is bought down with the harness the project spent phase 4 building.
  Scoping rule: user-data tables (imports, raw_positions, visits,
  activities, path_points, detection_runs, candidates, decisions, photos,
  photo_records) gain `user_id`; genuinely global tables stay global —
  `countries` (public reference data) and `route_cache` (derived road
  geometry keyed by rounded coordinates; an internal cache never exposed
  per-user, shared so one user's routing warms another's — stated here so
  the choice is deliberate, revisited if cache contents ever surface in
  any API response).

### B. Identity: which provider story

- **(1) OIDC with a configurable issuer, Google first — recommended.**
  One code path against the standard; Google is the first configured
  issuer because every pilot person has a Google account (their data
  export came from Google, after all). A self-hoster who wants multi-user
  points the same three env vars at their own issuer. Costs: multi mode
  requires a registered OAuth client (free, but a console step), and
  sign-in requires the provider to be reachable.
- **(2) Hand-rolled email+password.** Rejected by charter, and rightly:
  all liability, no value.
- **(3) Magic-link email.** No password storage, but it makes an SMTP
  sender a runtime dependency of sign-in and puts deliverability (spam
  folders) on the critical path — worse than (1) on every axis we care
  about, given every target user demonstrably has an OIDC identity.

### C. Share links: what is shared, and where the feature lives

- **(1) Per-adventure links as a checkpoint of this phase — recommended.**
  One confirmed adventure per token; view shows the plate: map, legs with
  the honesty channel intact (invariants 5 and 8 apply to strangers most
  of all), day narrative, countries, photos included — the sharer chose to
  share this adventure, and a shared plate with holes would misrepresent
  the product; the create dialog states photos are included, and revoke
  is one tap. `X-Robots-Tag: noindex`. Rationale for in-phase: sharing is
  one of the two features the maintainer named, and it needs exactly the
  machinery this phase builds (a public read path with authorization
  semantics); deferring it would leave the phase's hardest new surface
  untested by its first real consumer.
- **(2) A separate later phase.** Cleaner scope, but the share view would
  then be the *next* phase's first public read path — the risk moves, it
  does not shrink.
- **(3) Life-map sharing now too.** Rejected for now: a whole-life map is
  a categorically bigger disclosure than one adventure; it deserves its
  own decision when someone asks for it.

## 4. Constraints carried in

- The PRODUCT.md test, restated as this phase's acceptance bar: with
  `ROADBOOK_AUTH=off` and no new env, a fresh `docker compose up` behaves
  exactly as today — same pages, no sign-in surface, goldens and e2e
  byte-identical. Auth-off is not a degraded mode; it is the reference.
- Invariants bind unchanged; 9 (nothing user-specific hardcoded), 11
  (frontend never reaches the database), 13 (no unreproducible numbers),
  14 (no real location data in git) all have new ways to fail here.
- openapi.yaml stays the single contract; auth endpoints and 401/403
  semantics are spec first, generated both sides.
- Laptop is the only runtime. Nothing in this phase may depend on a
  public origin; OIDC redirect URLs are localhost until hosting returns.

## 5. Verification plan

- **Cross-tenant family:** for every store read and write path, a
  two-user harness test proving disjoint visibility; a new test is added
  with every new query as standing practice.
- **Auth-off regression:** full existing suite (make test, both goldens,
  demo pin, corpus pin, e2e) green with zero diffs against an auth-off
  stack — proven at every checkpoint, not once.
- **Auth-on walk:** two real Google identities on one instance; each
  uploads, detects, curates; each sees only their own everything; sign
  out kills the session server-side (revocation observable).
- **Share links:** view works signed-out; revoked token 404s; unshared
  adventure unreachable by construction; noindex header present.
- **Deletion:** a deleted user's rows are gone (count zero across every
  user-data table), the other user's untouched — harness-proven.

## 6. Checkpoints

1. **CP1 — identity schema and the scoping sweep.** Migration (users +
   owner columns, `'self'` backfill), store sweep, cross-tenant test
   family, auth-off regression green. No UI, no auth yet. The riskiest
   checkpoint, deliberately first and alone.
2. **CP2 — auth in Go.** OIDC + sessions + mode switch + openapi; proven
   with the test issuer in the harness and one live Google sign-in via
   curl/browser against localhost.
3. **CP3 — auth in the web app.** Sign-in page, (app) gate, account slot,
   cookie forwarding in the client wrapper; the two-identity walk passes;
   auth-off still renders no auth surface anywhere.
4. **CP4 — share links.** Schema + API + create/revoke UI + `/shared/
   [token]` view; e2e extended; the honesty channel visibly intact on a
   shared plate.
5. **CP5 — deletion and close.** Self-serve deletion both modes, README
   (dual-mode statement updated), LOG.md — phase not complete until the
   log exists.

## 7. What would change our mind

- If the scoping sweep turns out to touch materially more than the store
  package (leaking into the pure core), stop — the core must stay
  identity-free (a detector does not know who owns a point), and any
  pressure otherwise means the seam is drawn wrong.
- If OIDC-at-localhost fights Google's redirect rules in practice, CP2
  falls back to proving with the local test issuer alone and live-Google
  moves to the hosting phase's checklist — recorded, not silently skipped.
- If share links stall the phase, they detach into their own phase intact
  (choice C's option 2) rather than shipping half-authorized.
