# CLAUDE.md — Roadbook working agreement

Roadbook turns a Google Timeline export into a curated set of adventures. It detects
candidate journeys — far from home, with a real destination — and the user confirms or
dismisses each one. Confirmed adventures are drawn as routes in which observed and
inferred segments are always visibly distinct. Product definition: `docs/PRODUCT.md`.
Phase plan: `docs/PLAN.md`.

---

## Architecture — decided 2026-07-30, not open for preference

**Go owns all data. The frontend never touches Postgres.**

```
browser ──▶ Next.js ──HTTP──▶ Go API ──▶ Postgres
                                 ▲
                    Go CLI ──────┘   (import · detect · route)
```

- Go is both a batch CLI (import, detection, routing) and a long-running HTTP service
  that the frontend consumes. Postgres is reachable only from Go.
- The browser talks only to Next.js. Reads happen in server components calling the Go
  API; writes go through server actions calling the Go API. The Go service is not
  exposed to the browser — one network boundary visible to the user's machine, no CORS
  or auth questions before anything requires them.
- `openapi.yaml`, written by hand, is the single contract. Go server interfaces are
  generated from it with `oapi-codegen`; the TypeScript client with
  `openapi-typescript`. Neither side is ever hand-written; generated files are never
  edited.

**Why this shape.** It makes "no business logic in the frontend" a structural fact
rather than a discipline: the frontend has no connection string, so the rule cannot be
violated even by accident. There is one owner of the schema and one access path, so no
coordination rule between two database consumers is needed. The data contract sits in
Go, where this project's engineering strength lies and where it can be reviewed
properly. A second consumer later — a phone client, a shared view — gets an API that
already exists.

**Costs, accepted knowingly.** More phase-1 code that no user sees (routes, shapes,
error semantics, a generated client, loading and error states), and a local HTTP hop
when a server component renders a page. Both were weighed against the alternatives
(frontend reads Postgres via Drizzle introspection; all-TypeScript single runtime) and
accepted. Do not re-open without a substantive new fact.

## Stack

Go · Postgres · `goose` · `pgx` · Next.js 16 App Router · React 19 · TypeScript strict ·
Tailwind 4 · shadcn · MapLibre GL · OSRM (pluggable, offline batch).

**Next.js 16 postdates model training data.** Read `node_modules/next/dist/docs/`
before writing any Next-specific code, and heed the deprecation notices there.

**Do not add:** an ORM that owns the schema · a message queue, Redis, or Kubernetes ·
runtime dependence on a routing service · business logic in the Next layer, which
renders and interacts and nothing more · live sharing, WebSockets, multi-user, or
social features · list virtualization or geometry level-of-detail precompute — there
are tens of journeys, and building for thousands is theatre.

---

## Invariants

Not preferences. Breaking one is a bug. Each has a reason; the reason is the point.

1. **The detection and reconstruction core is pure.** No I/O, clock, randomness,
   network, or database. Identical inputs always yield identical output. *Why:* it makes
   the reference detector's output a real regression test, and it is the boundary that
   keeps the interesting logic testable without infrastructure.
2. **Inputs are immutable.** Parsed visits, activities, and points are never mutated or
   deleted. Derived output is written separately. *Why:* re-detection with different
   parameters must be free and safe. If the algorithm can corrupt its own input, no
   result is reproducible.
3. **Every threshold is a named parameter**, never a literal in an algorithm body, and
   every run records the parameters that produced it. *Why:* detection parameters
   changed four times in a single exploration session. Without this, no output can be
   explained after the fact.
4. **Parsers emit domain types and nothing else.** No parser type escapes its package.
   *Why:* adding GPX or another source later must require zero changes to detection.
5. **A journey is a list of legs, each observed or inferred.** Never one
   undifferentiated line. *Why:* the product's honesty depends on the distinction
   surviving all the way to the renderer. A single geometry loses it irrecoverably.
6. **Observed legs are never routed.** Routing belongs only to gaps. *Why:* routing
   replaces measurement with inference. Applying it to observed data destroys the only
   ground truth available.
