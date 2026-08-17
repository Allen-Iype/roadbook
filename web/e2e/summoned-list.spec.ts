// The summoned list is the accessible path to adventures (DESIGN §5): the
// map canvas is aria-hidden decoration, so this dialog's focus behaviour
// is load-bearing for keyboard and screen-reader users, and it is the
// primary navigation on small screens. The native <dialog> promises focus
// trap, Escape, and focus return — promises are exactly what regress, so
// they are asserted here rather than trusted.
import { expect, test } from "@playwright/test";

import { openSummonedList } from "./helpers";

test("List (n) opens the dialog and focus moves into it", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  await expect(dialog).toBeVisible();
  // showModal() moves focus into the dialog; anywhere inside it counts.
  const focusInside = await dialog.evaluate(
    (el) => el.contains(document.activeElement) || el === document.activeElement,
  );
  expect(focusInside, "focus must land inside the dialog").toBe(true);
  // The demo dataset has confirmed adventures; each is a link.
  await expect(dialog.getByRole("link").first()).toBeVisible();
});

test("Escape closes the dialog and focus returns to the button", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  const focusOnButton = await page.evaluate(
    () =>
      document.activeElement instanceof HTMLButtonElement &&
      /^List \(\d+\)$/.test(document.activeElement.textContent ?? ""),
  );
  expect(focusOnButton, "focus must return to the List button").toBe(true);
});

test("an adventure link navigates and closes the dialog", async ({ page }) => {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const first = dialog.getByRole("link").first();
  const href = await first.getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await first.click();
  await page.waitForURL(/\/adventure\/\d+$/);
  await expect(dialog).toBeHidden();
});
