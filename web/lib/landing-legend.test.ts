import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { describe, expect, it } from "vitest";

import { LEGEND_ENTRIES } from "./legend";

// The public landing page (landing/index.html) is a static file outside the
// app, so nothing in the build makes it read lib/legend.ts. Its legend is
// typed by hand — which is how the pre-phase-16 page kept "recorded fixes"
// for a month after the product's words changed. This pins the landing's
// legend to the one constant every other legend reads (DESIGN §6), and the
// landing's two structural promises with it: no script, and the domain left
// as the deploy-time placeholder (phase 15 BRIEF §9).
const LANDING = resolve(__dirname, "..", "..", "landing", "index.html");
const HOW = resolve(__dirname, "..", "..", "landing", "how-its-built.html");

describe("the landing page", () => {
  const html = readFileSync(LANDING, "utf8");

  it.each(LEGEND_ENTRIES)("carries the legend entry %o verbatim", (e) => {
    expect(html).toContain(`<b>${e.label}</b>`);
    expect(html).toContain(e.desc);
  });

  it("runs no script", () => {
    expect(html).not.toMatch(/<script\b/i);
  });

  it("names no domain of its own", () => {
    expect(html).toMatch(/https:\/\/app\.\{ROADBOOK_DOMAIN\}\/signin/);
    expect(html).not.toMatch(/getroadbook/);
  });
});

describe("the how-it's-built page", () => {
  const html = readFileSync(HOW, "utf8");

  it.each(LEGEND_ENTRIES)("explains kind %o under its own name", (e) => {
    expect(html).toContain(`>${e.label}<`);
  });

  it("runs no script and names no domain", () => {
    expect(html).not.toMatch(/<script\b/i);
    expect(html).not.toMatch(/getroadbook/);
  });
});
