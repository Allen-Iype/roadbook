// Captures the UI-record screenshots for docs/screens/.
//
//   node capture.js <set>            e.g. node capture.js phase6-cp2
//   node capture.js <set> --both     also capture prefers-color-scheme: dark
//                                    (only the pre-phase-6 UI had a dark scheme)
//
// Points at the compose demo instance (127.0.0.1:3000) — fictional data only.
// Needs puppeteer-core (npm install in any scratch dir, run from there) and a
// Chromium binary; set CHROME_BIN if yours is not in the list below.

const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');

const OUT = __dirname;
const BASE = 'http://127.0.0.1:3000';
const PAGES = [
  { path: '/', name: 'home' },
  { path: '/candidates', name: 'candidates' }, // 404 before phase 6 CP2 ships
  { path: '/imports', name: 'imports' },
  { path: '/adventure/1', name: 'adventure-1' },
  { path: '/adventure/2', name: 'adventure-2' },
  { path: '/adventure/3', name: 'adventure-3' },
];
const BINARIES = [
  process.env.CHROME_BIN,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser',
].filter(Boolean);

const set = process.argv[2];
if (!set || set.startsWith('-')) {
  console.error('usage: node capture.js <set> [--both]');
  process.exit(1);
}
const schemes = process.argv.includes('--both') ? ['light', 'dark'] : ['light'];
const executablePath = BINARIES.find((p) => fs.existsSync(p));
if (!executablePath) {
  console.error('no Chromium binary found; set CHROME_BIN');
  process.exit(1);
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath,
    headless: 'new',
    args: ['--hide-scrollbars'],
  });
  const page = await browser.newPage();
  const VIEWPORT = { width: 1600, height: 1000, deviceScaleFactor: 1 };
  await page.setViewport(VIEWPORT);

  for (const scheme of schemes) {
    await page.emulateMediaFeatures([{ name: 'prefers-color-scheme', value: scheme }]);
    for (const p of PAGES) {
      const res = await page.goto(BASE + p.path, { waitUntil: 'networkidle0', timeout: 30000 });
      if (!res.ok()) {
        console.log('skip', p.path, res.status());
        continue;
      }
      // Remote tiles on a cold cache can take several seconds; a screenshot
      // of a half-loaded map would read as a rendering bug in the record.
      await new Promise((r) => setTimeout(r, 8000));
      const suffix = scheme === 'light' ? '' : '-dark';
      const file = path.join(OUT, `${set}-${p.name}${suffix}.png`);
      // Never fullPage: puppeteer's fullPage mode composites beyond the
      // viewport, and a WebGL canvas rendered without preserveDrawingBuffer
      // (MapLibre's default) yields only whatever sliver intersected its
      // last presented frame — maps come out blank or partial while the
      // live page is fine. Growing the viewport to the page height and
      // taking a plain capture goes through the normal compositor path,
      // which always holds the map's current frame.
      const fullHeight = await page.evaluate(() =>
        Math.min(document.documentElement.scrollHeight, 5000),
      );
      if (fullHeight > VIEWPORT.height) {
        await page.setViewport({ ...VIEWPORT, height: fullHeight });
        await new Promise((r) => setTimeout(r, 2500)); // reflow + repaint
      }
      await page.screenshot({ path: file });
      if (fullHeight > VIEWPORT.height) {
        await page.setViewport(VIEWPORT);
        await new Promise((r) => setTimeout(r, 500));
      }
      console.log('wrote', file);
    }
  }
  await browser.close();
})();
