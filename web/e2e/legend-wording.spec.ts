// DESIGN §6: the legend's fixed wording, read from lib/legend.ts so the
// spec follows the one constant every legend reads (the page components
// and the exported image's margin alike). Copy pass (words.md decision 2,
// 2026-11-02): the legend is REACHABLE at every width — one "What the
// lines mean" button on the life map, /welcome, and the trip page's map
// margin (reached through the summoned list, as a reader would) — and
// opens with all four names and their descriptions.
import { expect, test } from "@playwright/test";

import { LEGEND_ENTRIES } from "../lib/legend";
import { openLegend, openSummonedList } from "./helpers";

async function expectFullLegend(page: import("@playwright/test").Page) {
  const legend = await openLegend(page);
  for (const e of LEGEND_ENTRIES) {
    await expect(legend.getByText(e.label, { exact: true })).toBeVisible();
    await expect(legend.locator("[data-legend-desc]", { hasText: e.desc })).toBeVisible();
  }
}

for (const path of ["/", "/welcome"]) {
  test(`the legend with its descriptions at ${path}`, async ({ page }) => {
    await page.goto(path);
    await expectFullLegend(page);
  });
}

test("the legend with its descriptions on the trip page", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  await page.goto(href!);
  await expectFullLegend(page);
});
