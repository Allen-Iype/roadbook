# UI screenshots — visual record over time

Every screenshot in this directory is captured from the compose demo
instance (`docker compose up`, the fictional Reykjavík persona at
`127.0.0.1:3000`). Never from a real-data instance: real location history
never enters git, and that includes pixels.

Naming: `<set>-<page>[-dark].png`. A set is `phase<N>-<milestone>` —
`phase6-baseline` is the UI before phase 6's redesign, `phase6-cp2` the UI
at that checkpoint's acceptance, and later phases add sets the same way.
`<page>` is `home`, `candidates`, `imports`, or `adventure-<id>` (demo
ids: 1 South coast to Höfn · 2 Westfjords · 3 Akureyri by air). The
`-dark` variants exist for `phase6-baseline` only — the pre-phase-6 UI
followed `prefers-color-scheme`; the Atlas plate design is single-theme.

To compare one page over time, list it across sets:
`ls docs/screens/*-home*.png`.

Reproduction: `capture.js` in this directory (puppeteer-core against a
local Chromium; it emulates the color scheme and waits out tile loading —
the MapLibre canvas paints fine this way, unlike the interactive driven
browser). Each file stays under 1 MB.

These files are a historical record, not a source of truth: if the
repository grows too heavy, old sets can simply be deleted
(`git rm docs/screens/phase6-*`) — a set remains regenerable by checking
out the commit of its era, rebuilding the compose web image, and running
`capture.js`.

Known artifact, kept deliberately: the `phase6-baseline` adventure pages
show the pre-fix (0,0)-stop bug — world-spanning bounds with a stop
marker at null island (fixed in ac06b50, which the baseline predates).
The baseline records the UI as it was, bug included.
