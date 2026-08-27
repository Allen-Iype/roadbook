# landing/ — the public landing page (phase 12)

The hosted service's public pitch surface: one static HTML file, served by
the host's Caddy front at a public hostname with no credential. It is
deliberately not part of the product app (`web/`): a stranger-facing page
with no runtime, no state, and no write path has almost no attack surface,
and PRODUCT.md keeps hosted-operator machinery out of product code.

## What references what

- `index.html` is self-contained except for two asset directories that are
  staged at deploy time, so nothing here is duplicated in git:
  - `fonts/` ← copied from `web/app/fonts/` (Source Serif 4 + IBM Plex
    Mono, OFL-licensed woff2, the same faces the app self-hosts);
  - `shots/` ← copied from `docs/screens/` (the committed demo-dataset
    captures; the mapping lives in `scripts/pilot/host/deploy-landing.sh`).
- The hero plate is the same SVG as the app's `/welcome` pitch plate
  (`web/components/welcome-plate.tsx`), converted to plain HTML attributes.
  Demo geometry only — nothing on this page derives from real location
  history.
- The CSS `:root` tokens are the Atlas values from `web/app/globals.css`.
  If a token changes there, it changes here in the same commit — this page
  must never drift from the product's inks.

## Honesty discipline

The page states only what the repository can demonstrate, in the factual
voice CLAUDE.md requires. When the product changes (a new source, a changed
retention fact, a different cap), this page changes in the same phase —
same standard as the README.

The waitlist is a plain HTML form (the page stays zero-JavaScript — a POST
needs no script): Caddy proxies `/waitlist` to a tiny loopback intake
service (`scripts/pilot/host/waitlistd.py`, installed by
`setup-waitlist.sh`) that appends timestamp + address to one 0600 file on
the host. `joined.html` and `oops.html` are the form's two endings. The
custody sentence on the page — one file, nothing else sent, deleted on
request — is the contract; the entries file is deliberately excluded from
nightly backups so the deletion promise is absolute.

## Deploying

On the host, from the repo checkout:

```
scripts/pilot/host/deploy-landing.sh
```

It stages `/srv/pilot/landing/site/`, writes the Caddy snippet for
`www.$ROADBOOK_DOMAIN` (the wildcard certificate covers `www.`; the bare
apex would need its own certificate entry, which is why the landing lives
on a subdomain), validates as the caddy user, and reloads. The Caddyfile
must carry `import /srv/pilot/landing/caddy.conf` inside the wildcard site
block, before the 404 fallback — the script checks and prints the line if
it is missing rather than editing `/etc/caddy/Caddyfile` itself.
