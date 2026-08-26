// The v1 loop pages, at every width: each loads and none scrolls
// sideways. Plus the structure each page owes the reader — the welcome
// page's walkthrough anchors (every rejection redirection must land
// somewhere real) and the two tables' breakpoint folding.
import { expect, test } from "@playwright/test";

import { WELCOME_SECTIONS } from "../lib/rejection-anchors";
import { MD, SM, expectNoHorizontalScroll, viewportWidth } from "./helpers";

test.describe("no horizontal scroll on any v1 loop page", () => {
  for (const path of ["/", "/welcome", "/adventures", "/candidates", "/imports"]) {
    test(`at ${path}`, async ({ page }) => {
      await page.goto(path);
      await expectNoHorizontalScroll(page);
    });
  }
});

test("welcome: every rejection anchor has a real target", async ({ page }) => {
  await page.goto("/welcome");
  // lib/rejection-anchors.ts types every sniffer label against this id
  // set; this closes the loop at runtime — the ids must exist in the DOM
  // or a rejection would send the reader to a dead anchor.
  for (const id of Object.values(WELCOME_SECTIONS)) {
    await expect(page.locator(`#${id}`), `#${id} must exist`).toBeAttached();
  }
});

test("welcome: both upload islands are present", async ({ page }) => {
  // Two doors since phase 11: the Timeline export upload and the photo
  // batch upload each carry their own file input.
  await page.goto("/welcome");
  await expect(page.locator('input[type="file"]')).toHaveCount(2);
});

test("candidates: every gallery card carries art, figures, and a decision", async ({ page }) => {
  // Triage is a gallery from phase 9 CP4 (T3, decided at the mockup STOP).
  // Cards do not fold — the same structure renders at every width; what
  // each card owes the decider is the route shape, the facts line with its
  // score, and the decide control.
  await page.goto("/candidates");
  const cards = page.locator("main ul > li");
  const n = await cards.count();
  expect(n).toBeGreaterThan(0);
  for (let i = 0; i < n; i++) {
    const card = cards.nth(i);
    await expect(card.locator("svg").first()).toBeVisible();
    await expect(card.getByText(/score (\d+|—)/)).toBeVisible();
    await expect(
      card.locator('a[href^="/adventure/"]').first(),
    ).toBeVisible();
  }
});

test("imports: format folds below sm, accounting below md", async ({ page }) => {
  await page.goto("/imports");
  const w = viewportWidth(page);
  const format = page.getByRole("columnheader", { name: "Format" });
  const window_ = page.getByRole("columnheader", { name: "Window" });
  if (w >= SM) {
    await expect(format).toBeVisible();
  } else {
    await expect(format).toBeHidden();
  }
  if (w >= MD) {
    await expect(window_).toBeVisible();
  } else {
    await expect(window_).toBeHidden();
  }
});
