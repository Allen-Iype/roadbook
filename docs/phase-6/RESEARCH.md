# Phase 6 research — the life map and the adventure narrative

Stage A of the phase 6 design process: the reference survey and information
inventory that ground the Stage B mockups. Nothing here is a decision;
decisions happen at the Stage B review and land in DESIGN.md and
DECISIONS.md. Product references were checked against their public
materials in August 2026.

## 0. The anchor vision, pinned

Stated by the maintainer at charter, refined in discussion:

- **Home is one big map** — the union of every *confirmed* adventure's
  route, bounds-fit to wherever the adventures are (India for the
  maintainer's data, Iceland for the demo; nothing regional assumed —
  invariant 9). Curation at life scale: commutes and dismissed candidates
  never appear, because the map draws adventures, not coverage.
- **The map is the navigation.** Click a route to enter its adventure.
  Any list is secondary.
- **An adventure is a narrative** — the detail page presents the journey
  as map plus day-sliced story: legs, stops, and photos organised by the
  civil days of the trip, each day selectable.
- **Aesthetic direction is open** — two genuinely different directions go
  to mockup ("surprise me").
- Triage (the current candidate table) and imports demote to entry
  points; the confirm flow itself is unchanged.

The hard design problem, named up front: **the honesty channel at country
zoom** (§3). Invariant 8 — no geometry ever hides its leg kind — applies
to the life map too, and the life map is exactly where per-leg texture
could collapse into noise. The mockups must attack this, not dodge it.

## 1. Information inventory — what the system can already show

Everything below exists behind the API today; the design chooses what
leads, what supports, and what stays a click away. Nothing new needs to be
computed for the anchor vision.

**Per confirmed adventure** (`/candidates/{id}` + `/journey` +
`/photos`): name, span with truncation flags, days, destination and its
distance, countries crossed in journey order, score with per-component
breakdown, mode mix; the journey as ordered legs — observed (with point
geometry), routed road (cached polyline), unknown (chord), air (arc) —
each with timestamps and distances; distance totals by class (observed /
routed / unknown / air) and the provenance fraction; Google's own distance
and the ground-to-ground divergence with its flag; stops with location,
arrival, dwell; photos placed on legs or stops with thumbnails, taken
times, position provenance, and the far-from-route flag; unplaced photos,
honestly listed.

**Global:** all candidates of the latest run with decisions attached;
orphaned decisions; import history with status, counters, and detected
format; detection-run parameters.

**Derivable client-side, no backend work:** day slices (legs, stops, and
photos already carry instants in the journey's own UTC offset — cutting
an adventure into civil days is arithmetic); per-day distances; the
life-map union (the home page can fetch the confirmed adventures'
journeys — eleven requests server-side at current scale, or one new
aggregate endpoint if composition wants it; see §7).

**Not exposed today:** home-base locations (derived per run, not served).
Relevant only if the life map wants a "home" marker — §7 argues it should
not.

## 2. References — what transfers, what stays out

### Polarsteps (trip storytelling; the closest product reference)

