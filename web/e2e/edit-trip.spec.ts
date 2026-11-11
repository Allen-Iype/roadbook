// A kept trip's name and notes (phase 17, issue #3). The second spec in
// this suite that writes: it renames the second kept trip and adds notes,
// reloads to prove they were stored, then puts the name back and clears
// the notes, leaving the stack as it found it. Desktop project only, like
// share.spec.ts, so three projects never race on one row; the second trip,
// so it never races share.spec.ts on the first.
import { expect, test, type Page } from "@playwright/test";

import { clickUntil, openSummonedList, viewportWidth } from "./helpers";

async function gotoSecondAdventure(page: Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").nth(1).getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await page.goto(href!);
}

test("edit name and notes: saved, shown after reload, then put back", async ({ page }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  await gotoSecondAdventure(page);
  // The trip's own heading: app/loading.tsx's h1 sits hidden in the DOM
  // for a moment while a reloaded page streams in.
  const heading = page.locator("article h1");
  const original = (await heading.textContent())?.trim() ?? "";
  expect(original).not.toBe("");
  const form = page.getByRole("form", { name: "Edit name and notes" });

  async function save(name: string, notes: string) {
    await clickUntil(page.getByRole("button", { name: /^Edit name/ }), form);
    await form.getByLabel("Name").fill(name);
    await form.getByLabel(/^Notes/).fill(notes);
    await form.getByRole("button", { name: "Save" }).click();
    await expect(form).toBeHidden({ timeout: 10_000 });
    await page.reload();
  }

  try {
    await save("Edited by the e2e walk", "First line.\nSecond line.");
    await expect(heading).toHaveText("Edited by the e2e walk");
    const notes = page.getByTestId("trip-notes");
    await expect(notes).toHaveText("First line.\nSecond line.");
    // The line break is kept as written, not run together. Measured in one
    // evaluate, and polled: a streamed page first delivers the node inside
    // a hidden container (height 0) before React moves it into place.
    await expect
      .poll(() =>
        notes.evaluate(
          (el) => el.getBoundingClientRect().height / parseFloat(getComputedStyle(el).lineHeight),
        ),
      )
      .toBeGreaterThan(1.9);
  } finally {
    // Put it back whatever happened above: the original name, no notes.
    await save(original, "");
  }
  await expect(heading).toHaveText(original);
  await expect(page.getByTestId("trip-notes")).toHaveCount(0);
});

test("a blank name cannot be saved, and Cancel changes nothing", async ({ page }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  await gotoSecondAdventure(page);
  const original = (await page.locator("article h1").textContent())?.trim() ?? "";

  const form = page.getByRole("form", { name: "Edit name and notes" });
  await clickUntil(page.getByRole("button", { name: /^Edit name/ }), form);
  await form.getByLabel("Name").fill("   ");
  await expect(form.getByRole("button", { name: "Save" })).toBeDisabled();
  await form.getByRole("button", { name: "Cancel" }).click();
  await expect(form).toBeHidden();
  await expect(page.locator("article h1")).toHaveText(original);
});
