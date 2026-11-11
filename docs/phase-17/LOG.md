# Phase 17 — log: fixes from the first testers

What the phase does, what broke, and why each fix took the form it did.
BRIEF.md and DECISIONS.md are the other two artifacts; this closes the set.

## What the phase does

The first testers raised four issues. This phase fixes all four, plus a
fifth found while fixing the first. All five are tracked as GitHub issues
under the "Phase 17" milestone and closed by one pull request.

- **#4 — an iPhone Timeline export is read.** Google's iPhone export is a
  different shape from the Android one:
  - a JSON array at the root, not an object;
  - positions written `geo:lat,lon`;
  - numbers as strings;
  - path points timed in minutes from their segment's start;
  - its own spellings ("Inferred Home", "in passenger vehicle").

  The parser (`internal/timeline/parse.go`) now reads both shapes. The
  first token chooses the path, and the spellings are normalised in the
  parser, so detection is unchanged (invariant 4). The format is recorded
  as `phone-timeline-ios`.
- **#7 — needles on iPhone maps.** Once the file read, some trip maps
  drew lines out to a far place and straight back. A second rejection
  pass in journey assembly sets those points aside from the drawn route
  by their shape (`internal/journey/journey.go`, three named parameters,
  rows untouched). Detection is unchanged.
- **#1 — newest first.** "Trips to review" opens on the latest trip. The
  score order is unchanged, and so is the API's order.
- **#2 — the plate numeral is gone.** "PLATE 2" / "PLATE II" left every
  surface it was on:
  - the covers and the trip page;
  - the map margin;
  - the exported image and overlay;
  - the landing's sample covers.

  The dates carry the order.
- **#3 — a kept trip's name and the owner's notes are edited on its
  page.** This adds migration 15 (`decisions.notes`) and a new
  `PATCH /candidates/{id}/decision` that changes the name and notes and
  nothing else. Notes are owner-only: the shared view never receives them
  and never renders the control. A re-decide never blanks them. Backup
  and restore carry them.

By checkpoint:

- **STOP 0 — brief** (943a1ef). Places was renumbered from phase 17 to
  phase 18.
- **STOP 1 — #4** (54c9d95).
- **STOP 1b — #7** (404c9bb). Added to the phase after STOP 1, with a
  brief section before the code.
- **STOP 2 — #1 and #2** (fb583c3). Web only, no Go diff.
- **STOP 3 — #3** (ba5f55b).

## What was verified

- **`make test` with the database suites at every STOP.** The new tests
  were confirmed as run, not skipped. All detection pins held:
  - fixture 18/1/32;
  - archive 66/4/238;
  - demo 3/1/0;
  - both journey goldens byte-identical, including after the needle pass.
- **The iPhone path is held to the reference detector on a real tester
  export.** That export was converted independently to the Android shape
  and run through `prototype/detect_fixture.py`. The result is 4 home
  bases and 60 candidates, every field equal to Go's
  (`TestIOSRegression`; private data, skips when absent). The demo is also
  written in both shapes (`testdata/demo/demo-ios.json`), and parity
  between them is tested.
- **Needles are counted on every private dataset**, before and after the
  pass (`internal/journey/needles_test.go`):

  | dataset | trips | needles before | after |
  |---|---|---|---|
  | tester's iPhone export | 60 | 67 (7 trips) | 4 |
  | Android fixture | 18 | 0 | 0 |
  | maintainer's archive | 66 | 3 | 0 |

  The same four leftovers were confirmed through the running API on a
  scratch stack.
- **Notes, through the store, the API and backup:**
  - the store edits a kept trip and refuses a dismissed one or an unknown
    id;
  - a re-decide keeps the notes;
  - another account cannot edit the row (`cross_tenant_db_test.go`);
  - the API counts its limits in characters, not bytes (120 two-byte
    characters pass, 121 fail);
  - the shared view's body has no notes;
  - the backup round trip restores them.
- **Web checks:**
  - tsc and lint clean (0 errors);
  - vitest 131;
  - Playwright 124 passed and 26 skipped across 390, 768 and 1280,
    against a scratch compose stack in the demo state;
  - the new writing spec `edit-trip.spec.ts` was repeated 20 times
    serially and passed every time.
