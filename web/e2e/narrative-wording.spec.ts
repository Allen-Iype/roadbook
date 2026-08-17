// Phase 9 BRIEF §5.2: a stationary gap (a routed leg whose ends coincide —
// the demo's overnight guesthouse gaps) must read as "Stationary gap", not
// "Routed transit — 0.0 km along roads". Checked on every adventure the
// instance has, discovered through the summoned list like a reader would.
import { expect, test } from "@playwright/test";

import { openSummonedList } from "./helpers";

test("no adventure narrates a 0.0 km routed transit", async ({ page }) => {
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
    // \b keeps "160.0 km along roads" (a real transit) from matching as a
    // substring — only a genuine standalone 0.0 figure may fail this.
    await expect(narrative.getByText(/\b0\.0 km along roads/)).toHaveCount(0);
  }
});
