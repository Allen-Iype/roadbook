// The adventure detail page, reached the way a user reaches it (through
// the summoned list — the suite never hardcodes a candidate id, because
// ids change with every detection run). Layout truths plus the two
// promises the plate makes at every width: the fixed legend wording
// (DESIGN §6) and a mounted map canvas. Canvas PIXELS are deliberately
// not asserted — see playwright.config.ts.
import { expect, test } from "@playwright/test";

import {
  expectNoHorizontalScroll,
  legendLocator,
  openSummonedList,
} from "./helpers";

async function gotoFirstAdventure(page: import("@playwright/test").Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await page.goto(href!);
}

test("adventure page: no horizontal scroll", async ({ page }) => {
  await gotoFirstAdventure(page);
  await expectNoHorizontalScroll(page);
});

test("adventure page: the plate legend names all four kinds", async ({ page }) => {
  await gotoFirstAdventure(page);
  // The detail plate mounts the compact legend (wordy={false}): drawn
  // samples + kind names in the plate margin, no descriptions at any
  // width. The full fixed wording lives on the life map and /welcome —
  // asserted in legend-wording.spec.ts.
  const legend = legendLocator(page);
  for (const label of ["Observed", "Routed", "Unknown", "Air"]) {
    await expect(legend.getByText(label, { exact: true })).toBeVisible();
  }
});

test("adventure page: the map mounts", async ({ page }) => {
  await gotoFirstAdventure(page);
  await expect(page.locator(".maplibregl-canvas")).toBeAttached({
    timeout: 20_000,
  });
});

test("life map: the map mounts", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".maplibregl-canvas")).toBeAttached({
    timeout: 20_000,
  });
});
