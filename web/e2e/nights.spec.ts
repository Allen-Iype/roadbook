// Nights on the trip page (phase 19): the trip block says how many nights
// and where, and the stay that held each one says so on its row. Trips are
// discovered through the summoned list, never by id. The wording rules are
// pinned in lib/nights.test.ts; this file is about the page.
import { expect, test } from "@playwright/test";

import { expectNoHorizontalScroll, openSummonedList } from "./helpers";

async function adventureHrefs(page: import("@playwright/test").Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const hrefs = await dialog
    .getByRole("link")
    .evaluateAll((els) => els.map((el) => el.getAttribute("href")));
  expect(hrefs.length).toBeGreaterThan(0);
  return hrefs as string[];
}

test("every trip counts its nights and says where they were spent", async ({ page }) => {
  for (const href of await adventureHrefs(page)) {
    await page.goto(href);
    const row = page.locator('section[aria-label="The trip"] [data-nights]');
    await expect(row).toBeVisible();
    const text = (await row.innerText()).trim();
    expect(text).toMatch(/^\d+ · \S/);

    // One night between each pair of days: the count is the days minus one.
    const days = await page.locator('section[aria-label="Days"] > section').count();
    expect(Number(text.split(" ")[0])).toBe(days - 1);

    // In the demo every night was spent at a stop, so as many stay rows
    // say so as there are nights.
    const held = page.locator('section[aria-label="Days"] li[data-night-held]');
    await expect(held).toHaveCount(days - 1);
    for (const words of await held.allInnerTexts()) {
      expect(words).toMatch(/Stayed the night|Stayed all day and the night|overnight/);
    }
    await expectNoHorizontalScroll(page);
  }
});

test("a night is said once: the day line does not repeat a night spent at a stop", async ({ page }) => {
  for (const href of await adventureHrefs(page)) {
    await page.goto(href);
    await expect(page.locator('section[aria-label="Days"]')).toBeVisible();
    await expect(page.locator("[data-day-line]").getByText(/overnight into Day/)).toHaveCount(0);
  }
});
