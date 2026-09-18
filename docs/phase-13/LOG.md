# Phase 13 — log: accounts, tenancy, and share links

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

Roadbook now runs in two modes, and the first is the reference. With
`ROADBOOK_AUTH=off` (the default) nothing changed for a self-hoster: one
instance, one person, no sign-in surface, every existing suite green with
zero diff in the pure core. With `oidc`, one instance holds several
people's data: every user-data row carries its owner, every store query
filters by it, identity is delegated to an OpenID Connect issuer, and
sessions are opaque rows in Postgres behind an HTTP-only cookie. Share
links open one confirmed adventure read-only for anyone holding the URL,
in either mode. Self-serve deletion removes everything of a person's,
structurally, in either mode.

By checkpoint:

- **CP1** — migration 00011 (`users`, `user_id` on the nine user-data
  tables, content-hash dedupe rescoped per user), the store sweep with
  `userID` as every method's first data argument, the API's single
  `currentUser` seam, and the cross-tenant test family.
- **CP2** — `internal/auth` (go-oidc + x/oauth2), migration 00012
  (sessions, hash-only), all four auth operations in the contract
  including the redirects, `AuthMiddleware` as the one place a request
  becomes a user, the fake issuer in `authtest`.
- **CP3** — session-cookie forwarding in the generated client wrapper,
  `requireUser` page gates, `/signin`, the proxy routes, the account slot.
- **CP4** — migration 00013 (`share_links`, hash-only, tied to the
  decision), six contract operations, `/shared/[token]` in the public
  shell drawing the owner's plate through the same helpers, noindex as
  header and metadata, the suite's one writing e2e spec.
- **CP5** — `DELETE /me` (one transaction over `store.UserDataTables`,
  files swept after commit when unreferenced by anyone), the imports-page
  control with its typed confirmation, compose passthrough of the auth
  variables, README's two-modes statement, this log.

## What was verified

- `make test` cold (`-count=1`) with every regression confirmed as run:
  fixture 18/1/32, archive, both goldens, demo, corpus; the cross-tenant
  family; share and deletion in both modes; the OIDC flow against the
  fake issuer.
- Pure core untouched all phase: `git diff --stat` over `internal/detect`,
  `internal/journey`, `internal/photosource`, `internal/timeline` is
  empty across every checkpoint.
- Web: tsc, lint (0 errors), production build, vitest 63; the e2e suite
  at 92 passing across three viewports against a scratch compose stack
  (demo dataset, three confirmed), including the share walk and the
  auth-surface spec.
- Live, on the scratch stack: a share minted before a re-detection kept
  opening after candidates renumbered; a photo thumbnail served through
  the token proxy; "Delete everything" through the browser left zero
  user rows, an empty photos and uploads volume, and landed on /welcome.

## What broke, and the shape of each fix

1. **The cross-tenant test family found nothing missing — because it was
   written first.** Not a break, recorded as the phase's central
   observation: the phase-4 harness's stated purpose ("the day every
   query gains a user filter") was the day, and the sweep's uniformity
   rule (owner as the first data argument, every statement filtering)
   made a missed filter a diff-visible fact rather than a runtime one.
2. **x/oauth2 v0.37 requires Go 1.26.** `go get` bumped the module's go
   directive silently. Fix: pin v0.36.0, `go mod edit -go=1.25.7`,
   `GOTOOLCHAIN=local go mod tidy`, keep the Dockerfile's golang:1.25.
   Recorded in DECISIONS because it will recur at the next dependency
   bump.
3. **Redirect endpoints inside a strict generated server.** The OIDC
   dance needs 302s with `Location` and `Set-Cookie`; hand-mounting them
   beside the mux would have broken invariant 10. Fix: declare the
   headers on the 302 responses in openapi.yaml — oapi-codegen generates
   typed header structs, and cookies reach handlers through the
   middleware's ctx stash. Nothing hand-mounted.
4. **The gate's placement.** A layout gate goes stale on soft navigation;
   `middleware.ts` would be a second enforcement point to keep honest.
   Fix: `requireUser()` at the top of each app page, UX only, with Go's
   401 as the enforcement. A bug here shows an empty shell, never data.
5. **Public shell importing from the app shell.** The shared page needed
   the adventure plate, which lived under `app/(app)/adventure/[id]/`.
   Fix: move the four plate components to `components/adventure/` and
   hand the owner's islands in as slots — the phase-9 seam stays a
   file-layout fact, and a stranger's plate is the owner's component.
6. **`notFound()` streams as 200 under the root loading boundary.** The
   shared not-found page carries the right content and the noindex
   header but the transport status is 200 (the NEXT_REDIRECT precedent).
   The API answers a true 404 (tested). The e2e spec asserts content,
   not status; recorded rather than worked around.
7. **Cascades were the wrong deletion mechanism.** `ON DELETE CASCADE`
   from `users` would have hidden the list of what "everything of yours"
   means. Fix: an exported table list, deleted in FK order in one
   transaction, with the test counting every table on it to zero for the
   deleted user and non-zero for the other. Files go after the commit,
   and only when no remaining row of anyone's names the hash.
8. **The store deletion test was vacuous for one table on the first run
   (sessions for user B).** The test seeds a session for both users now,
   and asserts every table held rows for the deleted user before
   deletion — a deletion proof over an empty table proves nothing.
9. **One flaky e2e failure on the first run after a stack rebuild** (the
   share spec's dialog on a cold web container); passed in isolation and
   on the full rerun. Recorded, not masked with a retry.

## Carried out of the phase

- Live-Google browser walk: needs an OAuth client from the maintainer's
  console; rides the hosting phase's checklist (BRIEF §7 tripwire, as
  planned).
- Switching an instance with off-mode data to oidc: the `self` rows are
  invisible to signed-in accounts by construction; a "claim this data"
  migration is not built and is stated as such in the README.
- Per-user data export for the user (not the operator): the backup format
  already carries one person's data; a user-facing download is its own
  item.
- Life-map sharing: deferred deliberately (a categorically bigger
  disclosure), its own STOP when asked for.
- Expiring or labelled share links: on evidence.
- Compose's `ROADBOOK_PUBLIC_URL` defaults to loopback; hosting sets it.
