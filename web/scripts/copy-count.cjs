// Counts the words a person is shown on each page — the one number the
// reading-load phase (phase 16) is measured by, before and after.
//
//   NODE_PATH=<dir with puppeteer-core>/node_modules node web/scripts/copy-count.cjs
//
// Points at a demo stack (ROADBOOK_SCREENS_URL, default 127.0.0.1:3000 —
// demo data only, the docs/screens rule). Needs puppeteer-core and a
// Chromium binary (CHROME_BIN overrides the list below), the same way
// docs/screens/capture.js does.
//
// What is counted, per page and per viewport (phone 390 / desktop 1280):
//   words      whitespace-separated runs of body.innerText — what is visible
//              at that width (display:none text is excluded by innerText, so
//              the legend's phone-hidden descriptions count on desktop only)
//   sentences  runs ending in . ? ! or a line break — approximate, stable
//   titles     elements carrying a title attribute: hover-only teaching,
//              invisible on touch
//   terms      occurrences of the atlas/pipeline vocabulary listed in TERMS
// The landing page is read from disk (landing/index.html) with its
// deploy-time placeholder left in; its words are the same either way.
//
// Output: one Markdown table on stdout, pasted into the phase documents.

/* eslint-disable @typescript-eslint/no-require-imports -- CommonJS on purpose:
   require() honours NODE_PATH, so puppeteer-core resolves from a scratch
   directory instead of becoming a web/ dependency (docs/screens/capture.js
   works the same way). */
const puppeteer = require('puppeteer-core');
const fs = require('fs');
const path = require('path');

const BASE = process.env.ROADBOOK_SCREENS_URL || 'http://127.0.0.1:3000';
const REPO = path.resolve(__dirname, '..', '..');
const PAGES = [
  { url: 'file://' + path.join(REPO, 'landing', 'index.html'), name: 'landing' },
  { url: BASE + '/welcome', name: 'welcome' },
  { url: BASE + '/', name: 'home (life map)' },
  { url: BASE + '/adventures', name: 'adventures' },
  { url: BASE + '/candidates', name: 'candidates' },
  { url: BASE + '/adventure/1', name: 'adventure 1' },
  { url: BASE + '/adventure/2', name: 'adventure 2' },
  { url: BASE + '/adventure/3', name: 'adventure 3' },
  { url: BASE + '/imports', name: 'imports' },
];
// Vocabulary a newcomer meets cold. Counted as whole words, case-insensitive;
// plural/verb forms listed where the pages use them.
const TERMS = [
  'plate', 'plates', 'provenance', 'legend', 'observed', 'routed', 'inferred',
  'unknown', 'fix', 'fixes', 'candidate', 'candidates', 'triage', 'decision',
  'decisions', 'dwell', 'dwelling', 'divergence', 'source', 'window', 'outliers',
  'detection', 'detector', 'score', 'instance', 'sidecar', 'sidecars', 'takeout',
  'great-circle', 'trace', 'raw', 'stop', 'stops', 'measured', 'reconstruction',
  'timeline', 'export', 'atlas',
];
const VIEWPORTS = [
  { name: 'phone', width: 390, height: 844, deviceScaleFactor: 1 },
  { name: 'desktop', width: 1280, height: 800, deviceScaleFactor: 1 },
];
const BINARIES = [
  process.env.CHROME_BIN,
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Brave Browser.app/Contents/MacOS/Brave Browser',
].filter(Boolean);

const executablePath = BINARIES.find((p) => fs.existsSync(p));
if (!executablePath) {
  console.error('no Chromium binary found; set CHROME_BIN');
  process.exit(1);
}

function measure(terms) {
  const text = document.body.innerText || '';
  const words = text.split(/\s+/).filter(Boolean);
  const sentences = text
    .split(/[.?!]\s|\n+/)
    .map((s) => s.trim())
    .filter((s) => s.split(/\s+/).length >= 3).length;
  const titles = document.querySelectorAll('[title]').length;
  const lower = text.toLowerCase();
  let hits = 0;
  const perTerm = {};
  for (const t of terms) {
    const re = new RegExp('(^|[^a-z-])' + t.replace('-', '\\-') + '(?=$|[^a-z-])', 'g');
    const n = (lower.match(re) || []).length;
    if (n) perTerm[t] = n;
    hits += n;
  }
  return { words: words.length, sentences, titles, terms: hits, perTerm };
}

(async () => {
  const browser = await puppeteer.launch({ executablePath, headless: 'new' });
  const page = await browser.newPage();
  const rows = [];
  const termTotals = {};
  for (const p of PAGES) {
    const row = { name: p.name };
    for (const vp of VIEWPORTS) {
      await page.setViewport(vp);
      let res;
      try {
        res = await page.goto(p.url, { waitUntil: 'networkidle0', timeout: 30000 });
      } catch (e) {
        row[vp.name] = null;
        console.error('skip', p.url, e.message);
        continue;
      }
      if (res && !res.ok() && !p.url.startsWith('file:')) {
        row[vp.name] = null;
        console.error('skip', p.url, res.status());
        continue;
      }
      await new Promise((r) => setTimeout(r, 1500)); // hydration, islands
      const m = await page.evaluate(measure, TERMS);
      row[vp.name] = m;
      if (vp.name === 'desktop') {
        for (const [t, n] of Object.entries(m.perTerm)) termTotals[t] = (termTotals[t] || 0) + n;
      }
    }
    rows.push(row);
  }
  await browser.close();

  const cell = (m) => (m ? `${m.words} / ${m.sentences} / ${m.titles} / ${m.terms}` : '—');
  console.log(`Source: ${BASE} · ${new Date().toISOString().slice(0, 10)}`);
  console.log('');
  console.log('| page | phone 390 (words / sentences / titles / terms) | desktop 1280 |');
  console.log('|---|---|---|');
  for (const r of rows) console.log(`| ${r.name} | ${cell(r.phone)} | ${cell(r.desktop)} |`);
  console.log('');
  console.log('Vocabulary hits, desktop, all pages, most frequent first:');
  console.log('');
  const sorted = Object.entries(termTotals).sort((a, b) => b[1] - a[1]);
  console.log(sorted.map(([t, n]) => `${t} ${n}`).join(' · '));
})();
