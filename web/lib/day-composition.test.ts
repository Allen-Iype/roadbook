// dayComposition (phase 16 CP1): the day heading's provenance bar must
// account for exactly the day's kilometres, by kind — on the three demo
// journeys as the API served them (fixtures/README.md), and on a
// fixes-only day (no distance, all observed).
import { describe, expect, it } from "vitest";

import { dayComposition, dayPaceKmh, sliceDays } from "@/lib/slice-days";
import type { components } from "@/lib/api/schema";

import journey1 from "./fixtures/demo-journey-1.json";
import journey2 from "./fixtures/demo-journey-2.json";
import journey3 from "./fixtures/demo-journey-3.json";

type Journey = components["schemas"]["Journey"];

const journeys: [string, Journey][] = [
  ["South coast drive", journey1 as Journey],
  ["Westfjords loop", journey2 as Journey],
  ["Akureyri weekend", journey3 as Journey],
];

describe("dayComposition sums to the day's km, by kind", () => {
  for (const [name, j] of journeys) {
    it(name, () => {
      for (const day of sliceDays(j)) {
        const c = dayComposition(day, j);
        expect(c.observed + c.routed + c.unknown + c.air).toBeCloseTo(day.km, 9);
        for (const v of Object.values(c)) expect(v).toBeGreaterThanOrEqual(0);
      }
    });
  }

  it("a fixes-only day is all observed and 0 km", () => {
    const j = {
      legs: [
        { kind: "observed", points: [{ t: "2026-05-22T15:10:00+00:00", lat: 65, lon: -22 }], distance_km: 0, start: "2026-05-22T15:10:00+00:00", end: "2026-05-22T15:10:00+00:00" },
      ],
    } as unknown as Pick<Journey, "legs">;
    expect(dayComposition({ legIndices: [0] }, j)).toEqual({ observed: 0, routed: 0, unknown: 0, air: 0 });
  });

  it("a road gap counts its routed km, an unrouted gap its chord", () => {
    const j = {
      legs: [
        { kind: "gap", gap_kind: "road", points: [], distance_km: 10, routed_km: 14, start: "", end: "" },
        { kind: "gap", gap_kind: "unknown", points: [], distance_km: 5, start: "", end: "" },
        { kind: "gap", gap_kind: "air", points: [], distance_km: 300, start: "", end: "" },
      ],
    } as unknown as Pick<Journey, "legs">;
    expect(dayComposition({ legIndices: [0, 1, 2] }, j)).toEqual({ observed: 0, routed: 14, unknown: 5, air: 300 });
  });
});

describe("dayPaceKmh mirrors the served pace, day by day", () => {
  for (const [name, j] of journeys) {
    it(`${name}: the days' moving observed legs reproduce summary.observed_pace_kmh`, () => {
      let km = 0;
      let hours = 0;
      for (const day of sliceDays(j)) {
        const pace = dayPaceKmh(day, j);
        if (pace === undefined) continue;
        // Recover the day's km and hours the way the helper summed them.
        for (const i of day.legIndices) {
          const leg = j.legs[i];
          if (leg.kind !== "observed" || leg.points.length === 1 || leg.distance_km === 0) continue;
          km += leg.distance_km;
          hours += (Date.parse(leg.end) - Date.parse(leg.start)) / 3_600_000;
        }
      }
      if (j.summary.observed_pace_kmh === undefined) {
        expect(hours).toBe(0);
      } else {
        expect(km / hours).toBeCloseTo(j.summary.observed_pace_kmh, 6);
      }
    });
  }

  it("a fixes-only day has no pace (absent, never zero)", () => {
    const j = {
      legs: [
        { kind: "observed", points: [{ t: "2026-05-22T15:10:00+00:00", lat: 65, lon: -22 }], distance_km: 0, start: "2026-05-22T15:10:00+00:00", end: "2026-05-22T15:10:00+00:00" },
      ],
    } as unknown as Pick<Journey, "legs">;
    expect(dayPaceKmh({ legIndices: [0] }, j)).toBeUndefined();
  });

  it("drawn-in stretches and flights never enter a day's pace", () => {
    const j = {
      legs: [
        { kind: "observed", points: [{}, {}], distance_km: 60, start: "2026-05-22T10:00:00+00:00", end: "2026-05-22T11:00:00+00:00" },
        { kind: "gap", gap_kind: "road", points: [], distance_km: 100, routed_km: 140, start: "2026-05-22T11:00:00+00:00", end: "2026-05-22T12:00:00+00:00" },
        { kind: "gap", gap_kind: "air", points: [], distance_km: 300, start: "2026-05-22T12:00:00+00:00", end: "2026-05-22T13:00:00+00:00" },
      ],
    } as unknown as Pick<Journey, "legs">;
    expect(dayPaceKmh({ legIndices: [0, 1, 2] }, j)).toBeCloseTo(60, 9);
  });
});
