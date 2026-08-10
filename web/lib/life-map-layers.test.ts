// The invariant-8 regression statement for T2 (DESIGN §2, BRIEF §4.2):
//
//   At no zoom does an unknown leg share the known-ground style.
//
// The life map merges observed and routed into one known-ground ink at
// overview zooms — that is T2's deliberate graduation. The boundary that
// makes it honest is that unknown and air NEVER join the merge: their color
// is a constant that is not the known-ground ink, and their dash pattern is
// present at every zoom. These tests assert that over the exact layer
// objects the island hands to MapLibre, so a future restyle that lets an
// inference class drift toward confident ground fails CI, not a reader.
import { describe, expect, it } from "vitest";

import {
  HIT_LAYER,
  T2_MERGE_MAX_ZOOM,
  T2_SPLIT_ZOOM,
  lifeMapLayers,
} from "@/lib/life-map-layers";
import { INK } from "@/lib/tokens";

const layers = lifeMapLayers();
const byId = (id: string) => {
  const layer = layers.find((l) => l.id === id);
  if (!layer) throw new Error(`layer ${id} missing`);
  return layer;
};

// Every string that appears anywhere in a value — for expression-typed
// paint, this walks the expression tree, so a color hidden inside a nested
// ["interpolate", ...] stop is still found.
function stringsIn(value: unknown): string[] {
  if (typeof value === "string") return [value];
  if (Array.isArray(value)) return value.flatMap(stringsIn);
  if (value && typeof value === "object") {
    return Object.values(value).flatMap(stringsIn);
  }
  return [];
}

describe("invariant 8: unknown and air never merge into known ground", () => {
  for (const id of ["life-unknown", "life-air"]) {
    const layer = byId(id);
    const paint = (layer as { paint?: Record<string, unknown> }).paint ?? {};

    it(`${id}: color is a constant — no zoom expression at any zoom`, () => {
      // A literal string is constant at every zoom by construction. This is
      // the structural form of "at no zoom": there is no zoom input at all.
      expect(typeof paint["line-color"]).toBe("string");
    });

    it(`${id}: color is never the known-ground ink (nor a split ink)`, () => {
      for (const s of stringsIn(paint["line-color"])) {
        expect(s.toLowerCase()).not.toBe(INK.known.toLowerCase());
        expect(s.toLowerCase()).not.toBe(INK.observed.toLowerCase());
        expect(s.toLowerCase()).not.toBe(INK.routed.toLowerCase());
      }
    });

    it(`${id}: dash pattern is present and constant at all zooms`, () => {
      const dash = paint["line-dasharray"];
      expect(Array.isArray(dash)).toBe(true);
      // Constant numeric array — not a zoom-driven expression that could
      // dissolve the dashes into a solid line somewhere.
      for (const v of dash as unknown[]) expect(typeof v).toBe("number");
      expect((dash as number[]).length).toBeGreaterThanOrEqual(2);
      // A dash pattern with no gap is a solid line wearing a dash's name.
      expect((dash as number[])[1]).toBeGreaterThan(0);
    });
  }

  it("the known-ground ink appears only on the graduating layers", () => {
    for (const layer of layers) {
      const hasKnown = stringsIn(layer).some(
        (s) => s.toLowerCase() === INK.known.toLowerCase(),
      );
      const graduates = layer.id === "life-observed" || layer.id === "life-routed";
      expect(hasKnown, `${layer.id} vs known ink`).toBe(graduates);
    }
  });
});

describe("the T2 graduation itself", () => {
  for (const [id, ink] of [
    ["life-observed", INK.observed],
    ["life-routed", INK.routed],
  ] as const) {
    it(`${id}: merged to known ground below z${T2_MERGE_MAX_ZOOM}, own ink at z${T2_SPLIT_ZOOM}`, () => {
      const paint = (byId(id) as { paint?: Record<string, unknown> }).paint!;
      const color = paint["line-color"] as unknown[];
      // Shape: ["interpolate", ["linear"], ["zoom"], MERGE, known, SPLIT, ink]
      expect(color[0]).toBe("interpolate");
      expect(color[2]).toEqual(["zoom"]);
      expect(color[3]).toBe(T2_MERGE_MAX_ZOOM);
      expect(color[4]).toBe(INK.known);
      expect(color[5]).toBe(T2_SPLIT_ZOOM);
      expect(color[6]).toBe(ink);
    });
  }

  it("the routed casing is invisible while merged", () => {
    const paint = (byId("life-routed-casing") as { paint?: Record<string, unknown> })
      .paint!;
    const opacity = paint["line-opacity"] as unknown[];
    expect(opacity[0]).toBe("interpolate");
    expect(opacity[3]).toBe(T2_MERGE_MAX_ZOOM);
    expect(opacity[4]).toBe(0);
  });

  it("the hit area renders nothing", () => {
    const paint = (byId(HIT_LAYER) as { paint?: Record<string, unknown> }).paint!;
    expect(paint["line-opacity"]).toBe(0);
  });
});
