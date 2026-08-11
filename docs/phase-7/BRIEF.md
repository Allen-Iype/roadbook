# Phase 7 design brief — the front door: import upload

**Goal:** a person with a link and a Timeline export reaches their candidates
without anyone running a CLI. This is the first half of the v1 loop (link →
pitch → walkthrough → upload → import → detect → life map); phase 8's hosting
work supplies the link. The phase serves self-hosters identically: after it, a
fresh `docker compose up` needs no `docker compose run api roadbook import`
step to become useful.

Phase 7 inherits every hard piece already built. `Parse(io.Reader)` streams any
reader at constant memory (phase 1 checkpoint 5). The sniffer rejects thirteen
known wrong inputs with actionable messages, and its stable format label is
recorded per import (`detected_format`, migration 00007). The imports table
carries `running | completed | failed` with error prose, and `/imports` renders
it (phase 5 §3B). Photo upload established multipart through the one API
boundary (phase 4 §1.3). What this phase builds is the connective tissue — an
upload endpoint, a background import, an auto-detect step — and the one thing
that exists nowhere yet: a page that carries a cold visitor from "what is
this?" to their own adventures.

Phase 5 §3B rejected an upload UI ("it turns a few-times-a-year batch
operation into a request/response problem"). The substantive new fact that
reopens it is the v1 definition (DIRECTIONS 2026-08-10): the target user is no
longer only the operator, and "hand the operator your location history" is the
wrong privacy shape for a hosted pilot — the export must go browser → own
instance, no relay copies. The rejection's technical content (multi-GB bodies,
progress, timeouts) was correct and is exactly what §§3A–3B answer.

---

## 1. Concepts this phase introduces

### 1.1 Streaming an upload end to end

A Timeline export can run to hundreds of megabytes (`rawSignals` inflates it;
the committed demo is 152 KB, a real multi-year archive is not). The rule that
makes this tractable is: **no process ever holds the file in memory.** The
browser streams the file from disk; the Next layer forwards the request body
without reading it; the Go handler copies the stream to disk as it arrives;
the parser later reads the file with the same constant-memory `Parse` that the
CLI uses. Three mechanisms, one per hop:

- **Browser → Next:** a POST from a client component. `fetch` cannot report
  upload progress, so the upload island uses `XMLHttpRequest`, whose
  `upload.onprogress` events drive the progress bar — the one legitimate use
  of XHR left in 2026. The body is `multipart/form-data`, produced by
  `FormData`, same as photos.
- **Next → Go:** a *route handler* (phase 4 introduced these for thumbnails),
  not a server action. It forwards the incoming request's `ReadableStream`
  body and `Content-Type` header verbatim to the Go API — it never parses the
  multipart at all, so nothing buffers. §3B carries the reasoning and the
  known verification point (Node fetch requires `duplex: "half"` for
  streaming request bodies — confirm against the local Next docs at
  implementation, per the standing Next 16 rule).
- **Go:** the oapi-codegen strict server hands the handler a multipart
  reader. The file part is `io.Copy`'d to a temp file in the uploads
  directory through an `io.TeeReader` into SHA-256, so the content hash is
  computed during the single pass. `http.MaxBytesReader` enforces the size
  cap. Nothing is parsed yet.

The upload and the import are therefore *two separate acts*: the upload's job
ends when the bytes are safely on the instance's disk; the import reads them
from there. That separation is what makes §3A's async answer safe — a
connection drop after upload loses nothing, and a crash mid-import can retry
from the retained file.

### 1.2 Background work without a queue

The import of a large archive takes real time — long enough that holding an
HTTP request open across it invites every timeout between the browser and the
Go handler. The charter forbids a message queue, and rightly: a queue's value
is durability and distribution across workers, and this system has exactly one
worker and already owns a durable record — the imports table.

So the pattern is the plainest one Go offers: the handler finishes the upload,
writes the `running` row (`BeginImport`, which exists), responds `202
Accepted` with the import id, and starts a goroutine that parses, inserts,
finalises the row (`ImportObservations` / `FailImport`, which exist), and runs
detection. The imports table is the only status channel; the frontend polls
`GET /imports/{id}` — no WebSockets (charter), no server-sent events, just the
page asking again every couple of seconds while visible. The `/imports` page
is the durable fallback: close the tab, come back, the row is still there.

