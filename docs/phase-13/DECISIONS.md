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

## 2026-09-16 — CP3: where the web gate lives, and what stays public

- **Chosen:** the gate is `requireUser()` at the top of each app PAGE, not
  the (app) layout and not proxy middleware — layouts don't re-run on
  soft navigation, and the check is UX only (Go's 401 is the enforcement;
  a bug here shows an empty shell, never data). Exactly one cookie
  crosses the Next→Go boundary (the session; client middleware fills
  only silence, so the callback's explicit state-cookie header wins).
  /welcome STAYS static and ungated: it is genuinely the public pitch,
  its islands fail honestly on Go's 401, and the signed-out path lands
  on /signin via every app page's gate — the phase-7 statically-rendered
  decision survives. /signin renders exactly one action (the OIDC
  redirect link, zero JS) and redirects home on an authless instance —
  no dead controls, the phase-9 rule kept now that auth is real.
  Sign-out is a plain form POST through a proxy route (leaving needs no
  JavaScript). Callback failures land on /signin?error=… — a stranded
  JSON body is not a page a person can act on.
- **Rejected:** gating in the (app) layout (soft-nav staleness); Next
  middleware.ts auth (a second enforcement point to keep honest when Go
  already is one); forwarding all cookies upstream (the boundary stays
  as narrow as the architecture drawing).
- **Would change our mind:** session expiry mid-browse surfacing as
  ugly boundary errors instead of the signin redirect (then data fetches
  gain a 401→redirect helper); a real marketing landing replacing
  /welcome's pitch role (then /welcome may move behind the gate).

## 2026-09-16 — CP4: share links

- **Chosen:** a link is a row in `share_links` tied to the adventure's
  DECISION (its durable identity — links ride re-detection the way photos
  do), storing only the token's SHA-256 exactly like sessions: the raw
  128-bit token is returned once at creation, the web island composes the
  URL on the browser's own origin (no public hostname assumed anywhere),
  and a lost link is replaced, never recovered. Revocation is a DELETE.
  Every way a link can fail to open — unknown, revoked, decision dismissed
  since, orphaned by re-detection — is one 404 from the API and one
  not-found page; the outside learns nothing about which. The three
  token-side reads (view + two thumbnail operations) are the only
  additions to the auth-exempt set: the token is the credential, the
  requester is nobody by design, and every read the token unlocks runs as
  the link's OWNER (`matchedStateFor`, `journeyFor`, `placedPhotos`,
  `placedImportPhotos` — the same helpers the owner's own page now uses,
  so a stranger's plate cannot differ from the owner's by construction).
  On the web the plate components moved from the app route into
  `components/adventure/` (a public-shell page must not import from the
  app shell — the phase-9 seam stays a file-layout fact); the owner's
  islands (photo upload, share controls) arrive as slots; the shared page
  lives in `(public)`, reads no session, and carries `noindex` both as a
  header (next.config, covering the thumbnail proxies too) and as page
  metadata. The e2e share spec is the suite's one writing test —
  mint/open-as-stranger/revoke, self-cleaning, desktop project only
  (three projects would race on one adventure's link list; it resizes
  its own viewport for the phone check).
- **Rejected:** storing the raw token for later re-copy (a leaked row
  would be a working link; the once-shown URL with a plain statement is
  the honest trade); a soft "revoked" state (nothing to reason about, and
  revoked-vs-never-existed must be indistinguishable anyway); per-link
  labels or expiry (no evidence yet); the owner's plate number on the
  shared cover (it is the owner's atlas register, meaningless to a
  stranger); any owner-identifying field in `SharedAdventure` (no
  candidate id, no score, no account — nothing about the owner).
- **Would change our mind:** people losing links repeatedly (then an
  opaque stored prefix for identification, or a per-link label);
  a request for expiring links (an `expires_at` column, checked in
  `ResolveShareLink`); a second consumer of the shared read (a life-map
  share) — that reopens the deliberately-deferred bigger disclosure at
  its own STOP.

## 2026-09-18 — CP5: deletion, and the close

- **Chosen:** self-serve deletion is one store transaction that deletes
  table by table, in foreign-key order, from an exported list
  (`store.UserDataTables`) — cascades were available and deliberately not
  used, because the list IS the statement of what "everything of yours"
  means, and the deletion test counts every table on it to zero (a new
  user-data table missing from the list fails at review, the cross-tenant
  family's rule applied to deletion). Files go after the commit and only
  when no remaining row of anyone's names the hash — identical bytes are
  one file on disk regardless of owner, so a photo two people hold
  survives one person's deletion. Mode off keeps the `self` row (every
  table's DEFAULT names it; the instance is "empty again"); mode oidc
  drops the user row so a re-sign-in starts from nothing, and the response
  clears the cookie. The operation takes the import lock and answers 409
  while an import runs — the goroutine would write rows behind the
  deletion otherwise. The control lives on the imports page (where the
  data story lives), behind a native dialog with a typed "delete" — the
  one typed confirmation in the product, warranted by no-undo on a
  person's history. Compose now passes the five auth variables through;
  `.env.example` documents the Google registration shape; the README
  carries the two-modes statement with off as the reference. The e2e
  suite gained the auth-surface spec (authless: /signin redirects home, no
  sign-out anywhere, the deletion dialog arms only on the typed word and
  is never submitted — Go owns the deletion proof).
- **Rejected:** `ON DELETE CASCADE` from `users` as the deletion mechanism
  (hides the list; a forgotten table would silently keep rows or silently
  lose them); deleting files inside the transaction (a rollback after an
  unlink is a row pointing at nothing — the photos rule); a per-user
  export bundled with deletion (backup already covers it for the operator;
  a user-facing export is its own item); a "delete this import" partial
  deletion (observations from overlapping exports dedupe by content hash,
  so per-import removal is not well-defined — all-or-nothing is honest).
- **Would change our mind:** a hosted instance where users ask to remove
  one export's contribution (then imports need an ownership graph over
  observations, a schema change); deletion taking long enough to need a
  background job (tens of thousands of rows delete in well under a second
  today — measured on the archive-scale scratch data).
