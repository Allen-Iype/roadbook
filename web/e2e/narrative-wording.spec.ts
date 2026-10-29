// Phase 9 BRIEF §5.2 found the demo's stationary overnight gaps narrated
// as "0.0 km" transits. Phase 16 removed transit rows from the page, so
// the remaining risk is a day line or stay row carrying a 0.0 km figure
// in words; the day heading shows "—" for a day with none. Checked on every adventure the
// instance has, discovered through the summoned list like a reader would.
import { expect, test } from "@playwright/test";

import { openSummonedList } from "./helpers";

test("no adventure narrates a 0.0 km figure", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const hrefs = await dialog
    .getByRole("link")
    .evaluateAll((els) => els.map((el) => el.getAttribute("href")));
  expect(hrefs.length).toBeGreaterThan(0);
  for (const href of hrefs) {
    await page.goto(href!);
    const narrative = page.locator('section[aria-label="Days"]');
    await expect(narrative).toBeVisible();
    // \b keeps "160.0 km" from matching as a substring — only a genuine
    // standalone 0.0 figure may fail this.
    await expect(narrative.getByText(/\b0\.0 km/)).toHaveCount(0);
  }
});
