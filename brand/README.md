# Brand assets

The single home for Roadbook's mark and the files made from it. The mark
was chosen on 2026-10-31 from the "Roadbook Logo Options" canvas (concept
A, "Rally"): a vertical tape with tick marks, a bent route line, a
waypoint dot and an arrow. No letters in the mark; the R and B variants on
the canvas were not chosen.

```
brand/
  logo/      source vectors, copied from the canvas — never redrawn here
  exports/   generated files only; `node brand/export.mjs` remakes every one
  export.mjs the generator (Chromium from web/'s Playwright, no new dependency)
```

## Which file where

| use | file | notes |
|---|---|---|
| Instagram profile picture | `exports/instagram-profile-1080.png` | 1080 square, mark only, sits in the middle 70% so the circle crop never cuts it |
| anything that takes a vector | `logo/rally-strip.svg` | 120×160 viewBox, 3:4 |
| browser tab | `web/app/favicon.ico` | **not the mark** — it is the Next.js scaffold's default icon from the phase-1 commit (16/32/48/256). Replacing it is a separate change: Next reads `app/favicon.ico` and `app/icon.png` by convention |
| phone home screen, social preview image | none yet | no `apple-icon`, `icon.png`, `manifest` or `opengraph-image` exists in `web/app/`, and the landing has no icon link at all |

There is no wordmark file. On the canvas "ROADBOOK" is live text in
Barlow Condensed 700, letter-spaced; no vector of it exists, and Barlow is
not among the repository's fonts (`web/app/fonts/` holds Source Serif 4
and IBM Plex Mono). Set it in type where needed, or decide a lockup first.

## Colours

The mark's own colours are the canvas's: tar black `#1B1A17`, sand
`#F3EDE2`, rally orange `#E8590C`. They are **not** the app's tokens.
The app's palette lives in `web/app/globals.css` under `:root` as
`--color-paper`, `--color-land`, `--color-ink`, `--color-ink-2`,
`--color-rule`, and the four leg inks `--color-observed`, `--color-routed`,
`--color-unknown`, `--color-air`. `export.mjs` reads a background by token
name from that file; nothing in `brand/` copies a hex from it by hand.
Which palette the brand uses is an open decision (see
`docs/phase-16/DECISIONS.md`, 2026-10-31); until it is made, the mark
keeps the canvas colours and sits on the app's paper.

## Clear space and size

Keep clear space around the mark of at least one quarter of its height on
every side (for the 120×160 mark, 40 units). Nothing else inside that
space: no text, no rule, no edge. Smallest use: 24 px tall; below that the
tick marks and the branch line vanish and only the tape and route remain.

## Licence

The Roadbook name and logo are not covered by the AGPL-3.0 licence. Forks
and self-hosted copies may use the code, but must not present themselves
as Roadbook or use this logo.

## Rules

- **No text in the profile picture.** At 110 px in a feed and 40 px in a
  comment, letters are noise. The mark alone, on a plain ground.
- Never edit a file in `exports/`; change the source or the generator and
  run it.
- Never redraw the mark here. The canvas is the drawing; `logo/` is a copy.
