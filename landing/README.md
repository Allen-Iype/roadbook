# landing/ — the public landing page (phase 12; redesigned phase 16 CP2b)

The hosted service's public face: static HTML, served by the host's Caddy
front at a public hostname with no credential. Deliberately not part of the
product app (`web/`): a stranger-facing page with no runtime, no state and
no write path has almost no attack surface, and PRODUCT.md keeps
hosted-operator machinery out of product code.

## The files

- `index.html` — the landing. Written for a traveller, not an engineer
  (phase 16 DECISIONS 2026-10-29): the copy is `docs/phase-16/LANDING-COPY.md`,
  approved word for word, 327 words; it never uses the product's internal
  vocabulary (plate, leg, fix, custody, provenance, candidate, civil day,
  great-circle). One action, "Make mine", wherever the sign-in appears.
  The shape is mockup A1, `docs/phase-16/mockups/landing-a1.html`.
- `how-its-built.html` — the engineer's page, linked from the footer: the
  four kinds and what each means, the rule that a gap is never drawn as a
  confident line, where the figures come from, where the data lives, and
  self-hosting as the reference deployment.
- `joined.html`, `oops.html` — the waitlist form's two endings.

## What references what

- The pages are self-contained except for two directories staged at deploy
  time, so nothing here is duplicated in git:
  - `fonts/` ← `web/app/fonts/` (Source Serif 4 + IBM Plex Mono, OFL woff2,
    the faces the app self-hosts);
  - `shots/app.png` ← `docs/screens/phase16-cp2b-adventure-2.png`, the one
    real screenshot on the page (the mapping lives in
    `scripts/pilot/host/deploy-landing.sh`; update the `<img>` width and
    height with it).
- The hero plate is the same SVG as the app's `/welcome` pitch plate
  (`web/components/welcome-plate.tsx`) in plain HTML attributes. The three
  trip covers are the demo adventures' route art exactly as `/adventures`
  draws it (`web/lib/route-thumb.ts`), with the kind inks as CSS variables.
  Demo geometry only — nothing on this page derives from real location
  history.
- The CSS `:root` tokens are the Atlas values from `web/app/globals.css`.
  If a token changes there, it changes here in the same commit.
- The legend's words are `web/lib/legend.ts` (DESIGN §6). The vitest
  `web/lib/landing-legend.test.ts` pins the landing's legend to that
  constant, and pins zero `<script>` and the domain placeholder; it fails
  the moment this page drifts.

## Honesty discipline

The page states only what the repository can demonstrate, in the factual
voice CLAUDE.md requires: "10 free spots" is `ROADBOOK_USER_CAP`; "delete
all of it, any time" is the Imports page's delete-my-data control; the
export steps are the welcome page's verified walkthrough. When the product
changes (a new source, a changed retention fact, a different cap), this
page changes in the same phase — same standard as the README.

The sign-in links name the app hostname as `{ROADBOOK_DOMAIN}`;
`deploy-landing.sh` substitutes the host's `front.env` value at staging and
refuses to deploy if the placeholder survives, so the committed pages name
no domain (phase 15 BRIEF §9).

The waitlist is a plain HTML form (the page stays zero-JavaScript — a POST
needs no script): Caddy proxies `/waitlist` to a tiny loopback intake
service (`scripts/pilot/host/waitlistd.py`, installed by
`setup-waitlist.sh`) that appends timestamp + address to one 0600 file on
the host. The sentence on the page — one message when a spot opens,
nothing else, deleted on request — is the contract; the entries file is
deliberately excluded from nightly backups so the deletion promise is
absolute.

## Deploying

On the host, from the repo checkout:

```
scripts/pilot/host/deploy-landing.sh
```

It stages `/srv/pilot/landing/site/`, writes the Caddy snippet for
`www.$ROADBOOK_DOMAIN`, validates as the caddy user, and reloads. The
Caddyfile must carry `import /srv/pilot/landing/caddy.conf` inside the
site block, before the 404 fallback — the script checks and prints the line
if it is missing rather than editing `/etc/caddy/Caddyfile` itself.
