# Copy pass — log

Closed 2026-11-03, branch `copy-pass`, twelve commits of one screen each
plus the docs. The brief: non-technical testers said Roadbook was too
technical and they could not work out how to move through it. The rule
and the word list are in `words.md`, the approved text screen by screen
in `copy.md`, the audit of every string in `copy-audit.md`, and the
decisions as they were made in `DECISIONS.md`.

## What the pass did

Every user-visible string in the web app, the landing, the two waitlist
replies and the messages the Go service sends to the browser was
rewritten under one rule: say what the person gets, not what Roadbook
does, in words a traveller would use. Nothing internal was renamed —
routes, the API, `openapi.yaml`, database columns, test identifiers and
the engineering documents keep their names.

The words that changed everywhere: *candidate* and *adventure* → **trip**
(to review / kept); *confirm* and *dismiss* → **keep** and **leave out**;
*import(s)* → **upload(s)**; *detection* → **finding your trips**;
*observation / position / fix* → **point**; *seat* → **spot**;
*instance* → **this Roadbook**; *provider* → **Google**; *revoke* →
**stop sharing**; *overlay* → **for a story**; *sidecar* → **info file**;
and the printed legend under every map → one **"What the lines mean"**
control. The four line styles are drawn exactly as before; the labels
inside the control are unchanged. The nav reads Trips · Trips to review ·
Uploads.

Three product decisions rode along, each the maintainer's and each
recorded in `words.md`: *plate* survives only as the printed numeral on
covers and map margins; the photos door left the start page (the start
page is for someone with a Timeline file; photos stay on Uploads); and
operator-facing hints (`roadbook serve`, "the server log has details")
appear only on an instance without sign-in.

## Measured

`web/scripts/copy-count.cjs` against the development server over the
scratch demo API, the same way phase 16 measured. "Before" is phase 16's
closing figure for each page (its DECISIONS tables); "terms" counts the
words on the tool's vocabulary list.

| page | before, phone (words / sentences / titles / terms) | after, phone | after, desktop |
|---|---|---|---|
| landing | 390 / 47 / 0 / 10 | 377 / 45 / 0 / 7 | 382 / 46 / 0 / 8 |
| welcome | 257 / 27 / 0 / 12 | 138 / 16 / 0 / 3 | 138 / 16 / 0 / 3 |
| home (life map) | — | 19 / 3 / 3 / 0 | 19 / 3 / 3 / 0 |
| adventures (Trips) | 99 | 78 / 10 / 3 / 3 | 78 / 10 / 3 / 3 |
| candidates (Trips to review) | 91 | 97 / 10 / 0 / 7 | 104 / 10 / 0 / 6 |
| adventure 1 | 355 / 42 / 6 / 11 | 310 / 39 / 15 / 6 | 310 / 39 / 15 / 6 |
| adventure 2 | 298 / 36 / 6 / 11 | 251 / 32 / 9 / 5 | 251 / 32 / 9 / 5 |
| adventure 3 | 330 / 42 / 6 / 12 | 286 / 38 / 11 / 7 | 286 / 38 / 11 / 7 |
| imports (Uploads) | 95 / 11 / 0 / 6 | 97 / 12 / 0 / 3 | 113 / 12 / 0 / 6 |

Vocabulary hits across all pages, desktop, after: plate 12 (the printed
numerals) · stops 12 · timeline 7 · stop 4 · export 3 · score 3 · source
2 · atlas 1. Every one of those is a word kept on purpose (`words.md`,
"kept on purpose"). The "titles" column on the trip pages went up because
the map's own controls (MapLibre's zoom buttons and attribution) carry
title attributes; no teaching text sits in a title any more.

Trips to review grew by six words: the intro is one sentence longer than
the old one-line term, and the keyboard hint is a sentence. The landing
barely moved because phase 16 CP2b had already rewritten it; its layout
cut is the next step (below).

## What broke, and why each fix took the form it did

- **The preview proxy reset connections under load.** To show the sign-in
  page on a stack that runs without sign-in, a small pass-through in
  front of the scratch API faked one session call. Under the Playwright
  suite it reset connections and failed a third of the tests. The fix was
  to use it only for the sign-in look and point the dev server straight at
  the API for everything else — a tool for one purpose, not a layer to
  keep.
- **A stuck "an import is already running" lock** on the scratch API, left
  by an aborted test upload before the pass. Rebuilding the API container
  cleared it and brought the new Go messages up at the same time.
- **Next 16 allows one dev server per checkout**, so a second instance on
  another port for the sign-in preview refused to start. Hence the proxy.
- **SWC drops the space before an inline element** (the phase 7 trap, again):
  the flag line's "65% — “How this route…" rendered without its space
  until the dash moved into a `{" — "}` expression.
- **Tests pinned sentences, not meaning.** Eleven e2e and unit pins
  changed with their screens, each in the same commit; the legend spec
  now opens the control first (`openLegend` in `e2e/helpers.ts`). The Go
  tests pin rejection *kinds*, not sentences, so every Go message changed
  without a Go test changing — the right division, kept.
- **The photos door's two anchors** (`photos`, `never-enabled`) were targets
  of a rejection link. Removing the section meant retiring them in
  `rejection-anchors.ts` and pointing a photo-in-the-Timeline-picker at
  "What the right file looks like" instead, with the Go message naming
  the Uploads page.

## Reported, not done

- **The landing is still cluttered** (eight blocks, two pictures of a
  trip, two "Make mine" buttons). A layout cut — one picture, one button,
  one-line steps and promises, the waitlist box reduced to a field — is
  the next step, mockup first on the maintainer's phone, then the
  five-second test with friends.
- **"One line"**: the maintainer's wish that a trip read as one route from
  A to B. Drawing the four kinds in one style is a rendering change; filling
  the dashed gaps with real roads is the routing work (OSRM on the host,
  a batch after each upload, coverage decisions). Either is a PRODUCT.md
  and CLAUDE.md amendment (invariants 5, 6, 8) before code. Its brief waits
  until the landing cut lands.
- **The review page's shape** (cards with a score) and **the Uploads table**
  (eleven columns) are unchanged; only their words are.

For PLAN.md (the maintainer edits PLAN on main): two backlog entries,
"Landing cut" and "One line", with the text above; and a note on the
phase 16 line that the atlas-vocabulary teaching it introduced was
superseded by this pass's word list.
