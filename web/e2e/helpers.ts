import { expect, type Locator, type Page } from "@playwright/test";

// Tailwind's default breakpoints, the ones the folding tables key on.
export const SM = 640;
export const MD = 768;

export function viewportWidth(page: Page): number {
  const vp = page.viewportSize();
  if (!vp) throw new Error("projects always set a viewport");
  return vp.width;
}

// The core layout truth, asserted on every page at every width: the
// document never scrolls sideways. Individual wide elements (tables, code)
// must scroll inside their own container instead.
export async function expectNoHorizontalScroll(page: Page) {
  const overflow = await page.evaluate(() => {
    const el = document.scrollingElement ?? document.documentElement;
    return el.scrollWidth - el.clientWidth;
  });
  expect(overflow, "document must not scroll horizontally").toBeLessThanOrEqual(0);
}

// Server components render a client island's initial HTML, so its buttons
// exist before hydration — and a click before hydration silently does
// nothing (the phase 7 trap, recorded in that phase's log). Clicking until
// the expected effect appears is the honest way to wait for hydration
// without reaching into framework internals.
export async function clickUntil(
  trigger: Locator,
  effect: Locator,
  tries = 5,
): Promise<void> {
  for (let i = 0; i < tries; i++) {
    await trigger.click();
    try {
      await expect(effect).toBeVisible({ timeout: 2_000 });
      return;
    } catch {
      // island not hydrated yet — click again
    }
  }
  await expect(effect).toBeVisible();
}

// The leg-kind legend renders as one <ul aria-label="Legend"> holding the
// four kind entries; each name is a Term button (its popover note repeats
// the name, so specs match the label BY ROLE, never by text alone), with a
// drawn SVG sample and its description visible at every width (phase 16).
// Scoping to it matters: the
// welcome pitch prose legitimately reuses legend phrases, and text
// filtering is case-insensitive, so the adventure disclosure's lowercase
// composition line ("observed 700.2 km · routed …") would also match.
export function legendLocator(page: Page): Locator {
  return page.locator('ul[aria-label="Legend"]').first();
}

// Opens the life map's summoned list ("List (n)") and returns the dialog.
export async function openSummonedList(page: Page): Promise<Locator> {
  const button = page.getByRole("button", { name: /^List \(\d+\)$/ });
  const dialog = page.locator('dialog[aria-label="Confirmed adventures"]');
  await clickUntil(button, dialog);
  return dialog;
}
