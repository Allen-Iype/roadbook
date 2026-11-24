// Phase 23 (#9): which line a leg is drawn with. Written before legLine
// existed. The rule under test is invariant 6 as amended: a recorded
// stretch may be drawn along the road its points lie on, it is still a
// recorded stretch, and a gap never borrows that geometry.
import { describe, expect, it } from "vitest";

import type { components } from "@/lib/api/schema";
import { legFeatures, legLine } from "@/lib/geo";

type Leg = components["schemas"]["Leg"];

const t = "2026-07-01T10:00:00Z";
const pt = (lat: number, lon: number) => ({ lat, lon, t, source: "trace" as const });

const recorded: Leg = {
  kind: "observed",
  states: [],
  points: [pt(10.0, 70.0), pt(10.01, 70.0), pt(10.02, 70.0)],
  distance_km: 2.2,
  start: t,
  end: t,
};

// The road: starts and ends a little off the recorded ends, as a matched
// line does (the matcher places each point on the nearest road).
const road = [
  { lat: 10.0001, lon: 70.0004 },
  { lat: 10.012, lon: 70.003 },
  { lat: 10.0199, lon: 70.0003 },
];

describe("legLine", () => {
  it("draws a recorded stretch point to point when it has no road", () => {
    expect(legLine(recorded)).toEqual([
      [70.0, 10.0],
      [70.0, 10.01],
      [70.0, 10.02],
    ]);
  });

  it("draws a recorded stretch along its road, joined to its own first and last recorded points", () => {
    const line = legLine({ ...recorded, snapped_points: road, snapped_km: 2.4 });
    expect(line).toEqual([
      [70.0, 10.0], // the first recorded point: the neighbouring leg ends here
      [70.0004, 10.0001],
      [70.003, 10.012],
      [70.0003, 10.0199],
      [70.0, 10.02], // the last recorded point
    ]);
  });

  it("does not repeat an end the road already starts or ends on", () => {
    const exact = [{ lat: 10.0, lon: 70.0 }, { lat: 10.012, lon: 70.003 }, { lat: 10.02, lon: 70.0 }];
    expect(legLine({ ...recorded, snapped_points: exact, snapped_km: 2.4 })).toHaveLength(3);
  });

  it("ignores a road of fewer than two points", () => {
    expect(legLine({ ...recorded, snapped_points: [road[0]] })).toHaveLength(3);
    expect(legLine({ ...recorded, snapped_points: [] })).toHaveLength(3);
  });

  it("never gives a gap the recorded-stretch road, whatever the response carries", () => {
    const gap: Leg = {
      kind: "gap",
      gap_kind: "unknown",
      states: [],
      points: [pt(10.0, 70.0), pt(10.2, 70.0)],
      distance_km: 22,
      snapped_points: road,
      start: t,
      end: t,
    };
    expect(legLine(gap)).toEqual([
      [70.0, 10.0],
      [70.0, 10.2],
    ]);
  });

  it("still draws a routed gap along its routed geometry", () => {
    const gap: Leg = {
      kind: "gap",
      gap_kind: "road",
      states: [],
      points: [pt(10.0, 70.0), pt(10.2, 70.0)],
      distance_km: 22,
      routed_points: road,
      start: t,
      end: t,
    };
    expect(legLine(gap)).toHaveLength(3);
  });
});

describe("legFeatures", () => {
  it("keeps the kind of a stretch drawn along its road: it is still recorded", () => {
    const [f] = legFeatures([{ ...recorded, snapped_points: road, snapped_km: 2.4 }]);
    expect(f.properties).toMatchObject({ kind: "observed", gap_kind: null });
    expect((f.geometry as GeoJSON.LineString).coordinates).toHaveLength(5);
  });
});
