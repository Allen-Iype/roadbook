// Day places (phase 18): the regions phrase on the day line and the town
// on a stay row — the wording rules on small hand-built inputs, then the
// three demo journeys as the API served them (fixtures/README.md), pinned.
import { describe, expect, it } from "vitest";

import {
  dayRegions,
  placePhrase,
  regionsPhrase,
  stayPhrases,
  type PlacesInput,
} from "@/lib/day-places";
import { sliceDays } from "@/lib/slice-days";
import type { components } from "@/lib/api/schema";

import journey1 from "./fixtures/demo-journey-1.json";
import journey2 from "./fixtures/demo-journey-2.json";
import journey3 from "./fixtures/demo-journey-3.json";

type Journey = components["schemas"]["Journey"];

const st = (name: string) => ({ code: name, name, country_code: "XX" });
const at = (h: number) => `2026-05-22T${String(h).padStart(2, "0")}:00:00+00:00`;
const leg = (h: number, ...names: string[]) => ({ start: at(h), states: names.map(st) });
const stop = (h: number, name?: string) => ({
  start: at(h),
  state: name === undefined ? undefined : st(name),
});
const phrase = (j: PlacesInput) =>
  regionsPhrase(
    dayRegions(
      { legIndices: j.legs.map((_, i) => i), stopIndices: j.stops.map((_, i) => i) },
      j,
    ),
  );

describe("the regions phrase", () => {
  it("one region is its name", () => {
    expect(phrase({ legs: [leg(8, "A"), leg(10, "A")], stops: [stop(9, "A")] })).toBe("A");
  });
  it("two regions read from one to the other", () => {
    expect(phrase({ legs: [leg(8, "A", "B")], stops: [] })).toBe("A to B");
  });
  it("three or more are a list, in the order reached", () => {
    expect(phrase({ legs: [leg(8, "A", "B"), leg(12, "B", "C")], stops: [] })).toBe("A, B, C");
  });
  it("out and back says so instead of repeating the first", () => {
    expect(phrase({ legs: [leg(8, "A", "B"), leg(15, "B", "A")], stops: [] })).toBe(
      "A to B and back",
    );
  });
  it("a road along a border does not stutter", () => {
    expect(phrase({ legs: [leg(8, "A", "B", "A", "B")], stops: [] })).toBe("A to B");
  });
  it("a stay that began before the day's first leg comes first", () => {
    expect(phrase({ legs: [leg(9, "B")], stops: [stop(2, "A")] })).toBe("A to B");
  });
  it("no region anywhere is no phrase", () => {
    expect(phrase({ legs: [leg(8)], stops: [stop(9)] })).toBeUndefined();
  });
});

describe("the place phrase", () => {
  const place = (name: string, distance_km: number, near: boolean) => ({
    name,
    distance_km,
    near,
    source: "osm",
  });
  it("near within the near distance, as the API decided", () => {
    expect(placePhrase(place("Höfn", 9.9, true))).toBe("near Höfn");
  });
  it("whole kilometres beyond it", () => {
    expect(placePhrase(place("Höfn", 15.6, false))).toBe("16 km from Höfn");
  });
  it("nothing when the stay has no name", () => {
    expect(placePhrase(undefined)).toBeUndefined();
  });
  it("a run of stays near one town names it once; no name ends the run", () => {
    const stops = [
      { place: place("A", 1, true) },
      { place: place("A", 2, true) },
      { place: place("A", 12, false) },
      { place: undefined },
      { place: place("A", 1, true) },
      { place: place("B", 1, true) },
    ].map((s) => ({ ...s, start: at(1), state: undefined }));
    expect(stayPhrases([0, 1, 2, 3, 4, 5], { stops })).toEqual([
      "near A",
      undefined,
      "12 km from A",
      undefined,
      "near A",
      "near B",
    ]);
  });
});

describe("the demo journeys, as served", () => {
  const lines = (j: Journey) =>
    sliceDays(j).map((d) => regionsPhrase(dayRegions(d, j)) ?? null);
  it("South coast drive", () => {
    expect(lines(journey1 as Journey)).toMatchInlineSnapshot(`
      [
        "Suðurland",
        "Suðurland to Austurland",
        "Austurland to Suðurland",
      ]
    `);
  });
  it("Westfjords loop", () => {
    expect(lines(journey2 as Journey)).toMatchInlineSnapshot(`
      [
        "Vesturland to Vestfirðir",
        "Vestfirðir",
        "Vestfirðir to Vesturland",
      ]
    `);
  });
  it("Akureyri weekend", () => {
    expect(lines(journey3 as Journey)).toMatchInlineSnapshot(`
      [
        "Suðurnes to Norðurland eystra",
        "Norðurland eystra",
        "Norðurland eystra to Suðurnes",
      ]
    `);
  });
});