Two failure shapes need naming because a goroutine has no terminal:

- **Crash mid-import.** The process dies, the row says `running` forever. At
  serve startup, any `running` row is finalised as `failed` with an
  "interrupted by restart" message — visible, honest, and recoverable because
  the uploaded file is retained (§3C): re-uploading or re-importing the same
  bytes is idempotent at the row level (proven: re-import adds 0). One
  sharp edge, accepted and documented: a CLI import running at the moment
  serve restarts would be falsely swept. The CLI operator is watching a
  terminal; the sweep exists for the browser user who is not.
- **Concurrent uploads.** One import at a time, enforced by an in-process
  guard in serve; a second upload while one runs gets `409` with a message
  pointing at `/imports`. Single instance, single user — a semaphore, not a
  scheduler. (Compose never runs replicas; phase 5 established this.)

One rule spans every failure shape, stated once because the UI leans on it:
**no import row exists until the upload has fully arrived and passed the
sniff.** An interrupted upload therefore leaves nothing behind — no partial
data, no stuck row, no temp file (the handler deletes its temp on any
incomplete copy) — and retrying is always safe, because everything downstream
is idempotent. Once a row exists, every later failure is visible on it. The
user always sees either an immediate message on the page or a row on
`/imports`; a spinner-forever state is a bug by definition.

### 1.3 The two rejection moments

The sniffer's taxonomy is the product's best error UX, but its rejections now
arrive at two different times, and the page must handle both:

- **Synchronous — at upload.** Once the file is on disk, the handler reads its
  head and sniffs *before* accepting. A PDF, a zip, a Records.json — the
  thirteen known wrong inputs — are rejected in the upload response itself
  (`400`, carrying both the message and the stable `detected_format` label),
  and the rejected file is deleted, not retained. The label is what lets the
  page *redirect* rather than merely apologise (§3E): `semantic-history` →
  "this is the old Takeout format — export from your phone instead, here's
  how", `zip` → "extract the archive and upload the Timeline file inside".
  Today `sniff` is unexported and runs inside `Parse`; checkpoint 1 exports a
  head-sniffing entry point from `internal/timeline` (pure function over
  bytes, no contract change, parser types still never escape).
- **Asynchronous — mid-parse.** Truncation and unrecognised-JSON are only
  discoverable while parsing, which now happens in the goroutine. These land
  exactly where phase 5 put them: a `failed` imports row with the actionable
  message, which the polling page renders with the same redirection treatment.

### 1.4 Onboarding for a file nobody has heard of

Everything above is plumbing. The product problem is that a link recipient has
never heard the words "Timeline export." The front door page (§3E) is the
onboarding surface: it opens with the pitch — what Roadbook is and what it can
show, demonstrated with demo-derived visuals — *before* asking for anything;
then per-platform export walkthroughs live in the page itself, each carrying a
"verified as of" date because Google changes these flows unannounced (the
`probe` lesson, applied to prose); then expectation setting — the file may be
large, upload → import → detection → "your candidates appear", everything
stays on this instance, `/imports` means you can close the tab. The honest
third branch is stated up front: Timeline never enabled means nothing to
export, full stop — this page never implies data exists that does not.

The page is built to grow: when photos-as-source or GPX land (their own
phases, on evidence), this becomes the one "add your data" surface listing
each source with its own how-to. Nothing in this phase builds for that beyond
not painting the page into a Timeline-only corner.

### 1.5 The phone is the whole funnel

The link arrives on WhatsApp and is opened on a phone — and the phone is not
merely the reading device: the Timeline export is *created on the phone* and
lands in its storage. The natural funnel is therefore entirely on-device: tap
the link → read the pitch → do the export on the same phone → pick the file
from Downloads/Files → upload over Wi-Fi. Asking for a computer anywhere in
that chain means asking the user to move a file between devices first — real
friction, and for most pilot users the step where the funnel dies. A laptop
is the fallback if a phone browser proves broken, never the plan.

Three consequences:

- **The front door is designed mobile-first**, and the small-screen pass the
  project has never driven (a known phase 6 delta) enters this phase, scoped
  to the v1 loop pages: front door, imports, candidates/confirm, life map,
  adventure detail. Not a general redesign — the loop must be walkable under
  a thumb, including its one interactive step (confirm/dismiss/name).
