// The adventures atlas (phase 9 CP4): structural truths of the grid as the
// CP3 review decided them — plate covers with the honest figures, newest
// first (no plate numeral since phase 17, #2), the legend on the page, and
// the header naming where the reader is.
import { expect, test } from "@playwright/test";

import { openLegend } from "./helpers";

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
    // Phase 17 (#2): the cover carries no plate numeral.
    await expect(cover.getByText(/PLATE/)).toHaveCount(0);
  }
});

test("newest first", async ({ page }) => {
  // The CP3 review's ordering decision, proven by the dates themselves now
  // the numeral is gone (phase 17, #2): reading down the grid, each trip
  // starts no later than the one above it.
  await page.goto("/adventures");
  const starts = await page
    .locator('main ul a[href^="/adventure/"]')
    .evaluateAll((els) => els.map((el) => el.getAttribute("data-start") ?? ""));
  expect(starts.length).toBeGreaterThan(1);
  for (let i = 1; i < starts.length; i++) {
    expect(starts[i] <= starts[i - 1]).toBe(true);
  }
});

test("one plain line above the covers: the count and the order", async ({ page }) => {
  // Copy pass (2026-11-03): one plain line under the heading — the
  // count, the order — and no taught term; the legend sits behind its
  // button at the foot.
  await page.goto("/adventures");
  const intro = page.locator("main > p").first();
  await expect(intro).toContainText(/\d+ trips?, the latest first\./);
  await expect(intro).not.toContainText(/numbered/);
  await expect(intro.locator("button.term")).toHaveCount(0);
  const firstCover = page.locator('main ul a[href^="/adventure/"]').first();
  expect((await intro.boundingBox())!.y).toBeLessThan((await firstCover.boundingBox())!.y);
});

test("the page carries the legend behind its button", async ({ page }) => {
  // Covers draw the four-way encoding, so the page owes the reader its
  // language — one tap away since the copy pass (words.md decision 2).
  await page.goto("/adventures");
  await expect(await openLegend(page)).toBeVisible();
});

test("the header marks Trips as the current page", async ({ page }) => {
  await page.goto("/adventures");
  await expect(
    page.locator('header a[aria-current="page"]', { hasText: "Trips" }),
  ).toBeVisible();
});

test("covers are keyboard-reachable and Enter opens the adventure", async ({ page }) => {
  // The §7 keyboard walk, made permanent: a cover is a link, so the
  // keyboard contract is focus + Enter — no pointer required anywhere on
  // the shelf.
  await page.goto("/adventures");
  const first = page.locator('main ul a[href^="/adventure/"]').first();
  const href = await first.getAttribute("href");
  // Focus set before hydration is dropped when React reconciles the node
  // (the clickUntil trap, in focus form — helpers.ts documents it), so
  // re-issue focus until it sticks; a human tabbing after paint never
  // races this.
  for (let i = 0; i < 5 && !(await first.evaluate((el) => el === document.activeElement)); i++) {
    await first.focus();
    await page.waitForTimeout(300);
  }
  await expect(first).toBeFocused();
  await page.keyboard.press("Enter");
  await page.waitForURL(`**${href}`);
});
