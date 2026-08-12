# Phase 7 log — the front door: import upload

Written at phase close. What the phase built, what broke while building it,
and why each fix took the form it did. The pure core (`internal/detect`,
`internal/journey`) was not touched at any point; both goldens and the
detection regressions closed byte-identical. Public numbers here come from
the committed demo dataset.

## What the phase does

A person with a link and a Timeline export reaches their candidates without
anyone running a CLI. The phase built the connective tissue in four
checkpoints: the upload path in Go, the upload loop in the browser, the
front door page, and the close.

- **The contract and the Go path (CP1).** `POST /imports` accepts a
  multipart upload: the file part streams to disk through a
  `TeeReader`-fed SHA-256 (no process ever holds the file in memory),
  `MaxImportUploadBytes` (2 GiB) is enforced during the same pass, and the
  head is sniffed synchronously — a wrong file is rejected in the response
  itself as an `ImportRejection` carrying the taxonomy's message and its
  stable `detected_format` label, and the file is deleted. An accepted
  file is promoted to `<hash>.json` under the uploads directory
  (`ROADBOOK_UPLOADS_DIR`, a named compose volume) and retained: an export
  can be irreplaceable, and the instance copy may become the only one. The
  rule that shapes every failure mode: **no import row exists until the
  upload has fully landed and passed the sniff** — an interrupted upload
  leaves nothing, so retrying is always safe. Import and auto-detect run
  in a goroutine behind a `202`; the imports row is the only status
  channel (`GET /imports/{id}` polls it); a `TryLock` guard answers `409`
  to a second concurrent upload; a startup sweep finalises `running` rows
  orphaned by a crash as visibly failed. Migration 00009 added
  `content_hash` (row ↔ retained file, the photos pattern),
  `detect_status` (a detect failure must survive a restart), and
  `inserted` (found writing tests: without it a duplicate upload's row was
  indistinguishable from its original).
- **The upload loop in the browser (CP2).** A route handler
  (`/api/imports`) forwards the request body as a stream — generated
  client, `bodySerializer` passing the `ReadableStream` through with
  `duplex: "half"`; the archive never materialises in the Node process.
  The island uses `XMLHttpRequest` for the one thing `fetch` cannot do —
  upload progress — and models the two async facts as explicit phases:
  uploading → importing → detecting → done. "Done" is declared only when
  `detect_status` resolves, so "see your candidates" never points at a
  list still being written; "imported but detection failed" renders
  instead of a spinner. The maintainer's real archive was uploaded through
  this path as the real-scale streaming proof (numbers stay private), and
  the scratch stack that held it was destroyed volumes-and-all — the
  handover-reset story working as designed.
- **The front door (CP3).** `/welcome` carries a cold visitor from "what
  is this?" to their candidates: pitch first (an inline SVG atlas plate of
  the three demo adventures, geometry reused verbatim from the phase 6
  mockups — 28 KB that demonstrates the leg-kind language rather than
  showing a picture of it), then "what you need" with the never-enabled
  branch stated before the walkthroughs, the per-platform walkthroughs
  with visible verification state, the embedded upload island, and "what
  happens next" naming `/imports` and the retention fact. Rejection is
  redirection: `lib/rejection-anchors.ts` maps every format label (13
  sniff-time + `truncated`/`json-unrecognised` from the parser) to a
  section id the page renders from the same module — a dead-end anchor is
  unrepresentable, and a vitest walks all fifteen labels as the
  regression. `/` redirects to `/welcome` only when `GET /imports`
  answers with zero rows; the life map's empty states, `/candidates`, and
  `/imports` chain back to it. The zero-candidates ending is designed
  copy on both surfaces that can show it ("detection ran and found no
  candidates … your data imported fine"), proven with a fixture that
  parses cleanly and detects nothing. The small-screen pass covered the
  v1 loop pages at a real 390 px viewport: tables fold to the columns a
  thumb needs, decide-cell controls grew padding-expanded 44 px tap
  targets with no visual change, and no loop page scrolls horizontally.
- **The close (CP4).** The README quickstart is browser-first and was
  executed from a clean clone: `docker compose up -d --build`, open the
  instance, upload the demo export at the front door — three candidates
  with zero CLI commands. The demo compose stack was rebuilt to the
  phase-7 image and restored *through the browser path itself* (upload,
  three confirmations in the UI), then countries (177) and the routing
  batch (13/13 pairs, Westfjords 0 unknown / 240 km routed, flights
  untouched as air) — matching the phase 5 record. The Android
  walkthrough was verified end-to-end from a real WhatsApp message on the
  maintainer's device and is stamped on the page; the iPhone branch ships
  with its honest "unverified" marker (the audit friend's device was not
  reachable; the BRIEF makes the marker the requirement, not the
  verification).

## What broke, and why each fix took its form

- **The uploads volume was unwritable on first mount.** Named volumes
  inherit the *image* directory's ownership when first mounted, and the
  image had no `/uploads`; the first real upload failed with a permission
  error. Fix in the Dockerfile — `mkdir` + `chown` at build — because
  that is why the photos volume had always worked: the precedent was
  already in the file, one directory short.
- **A duplicate upload was invisible in the record.** The store had
  always computed how many observations were genuinely new — and only
  ever printed it to a terminal. The front door needed to say "this file
  held nothing new", so the number became a column (`inserted`, migration
  00009) rather than prose parsing of counters.