- **WhatsApp opens links in its own in-app browser**, whose file pickers and
  large uploads are the classic trouble spots of embedded WebViews. Device
  verification (§7) starts from an actual WhatsApp message, not a browser
  address bar; if the in-app browser misbehaves, the likely fix is an "open
  in your browser" hint — discovered by testing, not guessed.
- **Phone browsers suspend background tabs**, which can kill an upload when
  the user switches apps. The expectation copy says so plainly: use Wi-Fi,
  keep the page open until the upload finishes. Mobile makes the
  resumable-upload trigger (§6) meaningfully more likely to fire; the
  design's job today is only that an interrupted upload fails clean and
  visibly (§1.1, §1.2).

---

## 2. What gets built

- `POST /imports` and `GET /imports/{id}` in `openapi.yaml` (§4), server and
  client regenerated, handlers in `internal/api`.
- Upload handling in Go: stream-to-disk with tee'd SHA-256, size cap
  (`MaxImportUploadBytes`, named parameter), synchronous head sniff, retention
  under an uploads directory (`ROADBOOK_UPLOADS_DIR`, compose named volume),
  single-import guard, background import goroutine, startup sweep for
  orphaned `running` rows.
- Auto-detect after successful import (§3D): the goroutine runs detection with
  default parameters, recorded as a normal run (invariant 3 — the run row
  carries its params like every run since phase 1).
- An exported head-sniff entry point in `internal/timeline`.
- Web: the streaming route-handler proxy (`/api/imports`), the upload island
  (file input, XHR progress, per-outcome rendering: accepted / rejected-with-
  redirection / failed-later), status polling against the generated client.
- The front door page: pitch, three walkthrough branches with verified-as-of
  dates, expectation setting, upload embedded, rejection-as-redirection
  mapping from `detected_format` to walkthrough anchors.
- Empty-state chaining: the life map's no-imports state and the `/imports`
  page link to the front door; `/` routes cold visitors there when the
  instance has no imports (§3E).
