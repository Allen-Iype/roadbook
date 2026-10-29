// The v1 loop pages, at every width: each loads and none scrolls
// sideways. Plus the structure each page owes the reader — the welcome
// page's walkthrough anchors (every rejection redirection must land
// somewhere real) and the two tables' breakpoint folding.
import { expect, test } from "@playwright/test";

import { WELCOME_SECTIONS } from "../lib/rejection-anchors";
import { MD, SM, clickUntil, expectNoHorizontalScroll, viewportWidth } from "./helpers";

test.describe("no horizontal scroll on any v1 loop page", () => {
  for (const path of ["/", "/welcome", "/adventures", "/candidates", "/imports"]) {
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

test("welcome: both upload islands are present", async ({ page }) => {
  // Two doors since phase 11: the Timeline export upload and the photo
  // batch upload each carry their own file input.
  await page.goto("/welcome");
  await expect(page.locator('input[type="file"]')).toHaveCount(2);
});

test("welcome: the primary action sits under the three steps and is a thumb-sized target", async ({ page }) => {
  // Phase 16 CP2 (BRIEF §4): one primary action, the Timeline picker,
  // directly under "your own, in three steps" — above the walkthroughs,
  // not after them — and at least 44px tall at every width.
  await page.goto("/welcome");
  const button = page.getByRole("button", { name: "Choose the Timeline file" });
  await expect(button).toBeVisible();
  const box = await button.boundingBox();
  expect(box, "button must lay out").not.toBeNull();
  expect(box!.height).toBeGreaterThanOrEqual(44);
  const steps = page.getByRole("heading", { name: "Your own, in three steps" });
  const stepsBox = await steps.boundingBox();
  const walkthroughs = page.locator(`#${WELCOME_SECTIONS.export}`);
  const walkBox = await walkthroughs.boundingBox();
  expect(stepsBox!.y).toBeLessThan(box!.y);
  expect(box!.y).toBeLessThan(walkBox!.y);
  // The photos door is present and second.
  const photos = page.getByRole("button", { name: "Choose photos" });
  await expect(photos).toBeVisible();
  expect((await photos.boundingBox())!.y).toBeGreaterThan(box!.y);
});

test("welcome: the folded sections start closed and a rejection anchor opens its fold", async ({ page }) => {
  // The fold is the point (RESEARCH §2); the one script on the page
  // (components/open-on-hash.tsx) opens a <details> when a rejection link
  // lands on it, so "What the right file looks like" is never a dead end
  // behind a closed summary.
  await page.goto("/welcome");
  const theFile = page.locator(`details#${WELCOME_SECTIONS.theFile}`);
  await expect(theFile).toBeAttached();
  await expect(theFile).not.toHaveAttribute("open", "");
  for (const id of [WELCOME_SECTIONS.exportAndroid, WELCOME_SECTIONS.exportIphone, WELCOME_SECTIONS.next]) {
    await expect(page.locator(`details#${id}`)).not.toHaveAttribute("open", "");
  }
  await page.goto(`/welcome#${WELCOME_SECTIONS.theFile}`);
  await expect(theFile).toHaveAttribute("open", "");
  await expect(theFile.getByText("Timeline.json")).toBeVisible();
  // The walkthroughs keep their verification state visible before the fold.
  await expect(page.locator(`#${WELCOME_SECTIONS.exportAndroid} summary`)).toContainText(/Verified \d+ \w+ \d{4}/);
  await expect(page.locator(`#${WELCOME_SECTIONS.exportIphone} summary`)).toContainText("Unverified");
});

test("candidates: every gallery card carries art, figures, and a decision", async ({ page }) => {
  // Triage is a gallery from phase 9 CP4 (T3, decided at the mockup STOP).
  // Cards do not fold — the same structure renders at every width; what
  // each card owes the decider is the route shape, the facts line with its
  // score, and the decide control.
  await page.goto("/candidates");
  const cards = page.locator("main ul > li");
  const n = await cards.count();
  expect(n).toBeGreaterThan(0);
  for (let i = 0; i < n; i++) {
    const card = cards.nth(i);
    await expect(card.locator("svg").first()).toBeVisible();
    await expect(card.getByText(/score (\d+|—)/)).toBeVisible();
    await expect(
      card.locator('a[href^="/adventure/"]').first(),
    ).toBeVisible();
  }
});

test("candidates: bulk triage — selection summons the bar; score order sorts", async ({ page }) => {
  // Read-only walk of phase 11 §6.1's surfaces: ticking a checkbox is
  // client state (no server mutation), so the bar's appearance is safe to
  // drive; its action buttons are never clicked here — the Go suite owns
  // the mutation path.
  await page.goto("/candidates");
  const boxes = page.locator('main input[type="checkbox"]');
  const n = await boxes.count();
  expect(n).toBeGreaterThan(0);

  const bar = page.getByRole("region", { name: "Bulk decision" });
  await expect(bar).toBeHidden();
  // clickUntil, not .check(): a click before hydration silently no-ops
  // (the phase 7 trap) — clicking until the bar appears waits it out.
  // Clicks toggle, so land on an odd count by pairing each retry.
  await clickUntil(boxes.first(), bar);
  await expect(bar).toBeVisible();
  await expect(bar.getByText("1 selected")).toBeVisible();
  await expect(
    bar.getByRole("button", { name: "Dismiss selected" }),
  ).toBeVisible();
  await bar.getByRole("button", { name: "Clear" }).click();
  await expect(bar).toBeHidden();

  // Select-all: scoped to undecided candidates, and the count says so.
  const selectAllBtn = page.getByRole("button", {
    name: /^Select all \d+ undecided$/,
  });
  if (await selectAllBtn.isVisible()) {
    const m = (await selectAllBtn.textContent())?.match(/(\d+)/);
    const undecided = Number(m?.[1] ?? 0);
    await selectAllBtn.click();
    await expect(bar.getByText(`${undecided} selected`)).toBeVisible();
    await bar.getByRole("button", { name: "Clear" }).click();
    await expect(bar).toBeHidden();
  }

  // The sweep order: ?sort=score renders scores non-increasing.
  await page.goto("/candidates?sort=score");
  // The score element, not card textContent: concatenated text runs the
  // score into the next figure ("score 100" + "28 days" → "10028").
  const scores = await page
    .locator('main ul > li [title^="Confidence"]')
    .evaluateAll((els) =>
      els.map((el) => {
        const m = el.textContent?.match(/score (\d+|—)/);
        return m && m[1] !== "—" ? Number(m[1]) : -1;
      }),
    );
  expect(scores.length).toBeGreaterThan(0);
  for (let i = 1; i < scores.length; i++) {
    expect(scores[i]).toBeLessThanOrEqual(scores[i - 1]);
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
