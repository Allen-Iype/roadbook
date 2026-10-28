# Phase 16 decisions

Written as decisions are made. Three lines each: what was chosen, what was
rejected, what would change our mind.

## 2026-10-28 — Phase 16 is reading load

- **Chosen:** the next phase addresses the maintainer's finding after the
  first live walk — the UI is "mostly words", confusing, unread. Named
  "reading load"; measured by `web/scripts/copy-count.cjs` before and after.
- **Rejected:** folding this into phase 15 (hosting), whose target is a
  zero product-code diff; a full redesign (the Atlas direction is kept).
- **Would change our mind:** nothing; the evidence is the maintainer's own
  walk and the measured baseline (RESEARCH §2).

## 2026-10-28 — Implementation after phase 15 CP3; planning in parallel

- **Chosen:** research, brief and shape work run alongside phase 15 CP3
  (durability); code starts only when CP3 is on main. CP3 landed the same
  day (`edcbeb7`), so the condition is met before CP1 begins.
- **Rejected:** pausing phase 15 for the UI — the live instance accepted
  real sign-ins with no backup chain yet.
- **Would change our mind:** nothing retroactively.

## 2026-10-28 — The work lives on a branch

- **Chosen:** `phase-16-ux`, created by the assistant, every commit the
  maintainer's; `docs/PLAN.md` edited on main; rebase after each main
  commit; merge at accepted checkpoints. The host pulls main only.
- **Rejected:** working on main beside phase 15's checkpoints — the
  dry-run review and the host's pull become harder to reason about.
- **Would change our mind:** phase 15 finishing entirely; then the branch
  is just a branch and merges as usual.

## 2026-10-28 — Atlas stays; treatments within it are explored

- **Chosen:** DESIGN.md binding; treatment options within the token
  system compared at a STOP; one wording amendment proposed (BRIEF §5C).
- **Rejected:** reopening the direction (a dark theme, a new type
  system) — the complaint is information design, not aesthetics.
- **Would change our mind:** a walk showing the aesthetic itself puts
  people off; none has.

## 2026-10-28 — Watched walks join at checkpoint boundaries, not before CP1

- **Chosen:** the maintainer's call after reading the findings: proceed
  on the measured audit and his own input (RESEARCH §9); walks (RESEARCH
  §1) happen on the live instance as they can be arranged and join at
  the next checkpoint boundary; the closing walk is CP4's acceptance.
  This supersedes the planning STOP's "walks within a week or two before
  design starts".
- **Rejected:** holding the phase for the walks — the research already
  located the words, and the maintainer has used the product.
- **Would change our mind:** a walk before CP2 contradicting §9.1; then
  CP2's shape is re-decided at its STOP.

## 2026-10-28 — Shape review as code on the branch, not static mockups

- **Chosen:** the maintainer's call ("start implementing and see the next
  UI"): the mockup bucket's questions are answered by working code on the
  branch, reviewed on a phone at each checkpoint's STOP against a scratch
  demo stack. The gate's purpose — nothing reaches main unreviewed — is
  unchanged.
- **Rejected:** `docs/phase-16/mockups/` as phases 6 and 9 did — the
  research fixed the treatments' direction, and the open questions (a
  disclosure's feel, a popover on a phone) are better judged in hand.
- **Would change our mind:** the CP1 STOP rejecting the shape rather than
  its details — then CP2 returns to the mockup gate before its code
  (BRIEF §5A fallback).

## 2026-10-28 — The map pages are out of scope

- **Chosen:** the life map, `/adventures` and `/candidates` are left as
  they are except for the plate-number caption and the teaching control
  where a load-bearing term appears; their own improvements go to the
  next iteration (maintainer, RESEARCH §9.3).
- **Rejected:** the research's candidate-card treatment and the
  `/candidates` copy items beyond folding the run line.
- **Would change our mind:** a walker stopping on `/candidates` in a way
  the bulk-triage tools of phase 11 do not explain.

## 2026-10-28 — The adventure page is the trip, not the reconstruction

- **Chosen:** the cover leads with the trip's figures (phase 14's summary
  block becomes the hero); the drawing's account moves into one
  disclosure; the floor — legend, provenance bar, divergence flag,
  truncation marks — stays visible (BRIEF §1, §4 CP1).
- **Rejected:** deleting the reconstruction figures (invariant 13 — they
  are what the CLI prints); a separate page for them (a new navigation
  element).
- **Would change our mind:** nothing about the split; the disclosure's
  copy gets its own diet if a walker opens it and is confused.

## 2026-10-28 — Gate 1 passed

- **Chosen:** BRIEF.md approved as written; all §5 recommendations stand
  (disclosure for the drawing's account, legend wording to be decided at
  the CP1 STOP, the `Term` popover control, the folded front door). CP1
  begins in a fresh session from the brief.
- **Rejected:** continuing straight into CP1 in the planning session —
  the brief is the source CP1 builds from, and a cold read of it is the
  check that it is complete.
- **Would change our mind:** nothing; a process choice.
