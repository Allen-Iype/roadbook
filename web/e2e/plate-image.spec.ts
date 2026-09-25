// "Download as image" (phase 14 CP3, BRIEF §6): the export produces a real
// PNG of 2400×1600 whose map area is not all paper and whose margin has
// ink — on the owner's page and on a share link opened signed-out — and a
// basemap the browser cannot read yields the worded error, never a silent
// nothing. The suite's second writing spec (after share.spec.ts): the
// shared-view walk mints a link and revokes it again, so it runs at the
// desktop project only, and self-cleans — on the LAST adventure in the
// summoned list, because share.spec.ts works the first one and the suite
// runs fully parallel: two writers counting "Revoke" buttons on the same
// page race each other (seen once at CP3). Writing specs must not share
// an adventure.
//
// Pixels are read by the browser, not decoded in node: the downloaded file
// is handed back to the page as a data URL, drawn on a 2D canvas, and
// sampled with getImageData — no PNG decoder dependency. This is a check
// on the export's OWN canvas (a 2D composition), not on the WebGL canvas
// the config warns about; the map bitmap inside it was read with the
// drawing buffer preserved, which is the point under test.
import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";

import { clickUntil, openSummonedList, viewportWidth } from "./helpers";

const PAPER = [0xf5, 0xf2, 0xe8] as const;

async function gotoAdventure(page: Page, which: "first" | "last") {
  await page.goto("/");
  const dialog = await openSummonedList(page);
  const links = dialog.getByRole("link");
  const href = await (which === "first" ? links.first() : links.last()).getAttribute("href");
  expect(href).toMatch(/^\/adventure\/\d+$/);
  await page.goto(href!);
}

type Sample = {
  width: number;
  height: number;
  /** Fraction of pixels on a row through the map area that are not paper. */
  mapRowNonPaper: number;
  /** Fraction of pixels in the margin band that are darker than paper. */
  marginInk: number;
};

async function samplePng(page: Page, file: string): Promise<Sample> {
  const b64 = readFileSync(file).toString("base64");
  return page.evaluate(
    async ([data, paper]) => {
      const img = new Image();
      img.src = `data:image/png;base64,${data}`;
      await img.decode();
      const c = document.createElement("canvas");
      c.width = img.naturalWidth;
      c.height = img.naturalHeight;
      const ctx = c.getContext("2d")!;
      ctx.drawImage(img, 0, 0);
      const isPaper = (r: number, g: number, b: number) =>
        Math.abs(r - paper[0]) < 6 && Math.abs(g - paper[1]) < 6 && Math.abs(b - paper[2]) < 6;
      // A row through the middle of the map slot (MAP_RECT y 32..560 → the
      // file's 64..1120): basemap land/sea and route inks, never paper.
      const rowY = Math.round(c.height * 0.35);
      const row = ctx.getImageData(0, rowY, c.width, 1).data;
      let nonPaper = 0;
      for (let i = 0; i < row.length; i += 4)
        if (!isPaper(row[i], row[i + 1], row[i + 2])) nonPaper++;
      // The margin band (below the frame): text ink on paper.
      const y0 = Math.round(c.height * 0.72);
      const band = ctx.getImageData(0, y0, c.width, Math.round(c.height * 0.26)).data;
      let ink = 0;
      for (let i = 0; i < band.length; i += 4)
        if (band[i] + band[i + 1] + band[i + 2] < 3 * 160) ink++;
      return {
        width: c.width,
        height: c.height,
        mapRowNonPaper: nonPaper / c.width,
        marginInk: ink / (band.length / 4),
      };
    },
    [b64, PAPER] as const,
  );
}

async function downloadPlate(page: Page): Promise<Sample> {
  const status = page
    .getByTestId("plate-export-status")
    .filter({ hasText: /./ });
  const downloadPromise = page.waitForEvent("download", { timeout: 90_000 });
  // The island's button exists before hydration; clicking until the
  // status line appears is the honest wait (clickUntil's contract).
  await clickUntil(page.getByRole("button", { name: "Download as image" }), status);
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^roadbook-plate-[a-z0-9-]+\.png$/);
  await expect(status).toContainText(`Downloaded ${download.suggestedFilename()}`, {
    timeout: 30_000,
  });
  const file = await download.path();
  expect(file).toBeTruthy();
  const sample = await samplePng(page, file!);
  expect([sample.width, sample.height]).toEqual([2400, 1600]);
  expect(sample.mapRowNonPaper, "the map row is drawn, not paper").toBeGreaterThan(0.5);
  expect(sample.marginInk, "the margin carries text").toBeGreaterThan(0.002);
  return sample;
}

test("owner page: Download as image yields a 2400×1600 plate", async ({ page }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  await gotoAdventure(page, "first");
  // The control states, before anything renders, what the image carries.
  await expect(page.getByText("Photos are not drawn.")).toBeVisible();
  await downloadPlate(page);
});

test("shared view: the same control, signed out", async ({ page, browser }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  await gotoAdventure(page, "last");

  const dialog = page.locator('dialog[aria-label="Share this adventure"]');
  await clickUntil(page.getByRole("button", { name: "Create a share link" }), dialog);
  await dialog.getByRole("button", { name: "Create link" }).click();
  const urlInput = dialog.getByLabel("Share link");
  await expect(urlInput).toBeVisible({ timeout: 10_000 });
  const url = await urlInput.inputValue();
  await dialog.getByRole("button", { name: "Done" }).click();

  const stranger = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    acceptDownloads: true,
  });
  const view = await stranger.newPage();
  try {
    await view.goto(url);
    await expect(view.getByText("SHARED ADVENTURE", { exact: true })).toBeVisible();
    await downloadPlate(view);
  } finally {
    // Self-clean: the newest link is listed last; the proof of revocation
    // is the stranger's URL going dead, not a button count (counts race
    // any other writer on the page).
    await page.getByRole("button", { name: "Revoke" }).last().click();
    await expect
      .poll(async () => {
        await view.goto(url);
        return view.getByText("This link doesn't open anything.").isVisible();
      }, { timeout: 15_000 })
      .toBe(true);
    await stranger.close();
  }
});

test("a basemap the browser cannot read is a worded error, not a silent nothing", async ({ page }) => {
  test.skip(viewportWidth(page) !== 1280, "runs once, at the desktop project");
  // Every basemap resource fails at the network layer — what a tile
  // server without CORS headers looks like from inside a browser (fetch
  // rejects; MapLibre reports status 0). The on-screen map fails too; the
  // page must still stand and the export must say what happened.
  await page.route("**/tiles.openfreemap.org/**", (route) => route.abort("failed"));
  await gotoAdventure(page, "first");
  const status = page.getByTestId("plate-export-status").filter({ hasText: /./ });
  let downloaded = false;
  page.on("download", () => {
    downloaded = true;
  });
  await clickUntil(page.getByRole("button", { name: "Download as image" }), status);
  await expect(status).toContainText("could not be read", { timeout: 60_000 });
  await expect(status).toContainText("cross-origin");
  await expect(status).toContainText("Nothing was downloaded");
  await expect(status).toHaveAttribute("role", "alert");
  expect(downloaded).toBe(false);
  // The control recovers: it is enabled again for a retry.
  await expect(page.getByRole("button", { name: "Download as image" })).toBeEnabled();
});
