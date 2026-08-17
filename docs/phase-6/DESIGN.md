# Phase 6 — design decisions (Stage B review outcome)

Decided at the Stage B mockup review, 2026-08-08. The mockups under
`docs/phase-6/mockups/` are the visual reference; where this document and a
mockup disagree, this document wins. Stage C turns this into an
implementation brief; nothing here is code yet.

## 1. Direction: A — "Atlas plate"

The printed road atlas and the rally roadbook as ancestors. Light paper
ground, cartographic route inks, serif display over quiet sans body with
mono data. The full token system is `mockups/tokens-a.html`; the tokens in
§5 below are the binding subset.

Direction B ("Night expedition") is not discarded: its token sheet remains
in the mockups, and an optional dark theme is parked in the backlog (§7).
Nothing in Stage C may depend on there being exactly one theme, but nothing
is built for theming now either — no theme machinery, one stylesheet.

## 2. Honesty channel at life-map zoom: T2 — zoom-graduated emphasis

- Kind is encoded at every zoom, always (invariant 8).
- At overview (life-map) zoom, observed and routed merge into one "known
  ground" ink (#3A3A34). Unknown stays dashed grey; air stays a dotted
  arc. The two distinctions that survive at country scale — known vs
  unknown, ground vs air — are exactly the ones a reader can still see.
- The observed/routed split returns as zoom increases and is fully split
  at adventure-detail zoom. Implementation shape (Stage C): MapLibre
  zoom-interpolated paint properties on the existing layer-per-class
  rendering; no new geometry, no precompute.
- The boundary condition that makes T2 honest: the overview never renders
  unknown or air as confident ground. A regression-proof statement of it:
  at no zoom does an unknown leg share the known-ground style.
- The adventure detail map keeps the full four-way encoding at all times —
  T2 applies to the life map's zoom range only.

## 3. Day slices: "Day", not "Stage"

Day headings read "Day 3 · Sunday 24 May · 80 km". Plain and instantly
understood; the rally-roadbook register stays in the product's name and
the atlas plate framing, not in the labels. (Mockup pages show "Stage" on
direction A — superseded by this decision.)

## 4. Score: retired from the confirmed adventure's cover

A confirmed adventure's page carries no score. Ranking is triage
machinery: it earns its keep in the candidates table and in the confirm
cell's breakdown, both of which keep it. The cover keeps the honest
figures only — distance with provenance bar, dates, days, fixes,
countries, and the divergence flag when present.

## 5. Life-map list rail: summoned

The life map is map-maximal: a "List (n)" control summons the adventure
list as an overlay panel; it is not visible by default. Requirements that
keep it first-class rather than a consolation (RESEARCH §6):

- Keyboard: the control is focusable, the panel is a normal focus target,
  every adventure in it is a link. Nothing requires hovering the map.
- Screen readers: the list is the accessible enumeration of adventures;
  the map is `aria-hidden` decoration apart from its route links.
- Small screens: the summoned list is the primary navigation.

## 6. Binding tokens (from mockups/tokens-a.html)

Leg-kind inks, reserved first — nothing else on any surface uses these
hues at line weight, and kind is never hue alone:

| kind | ink | non-color channel |
|---|---|---|
| observed | #A81E22 | solid, widest, uncased |
| routed | #2A5DA8 | solid over paper casing |
| unknown | #8A8375 | dashed |
| air | #3F7069 | round-dotted arc, plane mark |
| known ground (T2 overview only) | #3A3A34 | solid |

Ground: paper #F5F2E8 · land #EFECE2 · sea #D9E2E4 · ink #26251F ·
secondary #6E6A5E · rule #C9C3B2. Flag amber #B97F10 marks honesty
warnings only (divergence, far-from-route photos); it never decorates.

*Amendment (2026-08-17, phase 9 CP2, maintainer-approved):* flag amber is
now **#8A5C0B**. The original measured 3.07:1 on paper and 2.91:1 on land —
under the 3:1 non-text bar on the divergence box's own background. The
replacement measures 5.18:1 / 4.91:1 (`node web/scripts/contrast.mjs`),
passing AA as glyph and as text on both grounds. Decision record:
`docs/phase-9/DECISIONS.md`.

Type roles: characterful serif display (adventure names, plate labels),
quiet sans body, mono for every figure (distances, coordinates, times).
Faces are a Stage C decision; mockups used system stacks, candidates are
named on the token sheet.

Signature: the plate margin — maps framed with their legend, scale bar,
and plate label set as printed marginalia. The legend is a permanent
fixture (treatment T3) with fixed wording everywhere:
"Observed — recorded fixes · Routed — inferred along roads · Unknown —
straight line, nothing inferred · Air — great-circle arc".

Shared invention kept: the provenance bar — headline distances carry a
hairline stacked bar of their kind-composition.

Standing recommendation confirmed: no home markers on the life map.

## 7. Parked

- **Optional themes** (maintainer, at review): direction B exists as a
  complete token system; an operator-selectable dark theme is a future
  option. Backlog, no trigger set, no theming machinery built now.

## 8. Superseded mockup details

The mockups predate three of the five decisions; when reading them:
"Stage" headings read as "Day" (§3), the cover score line is gone (§4),
and direction A's visible index rail becomes the summoned list (§5). The
mockups are not being regenerated — they are review artifacts, and this
document records the deltas.