- The small-screen pass over the v1 loop pages (§1.5) — layout and touch
  interaction only, no redesign — and the designed zero-candidates ending:
  an import that completes but detects nothing says so in words ("your data
  imported fine — no journeys far from home were found"), never an
  accidental empty screen.
- Migration 00009: `content_hash` on `imports` (nullable — CLI imports and
  historical rows have none). The upload path records the hash of the file
  that produced the import, tying row to retained file the way photos tie
  rows to thumbnails.
- README: the quickstart gains the browser path; the CLI import section
  remains for operators.

The pure core is untouched: no change to detection, journey assembly, or any
`internal/detect` / `internal/journey` code. Goldens stay byte-identical.

---

## 3. The real choices

### 3A. Sync vs async

**The choice: does the upload request block until the import finishes?**

Recommended: **the hybrid of §§1.1–1.3.** Upload and sniff are synchronous —
the user learns "wrong file" immediately, in the response, while they are
looking at the page; parse-and-insert is asynchronous behind a `202`, with the
imports table as the status channel and polling as the transport.

Rejected: fully synchronous (hold the request through parse) — a multi-hundred-
MB parse behind three hops of HTTP invites browser, proxy, and handler
timeouts, and a dropped connection would abandon work invisibly mid-insert.
Rejected: fully asynchronous (accept bytes, return, sniff later) — it throws
away the best moment for the taxonomy to act as navigation; a user who
uploaded the wrong file should not have to poll to find out. Rejected: a job
queue — charter, and §1.2 shows the imports table already is the durable
record a queue would add.

What would change our mind: if real archives turn out to parse in hundreds of
milliseconds, sync-everything becomes simpler and the status machinery
overbuilt — but the machinery (imports rows, polling `/imports`) exists
regardless for the CLI path, so the async choice costs almost nothing extra.

### 3B. Transport — streaming route handler, not a server action

**The choice: how the bytes cross the Next layer.**

Recommended: a **route handler that forwards the request body as a stream**,
never parsing it. The photo precedent (server action, `bodySizeLimit: 64mb`)
does not scale here: a server action materialises `File` objects — the whole
archive in the Node process's memory — and its encoding is designed for
React-invoked mutations, not gigabyte bodies. The route handler treats the
upload as what it is, plain HTTP: forward `Content-Type` (which carries the
multipart boundary) and the `ReadableStream` body verbatim to
`POST /imports` on the Go API; return Go's response verbatim. Invariant 11
holds — the browser still talks only to Next — and the Go API stays the only
process that touches disk or database.

Named limits, one per layer, Go authoritative: `MaxImportUploadBytes`
(recommended default 2 GiB — generous because an archive's size is not the
user's fault) enforced via `http.MaxBytesReader`; the Next route handler
passes streams and needs no body-size configuration of its own (verify
against the local Next 16 docs — route handlers, unlike server actions, do
not buffer by default; if implementation finds otherwise, that surfaces as a
checkpoint 1 finding).

The known verification point, stated in §1.1: Node's `fetch` streams request
bodies only with `duplex: "half"`. If streaming forwarding proves unworkable
in Next 16's runtime, the fallback is the route handler spooling to a temp
file and forwarding from disk — memory stays constant, one extra disk copy,
and the brief's shape survives. Rejected outright: raising the server-action
limit to archive scale (buffers the archive in memory by design); the browser
talking to Go directly (the one-boundary rule is architecture, not
preference).

### 3C. File retention

**The choice: is the uploaded export kept after import, and where?**

Recommended: **keep it.** The strongest reason is the one the project already
lives by: a Timeline export can be irreplaceable. Its `rawSignals` window
expires at the source (the maintainer's own archive proves it), and Google
has purged Timeline data server-side before (2024). For a pilot user whose
downloaded export lives in a phone's Downloads folder, the instance copy may
soon be the *only* copy. Deleting it after parse would make the parser's
current field coverage a one-shot: any future parser improvement (new fields,
legacy formats, `timelineMemory`) could never benefit an existing user
without a re-upload they may no longer be able to produce.

Mechanics: files land in `ROADBOOK_UPLOADS_DIR` (compose: a named `uploads`
volume, same pattern as `photos`; dev default under `data/`, same pattern as
`data/photos/`), named by content hash — so re-uploading identical bytes
overwrites nothing and stores nothing new, and the imports row's
`content_hash` column ties row to file exactly as photos tie rows to
thumbnails. A duplicate upload simply produces an import row whose counters
show 0 new observations (row-level idempotency, proven since phase 1) — no
special-case response needed. Rejected files (§1.3) are deleted, not
retained: they are the wrong file, possibly wholly unrelated to location
data, and keeping them serves nothing.

The UI states retention plainly beside the upload control — "your export is
stored on this instance" — because a privacy-shaped feature must not be
silently custodial. Backup scope is *unchanged this phase*: `roadbook backup`
still archives decisions and photos only. The retained exports are now a
third class of hard-to-replace data, and that tension is real — carried to
the backlog explicitly rather than solved here, because archive-scale backups
are a different design (phase 5 §3D rejected including source exports for
exactly the size reason).

Rejected: delete-after-parse (the irreplaceability argument above). Rejected:
retention opt-out this phase — a knob nobody asked for yet; add it on the
first real request. Rejected: an uploads management UI (list/delete) — same.

### 3D. Auto-detect after import

**The choice: does a successful upload-import run detection, or does the user
click a second button?**

Recommended: **yes, automatically, with default parameters.** The v1 loop
ends at "your candidates appear"; a user who has never heard the word
"detection" cannot be asked to invoke it. Detection is pure and fast
(seconds), its run row records its parameters like every run (invariant 3),
re-detection is free and safe (invariant 2), and decisions re-attach by
anchored identity — the most battle-tested path in the project. The CLI
`detect` remains the parameter-exploration surface; the automatic run is
always defaults, so any custom exploration a power user has done is
reproducible from run params as ever.

The goroutine's order is import → finalise import row → detect. A detection
failure after a successful import must not mark the *import* failed — the
import row's status describes the import (its semantics predate this phase
and the CLI shares them). A failed auto-detect is rare (pure function over
just-validated data) but must not vanish: the polling response carries the
auto-detect outcome so the page can say "imported, but detection failed"
instead of stranding the user on a spinner (§4 puts a small optional field on
`Import` for this rather than inventing a second status channel).

One visible consequence to design for in checkpoint 2, not paper over: the
page learns "import completed" from polling, but detection completes a beat
later. The front door waits for the detect outcome before declaring victory
and linking to candidates — polling the same endpoint, reading the same
field. Rejected: a "Run detection" button (a concept tax on the one user
least equipped to pay it). Rejected: auto-detect for CLI imports too — the
CLI user chose a CLI precisely to control the steps; no behaviour change
there.

### 3E. The front door itself

**The choice: what the page says, where it lives, and what "verified" means.**

*Structure.* One page, recommended route `/welcome`. Order is fixed by the
cold-visitor test — each section answers the question the previous one
raises: (1) the pitch: what Roadbook is, shown not told — demo-derived
visuals (the committed `docs/screens` demo captures, or live demo-data
rendering; never real-data screenshots); (2) "what you need": a Google
Timeline export, and the honest branch for people who never enabled Timeline
— stated before the walkthroughs, not buried after them; (3) the per-platform
walkthroughs; (4) the upload control with expectation setting; (5) what
happens next: upload → import → detection → candidates, with `/imports` named
as the place to come back to.

*Routing.* `/` keeps being the life map. When the instance has zero imports,
`/` redirects to `/welcome` — a friend's link lands on the pitch, and the
same link lands on the life map once their adventures exist. The life map's
no-candidates empty state and the `/imports` page link to `/welcome`
permanently ("add your data"). Rejected: making the front door the home page
outright — the life map earned that in phase 6, and the front door is a
doorway, not a destination.

*Walkthroughs.* Three branches, drafted at checkpoint 3 and **verified on
real devices before phase close** (§7): Android (on-device Timeline: Settings
→ Location → Location Services → Timeline → Export Timeline data — to be
verified, not asserted from memory); iPhone (Google Maps app → profile → Your
Timeline → settings → Export Timeline data — the iPhone friend from the audit
is the natural tester); and never-enabled (no retroactive data exists; what
enabling now would mean for future use — plainly, without upsell). Each
walkthrough carries "verified <date>, <platform>, <app version where
relevant>" as visible page text. An unverified walkthrough does not ship;
if a device cannot be reached this phase, its branch says "unverified — steps
may have changed" rather than carrying a false date. This is invariant 13's
discipline applied to prose: no claim the project has not checked.

*Rejection as redirection.* A static mapping from the sniffer's
`detected_format` labels to page anchors: the old-Takeout formats point at
the phone walkthroughs ("this is the old export format — your phone has a
newer one"), `zip`/`gzip` at "extract first and upload the file inside",
image formats at "that's a photo — photo import is a future feature; this
page wants your Timeline file", the generic branches at the format
statement. Every rejection lands the user on the next thing to try; no
dead ends. The mapping lives in the web layer beside the page (presentation
of an API-provided label — no business logic added to Next).

*Mobile first.* The page is designed at phone width and verified from a
WhatsApp-opened browser (§1.5); desktop is the widening, not the target. The
expectation copy carries the two mobile facts — Wi-Fi recommended, keep the
page open until the upload finishes — and the walkthroughs assume the export
happens on the same device doing the uploading, naming where the file lands
on each platform and how the upload control's file picker finds it there.

*The zero-candidates ending.* The loop's honest non-failure outcome is
designed copy, not an accident: data that imports cleanly but contains no
qualifying journeys says exactly that, beside a pointer back to what
detection looks for. An empty screen after a successful upload would read as
breakage and is treated as a bug.

*Copy discipline.* Factual voice per the charter — the pitch says what the
product does and shows it doing it; no marketing superlatives. The
expectation-setting section states the privacy property in one sentence
(the export goes to this instance and stays there) and the retention fact
(§3C), because those are product claims the repository can demonstrate.

---

## 4. The contract changes (openapi.yaml)

Named here per the working agreement: the contract is hand-edited first,
both sides regenerated after (invariant 10). No existing operation changes.

- **`POST /imports`** — multipart request body, one required `file` part
  (binary), optional `label` part (defaults to the uploaded filename, the
  existing `source_label` convention). Responses: `202` with the created
  `Import` (status `running`) — the id is the polling handle; `400` with a
  new **`ImportRejection`** schema `{ error, detected_format? }` — the
  sniffer's message and stable label, so the front door can redirect
  (`Error`'s single field cannot carry the label, and overloading prose
  parsing is exactly what `detected_format` exists to prevent); `409` with
  `Error` when an import is already running; `413` with `Error` when the
  body exceeds `MaxImportUploadBytes`.
- **`GET /imports/{id}`** — one import by id, `200` with `Import`, `404`
  with `Error`. The polling endpoint; `GET /imports` (the list) is unchanged.
- **`Import` schema** gains two optional fields: `content_hash` (the
  retained file's SHA-256, upload-path imports only — mirrors migration
  00009) and `detect_status` (`running | completed | failed`, present only
  on imports that triggered auto-detect — §3D's one-field answer to "the
  import succeeded but what about detection", in the resource the page is
  already polling rather than a second status surface).

The description prose in the yaml carries the async semantics (202 means
"accepted, poll this"), the retention statement, and the one-at-a-time rule,
so the generated client's consumers read the behaviour where they read the
types.

---

## 5. Data safety

- **Uploaded exports are real location history** — the same class as `data/`.
  They live only in `ROADBOOK_UPLOADS_DIR` (named volume in compose; under
  gitignored `data/` in dev, the photos precedent). Nothing under an uploads
  directory is ever committed, and no test fixture is ever a real upload:
  upload tests use the committed demo dataset and `testdata/` synthetics.
- **The dev default must not collide with the read-only mount.** In compose,
  `./data` is mounted `:ro` deliberately; the uploads volume is separate and
  writable. The two paths never overlap.
- **Rejected files are deleted at rejection** (§1.3) — the server does not
  accumulate unknown binaries.
- **No upload content in logs or errors.** Failure messages carry the
  taxonomy's prose and the format label, never file contents; coordinates
  never appear in any log line (standing rule, restated because a new binary
  ingress path is exactly where it would slip).
- Standing rules unchanged: `git add -A --dry-run` before every commit,
  nothing from `data/`, nothing near 1 MB, screenshots and README figures
  from demo data only.

---

## 6. Excluded, and stays excluded

- **Auth, accounts, tenancy** — v2 by definition (DIRECTIONS Direction 6);
  pilot privacy is instance isolation plus proxy auth, phase 8's concern.
- **Resumable/chunked upload protocols** (tus and kin) — a single POST
  suffices until a real failed-upload report says otherwise; that report is
  the trigger, and the mobile funnel (§1.5, tab suspension) makes it
  likelier to arrive — the trigger working as designed, not a reason to
  pre-build.
- **WebSockets / SSE for progress** — charter; polling an existing resource
  is the whole mechanism.
- **A queue, workers, or job tables** — §1.2; the imports table is the
  record.
- **Photos-as-source and GPX** — their own phases on audit evidence; the
  front door grows sections then, not now.
- **Legacy Timeline formats** — backlog behind the queryable
  `detected_format` trigger, unchanged; this phase makes the trigger *more*
  likely to fire (uploads from friends), which is the trigger working, not a
  reason to pre-empt it.
- **Upload management UI, retention opt-out, backup of retained exports** —
  §3C carries the tension to the backlog explicitly.
- **A standalone marketing site** — launch item; the front door's pitch is
  the v1 homepage for cold visitors.
- **CLI behaviour changes** — import and detect flags, semantics, and output
  are untouched; the CLI gains nothing and loses nothing this phase.

---

## 7. Checkpoint order — four slices, each visible, each a STOP

1. **The contract and the Go path, proven by curl.** `openapi.yaml` edits
   (§4), both generators run; migration 00009; exported head-sniff;
   stream-to-disk upload with hash + cap + retention; sync sniff rejection;
   single-import guard; background import goroutine; startup sweep;
   auto-detect with `detect_status`. *Visible: `curl -F` of the demo export
   against a fresh compose instance returns 202; polling shows running →
   completed with detect completed; candidates exist with zero CLI commands;
   a second identical upload yields an all-skipped import row and no second
   file; `curl -F` of a PDF returns 400 with `detected_format`; an upload
   killed mid-stream leaves nothing behind — no imports row, no temp file
   (proven with a deliberate client abort); a mid-import serve restart
   leaves a visible failed "interrupted" row.*
   The phase-1 discipline holds: no frontend work before curl proves the
   API. STOP.
2. **The upload loop in the browser.** Route-handler streaming proxy, upload
   island (XHR progress, per-outcome rendering), polling, the
   wait-for-detect beat, landing on candidates. *Visible: on a fresh compose
   instance, selecting the demo export in the browser carries through
   progress → importing → detected → "see your candidates", and the life map
   fills; the same walk succeeds at phone width; an upload aborted mid-flight
   (tab closed) shows a clean try-again with nothing recorded; a wrong file
   shows the rejection with its redirection text inline; `/imports` shows
   every attempt.* STOP.
3. **The front door.** The `/welcome` page per §3E: pitch with demo visuals,
   the three walkthrough branches (drafted, marked unverified until §7's
   device pass), expectation setting, embedded upload, rejection anchors,
   `/` redirect on empty instance, empty-state chaining from life map and
   `/imports`; the small-screen pass over the v1 loop pages (§1.5) and the
   zero-candidates copy. *Visible: a cold visit to `/` on an empty instance
   lands on the pitch and ends at candidates without leaving the page flow;
   every sniffer label routes to a sensible anchor (walked for all
   thirteen); the v1 loop pages walk cleanly at phone width, confirm and
   dismiss included; an import of qualifying-journey-free data renders the
   designed zero-candidates copy, not an empty screen.* STOP.
4. **Close.** Device verification of walkthroughs (Android on the
   maintainer's device; iPhone via the audit's iPhone friend), dates
   stamped — each device pass **starts from an actual WhatsApp message**:
   tap the link in the chat, walk export → upload → candidates entirely on
   the phone, in whatever browser WhatsApp opens; README browser path;
   cold-cache `make test` green including the new store/API tests; goldens
   byte-identical; fixture still 18/1/32; LOG.md. *Visible: the README
   quickstart executed from a clean clone using only the browser after
   `docker compose up`; the WhatsApp-origin walk completed on each verified
   platform; each walkthrough carries a real verified date or an honest
   "unverified" marker.* STOP.

Phase close per the standing rule: not complete until LOG.md exists.

---

## 8. Risks

- **Next streaming forwarding is unverified territory.** Route-handler
  request-body streaming with `duplex: "half"` must be confirmed against the
  local Next 16 docs and tested with an archive-scale file early in
  checkpoint 2; the spool-to-temp-file fallback (§3B) is the named escape
  hatch. This is the phase's main technical unknown.
- **Real archive scale is unmeasured on this path.** The demo is 152 KB; the
  private archive is the only real-scale test and stays private — checkpoint
  2's verification includes a local upload of the real archive (never leaving
  the machine) to observe memory flatness and duration. Numbers from that run
  stay out of committed docs (invariant 13 discipline: only demo numbers are
  public).
- **The detect-after-import beat** (§3D) is a small state machine across two
  async facts; getting the UI honest about "imported but detection failed" is
  fiddlier than it looks. The `detect_status` field keeps it one resource.
- **Google changes export flows unannounced.** The verified-as-of discipline
  contains this; the risk is a walkthrough silently rotting between
  verifications — accepted, mitigated by the visible date (a stale date is
  itself information).
- **Device access gates the close.** The iPhone walkthrough needs the
  audit's iPhone friend; if unavailable, the branch ships marked unverified
  (§3E) and the phase still closes — the marker, not the verification, is
  the hard requirement.
- **Sweep vs CLI race** (§1.2): accepted and documented rather than solved;
  revisit only if it bites in practice.
- **The WhatsApp in-app browser is unaudited territory.** File pickers and
  large uploads inside embedded WebViews are classic breakage; the §7 device
  pass starts from a real WhatsApp message precisely to find out. The escape
  hatch if it misbehaves is an "open in your browser" hint — a copy fix, not
  a redesign — and a laptop remains the documented fallback, never the plan.
- **Small-screen scope creep.** The pass is scoped to the v1 loop pages and
  to layout and touch only; the life map's cartography is not being
  redesigned for phones this phase. Anything deeper the pass uncovers goes
  to the backlog, not into the checkpoint.

## 9. Verification at phase close

- Cold-cache `make test` green; new tests cover: upload handler
  (stream/hash/cap/reject/retain), sweep, single-import guard, auto-detect
  ordering and `detect_status`, `ImportRejection` shape.
- Goldens byte-identical; fixture detection still 18/1/32; archive
  expectations unchanged; demo regression (3/1/0) unchanged.
- The checkpoint 1 curl walk and checkpoint 2 browser walk repeated on a
  fresh compose instance (`down -v` first).
- The device pass performed from a real WhatsApp message on each verified
  platform; the v1 loop pages verified at phone width.
- Real-archive upload exercised locally for memory flatness (private,
  unpublished).
- `git add -A --dry-run` clean of `data/`, uploads, `.env`, archives, and
  anything near 1 MB.
