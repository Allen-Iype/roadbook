# Phase 11 — decision log

Three lines each: what was chosen, what was rejected, what would change our mind.
Written as decisions are made, not reconstructed.

## 2026-08-24 — Charter: ingestion ahead of the front gate

- **Chosen:** photos-as-import-source as phase 11; front gate shifts to 12,
  accounts/tenancy to 13. Decided by the maintainer at the charter STOP after
  hearing the trade-offs.
- **Rejected:** building the front gate first — it would recruit strangers into a
  funnel the pilot proved closed to the iPhone majority (four of six pilot
  instances at zero imports because no Timeline data exists to export).
- **Would change our mind:** nothing retroactively; the roadmap's own
  resequencing clause pre-decided this and fired on pilot evidence stronger than
  the planned audit.

## 2026-08-24 — Binding scope input

- **Chosen:** friend-5's two report items enter the brief as named scope
  candidates (§6): bulk triage actions and per-mode distance breakdown.
- **Rejected:** treating the report as backlog-only; the maintainer's explicit
  instruction was that this brief must argue them.
- **Would change our mind:** n/a — instruction, not inference.

## 2026-08-24 — Gate 1 review input: three photo futures (maintainer)

- **Chosen:** the maintainer's three raised scenarios sorted per the standing
  buckets — proof-of-location is in-system and foundational (§6.3.1);
  road-condition evidence is the parked community direction, no design (§6.3.2);
  a points-not-routes view is mockup-gated new IA for a future phase (§6.3.3).
  One recommendation amended: ingestion keeps **photo records** (hash +
  metadata + thumbnail where decodable), not bare fixes (§4D) — all three
  futures need the photo to survive as a referenceable thing at a point.
- **Rejected:** fixes-only retention (forecloses all three futures, forcing a
  full re-upload later); full-original retention (the photo-host storage and
  privacy posture phase 4 already rejected); designing anything for the parked
  direction beyond not destroying evidence.
- **Would change our mind:** on records — storage evidence that thumbnails at
  camera-roll scale strain real instances (then metadata-only records keep the
  seam at near-zero cost); on the parked direction — only its own charter
  change, never this phase.

## 2026-08-24 — CP1: home-evidence precedence is fallback-on-failure

- **Chosen:** synthetic home evidence is consulted when INFERRED_HOME
  *derivation yields zero bases*, not merely when zero INFERRED_HOME visits
  exist. A user with a handful of Timeline home visits (below MinVisits) plus
  a camera roll still gets a home.
- **Rejected:** presence-based precedence — it would strand exactly the mixed
  thin-Timeline case this phase serves, for no honesty gain.
- **Would change our mind:** a real dataset where weak INFERRED_HOME evidence
  should have beaten strong synthetic evidence. Byte-identity is unaffected
  either way: without photo fixes there is no synthetic evidence to fall to.

## 2026-08-24 — CP1: HomeMinDays day-spread guard on synthetic bases

- **Chosen:** a synthetic cluster must recur across ≥ HOME_MIN_DAYS (8)
  distinct civil days, on top of the MinVisits count — recurrence alone
  cannot tell a residence from a week's hotel.
- **Rejected:** count-only qualification (a 12-stay hotel week would become a
  "home" and erase its own trip); applying the guard to INFERRED_HOME
  evidence too (byte-identity — Google already asserted home there).
- **Would change our mind:** corpus or real-data evidence of a true home
  lost to the guard (a weekend-only residence, e.g.); the parameter is the
  remedy, not code.

## 2026-08-24 — CP1 deviation from brief §4B: no on/off parameter for
## photo-fix observation inclusion

- **Chosen:** photo-sourced fixes always join the observation stream —
  inclusion is scoped by the source class itself, with no boolean parameter.
  The brief promised "behind a named parameter, default on".
- **Rejected:** the boolean — a dead knob nobody would turn off; invariant 3
  governs thresholds, and this is a structural rule made safe by scoping
  (data without PHOTO rows is byte-identical by construction, proven by the
  fixture/archive/demo regressions running green with synthesis live).
- **Would change our mind:** evidence that direct photo observations harm
  detection somewhere synthesis alone would not — then the knob earns its
  existence. Flagged for the maintainer at the CP1 STOP.

## 2026-08-24 — Gate 1 PASSED: brief accepted as amended

- **Chosen:** all §9 recommendations stand — A/B (photo records in their own
  table, fixes in `raw_positions` `Source="PHOTO"` with a back-reference to the
  record; detection fed via synthesis plus photo-scoped flatten inclusion),
  C (picker + CLI directory now, Takeout zip follow-up), D (photo records:
  hash + metadata + thumbnail where decodable, originals discarded), bulk
  triage in scope as CP4 with the auto-confirm PRODUCT amendment declined,
  per-mode breakdown in CP4, synthesis defaults as starting values, PLAN
  resequencing as prepared, §6.3 three-futures sorting as recorded.
- **Rejected (the maintainer's one raised concern, resolved in discussion):**
  a separate table for photo *fixes* — it would fork every pipeline consumer
  into a permanent two-table UNION while adding no queryable fact the photo
  record doesn't hold; invariant 4's one-seam rule is the deciding reason.
  The separate-table instinct is satisfied by the §4D photo-record table.
- **Would change our mind:** a real consumer that needs fix-level photo
  attributes the record link cannot serve — none is known or foreseen.
