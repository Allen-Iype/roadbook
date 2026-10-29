// DESIGN §6: the legend's fixed wording, read from lib/legend.ts so the
// spec follows the one constant every legend reads (the page components
// and the exported image's margin alike). Phase 16 CP1: the descriptions
// are visible at EVERY width — the phone used to get four names and no
// explanation — on the life map, /welcome, and the adventure plate's
// margin (reached through the summoned list, as a reader would).
import { expect, test } from "@playwright/test";

import { LEGEND_ENTRIES } from "../lib/legend";
import { legendLocator, openSummonedList } from "./helpers";

async function expectFullLegend(page: import("@playwright/test").Page) {
  const legend = legendLocator(page);
  for (const e of LEGEND_ENTRIES) {
    await expect(legend.getByRole("button", { name: e.label, exact: true })).toBeVisible();
    // The description span by its hook: the Term note behind each name
    // repeats the phrase, so a bare text match would resolve to two.
    await expect(legend.locator("[data-legend-desc]", { hasText: e.desc })).toBeVisible();
  }
}

for (const path of ["/", "/welcome"]) {
  test(`the legend with its descriptions at ${path}`, async ({ page }) => {
    await page.goto(path);
    await expectFullLegend(page);
  });
}

test("the legend with its descriptions on the adventure plate", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  await page.goto(href!);
  await expectFullLegend(page);
});
