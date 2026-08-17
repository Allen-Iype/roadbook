// Phase 7's tap-target rule, made permanent: on a phone, every control a
// thumb must hit presents at least a 44 px target (achieved with
// negative-margin padding, so the visual size stays small while the
// touchable box grows — which is exactly why this must be measured, not
// eyeballed). Phone viewport only: on wider screens a pointer is assumed.
//
// Measurement is one atomic evaluateAll snapshot. Per-button
// isVisible()/boundingBox() round-trips race React hydration and produced
// flaky reads while the suite was being built; a single DOM pass cannot.
//
// The first run of this suite caught the decided-state "change" /
// "keep as is" buttons at 40 px (phase 9 CP1 finding, fixed in CP2): the
// phase 7 verification only ever saw the undecided state's buttons.
import { expect, test, type Locator, type Page } from "@playwright/test";

import { clickUntil, viewportWidth } from "./helpers";

const MIN_TARGET_PX = 44;

async function expectThumbSized(scope: Page | Locator, selector: string) {
  const controls = scope.locator(selector);
  await expect(controls.first()).toBeVisible();
  const measured = await controls.evaluateAll((els) =>
    els
      .filter((el) => el.getClientRects().length > 0)
      .map((el) => ({
        text: (el.textContent ?? "").trim().slice(0, 40),
        height: el.getBoundingClientRect().height,
      })),
  );
  expect(measured.length).toBeGreaterThan(0);
  for (const m of measured) {
    expect
      .soft(
        m.height,
        `"${m.text}" must present a ≥${MIN_TARGET_PX}px target`,
      )
      .toBeGreaterThanOrEqual(MIN_TARGET_PX);
  }
}

test.beforeEach(({ page }) => {
  test.skip(viewportWidth(page) >= 640, "tap targets are a phone concern");
});

test("candidates: table buttons and header links are thumb-sized", async ({ page }) => {
  await page.goto("/candidates");
  await expectThumbSized(page, "table button");
  await expectThumbSized(page, "header a");
});

test("life map: the chrome nav and the dialog controls are thumb-sized", async ({ page }) => {
  await page.goto("/");
  await expectThumbSized(page, "nav button, nav a");
  // Inside the summoned list — the phone's primary navigation (DESIGN §5):
  // every adventure link and the close control.
  const button = page.getByRole("button", { name: /^List \(\d+\)$/ });
  const dialog = page.locator('dialog[aria-label="Confirmed adventures"]');
  await clickUntil(button, dialog);
  await expectThumbSized(dialog, "a, button");
});

test("adventure: the day-heading map toggles are thumb-sized", async ({ page }) => {
  await page.goto("/");
  const button = page.getByRole("button", { name: /^List \(\d+\)$/ });
  const dialog = page.locator('dialog[aria-label="Confirmed adventures"]');
  await clickUntil(button, dialog);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  await page.goto(href!);
  await expectThumbSized(page, 'section[aria-label="Days"] h2 button');
});