- **Looked at on the scratch stack:** the review list and the Trips page
  ordered newest first, and the trip page without the numeral. The edit
  flow at 390 and 1280: reading, editing, saved.

## What broke, and the shape of each fix

1. **The iPhone file was refused with a misleading message.** The parser
   assumed an object at the root, so the tester saw "not valid JSON" for
   a valid file. The fix reads the first token and takes the array path
   for `[`. An array that holds no Timeline segments is still refused with
   its existing kind, so the My Activity export's refusal is unchanged.
2. **The iPhone spellings would have produced one endless trip.**
   Detection finds home by the literal `INFERRED_HOME`. Passed through as
   "Inferred Home", an iPhone user would have had no home base, and every
   observation would have counted as away. The fix normalises
   (upper-case, words joined by `_`) in the parser for both shapes. That
   is idempotent on Android input, which is why the goldens did not move.
3. **The needles were in the file, not in the parser.** The first
   suspicion was the new offset timing. It was ruled out: 829 of the
   jumps sit inside a single path segment whose offsets only increase. The
   existing teleport filter missed them because their implied speeds
   (24–471 km/h) are under its 900 km/h limit. Lowering that limit would
   have cut real driving and flights, and exact-coordinate recurrence did
   not separate the needles from home points. So the rule is by shape: a
   cluster far from both neighbours, with the neighbours close together,
   inside 120 minutes. The window was measured, not chosen: 60 minutes
   left 13 needles, and 180 began removing hour-long clusters from the
   archive.
4. **A plain `git push` from the branch would not have gone to its own
   branch on GitHub.** `phase-17` had been created tracking the local
   `main`, so `git status` read "ahead of main". The upstream was unset
   and the branch pushed with `-u`. No commit was affected.
5. **The tester's export was loaded into a local scratch stack before
   the maintainer was told.** It was on loopback only and never left the
   machine. The stack was removed with `down -v` on the maintainer's word
   before STOP 2. Every later check ran on the fictional demo. The rule
   kept from it: say where real data is going before it goes there.
6. **The e2e that writes needed three fixes before it was stable.**
   - `app/loading.tsx`'s own `h1` sits hidden in the DOM while a reloaded
     page streams in, so the spec reads `article h1`.
   - A streamed node first arrives in a hidden container with zero
     height, so the line-break check polls until it has laid out.
   - A run stopped from outside left trip 2 edited and two share links
     behind. They were reset through the API, and the spec now puts the
     trip back in a `finally`.

   The same interrupted run showed four tests "failing" after 14 minutes
   each. The stack answered in milliseconds right after, and the rerun was
   green; the evidence points to the machine pausing, not to the code.
7. **A name could have been saved that the edit form would refuse.**
   With a limit only on the edit, a 130-character name kept through
   triage could not be saved unchanged. The 120 limit now applies to every
   write of a name: the edit, the single confirm and the bulk confirm.

## Carried out of the phase

- **The four needles left on the tester's trips** are out-and-backs
  longer than two hours. They are left drawn on purpose (#7, DECISIONS).
- **#8, far trips without a long stop.** Two of 28 trips Google itself
  grouped at 100 km or more were missed. Backlog; needs a brief.
- **#9, snapped lines.** Recorded stretches drawn along the road by OSM
  map matching. Backlog; changes invariant 6, needs a brief and its own
  line kind.
- **The trip page in a browser without WebGL** shows the page-level error
  instead of the trip without its map. Seen in a headless screenshot, and
  it predates this phase. To be filed as a backlog issue if the
  maintainer agrees.
- **Migration 15's down step** (one `DROP COLUMN`) was not exercised on
  its own.
- **Questions for the tester:**
  - how long the all-India trip lasted, to check how a long journey with
    thin recording reads;
  - whether trips under 100 km matter to them.
- **Host deploy after the merge:**
  - app and demo stacks rebuilt (the API applies migration 15 on start);
  - landing redeployed for the covers;
  - checked from outside;
  - a comment on #4 asking the tester to try again.
- **Scratch stacks:** `roadbook-p17` (3017/8097) served STOP 2 and STOP 3
  in the demo state; `roadbook-p16` is still up from phase 16. Tear both
  down with `-v`.
- **Phase 18, Places:** brief before any code.
