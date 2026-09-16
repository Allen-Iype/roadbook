# Phase 13 — decision log

Three lines each: chosen, rejected, what would change our mind. Written as
decisions are made, per the working agreement.

## 2026-09-16 — Gate 1: brief approved as written

- **Chosen:** all three §3 recommendations stand — row-scoped tenancy in
  one instance; OIDC with a configurable issuer, Google first; share links
  in-phase as CP4. Checkpoints and verification plan as briefed.
- **Rejected:** automated instance-per-user (signup as infrastructure
  orchestration — the torn-down pilot is the lived evidence of that
  model's ceiling); hand-rolled passwords (charter); magic-link email
  (SMTP deliverability on sign-in's critical path); life-map sharing
  (categorically bigger disclosure, own decision later).
- **Would change our mind:** the §7 tripwires — scoping pressure reaching
  the pure core stops the phase; Google-at-localhost friction demotes
  live-Google proof to the hosting phase's checklist; share links
  stalling detach into their own phase intact.

## 2026-09-16 — Tenancy scoping boundary: which tables carry an owner

- **Chosen:** `user_id` on every user-data table (imports, raw_positions,
  visits, activities, path_points, detection_runs, candidates, decisions,
  photos, photo_records); `countries` and `route_cache` stay global.
  Uniqueness that expressed "one per instance" becomes "one per user"
  (content-hash dedupe scopes to the owner; two users may hold the same
  photo or export).
- **Rejected:** scoping the route cache per user — it is derived road
  geometry keyed by rounded coordinate pairs, never exposed per-user, and
  sharing it warms routing across users.
- **Would change our mind:** any API response that would surface route
  cache contents attributable to a user — the moment that exists, the
  cache joins the scoped set.

## 2026-09-16 — CP1: the shape of the sweep

- **Chosen:** every user-data store method takes `userID` as its first
  data argument and every statement filters or sets it — uniformity over
  cleverness, so a missed filter is visible in the diff, not hidden in a
  helper. The API routes all handlers through one `currentUser(ctx)` seam
  returning `store.SelfUser`; CP2 swaps that one body for the session
  lookup and no call site moves. Backup `Write`/`Restore` take `userID`
  explicitly (the archive stays one person's data; CP5's per-user export
  reuses it). Background import goroutines capture the user at request
  time — a session must never be read from inside `context.Background()`.
- **Rejected:** scoping `SweepRunningImports` (a crash killed every
  user's goroutine — the startup sweep is global by meaning, documented
  in place); a per-user thumbnail namespace (files stay content-addressed
  and shared — identical bytes are one file regardless of owner).
- **Would change our mind:** if CP2's session plumbing can't reach the
  seam through ctx cleanly, the seam moves to the generated middleware
  layer — the uniformity rule (one place decides the user) survives
  either way.

## 2026-09-16 — CP2: auth mechanics

- **Chosen:** all four auth operations live in openapi.yaml and the strict
  generated server — including the redirect endpoints, via declared 302
  response headers (Location, Set-Cookie) — so invariant 10 holds with no
  hand-mounted HTTP anywhere. Cookies reach strict handlers through the
  one AuthMiddleware (it stashes request cookies in ctx and resolves the
  session), which is also where 401 enforcement lives; exempt operations
  are exactly healthz + the auth surface. Session TTL 30 days, table
  stores only the token's SHA-256 (a leaked row cannot be replayed), state
  and nonce ride one 10-minute HttpOnly cookie. Client secret is env-only
  (ROADBOOK_OIDC_CLIENT_SECRET) — flags land in `ps` output. x/oauth2
  pinned v0.36.0 (v0.37 requires go 1.26; the module stays 1.25.7 with
  the Dockerfile's golang:1.25 pin). Existing API e2e tests now run
  through AuthMiddleware, so the whole prior suite doubles as the
  auth-off regression.
- **Rejected:** hand-mounted auth routes beside the generated mux
  (invariant 10 says the interface is generated, so it is); PKCE (a
  confidential client with a secret; add it if a public-client flow ever
  appears); refresh tokens (a 30-day opaque session re-minted by
  sign-in is enough machinery for this product).
- **Would change our mind:** a second provider whose claims differ
  (Google's email is enough today — a provider without email would make
  the account slot show nothing and force a display-name claim);
  session-fixation-grade issues found at CP3's browser walk.
