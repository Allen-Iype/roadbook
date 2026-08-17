// The adventures atlas (phase 9 CP4): structural truths of the grid as the
// CP3 review decided them — plate covers with the honest figures, newest
// first with chronological plate numbers, the legend on the page, and the
// header naming where the reader is.
import { expect, test } from "@playwright/test";

import { legendLocator } from "./helpers";

test("every cover is a link carrying art, figures, and a provenance bar", async ({ page }) => {
  await page.goto("/adventures");
  const covers = page.locator('main ul a[href^="/adventure/"]');
  const n = await covers.count();
  expect(n).toBeGreaterThan(0);
  for (let i = 0; i < n; i++) {
    const cover = covers.nth(i);
    await expect(cover.locator("svg").first()).toBeVisible();
    // The mono dateline pairs the range with the km figure.
    await expect(cover.getByText(/\d+ km/)).toBeVisible();
    await expect(cover.getByText(/PLATE \d+/)).toBeAttached();
  }
});

test("newest first, plate numbers chronological", async ({ page }) => {
  // The CP3 review's ordering decision, as a testable statement: display
  // order is recency, the number is the order travelled — so reading down
  // the grid, plate numbers strictly decrease.
  await page.goto("/adventures");
  const labels = await page
    .locator('main ul a[href^="/adventure/"]')
    .allTextContents();
  const plates = labels.map((t) => Number(/PLATE (\d+)/.exec(t)?.[1]));
  expect(plates.length).toBeGreaterThan(0);
  for (const p of plates) expect(Number.isNaN(p)).toBe(false);
  for (let i = 1; i < plates.length; i++) {
    expect(plates[i]).toBeLessThan(plates[i - 1]);
  }
});

test("the page carries the leg-kind legend", async ({ page }) => {
  // Covers draw the four-way encoding, so the page owes the reader its
  // language (DESIGN §6: the legend is a permanent fixture).
  await page.goto("/adventures");
  await expect(legendLocator(page)).toBeVisible();
});

test("the header marks Adventures as the current page", async ({ page }) => {
  await page.goto("/adventures");
  await expect(
    page.locator('header a[aria-current="page"]', { hasText: "Adventures" }),
  ).toBeVisible();
});
