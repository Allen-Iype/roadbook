// Phase 7's tap-target rule, made permanent: on a phone, every button in
// the triage table presents at least a 44 px hit target (achieved there
// with negative-margin padding, so the visual size stays small while the
// touchable box grows — which is exactly why this must be measured, not
// eyeballed). Phone viewport only: on wider screens a pointer is assumed.
//
// Measurement is one atomic evaluateAll snapshot. Per-button
// isVisible()/boundingBox() round-trips race React hydration and produced
// flaky reads while the suite was being built; a single DOM pass cannot.
import { expect, test } from "@playwright/test";

import { viewportWidth } from "./helpers";

const MIN_TARGET_PX = 44;

test("candidates: every table button is thumb-sized on a phone", async ({ page }) => {
  test.skip(viewportWidth(page) >= 640, "tap targets are a phone concern");
  // CP1 finding (2026-08-17, the harness's first catch): the decided
  // state's "change" / "keep as is" buttons measure 40 px — text-xs (16px
  // line) + py-3 (24px) — while phase 7's verified 44 px only covered the
  // undecided state's text-sm confirm/dismiss. The CP2 refinement sweep
  // fixes the product; removing this test.fail() is part of that fix.
  test.fail(true, "known 40px 'change' button, fixed in phase 9 CP2");
  await page.goto("/candidates");
  const buttons = page.locator("table button");
  await expect(buttons.first()).toBeVisible();
  const measured = await buttons.evaluateAll((els) =>
    els
      .filter((el) => el.getClientRects().length > 0)
      .map((el) => ({
        text: (el.textContent ?? "").trim(),
        height: el.getBoundingClientRect().height,
      })),
  );
  expect(measured.length, "the demo dataset renders decide buttons").toBeGreaterThan(0);
  for (const m of measured) {
    expect(
      m.height,
      `button "${m.text}" must present a ≥${MIN_TARGET_PX}px target`,
    ).toBeGreaterThanOrEqual(MIN_TARGET_PX);
  }
});