The trip page pairs a map with a step timeline — scrollable steps (name,
dates, photos, notes) synced to map focus — and the product's emotional
payoff is the auto-generated Travel Book: itinerary, photos, and notes
composed into a printed keepsake.
([product](https://www.polarsteps.com/),
[travel book](https://support.polarsteps.com/hc/en-us/articles/24267055732114-How-do-I-create-a-Travel-Book-of-my-trip))

*Transfers:* the step-timeline-synced-to-map pattern is the skeleton for
our day-sliced detail page (our "steps" are stops and days, discovered
from data rather than typed in); trip cover framing (name, date range,
one hero stat, one photo) for adventure cards on the life map; the
print-keepsake instinct validates the backlog's poster view and pulls
toward the atlas aesthetic (§5A).
*Stays out:* the social layer entirely; the undifferentiated route line —
Polarsteps draws one confident line whatever the evidence, which is
precisely what invariants 5/8 forbid. Our line *is* the differentiator.

### Rally roadbooks (the product's own name, taken seriously)

A roadbook in rally navigation is a stage-by-stage document: numbered
stages, distances between instructions, tulip diagrams. The vocabulary
maps directly: adventures have days (stages), legs with distances, stops
as waypoints. *Transfers:* stage-based day navigation ("Day 3 · 214 km ·
2 stops"), the discipline of distance-annotated segments, and — for the
atlas direction — the aesthetic of a navigation document rather than a
dashboard. *Stays out:* literal tulip diagrams (charming, but decoration
for us — our reader navigates memory, not a route ahead).

### Komoot (the field-tested categorical line channel)

Komoot renders route lines in categorical styles — solid for paved,
dashed borders for loose surfaces, distinct colors per way type — with a
persistent legend, and users read it while riding.
([map legend](https://support.komoot.com/hc/en-us/articles/360024441552-Legend-of-the-komoot-Map),
[tour characteristics](https://www.komoot.com/tour-characteristics))
*Transfers:* proof that a texture-plus-legend channel on route lines is
readable in the field, not just in theory — our observed/routed/unknown/
air encoding is the same class of channel with fewer categories; also the
detail-page composition of map plus segment breakdown. *Stays out:*
planning features, elevation-first framing (elevation is our backlog, not
our lead).

### Strava (map-and-stats composition; hover synchronisation)

The activity page composes hero map, stat block, and analysis charts with
hover-sync between chart position and map position. *Transfers:* the
synchronisation idea — hovering a day, stop, or photo in the narrative
highlights its geometry on the map; restrained stat hierarchy (one hero
number, quiet supporting figures — ours is honest distance, with the
provenance fraction attached). *Stays out:* the fitness-dashboard
aesthetic and any leaderboard/social framing; speed and effort are not
our subject.

### Google Timeline (the anti-reference)

The source of our data, and the design we exist to not be: a day browser
over everything, one blue line whatever the confidence, coverage without
curation. Useful mainly as the contrast that defines us; its one
transferable element is the visit/activity chip vocabulary users already
recognise from it.

### Dawarich (already mined; §2 rejections final)

Phase 2 took its map *mechanics* (GeoJSON layout, layer ordering, bounds
fitting). Its presentation layer — timeline panels, calendars, heatmaps,
fog of war — remains rejected by the comparison doc, and nothing in this
phase re-opens that. The life map is not fog of war precisely because it
draws confirmed adventures only.

## 3. The honesty channel at life-map scale

The problem: at z4–z5 (country scale), eleven adventures' routes are thin
lines a few hundred pixels long. Per-leg texture switching every few
kilometres cannot read at that size; but flattening each adventure to one
confident line is the exact lie the product refuses.

Candidate treatments for Stage B to draw (not decide here):

- **T1 — the channel survives, quietly.** The same encoding at every
  zoom: solid observed, solid-but-distinct routed, dashed unknown, arcs
  for air. At country zoom the dashes and arcs remain visible (air arcs
  especially — flights are long and unmissable); the observed/routed
  distinction admittedly compresses. Komoot proves texture reads at
  planning zooms; the question is z4.
- **T2 — zoom-graduated emphasis.** Kind is always encoded, but the
  overview leans on the two distinctions that stay legible at scale
  (ground vs air; known vs unknown), and the observed-vs-routed
  distinction gains contrast as zoom increases. Never absent, louder as
  you approach — a defensible reading of invariant 8 if and only if the
  overview never renders unknown as confident.
- **T3 — the legend as a permanent fixture.** Whatever T1/T2 choice,
  phase 4's precedent (legend entries in words) scales up: the life map
  carries a quiet persistent legend, so the channel is always decodable.

The Stage B mockups will render T1 and T2 on the real demo geometry (a
dense route, a sparse one, and flights — the demo was built to exercise
exactly this) at both overview and detail zooms.

## 4. Basemap facts that constrain the aesthetic

- OpenFreeMap serves six styles: Liberty (current default), Bright,
  Positron, Dark, Fiord, 3D. **Liberty is the actively maintained one**;
  Positron is clean and label-sparse (good atlas base) but upstream-
  abandoned; Dark and Fiord are explicitly incomplete.
  ([styles repo](https://github.com/hyperknot/openfreemap-styles))
- A custom style is a JSON document editable in Maputnik and servable
  from anywhere — `ROADBOOK_MAP_STYLE` already accepts any style URL, so
  a Roadbook-tuned style (muted base, our route palette reserved) is a
  self-host-friendly option, not a new dependency.
- Whatever the direction, the basemap's job is to recede: the route
  lines own the strongest visual channel on the page (this is invariant
  8 expressed as a styling budget — nothing on the basemap may compete
  with the leg-kind encoding).

## 5. Two aesthetic directions for Stage B

Named here so the mockups have a thesis each; both keep the honesty
channel loudest, both are built from the demo dataset only.

**A — "Atlas plate."** The printed road atlas and the rally roadbook as
ancestors. Light, terrain-muted base (Positron-class or custom); routes
as inked itineraries; editorial typography (a characterful serif display
for adventure names, disciplined sans for data); days as numbered stages;
the page reads as plates in a travel atlas of your own life. Risk to
embrace deliberately: print-flavoured restraint on an interactive
surface. Natural kinship with the future poster view.

**B — "Night expedition."** The map room at night. Dark base
(custom-tuned, given upstream dark styles are incomplete); routes as
luminous traces; grotesk/mono typography; stat blocks as quiet
instruments; photos as lit windows on the dark field. Risk: dark
cartography makes the unknown-gap grey harder to keep honest — the
mockup must prove the channel, not assume it.

Per the design-process discipline: each direction commits to a compact
token system (palette with the leg-kind colors reserved first, two-role
type pairing, one signature element) before any implementation CSS, and
the life map itself is the signature in both — the directions differ in
how it is framed, not what it is.

## 6. Information architecture to mock

- **Home = the life map.** Full-viewport map, bounds-fit to confirmed
  adventures. Hovering (or tapping) a route raises that adventure's
  card — name, dates, one hero stat, one photo. Click enters the
  adventure. A quiet corner header carries the product name and the two
  workbench entries (Candidates — the current triage table, with its
  undecided count as the one attention signal; Imports). A compact
  adventure list exists as an overlay or rail — map-as-navigation is the
  primary path, but keyboards, screen readers, and small screens get a
  first-class list, not a consolation.
- **Empty states as invitations** (a self-hoster's first run): no data →
  point at the import quickstart; data but no confirmations → point at
  triage with the candidate count. The demo dataset makes both states
  screenshot-able.
- **Adventure detail = cover + map + day narrative.** Cover block (name,
  dates, countries, honest distance with provenance fraction, score
  quietly). Map beside/above a vertical narrative of days: each day a
  stage heading (Day n · date · km · stops) with its stops, photos, and
  leg summaries beneath; selecting a day highlights its geometry on the
  map (the Strava sync pattern). Unplaced photos and the divergence
  note keep their honest, quiet placements. Truncation flags render as
  words, phase-1 style.
- **Unchanged:** the confirm/dismiss cell and its score breakdown; the
  imports page; the adventure map's existing leg rendering at detail
  zoom.

## 7. Backend gaps the design may (and may not) create

- **Life-map data:** either the home page server component fans out over
  the confirmed candidates' journey endpoints (eleven requests,
  server-side, at a scale the charter calls "tens" — acceptable and
  zero new API), or one `GET /adventures/geometry` aggregate lands in
  `openapi.yaml`. Decide in Stage C by measuring the fan-out, not by
  taste; no geometry level-of-detail precompute either way (CLAUDE.md
  do-not-add list).
- **Day slicing:** client-side arithmetic on served instants. The one
  real decision is the midnight rule (a leg crossing midnight belongs to
  the day it starts in, or is split at the boundary) — a Stage C
  decision that must match between map highlight and narrative totals.
- **Home markers: recommend against.** The life map should not mark home
  bases: they are not exposed by the API today, the adventures' radial
  pattern already implies them, and a life map is the screen most likely
  to be screenshotted and shared — leaving home off it is the
  privacy-respecting default and costs nothing. (Self-hosters' data
  never leaves their machine; this is about what *they* then share.)
- **No other backend work is implied.** Photos, stops, countries,
  scores, divergence, truncation all serve already.

## 8. Stage B plan

Deliverables, all built on demo-dataset content, committed under
`docs/phase-6/mockups/` as static HTML (no live map — captured geometry
rendered as SVG/static frames, so the mockups need no tile server and
review in any browser):

1. Direction A and Direction B, each as two screens: the life map
   (overview zoom, Iceland demo bounds) and one adventure detail (the
   sparse Westfjords trip — the honesty-channel stress case — with its
   day narrative).
2. The T1 vs T2 channel treatment rendered in both directions at both
   zooms — four small map panels that make the invariant-8 question
   concrete enough to decide on sight.
3. A one-page token sheet per direction (palette with leg-kind colors
   first, type pairing, the signature element named).

Review questions Stage B puts to the maintainer: which direction (or
which hybrid); T1 or T2; day-stage vocabulary ("Day 3" vs "Stage 3");
how loud the score should be on a confirmed adventure's cover (or
whether it retires to the triage table); whether the adventure list rail
is visible by default or summoned.

Decisions from that review land in DESIGN.md, and only then does the
implementation BRIEF (Stage C) get written.