7. **Routing is pluggable** — one interface, three implementations: self-hosted OSRM, a
   public endpoint, and a null router that draws straight lines and says so. *Why:* a
   self-hoster must be able to see their adventures without building an OSM extract.
   The product degrades visibly instead of failing.
8. **Confidence is never hidden.** Any geometry on a map carries its leg kind. *Why:* a
   gap drawn as a confident solid line is a lie, not a styling choice.
9. **Nothing is hardcoded about any specific user.** Home is derived from the data;
   thresholds are parameters; no regional assumptions; no coordinates in config. *Why:*
   self-hostability is a stated goal, and the detection rule was only correct once it
   stopped assuming a fixed home.
10. **`openapi.yaml` is the contract.** Go server interfaces are generated from it with
    `oapi-codegen`; the TypeScript client is generated from it with
    `openapi-typescript`. Neither side is ever hand-written, and generated files are
    never edited. *Why:* the whole cost of a service boundary is keeping two type
    systems agreeing. Generating both from one spec makes drift impossible by
    construction rather than by vigilance.
11. **The frontend never reaches the database.** It has no connection string, no ORM,
    no SQL. Every read and write goes through the API. *Why:* this is what makes "no
    business logic in the frontend" structurally impossible to violate rather than a
    rule someone has to remember at 11pm two years from now.
12. **TypeScript is `strict`, zero `any`.** `unknown` plus a type guard is the escape
    hatch.
13. **Never state a number the repository cannot reproduce** — in the README, the UI,
    or any document. Every public figure must trace to a command that regenerates it.
14. **Real location data never enters git.** See "Data safety" below. This has been the
    single most recurring hazard in this project.

---

## Data safety

```
data/        private. Real location history. Gitignored wholesale. NEVER committed.
testdata/    committed. Anonymised fixtures and expected values only.
prototype/   detect_fixture.py — the reference detector.
```

- **Never commit anything under `data/`.** Location data reached stageable state twice
  during early setup — once through a `.gitignore` pattern anchored to the repository
  root that did not match a subdirectory, and once through a script writing output to
  an unanticipated path. The rule exists because vigilance alone failed.
- **Run `git add -A --dry-run` before every commit** and confirm that nothing from
  `data/` and no file over 1 MB appears. A standing rule, not a one-time check.
- **Never modify or delete anything in `data/`.** Files there can be irreplaceable:
  a Timeline export's `rawSignals` window expires at the source and cannot be
  re-downloaded.
- Any fixture that is committed must be anonymised first, into `testdata/`.
- Scripts and tools must never write output containing real coordinates outside
  `data/` (this caused one of the two near-leaks).

---

## Documentation

Every phase must explain the concepts it introduces: **what, how, why, and where in
this codebase.** Not links to documentation — explanation in context, against the code
actually being written, including the alternatives that were rejected and the reason.
Where the territory is unfamiliar, be explicit rather than efficient.

Each phase produces three artifacts:

- **A design brief, before any code.** Opens with the concepts the phase introduces.
  States what gets built, the two or three real choices inside it, and a recommendation
  with reasoning. Reviewed and agreed before implementation begins.
- **A decision log, written as decisions are made.** Three lines each: what was chosen,
  what was rejected, what would change our mind. Written at the moment, not
  reconstructed later.
- **A phase log, at the end.** What the phase does, what broke, and why each fix took
  the form it did. **A phase is not complete until its log exists.**

**Prose:** factual voice throughout. No "production-ready," no "beautiful" or
"modern," no marketing language, no emoji headers. State no claim the repository
cannot demonstrate.

**Commits:** conventional commits (`feat:`, `fix:`, `chore:`, `docs:`, …).

**Claude never commits.** Claude prepares changes and reports what is ready; the
maintainer reviews the `git add -A --dry-run` output and makes every commit
themselves. No exceptions, including "trivial" ones — the dry-run review in "Data
safety" only works if the person accountable for it is the one running it.
