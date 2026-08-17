// The layout harness (phase 9 CP1). This suite makes phase 7's throwaway
// puppeteer walk permanent: structural layout truths — no horizontal
// scroll, tap-target sizes, table folding, dialog focus — asserted at
// three viewports against a RUNNING stack with the demo dataset loaded
// (`docker compose up` in the repo root; the tests assume the demo's three
// confirmed adventures).
//
// Deliberately out of scope: map-pixel visual regression. A WebGL canvas
// without preserveDrawingBuffer captures blank, occluded windows suspend
// rAF, and tiles come from a third party — all documented traps
// (docs/phase-6/LOG.md). The map gets presence smoke checks only.
// docs/screens/capture.js remains the separate screenshot-record tool;
// two tools, two jobs, on purpose.
//
// Playwright is not vitest: vitest runs the pure lib/ modules in node,
// this runs a real Chromium against real pages. `npm run test:e2e`.
import { defineConfig } from "@playwright/test";

// Override with ROADBOOK_E2E_URL to point at another stack (e.g. a scratch
// compose project on a different port). A connection-refused failure here
// means the stack is not up — start it first; the suite never boots one.
const baseURL = process.env.ROADBOOK_E2E_URL ?? "http://127.0.0.1:3000";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  // The suite is read-only against a shared stack, so parallel workers are
  // safe: nothing decides, uploads, or deletes.
  reporter: [["list"]],
  timeout: 30_000,
  use: {
    baseURL,
    trace: "retain-on-failure",
  },
  // Three widths, one browser. The breakpoints under test are Tailwind's
  // sm (640) and md (768): phone sits below both, tablet exactly at md,
  // desktop above everything. Chromium only — the assertions are about CSS
  // layout, which does not differ enough across engines to buy three
  // browser downloads (proportionality: small suite, structural checks).
  projects: [
    { name: "phone-390", use: { viewport: { width: 390, height: 844 } } },
    { name: "tablet-768", use: { viewport: { width: 768, height: 1024 } } },
    { name: "desktop-1280", use: { viewport: { width: 1280, height: 800 } } },
  ],
});
