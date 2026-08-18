# Roadbook — what it is

**A place to see the adventures you actually took.**

You import a Google Timeline export. Roadbook finds the journeys that look like
adventures — where you went far from home and stayed somewhere — and offers them as
candidates. You confirm the ones that were real, name them, and reject the rest. For
each confirmed adventure it draws the route travelled, reports distance and stops, and
marks honestly which parts were inferred rather than observed.

Three properties follow. Check every design decision against them.

**1. Curation, not coverage.** This is not a timeline browser. The user chooses which
journeys count, and that choice is the feature. Expect tens of records, not thousands.
Anything built for scale — list virtualization, geometry level-of-detail precompute,
viewport-indexed spatial queries — is unnecessary weight.

**2. Post-hoc assembly is expected.** The tool proposes, the human decides. Optimising
for seamlessness at the cost of user control is the wrong trade. A confirmation step
is not friction to be removed.

**3. The satisfaction is seeing the route.** The source data contains no routes, only
scattered points. Inferring the road between them is therefore core, not polish. A
build that cannot draw a route has not shipped.

**Self-hostable, open source.** Anyone runs this with their own export. Nothing about
any specific user is hardcoded — no coordinates in config, no regional assumptions,
every threshold a named parameter with a sensible default. Setup must not begin with
"obtain an OSM extract."

**Hostable as a service; self-host is the reference.** Roadbook serves two kinds of
user with one codebase: people who run it themselves, and people who use an instance
someone runs for them. Self-host is the reference deployment forever; hosted is a
superset, never a fork. The test every feature must pass: it works single-user and
authless, on a plain `docker compose up`, before anything hosted builds on it.
Whatever an operator adds to host for someone else — provisioning, reverse proxies,
authentication in front — lives outside this repository's product code until
multi-user tenancy earns its own chartered phase, a deliberate launch decision.
Hosting for strangers also means holding a new category of data — contact details
such as a waitlist's email addresses — which is held minimally, stated plainly to
the person, and deletable on request.

## Out of scope

Full timeline browsing. Live tracking. A mobile app. Live sharing, social features, or
community content such as road conditions and points of interest. Accounts,
sign-up, and read-only share links are not in scope until multi-user tenancy's
chartered phase — they are staged, not rejected. The community direction is a
plausible future direction and a different product; the only accommodation now is a
`user_id` column that always holds the same value, so that adding it later is not a
migration nightmare.
