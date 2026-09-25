// The detail plate's layer list, asserted over the same objects both maps
// draw (phase 14 CP3): every leg kind has its own layer with its own
// non-color channel, routed is cased, and the highlight dims rather than
// hides. The on-screen map and the exported image cannot drift from each
// other because they share this list — this test pins what the list says.
import { describe, expect, it } from "vitest";

import {
  DIM_OPACITY,
  ROUTE_LAYERS,
  ROUTE_SOURCE,
  highlightPaint,
  routeFeatures,
} from "@/lib/route-layers";
import { INK } from "@/lib/tokens";

const t = "2026-04-24T09:54:05Z";
const pt = (lat: number, lon: number) => ({ t, lat, lon });

function layer(id: string) {
  const l = ROUTE_LAYERS.find((x) => x.id === id);
  if (!l || !("paint" in l) || !l.paint) throw new Error(`no layer ${id}`);
  return l.paint as Record<string, unknown>;
}

describe("route layers — the four-way encoding at every zoom", () => {
  it("paints bottom-up: unknown, air, routed over casing, observed, fixes, stops", () => {
    expect(ROUTE_LAYERS.map((l) => l.id)).toEqual([
      "unknown-legs",
      "air-legs",
      "road-casing",
      "road-legs",
      "observed-legs",
      "fixes",
      "stops",
    ]);
    for (const l of ROUTE_LAYERS) expect("source" in l && l.source).toBe(ROUTE_SOURCE);
  });

  it("kind is never hue alone", () => {
    expect(layer("unknown-legs")["line-dasharray"]).toEqual([2, 3]);
    expect(layer("air-legs")["line-dasharray"]).toEqual([0.1, 2]);
    expect(layer("road-casing")["line-color"]).toBe(INK.paper);
    expect(layer("road-legs")["line-dasharray"]).toBeUndefined();
    expect(layer("observed-legs")["line-dasharray"]).toBeUndefined();
    expect(layer("observed-legs")["line-width"]).toBeGreaterThan(
      layer("road-legs")["line-width"] as number,
    );
    // No zoom expression anywhere: the detail plate keeps the full split.
    for (const l of ROUTE_LAYERS)
      expect(JSON.stringify(l.paint)).not.toContain('"zoom"');
  });

  it("a highlight dims the rest, never hides it", () => {
    for (const c of highlightPaint(2)) {
      expect(JSON.stringify(c.value)).toContain(String(DIM_OPACITY));
    }
    for (const c of highlightPaint(null)) expect(c.value).toBe(1);
    expect(DIM_OPACITY).toBeGreaterThan(0);
  });

  it("a stationary observed leg becomes a fix point; a (0,0) stop is absent", () => {
    const fc = routeFeatures({
      legs: [
        { kind: "observed", points: [pt(64, -21)], distance_km: 0, start: t, end: t },
        { kind: "gap", gap_kind: "unknown", points: [pt(64, -21), pt(64, -20)], distance_km: 10, start: t, end: t },
      ],
      stops: [
        { start: t, end: t, loc: { lat: 0, lon: 0 }, points: 0, displacement_km: 0 },
        { start: t, end: t, loc: { lat: 64, lon: -20 }, points: 3, displacement_km: 0 },
      ],
    });
    expect(fc.features.map((f) => f.properties?.kind)).toEqual([
      "observed",
      "gap",
      "fix",
      "stop",
    ]);
  });
});
