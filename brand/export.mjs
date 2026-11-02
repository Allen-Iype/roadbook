// Regenerate every file under brand/exports/ from brand/logo/ (2026-10-31).
//
//   node brand/export.mjs
//
// Renders the SVG in the Chromium that web/'s Playwright already installs
// (no new dependency), at exactly the pixel size each export needs. Edit the
// canvas, copy the SVG into brand/logo/, run this; never edit an export by
// hand. Background colours are read from web/app/globals.css by token name
// so the brand never carries a hand-copied hex.
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";

const here = dirname(fileURLToPath(import.meta.url));
const require = createRequire(join(here, "..", "web", "package.json"));
const { chromium } = require("playwright-core");

const css = readFileSync(join(here, "..", "web", "app", "globals.css"), "utf8");
function token(name) {
  const m = css.match(new RegExp(`--color-${name}:\\s*(#[0-9a-fA-F]{3,8})`));
  if (!m) throw new Error(`token --color-${name} not found in web/app/globals.css`);
  return m[1];
}

const mark = readFileSync(join(here, "logo", "rally-strip.svg"), "utf8");

// Instagram profile: 1080 square, shown as a circle. The mark sits in the
// middle 70% (a 756 px box) so the circle crop never touches it; it is 700
// px tall, 525 wide (the SVG is 3:4), centred. No text: at 110 px on a
// phone, text in a profile picture is noise.
const EXPORTS = [
  {
    file: "instagram-profile-1080.png",
    size: 1080,
    background: token("paper"),
    markHeight: 700,
  },
];

const browser = await chromium.launch();
try {
  for (const e of EXPORTS) {
    const page = await browser.newPage({
      viewport: { width: e.size, height: e.size },
      deviceScaleFactor: 1,
    });
    const svg = mark.replace(/<svg ([^>]*?)width="\d+" height="\d+"/, `<svg $1width="${Math.round(e.markHeight * 0.75)}" height="${e.markHeight}"`);
    await page.setContent(
      `<!doctype html><html><body style="margin:0;width:${e.size}px;height:${e.size}px;background:${e.background};display:grid;place-items:center">${svg}</body></html>`,
    );
    const png = await page.screenshot({ type: "png", clip: { x: 0, y: 0, width: e.size, height: e.size } });
    writeFileSync(join(here, "exports", e.file), png);
    console.log(`${e.file}: ${e.size}×${e.size}, background ${e.background}, ${png.length} bytes`);
    await page.close();
  }
} finally {
  await browser.close();
}
