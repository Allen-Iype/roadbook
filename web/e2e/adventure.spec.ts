// The adventure detail page, reached the way a user reaches it (through
// the summoned list — the suite never hardcodes a candidate id, because
// ids change with every detection run). Layout truths plus the floor the
// page keeps at every width (phase 16 BRIEF §1 "progressive disclosure
// with a floor"): the legend with its four kinds, the provenance bar under
// the headline distance, the drawing's account closed by default, and a
// flagged divergence visible outside it. Canvas PIXELS are deliberately
// not asserted — see playwright.config.ts.
import { expect, test } from "@playwright/test";

import {
  expectNoHorizontalScroll,
  legendLocator,
  openSummonedList,
} from "./helpers";

async function gotoFirstAdventure(page: import("@playwright/test").Page) {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const href = await dialog.getByRole("link").first().getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await page.goto(href!);
}

test("adventure page: no horizontal scroll", async ({ page }) => {
  await gotoFirstAdventure(page);
  await expectNoHorizontalScroll(page);
});

test("adventure page: the plate legend names all four kinds", async ({ page }) => {
  await gotoFirstAdventure(page);
  const legend = legendLocator(page);
  for (const label of ["Observed", "Routed", "Unknown", "Air"]) {
    await expect(legend.getByRole("button", { name: label, exact: true })).toBeVisible();
  }
});

test("adventure page: the floor — bar, caption, disclosure closed, flag outside", async ({ page }) => {
  await gotoFirstAdventure(page);
  const trip = page.locator('section[aria-label="The trip"]');
  await expect(trip).toBeVisible();
  // The provenance bar is aria-hidden (its words are the caption and the
  // disclosure), so it is found by its data hook, not a role.
  await expect(trip.locator("[data-provenance-bar]")).toBeAttached();
  await expect(
    trip.getByRole("button", { name: "How much of this was recorded, how much drawn in" }),
  ).toBeVisible();

  const details = page.locator("details", { hasText: "How this route was drawn" });
  await expect(details).toBeVisible();
  await expect(details).toHaveJSProperty("open", false);
  // The composition line is inside, hidden until opened.
  const composition = details.getByText(/observed [\d.]+ km · routed/);
  await expect(composition).toBeHidden();
  await details.locator("summary").click();
  await expect(details).toHaveJSProperty("open", true);
  await expect(composition).toBeVisible();

  // When the distance check is flagged, its flag line sits OUTSIDE the
  // disclosure and is visible with it closed (at the page foot, above
  // it). Not every adventure is flagged; the assertion holds wherever the
  // line exists.
  await details.locator("summary").click();
  const flag = page.getByText(/differ by \d+% — the account below says why/);
  if ((await flag.count()) > 0) {
    await expect(flag.first()).toBeVisible();
    expect(await details.getByText(/differ by \d+%/).count()).toBe(0);
  }
});

test("adventure page: each day is a heading, a line, and stay rows", async ({ page }) => {
  await gotoFirstAdventure(page);
  const days = page.locator('section[aria-label="Days"] > section');
  await expect(days.first()).toBeVisible();
  const shape = await days.evaluateAll((els) =>
    els.map((sec) => ({
      heading: sec.querySelector("h2 button")?.textContent?.trim() ?? "",
      line: sec.querySelector("[data-day-line]")?.textContent?.trim() ?? "",
      rows: Array.from(sec.querySelectorAll("ol > li")).map((li) => ({
        glyph: li.children[0]?.querySelector("svg, span") !== null,
        time: (li.children[1]?.textContent ?? "").trim().length > 0,
        words: (li.children[2]?.textContent ?? "").trim(),
      })),
      text: sec.textContent ?? "",
    })),
  );
  expect(shape.length).toBeGreaterThan(0);
  for (const d of shape) {
    expect(d.heading).toMatch(/^DAY \d+/);
    expect(d.line.length).toBeGreaterThan(0);
    for (const r of d.rows) {
      expect(r.glyph).toBe(true);
      expect(r.time).toBe(true);
      expect(r.words).toMatch(/^(Stayed|Moved on)/);
    }
    // Coordinates left the narrative (phase 16 BRIEF §4): no "65.71°N".
    expect(d.text).not.toMatch(/°[NSEW]/);
  }
});

test("adventure page: the map mounts", async ({ page }) => {
  await gotoFirstAdventure(page);
  await expect(page.locator(".maplibregl-canvas")).toBeAttached({
    timeout: 20_000,
  });
});

test("life map: the map mounts", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".maplibregl-canvas")).toBeAttached({
    timeout: 20_000,
  });
});
