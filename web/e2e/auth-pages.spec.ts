// The auth surfaces on the authless reference stack (phase 13): /signin
// does not exist as a surface — it redirects home — and no sign-in or
// sign-out control renders anywhere. The deletion control renders on
// /imports with its dialog, but is never submitted: the suite is read-only
// against a shared stack, and the Go tests own the deletion proof.
import { expect, test } from "@playwright/test";

import { clickUntil, expectNoHorizontalScroll } from "./helpers";

test("/signin redirects home on an authless instance", async ({ page }) => {
  await page.goto("/signin");
  await expect(page).not.toHaveURL(/\/signin/);
  await expect(page.getByRole("link", { name: "Sign in to continue" })).toHaveCount(0);
});

test("no account controls on the authless reference", async ({ page }) => {
  for (const path of ["/", "/adventures", "/candidates", "/imports"]) {
    await page.goto(path);
    await expect(page.getByRole("button", { name: "Sign out" })).toHaveCount(0);
  }
});

test("/imports: the deletion dialog arms only on the typed word", async ({ page }) => {
  await page.goto("/imports");
  const dialog = page.locator('dialog[aria-label="Delete everything"]');
  await clickUntil(page.getByRole("button", { name: "Delete everything…" }), dialog);
  const confirm = dialog.getByRole("button", { name: "Delete everything" });
  await expect(confirm).toBeDisabled();
  await dialog.getByLabel(/Type delete to confirm/).fill("delet");
  await expect(confirm).toBeDisabled();
  await dialog.getByLabel(/Type delete to confirm/).fill("delete");
  await expect(confirm).toBeEnabled();
  // Never submitted. Leave by the safe control.
  await dialog.getByRole("button", { name: "Keep my data" }).click();
  await expect(dialog).toBeHidden();
  await expectNoHorizontalScroll(page);
});
