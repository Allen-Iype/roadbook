// The v1 loop pages, at every width: each loads and none scrolls
// sideways. Plus the structure each page owes the reader — the welcome
// page's walkthrough anchors (every rejection redirection must land
// somewhere real) and the two tables' breakpoint folding.
import { expect, test } from "@playwright/test";

import { WELCOME_SECTIONS } from "../lib/rejection-anchors";
import { MD, SM, expectNoHorizontalScroll, viewportWidth } from "./helpers";

test.describe("no horizontal scroll on any v1 loop page", () => {
  for (const path of ["/", "/welcome", "/candidates", "/imports"]) {
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

test("welcome: the upload island is present", async ({ page }) => {
  await page.goto("/welcome");
  await expect(page.locator('input[type="file"]')).toBeAttached();
});

test("candidates: secondary columns fold below sm", async ({ page }) => {
  await page.goto("/candidates");
  const wide = viewportWidth(page) >= SM;
  for (const th of ["Track", "Score"]) {
    const header = page.getByRole("columnheader", { name: th });
    if (wide) {
      await expect(header).toBeVisible();
    } else {
      await expect(header).toBeHidden();
    }
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
