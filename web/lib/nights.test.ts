// Nights (phase 19): the trip block's Nights row and which stay row a
// night is said on — the rules on small hand-built inputs, then the three
// demo journeys as the API served them (fixtures/README.md), pinned.
import { describe, expect, it } from "vitest";

import {
  dateBefore,
  nightsSummary,
  stayNight,
  NIGHT_PLACES_SHOWN,
  type NightsInput,
} from "@/lib/nights";
import type { components } from "@/lib/api/schema";

import journey1 from "./fixtures/demo-journey-1.json";
import journey2 from "./fixtures/demo-journey-2.json";
import journey3 from "./fixtures/demo-journey-3.json";

type Journey = components["schemas"]["Journey"];
type Night = components["schemas"]["Night"];

const town = (name: string, km = 2) => ({
  start: "2026-05-22T18:00:00+00:00",
  place: { name, distance_km: km, near: km <= 10, source: "osm" },
});
const noTown = { start: "2026-05-22T18:00:00+00:00", place: undefined };
const at = (day: number, stop: number): Night => ({
  date: `2026-05-${String(day).padStart(2, "0")}`,
  kind: "stop",
  stop_index: stop,
  hours: 8,
});
const moving = (day: number): Night => ({ date: `2026-05-${day}`, kind: "moving", hours: 6 });
const unknown = (day: number): Night => ({ date: `2026-05-${day}`, kind: "unknown" });

describe("the Nights row", () => {
  it("a day trip has none", () => {
    expect(nightsSummary({ nights: [], stops: [] })).toEqual({ count: 0, parts: [] });
  });
  it("one place for every night drops the number", () => {
    const j: NightsInput = { stops: [town("A"), town("A")], nights: [at(22, 0), at(23, 1)] };
    expect(nightsSummary(j)).toEqual({ count: 2, parts: ["near A"] });
  });
  it("groups by town, most nights first, ties in the order reached", () => {
    const j: NightsInput = {
      stops: [town("A"), town("B"), town("B"), town("C")],
      nights: [at(22, 0), at(23, 1), at(24, 2), at(25, 3)],
    };
    expect(nightsSummary(j).parts).toEqual(["near B (2)", "near A", "near C"]);
  });
  it("a town never within the near distance is said by its closest night", () => {
    const j: NightsInput = {
      stops: [town("A", 18.4), town("A", 12.6), town("B")],
      nights: [at(22, 0), at(23, 1), at(24, 2)],
    };
    expect(nightsSummary(j).parts).toEqual(["13 km from A (2)", "near B"]);
  });
  it("on the move, not recorded and no town nearby are said, and the numbers add up", () => {
    const j: NightsInput = {
      stops: [town("A"), noTown],
      nights: [at(22, 0), at(23, 1), moving(24), unknown(25), unknown(26)],
    };
    const s = nightsSummary(j);
    expect(s.parts).toEqual(["near A", "no town nearby", "on the move", "not recorded (2)"]);
    const n = (p: string) => Number(/\((\d+)\)$/.exec(p)?.[1] ?? 1);
    expect(s.parts.reduce((a, p) => a + n(p), 0)).toBe(s.count);
  });
  it("a long tour names the first few towns and counts the rest as elsewhere", () => {
    const names = ["A", "B", "C", "D", "E", "F"];
    const j: NightsInput = {
      stops: names.map((n) => town(n)),
      nights: names.map((_, i) => at(10 + i, i)).concat([at(20, 0)]),
    };
    const s = nightsSummary(j);
    expect(s.parts).toEqual(["near A (2)", "near B", "near C", "elsewhere (3)"]);
    expect(s.parts.length).toBe(NIGHT_PLACES_SHOWN + 1);
  });
  it("only nights on the move reads without a number", () => {
    expect(nightsSummary({ stops: [], nights: [moving(22)] }).parts).toEqual(["on the move"]);
  });
});

describe("which stay row a night is said on", () => {
  it("the date before", () => {
    expect(dateBefore("2026-05-23")).toBe("2026-05-22");
    expect(dateBefore("2026-03-01")).toBe("2026-02-28");
    expect(dateBefore("2026-01-01")).toBe("2025-12-31");
  });
  const stops = [
    { start: "2026-05-22T19:00:00+00:00", place: undefined }, // evening arrival
    { start: "2026-05-24T02:00:00+00:00", place: undefined }, // arrives after midnight
  ];
  const j: NightsInput = { stops, nights: [at(22, 0), at(23, 1)] };
  it("a stay from the evening says it on the day the night began", () => {
    expect(stayNight(j, 0, "2026-05-22")).toBe("tonight");
    expect(stayNight(j, 0, "2026-05-23")).toBeUndefined();
  });
  it("a late arrival says it on its own first day", () => {
    expect(stayNight(j, 1, "2026-05-24")).toBe("lastNight");
    expect(stayNight(j, 1, "2026-05-23")).toBe("tonight");
  });
  it("a stop that holds no night says nothing", () => {
    expect(stayNight({ stops, nights: [unknown(22)] }, 0, "2026-05-22")).toBeUndefined();
  });
});

describe("the demo journeys, as served", () => {
  const row = (j: Journey) => {
    const s = nightsSummary(j);
    return `${s.count} · ${s.parts.join(", ")}`;
  };
  it("South coast drive", () => {
    expect(row(journey1 as Journey)).toMatchInlineSnapshot(`"2 · near Höfn, no town nearby"`);
  });
  it("Westfjords loop", () => {
    expect(row(journey2 as Journey)).toMatchInlineSnapshot(`"2 · near Ísafjörður"`);
  });
  it("Akureyri weekend", () => {
    expect(row(journey3 as Journey)).toMatchInlineSnapshot(`"2 · near Akureyri"`);
  });
});