- **`detect_status` had to be a column, not process state.** The brief
  named the API field; writing the sweep made it obvious that serve
  memory would turn "detection failed" into silence across a restart. A
  failed detect leaves no run row to derive from, so the imports row
  carries it.
- **Node streaming forwarding simply worked.** The phase's named main
  unknown — `duplex: "half"` through Next 16 standalone — held up at
  archive scale; the spool-to-disk fallback was never built. Recorded
  because a risk that dissolves is still a result.
- **SWC swallowed one inter-tag space.** "Timeline export**from**" — the
  JSX compiler dropped a plain space after a closing tag in exactly one
  construct on the page. Fixed with the explicit `{" "}` idiom at the two
  affected boundaries; caught by proofreading the rendered screenshot,
  which is why the walk captures full-page images and not just assertions.
- **Touch targets measured 36 px.** The walk asserts tap-target height;
  text links in the decide cell failed it. Fix is padding expansion
  (`-mx-2 -my-3 px-2 py-3`) — the hit area reaches 44 px while the
  rendered table does not change at any width, honouring the "layout and
  touch only" scope.
- **The imports table clipped its Status column at 390 px.** The phone
  needs When/Source/New/Status; Format folds to `sm:` and the counts to
  `md:`. Found by looking at the screenshot, not the assertion — the
  page technically fit; the story didn't.
- **The cold-instance redirect streams as a 200.** The root
  `loading.tsx` boundary flushes the shell before `redirect()` runs, so
  Next emits its client-side `NEXT_REDIRECT` instruction instead of a
  307. Accepted: every browser with JavaScript follows it, and the funnel
  requires JavaScript anyway; noted here because a curl of `/` on an
  empty instance shows a 200 and that is not a bug.
- **The db healthcheck can pass during first-volume init.** Postgres
  restarts itself once while initialising a fresh volume; `pg_isready`
  answered in the gap and the api container hit "the database system is
  starting up" and exited. A second `docker compose up` recovers; known
  compose-ecosystem behaviour, tolerable for a demo stack because the
  restart policy and re-up are both trivial.
- **The browser path does not populate countries.** Country attribution
  still needs the operator's one-time `roadbook countries`; the README
  states it as the one CLI step remaining. Folding it into the compose
  startup (it is idempotent and wholesale) is a small candidate change,
  carried forward rather than slipped into a close checkpoint.
- **Two stale dev servers were found bound to all interfaces at close.**
  A CP2-era `next-server` on `*:3002` and the phase-6 `next dev` on
  `*:3001` (fronting the real local API) had outlived their sessions —
  on a home LAN, but contrary to the loopback discipline the compose
  stack enforces. Both killed. Lesson recorded: host-run dev servers
  bind all interfaces by default; the loopback rule lives in compose,
  not in `next dev`.

## Verification at close

- Cold-cache `make test` green, DB-backed suites included (api, store,
  backup ran against scratch databases). Fixture regression 18/1/32 and
  archive regression pass against the real data; both goldens
  byte-identical; demo regression 3/1/0; web: 53 vitest tests across 3
  files, 17 of them the rejection-anchor walk.
- README quickstart executed verbatim from a clean `git clone` — fresh
  volumes, no `.env`, no `data/` — ending at 3 candidates using only the
  browser.
- The CP3 walk (30 checks) passed at desktop and at a real 390×844
  viewport on a fresh scratch stack, including the PDF rejection with its
  redirection link, both zero-candidates endings, and confirm-under-thumb.
- Device pass: Android walked end-to-end from a WhatsApp-opened link
  (steps matched the phone's menus; upload completed in the in-app
  browser); stamped on the page. iPhone ships marked unverified.
- Data safety: the device-pass instance (which held a real export) was
  destroyed volumes-and-all and its temporary LAN publish reverted to
  loopback; `git add -A --dry-run` clean of `data/`, uploads, and
  anything near 1 MB.

## Carried forward

- **iPhone walkthrough stamp** — lands in a follow-up commit when the
  audit friend walks it; the branch ships honestly unverified until then.
- **Countries at startup** — fold `roadbook countries` into the compose
  startup path so the browser-only quickstart also yields country
  attribution; small, evidence in hand.
- **Resumable uploads** — trigger unchanged (a real failed-upload report
  from the pilot); the mobile funnel makes it likelier to fire.
- **Retained exports vs backup scope** — uploads are now a third class of
  hard-to-replace data that `roadbook backup` does not cover; carried in
  the BRIEF (§3C), still backlog.
- **Phase 8** — friend pilot hosting supplies the link this front door
  was built for.
