// Cross-language parity (phase 14 CP2): the cover's day count is the SERVED
// figure, computed in Go by the civil-day rule and printed by the CLI; the
// narrative still slices days in TypeScript. Two implementations of one
// rule is how they eventually disagree, so this test holds real journeys —
// the three demo adventures exactly as the API served them (fictional
// Reykjavík persona; see fixtures/README.md) — against both: the number of
// days the narrative renders must equal summary.civil_days.
import { describe, expect, it } from "vitest";

import { sliceDays } from "@/lib/slice-days";
import { fmtHours } from "@/lib/format";
import type { components } from "@/lib/api/schema";

import journey1 from "./fixtures/demo-journey-1.json";
import journey2 from "./fixtures/demo-journey-2.json";
import journey3 from "./fixtures/demo-journey-3.json";

type Journey = components["schemas"]["Journey"];

const journeys: [string, Journey][] = [
  ["South coast drive (dense, observed)", journey1 as Journey],
  ["Westfjords loop (sparse, fixes only)", journey2 as Journey],
  ["Akureyri weekend (flights)", journey3 as Journey],
];

describe("summary.civil_days agrees with sliceDays on real journeys", () => {
  for (const [name, j] of journeys) {
    it(name, () => {
      expect(sliceDays(j)).toHaveLength(j.summary.civil_days);
    });
  }
});

describe("summary figures the fixtures pin", () => {
  it("a fixes-only journey has no pace, not a zero one", () => {
    const j = journey2 as Journey;
    expect(j.legs.every((l) => l.kind === "gap" || l.distance_km === 0)).toBe(true);
    expect(j.summary.observed_pace_kmh).toBeUndefined();
    expect(j.summary.observed_hours).toBeUndefined();
  });
  it("mode hours ride beside mode km", () => {
    const j = journey3 as Journey;
    expect(j.mode_breakdown?.map((m) => m.mode)).toEqual([
      "FLYING",
      "IN_PASSENGER_VEHICLE",
    ]);
    for (const m of j.mode_breakdown ?? []) expect(m.hours).toBeGreaterThan(0);
  });
});

describe("fmtHours", () => {
  it("reads minutes, hours, then days", () => {
    expect(fmtHours(0.5)).toBe("30 min");
    expect(fmtHours(9.667)).toBe("9 h 40 min");
    expect(fmtHours(24)).toBe("1 d");
    expect(fmtHours(45.9)).toBe("1 d 22 h");
    expect(fmtHours(47.99)).toBe("2 d");
  });
});
