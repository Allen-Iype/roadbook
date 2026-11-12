// Places on the trip page (phase 18): every demo trip's days say which
// regions they were in, its stays say which town they were near, a name
// that rests on Google's visit record explains itself where it stands,
// and the list's credit sits in the drawing's account. Trips are
// discovered through the summoned list, never by id. The wording itself
// is pinned in lib/day-places.test.ts; this file is about the page.
import { expect, test } from "@playwright/test";

import { clickUntil, expectNoHorizontalScroll, openSummonedList } from "./helpers";

async function adventureHrefs(page: import("@playwright/test").Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const hrefs = await dialog
    .getByRole("link")
    .evaluateAll((els) => els.map((el) => el.getAttribute("href")));
  expect(hrefs.length).toBeGreaterThan(0);
  return hrefs as string[];
}

test("every trip names a region on a day and a town on a stay", async ({ page }) => {
  for (const href of await adventureHrefs(page)) {
    await page.goto(href);
    const days = page.locator('section[aria-label="Days"]');
    await expect(days).toBeVisible();
    expect(await days.locator("[data-day-regions]").count()).toBeGreaterThan(0);
    const places = days.locator("[data-stay-place]");
    expect(await places.count()).toBeGreaterThan(0);
    // The two wordings, and nothing else: "near X" or "N km from X".
    for (const text of await places.allInnerTexts()) {
      expect(text).toMatch(/· (near |\d+ km from )\S/);
    }
    await expectNoHorizontalScroll(page);
  }
});

test("a town taken from Google's visit record says so when pressed", async ({ page }) => {
  for (const href of await adventureHrefs(page)) {
    await page.goto(href);
    const term = page.locator("[data-stay-place] button.term").first();
    if ((await term.count()) === 0) continue;
    const note = page.getByText(/Google's own record of the visit/).first();
    await expect(note).toBeHidden();
    await clickUntil(term, note);
    // The open note must not push the page sideways at any width.
    await expectNoHorizontalScroll(page);
    return;
  }
  throw new Error("the demo has stays named from the visit record; none found");
});

test("the place list's credit is in the drawing's account", async ({ page }) => {
  const [href] = await adventureHrefs(page);
  await page.goto(href);
  const details = page.locator("details", { hasText: "How this route was drawn" });
  const credit = details.locator("[data-place-credit]");
  await expect(credit).toBeHidden();
  await clickUntil(details.locator("summary"), credit);
  await expect(credit).toHaveText("Place names © OpenStreetMap contributors");
});
