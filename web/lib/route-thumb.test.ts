// The thumbnail builder's regression table. The invariant-8 statement at
// thumbnail scale: every path carries its leg kind and no two kinds merge
// into one path — plus the projection truths (everything inside the
// viewBox, aspect preserved via the shared scale, air legs actually arc).
import { describe, expect, it } from "vitest";

import { routeThumb } from "@/lib/route-thumb";
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];

const t = "2026-04-01T10:00:00Z";
const pt = (lat: number, lon: number) => ({ t, lat, lon });

const observed = (coords: [number, number][]): Leg => ({
  kind: "observed",
  points: coords.map(([lat, lon]) => pt(lat, lon)),
  distance_km: 10,
  start: t,
  end: t,
});

const gap = (
  gapKind: "unknown" | "road" | "air",
  a: [number, number],
  b: [number, number],
  routed?: [number, number][],
): Leg => ({
  kind: "gap",
  gap_kind: gapKind,
  points: [pt(...a), pt(...b)],
  distance_km: 100,
  ...(routed
    ? { routed_points: routed.map(([lat, lon]) => ({ lat, lon })) }
    : {}),
  start: t,
  end: t,
});

const FOUR_KINDS: Leg[] = [
  observed([
    [64.1, -21.9],
    [64.3, -21.2],
    [64.6, -20.5],
  ]),
  gap("road", [64.6, -20.5], [65.0, -19.4], [
    [64.6, -20.5],
    [64.8, -20.0],
    [65.0, -19.4],
  ]),
  gap("unknown", [65.0, -19.4], [65.6, -18.1]),
  gap("air", [65.6, -18.1], [64.1, -21.9]),
];

// Every "x y" pair drawn in a path's d attribute.
function coordsOf(d: string): [number, number][] {
  return [...d.matchAll(/[ML]([\d.]+) ([\d.]+)/g)].map((m) => [
    Number(m[1]),
    Number(m[2]),
  ]);
}

describe("routeThumb", () => {
  it("keeps all four kinds as separate paths — nothing merges", () => {
    const thumb = routeThumb(FOUR_KINDS)!;
    expect(thumb.paths.map((p) => p.kind).sort()).toEqual([
      "air",
      "observed",
      "road",
      "unknown",
    ]);
  });

  it("paints unknown under air under road under observed", () => {
    const thumb = routeThumb(FOUR_KINDS)!;
    expect(thumb.paths.map((p) => p.kind)).toEqual([
      "unknown",
      "air",
      "road",
      "observed",
    ]);
  });

  it("projects every coordinate inside the padded viewBox", () => {
    const thumb = routeThumb(FOUR_KINDS, 224, 120, 8)!;
    for (const p of thumb.paths)
      for (const [x, y] of coordsOf(p.d)) {
        expect(x).toBeGreaterThanOrEqual(8 - 0.05);
        expect(x).toBeLessThanOrEqual(224 - 8 + 0.05);
        expect(y).toBeGreaterThanOrEqual(8 - 0.05);
        expect(y).toBeLessThanOrEqual(120 - 8 + 0.05);
      }
  });

  it("draws the air leg as an arc, not a chord", () => {
    const thumb = routeThumb(FOUR_KINDS)!;
    const air = thumb.paths.find((p) => p.kind === "air")!;
    // 24 interpolation segments → 25 points; a chord would have 2.
    expect(coordsOf(air.d).length).toBeGreaterThan(10);
  });

  it("road rides its routed polyline, not its endpoints", () => {
    const thumb = routeThumb(FOUR_KINDS)!;
    const road = thumb.paths.find((p) => p.kind === "road")!;
    expect(coordsOf(road.d).length).toBe(3);
  });

  it("north is up: greater latitude projects to smaller y", () => {
    const thumb = routeThumb([
      observed([
        [64.0, -21.0],
        [66.0, -21.0],
      ]),
    ])!;
    const [[, ySouth], [, yNorth]] = coordsOf(thumb.paths[0].d);
    expect(yNorth).toBeLessThan(ySouth);
  });

  it("preserves aspect: a wide route does not fill the thumb's height", () => {
    // 2° of longitude at ~64°N vs 0.2° of latitude — much wider than tall.
    const thumb = routeThumb(
      [
        observed([
          [64.0, -22.0],
          [64.2, -20.0],
        ]),
      ],
      224,
      120,
      8,
    )!;
    const ys = thumb.paths.flatMap((p) => coordsOf(p.d)).map(([, y]) => y);
    const used = Math.max(...ys) - Math.min(...ys);
    expect(used).toBeLessThan((120 - 16) / 2);
  });

  it("returns null when nothing is drawable", () => {
    expect(routeThumb([])).toBeNull();
    expect(routeThumb([observed([[64.1, -21.9]])])).toBeNull();
  });

  it("survives a degenerate zero-extent journey", () => {
    const thumb = routeThumb([
      observed([
        [64.1, -21.9],
        [64.1, -21.9],
      ]),
    ]);
    expect(thumb).not.toBeNull();
    for (const [x, y] of coordsOf(thumb!.paths[0].d)) {
      expect(Number.isFinite(x)).toBe(true);
      expect(Number.isFinite(y)).toBe(true);
    }
  });
});
