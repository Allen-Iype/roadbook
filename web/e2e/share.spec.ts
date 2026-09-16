// Share links (phase 13 CP4): the one spec in this suite that writes — it
// mints a link on the first adventure, opens it as a stranger (a fresh
// browser context: no cookies, no session), and revokes it again, leaving
// the stack as it found it. Because three projects would race each other
// on the same adventure's link list (revoke picks a row), the walk runs at
// the desktop project only and checks the phone layout of the shared page
// by resizing its own viewport — Playwright can, unlike driven Chrome.
import { expect, test, type Page } from "@playwright/test";

import {
  clickUntil,
  expectNoHorizontalScroll,
  legendLocator,
  openSummonedList,
  viewportWidth,
} from "./helpers";

async function gotoFirstAdventure(page: Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await page.goto(href!);
}

test("share link: mint, open signed-out, revoke", async ({ page, browser }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  await gotoFirstAdventure(page);

  // Mint. The dialog states what is shared before anything is created.
  const dialog = page.locator('dialog[aria-label="Share this adventure"]');
  await clickUntil(page.getByRole("button", { name: "Create a share link" }), dialog);
  await expect(dialog).toContainText("without signing in");
  await expect(dialog).toContainText("photo imports");
  await dialog.getByRole("button", { name: "Create link" }).click();
  const urlInput = dialog.getByLabel("Share link");
  await expect(urlInput).toBeVisible({ timeout: 10_000 });
  const url = await urlInput.inputValue();
  expect(url).toMatch(/\/shared\/[0-9a-f]{32}$/);
  await expect(dialog).toContainText("cannot be shown again");
  await dialog.getByRole("button", { name: "Done" }).click();
  await expect(page.getByText(/Link created \d+ \w+ \d{4}/).first()).toBeVisible();

  // Open as a stranger: a fresh context has no cookies at all.
  const stranger = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const view = await stranger.newPage();
  const response = await view.goto(url);
  expect(response?.status()).toBe(200);
  expect(response?.headers()["x-robots-tag"]).toContain("noindex");
  await expect(view.getByText("SHARED ADVENTURE", { exact: true })).toBeVisible();
  await expect(view.getByText("Read-only · shared by its owner")).toBeVisible();
  // The honesty channel survives to the stranger: the plate legend names
  // all four kinds, and the cover prints the provenance split.
  const legend = legendLocator(view);
  for (const label of ["Observed", "Routed", "Unknown", "Air"]) {
    await expect(legend.getByText(label, { exact: true })).toBeVisible();
  }
  await expect(view.getByText(/observed [\d.]+ km · routed/)).toBeVisible();
  await expect(view.locator(".maplibregl-canvas")).toBeAttached({ timeout: 20_000 });
  // No owner controls leak into the shared surface.
  await expect(view.getByRole("button", { name: "Create a share link" })).toHaveCount(0);
  await expect(view.getByRole("button", { name: "Sign out" })).toHaveCount(0);
  await expect(view.getByText("Candidates")).toHaveCount(0);
  await expectNoHorizontalScroll(view);
  await view.setViewportSize({ width: 390, height: 844 });
  await expectNoHorizontalScroll(view);

  // Revoke: the newest link is listed last; then the stranger's URL is dead.
  const revoke = page.getByRole("button", { name: "Revoke" }).last();
  const before = await page.getByRole("button", { name: "Revoke" }).count();
  await revoke.click();
  await expect(page.getByRole("button", { name: "Revoke" })).toHaveCount(before - 1, { timeout: 10_000 });
  // The page's notFound() streams under the root loading boundary, so the
  // HTTP status is 200 with the not-found content (the same transport fact
  // as NEXT_REDIRECT streaming as 200, phase 7); the API itself answers a
  // true 404 (internal/api/share_test.go). Assert the content, and that the
  // shared plate is gone.
  await view.goto(url);
  await expect(view.getByText("This link doesn't open anything.")).toBeVisible();
  await expect(view.getByText("SHARED ADVENTURE", { exact: true })).toHaveCount(0);
  await stranger.close();
});

test("shared view: a bad token is a designed not-found", async ({ page }) => {
  const response = await page.goto("/shared/00000000000000000000000000000000");
  expect(response?.headers()["x-robots-tag"]).toContain("noindex");
  await expect(page.getByText("This link doesn't open anything.")).toBeVisible();
  await expectNoHorizontalScroll(page);
});
