I have a project summary already prepared in this conversation. I want you to
  compare that project against a second project of mine, Roadbook, described
  below, and identify features from the first project that Roadbook should
  consider adopting.

  ## Roadbook, briefly
  
  A self-hostable, open-source web app: you import your Google Timeline export,
  it detects "adventure candidates" — journeys far from home with a real
  destination — and you confirm the real ones with a name or dismiss them. For
  each confirmed adventure it draws the route travelled and reports distance and
  stops. Product principles, in priority order:

  1. Curation, not coverage. Tens of records, not thousands. The user choosing
     which journeys count IS the feature. It is not a timeline browser.
  2. Post-hoc assembly is expected: the tool proposes, the human decides. A
     confirmation step is not friction to be removed.
  3. The satisfaction is seeing the route. The source data has only scattered
     points; inferring the road between them is core. Observed parts of a route
     are always visibly distinct from inferred parts — honesty about confidence
     is non-negotiable.

  Explicitly out of scope: full timeline browsing, live tracking, a mobile app,
  sharing/social/community features, multi-user.

  Stack: Go pipeline (import, detection, routing) + Postgres, an OpenAPI
  contract, Next.js frontend that only talks to the Go API. Detection is a pure
  function; every threshold is a named parameter; user decisions survive
  re-detection. Routing will be pluggable (OSRM / public endpoint / straight
  lines) with no runtime dependency on a routing service.

  Current state: phase 1 (candidate list + confirm/dismiss) is nearly done.
  Planned phases: 2 — one adventure on a map with gaps marked unknown; 3 —
  routing fills gaps, validated against the source's own distance figures; 4 —
  photos with EXIF positions as independent route validation; 5 — docker compose
  self-hosting.

  ## What I want from you

  Compare the project summarized earlier in this conversation with Roadbook:

  1. List candidate features from that project that could transfer to Roadbook.
     For each: what it does there, what it would look like in Roadbook, which
     Roadbook phase it would attach to, and rough complexity (small / medium /
     large).
  2. Filter honestly. Reject any feature that conflicts with Roadbook's three
     principles or its out-of-scope list, and say which principle it fails —
     rejected-with-reason is as valuable to me as recommended.
  3. Note ideas that are good but premature, and what should exist first.
  4. End with your top 3 recommendations ranked by value-to-effort, each in
     two or three sentences of factual justification — no marketing language.

  Format the whole answer as a markdown document titled "Feature comparison:
  dawarich → Roadbook" that I can save into Roadbook's docs/ for review.