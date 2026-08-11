# Phase 7 decision log

Three lines each: what was chosen, what was rejected, what would change our
mind. Written at the moment of decision, not reconstructed.

## 2026-08-10 — Brief drafted; recommendations are proposals until Gate 1

Chosen: BRIEF.md drafted with five recommendations (hybrid sync/async,
streaming route handler, retain uploads, auto-detect on defaults, front door
at `/welcome` with `/` redirect when empty) — all pending maintainer review.
Rejected: treating any of them as decided before the Gate 1 STOP.
Would change: Gate 1 amendments supersede the brief text; log them here.

## 2026-08-10 — Charter amendment draft lives in docs/private/, not in the tree

Chosen: the proposed CLAUDE.md / PRODUCT.md edits are drafted as a diff in
`docs/private/CHARTER-AMENDMENT-DRAFT-2026-08.md` (gitignored), applied only
if the maintainer accepts after re-reading parked-road-community.md.
Rejected: editing the charter files directly this session — an unaccepted
amendment must not exist in the working tree where it could be committed.
Would change: acceptance at the STOP; the maintainer applies and commits.

## 2026-08-11 — Mobile is the primary path, not a rendering target

Chosen: the funnel is on-device end to end (the link arrives on WhatsApp and
the export is created on the phone — BRIEF §1.5); front door mobile-first;
the never-driven small-screen pass enters the phase scoped to the v1 loop
pages, layout and touch only; device verification starts from a real
WhatsApp message.
Rejected: desktop-first with "open it on a laptop" as the plan — the export
file lives on the phone, so a laptop path adds a file transfer; laptop stays
the documented fallback if a phone browser proves broken.
Would change: the WhatsApp in-app browser failing the device pass in a way
copy cannot fix — then the laptop fallback gets promoted in the walkthroughs.

## 2026-08-11 — Failure shape: all-or-nothing upload, every failure visible

Chosen: no import row until the upload fully lands and passes the sniff, so
an interrupted upload leaves nothing (no row, no temp file) and retry is
always safe; every post-row failure is visible on the row; two additions from
walking the failure modes — a kill-mid-upload checkpoint test, and designed
zero-candidates copy so a successful import of adventure-free data never
looks broken.
Rejected: partial-upload recovery UI, resumable protocols (trigger
unchanged), and any state where the user sees a spinner with no row behind it.
Would change: real failed-upload reports from the pilot — the resumable
trigger firing.

## 2026-08-11 — Pilot handover and data deletion: operator reset, no button

Chosen: sequential testers on one instance are handled by volume-level reset
(`docker compose down -v` — structurally complete across DB, thumbnails,
retained uploads) plus a fresh link per person (old WhatsApp chats keep old
links; the link is the only secret on an authless instance). Self-serve
"delete my data" goes to the PLAN backlog behind the multi-user/accounts
trigger. Both recorded in DIRECTIONS (phase 8 runbook material).
Rejected: a delete-everything button in phase 7 — a destructive control on an
authless instance, and row-chasing deletion is less trustworthy than volume
destruction.
Would change: accounts arriving (v2) — deletion then becomes a user-facing
legal requirement, not an operator courtesy.

## 2026-08-10 — Phase 5's upload-UI rejection reopened on a named new fact

Chosen: the brief's opening names the v1 definition (DIRECTIONS, 2026-08-10)
as the substantive new fact that reopens phase 5 §3B's rejection of an
import upload UI, and keeps that rejection's technical reasoning as design
input (§§3A–3B answer it).
Rejected: reopening silently, as if phase 5 had not decided.
Would change: nothing — the reopening standard itself is charter.

## 2026-08-11 — Migration 00009 carries detect_status as well as content_hash

Chosen: persist detect_status on the imports row (the brief's §4 named the
API field but the migration named only content_hash — found at
implementation: the front door polls detect_status, and serve memory would
turn "detection failed" into silence across a restart).
Rejected: in-process detect state (lost on restart), and deriving it from
the runs table (a failed detect leaves no run row to derive from).
Would change: nothing foreseeable; the column is additive and nullable.

## 2026-08-11 — imports gains an `inserted` column (found writing the tests)

Chosen: record the genuinely-new observation count on the imports row
(migration 00009, `inserted`; API field `inserted`). Without it a duplicate
upload's row is indistinguishable from its original — the store computed
the number all along and only ever printed it to a terminal, so the front
door could not say "nothing new".
Rejected: leaving duplicate detection to prose comparison of counters, and
per-type inserted counters (one total answers the UX question).
Would change: nothing foreseeable; additive, NULL on historical rows.
