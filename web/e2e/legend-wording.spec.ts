// DESIGN §6: the legend's fixed wording — "Observed — recorded fixes ·
// Routed — inferred along roads · Unknown — straight line, nothing
// inferred · Air — great-circle arc". The wordy legend lives on the life
// map and /welcome; its descriptions yield at phone width (the drawn
// samples and kind names still carry the channel) and return from sm: up.
// Both halves of that behaviour are the layout truth pinned here. The
// adventure plate's compact variant is covered in adventure.spec.ts.
import { expect, test } from "@playwright/test";

import { SM, legendLocator, viewportWidth } from "./helpers";

const DESCRIPTIONS = [
  "recorded fixes",
  "inferred along roads",
  "straight line, nothing inferred",
  "great-circle arc",
];

for (const path of ["/", "/welcome"]) {
  test(`the wordy legend at ${path}`, async ({ page }) => {
    await page.goto(path);
    const legend = legendLocator(page);
    for (const label of ["Observed", "Routed", "Unknown", "Air"]) {
      await expect(legend.getByText(label, { exact: true })).toBeVisible();
    }
    const wordy = viewportWidth(page) >= SM;
    for (const desc of DESCRIPTIONS) {
      const el = legend.getByText(desc);
      if (wordy) {
        await expect(el).toBeVisible();
      } else {
        await expect(el).toBeHidden();
      }
    }
  });
}
