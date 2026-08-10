// Table tests for sliceDays (phase 6 BRIEF §5: date arithmetic across civil
// offsets is the checkpoint's likeliest bug source, so the rules are pinned
// here before any page consumes them). The fixtures are hand-built minimal
// journeys plus the Westfjords demo journey exactly as the demo API serves
// it — fictional data, committable.
import { describe, expect, it } from "vitest";

import {
  dayLabel,
  fmtDateRange,
  sliceDays,
  type SliceInput,
} from "@/lib/slice-days";

type Leg = SliceInput["legs"][number];
type Stop = SliceInput["stops"][number];

// Minimal constructors: only the fields sliceDays reads are meaningful; the
// rest satisfy the API types.
function pt(t: string, lat = 65.0, lon = -21.0) {
  return { t, lat, lon };
}

function observed(start: string, end: string, km: number, points: number): Leg {
  return {
    kind: "observed",
    points: Array.from({ length: points }, (_, i) =>
      pt(i === 0 ? start : end),
    ),
    distance_km: km,
    start,
    end,
  };
}

function gap(
  gapKind: "unknown" | "road" | "air",
  start: string,
  end: string,
  chordKm: number,
  routedKm?: number,
): Leg {
  return {
    kind: "gap",
    gap_kind: gapKind,
    points: [pt(start), pt(end)],
    distance_km: chordKm,
    ...(routedKm !== undefined ? { routed_km: routedKm } : {}),
    start,
    end,
  };
}

function stop(
  start: string,
  end: string,
  loc: { lat: number; lon: number } = { lat: 65.7, lon: -21.7 },
): Stop {
  return { start, end, loc, points: 1, displacement_km: 0 };
}

function journey(
  windowStart: string,
  windowEnd: string,
  legs: Leg[],
  stops: Stop[] = [],
): SliceInput {
  return { window_start: windowStart, window_end: windowEnd, legs, stops };
}

// The Westfjords demo journey, verbatim from the demo API (candidate 2):
// two sparse road gaps, single-fix observed legs between them, and a
// position-unobserved dwell spanning two midnights inside the second gap.
const westfjords = journey(
  "2026-05-22T15:10:00Z",
  "2026-05-24T13:01:00Z",
  [
    observed("2026-05-22T15:10:00Z", "2026-05-22T15:10:00Z", 0, 1),
    gap("road", "2026-05-22T15:10:00Z", "2026-05-22T17:30:00Z", 130.294, 159.9702),
    observed("2026-05-22T17:30:00Z", "2026-05-22T17:30:00Z", 0, 1),
    gap("road", "2026-05-22T17:30:00Z", "2026-05-24T13:00:00Z", 66.877, 80.2611),
    observed("2026-05-24T13:00:00Z", "2026-05-24T13:01:00Z", 0, 2),
  ],
  [stop("2026-05-22T20:00:00Z", "2026-05-24T10:00:00Z", { lat: 0, lon: 0 })],
);

describe("sliceDays — Westfjords (the acceptance journey)", () => {
  const days = sliceDays(westfjords);

  it("cuts the window into three civil days", () => {
    expect(days.map((d) => d.date)).toEqual([
      "2026-05-22",
      "2026-05-23",
      "2026-05-24",
    ]);
    expect(days.map((d) => d.index)).toEqual([1, 2, 3]);
    expect(days.map((d) => d.label)).toEqual([
      "Friday 22 May",
      "Saturday 23 May",
      "Sunday 24 May",
    ]);
  });

  it("Day 1: window edge, departure fix, both transits (start-day rule), dwell begins", () => {
    expect(days[0].events.map((e) => e.type)).toEqual([
      "windowEdge",
      "fix",
      "transit",
      "fix",
      "transit",
      "dwellBegin",
    ]);
    const transits = days[0].events.filter((e) => e.type === "transit");
    // Drawn km is the routed figure; the chord stays available for the
    // "straight line n km" statement.
    expect(transits[0]).toMatchObject({
      kind: "road",
      km: 159.9702,
      chordKm: 130.294,
      overnight: null,
    });
    // The second gap crosses two midnights: it appears once, under its
    // start day, noted as ending on Day 3 (BRIEF §3A — never split).
    expect(transits[1]).toMatchObject({ kind: "road", overnight: 3 });
    const begin = days[0].events.find((e) => e.type === "dwellBegin");
    expect(begin).toMatchObject({ stopIndex: 0, endsDay: 3 });
  });

  it("Day 2 is honest: the dwell's intermediate day, and nothing else", () => {
    expect(days[1].events).toEqual([{ type: "dwellDay", stopIndex: 0 }]);
    expect(days[1].km).toBe(0);
    expect(days[1].legIndices).toEqual([]);
  });

  it("Day 3: dwell ends with its true duration, arrival fixes, window closes", () => {
    expect(days[2].events.map((e) => e.type)).toEqual([
      "dwellEnd",
      "fix",
      "windowEdge",
    ]);
    const end = days[2].events.find((e) => e.type === "dwellEnd");
    // Fri 20:00 → Sun 10:00 is 38 h exactly.
    expect(end).toMatchObject({ stopIndex: 0, minutes: 38 * 60 });
    const fix = days[2].events.find((e) => e.type === "fix");
    expect(fix).toMatchObject({ legIndex: 4, points: 2 });
  });

  it("per-day km sums drawn distances by start day", () => {
    expect(days[0].km).toBeCloseTo(159.9702 + 80.2611, 6);
    expect(days[1].km).toBe(0);
    expect(days[2].km).toBe(0);
  });

  it("map highlight uses the same assignment: legs by start day, stops by overlap", () => {
    expect(days[0].legIndices).toEqual([0, 1, 2, 3]);
    expect(days[2].legIndices).toEqual([4]);
    // The dwell overlaps all three civil days.
    expect(days.map((d) => d.stopIndices)).toEqual([[0], [0], [0]]);
  });
});

describe("sliceDays — midnight rule", () => {
  it("a transit crossing midnight belongs to its start day, once", () => {
    const days = sliceDays(
      journey("2026-05-22T23:00:00Z", "2026-05-23T01:00:00Z", [
        gap("unknown", "2026-05-22T23:59:00Z", "2026-05-23T00:30:00Z", 12),
      ]),
    );
    expect(days).toHaveLength(2);
    expect(days[0].events.filter((e) => e.type === "transit")).toHaveLength(1);
    expect(days[1].events.filter((e) => e.type === "transit")).toHaveLength(0);
    expect(days[0].km).toBe(12);
    expect(days[1].km).toBe(0);
    const t = days[0].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({ overnight: 2 });
  });

  it("a transit ending exactly at 00:00 counts as ending on the next civil day", () => {
    const days = sliceDays(
      journey("2026-05-22T20:00:00Z", "2026-05-23T01:00:00Z", [
        gap("unknown", "2026-05-22T22:00:00Z", "2026-05-23T00:00:00Z", 5),
      ]),
    );
    const t = days[0].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({ overnight: 2 });
  });

  it("a transit starting at 00:00 belongs to that day with no overnight note", () => {
    const days = sliceDays(
      journey("2026-05-22T20:00:00Z", "2026-05-23T23:00:00Z", [
        gap("unknown", "2026-05-23T00:00:00Z", "2026-05-23T02:00:00Z", 5),
      ]),
    );
    const t = days[1].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({ overnight: null });
    expect(days[1].km).toBe(5);
  });
});

describe("sliceDays — the journey's own offsets, not UTC", () => {
  it("civil days follow the recorded offset", () => {
    // 23:45 → 00:30 local (+05:30) is 18:15Z → 19:00Z — one UTC day, but
    // two civil days as the traveller lived them.
    const days = sliceDays(
      journey("2026-05-22T23:00:00+05:30", "2026-05-23T01:00:00+05:30", [
        gap(
          "unknown",
          "2026-05-22T23:45:00+05:30",
          "2026-05-23T00:30:00+05:30",
          8,
        ),
      ]),
    );
    expect(days.map((d) => d.date)).toEqual(["2026-05-22", "2026-05-23"]);
    const t = days[0].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({ overnight: 2 });
  });

  it("durations are real elapsed time even across offset changes", () => {
    // A 2 h dwell whose end clock reads 30 min earlier than its start
    // (offset change mid-dwell): minutes come from the instants.
    const days = sliceDays(
      journey(
        "2026-05-22T10:00:00+05:30",
        "2026-05-22T18:00:00+04:00",
        [],
        [stop("2026-05-22T12:00:00+05:30", "2026-05-22T12:30:00+04:00")],
      ),
    );
    const d = days[0].events.find((e) => e.type === "dwell");
    expect(d).toMatchObject({ minutes: 120 });
  });

  it("a westward overnight flight may end on an earlier civil day — stated, not crashed", () => {
    const days = sliceDays(
      journey("2026-05-22T20:00:00+05:30", "2026-05-23T23:00:00-08:00", [
        gap(
          "air",
          "2026-05-23T01:00:00+05:30",
          "2026-05-22T23:00:00-08:00",
          9000,
        ),
      ]),
    );
    // The leg starts on the 23rd local and ends on the 22nd local: it lives
    // under Day 2 and honestly notes it ends on Day 1.
    const t = days[1].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({ kind: "air", overnight: 1 });
  });
});

describe("sliceDays — dwells", () => {
  it("a same-day dwell is one event with its duration", () => {
    const days = sliceDays(
      journey(
        "2026-05-22T08:00:00Z",
        "2026-05-22T20:00:00Z",
        [],
        [stop("2026-05-22T12:00:00Z", "2026-05-22T14:30:00Z")],
      ),
    );
    expect(days[0].events.filter((e) => e.type === "dwell")).toEqual([
      {
        type: "dwell",
        stopIndex: 0,
        start: "2026-05-22T12:00:00Z",
        end: "2026-05-22T14:30:00Z",
        minutes: 150,
      },
    ]);
  });
});

describe("sliceDays — tie at the same instant", () => {
  it("a dwell's end precedes the transit departing at that minute", () => {
    // Seen on real data: overnight dwell ends 06:48, observed run starts
    // 06:48 — the narrative must end the dwell before it departs.
    const days = sliceDays(
      journey(
        "2026-05-22T18:00:00Z",
        "2026-05-23T12:00:00Z",
        [observed("2026-05-23T06:48:00Z", "2026-05-23T09:03:00Z", 25, 30)],
        [stop("2026-05-22T21:00:00Z", "2026-05-23T06:48:00Z")],
      ),
    );
    expect(days[1].events.map((e) => e.type)).toEqual([
      "dwellEnd",
      "transit",
      "windowEdge",
    ]);
  });
});

describe("sliceDays — days nothing describes", () => {
  it("a day inside a multi-day gap with no dwell says the transit is under way", () => {
    const days = sliceDays(
      journey("2026-05-22T20:00:00Z", "2026-05-24T10:00:00Z", [
        gap("unknown", "2026-05-22T22:00:00Z", "2026-05-24T08:00:00Z", 40),
      ]),
    );
    expect(days[1].events).toEqual([
      { type: "transitDay", legIndex: 0, kind: "unknown" },
    ]);
  });

  it("a day nothing spans at all stays empty — absence rendered as absence", () => {
    const days = sliceDays(
      journey("2026-05-22T08:00:00Z", "2026-05-24T20:00:00Z", [
        observed("2026-05-22T09:00:00Z", "2026-05-22T10:00:00Z", 30, 12),
        observed("2026-05-24T09:00:00Z", "2026-05-24T10:00:00Z", 25, 9),
      ]),
    );
    expect(days[1].events).toEqual([]);
  });
});

describe("sliceDays — fixes vs transits", () => {
  it("a zero-km observed leg is a fix, a moving one is a transit", () => {
    const days = sliceDays(
      journey("2026-05-22T08:00:00Z", "2026-05-22T20:00:00Z", [
        observed("2026-05-22T09:00:00Z", "2026-05-22T09:00:00Z", 0, 1),
        observed("2026-05-22T10:00:00Z", "2026-05-22T12:00:00Z", 45.2, 38),
      ]),
    );
    expect(days[0].events.map((e) => e.type)).toEqual([
      "windowEdge",
      "fix",
      "transit",
      "windowEdge",
    ]);
    const t = days[0].events.find((e) => e.type === "transit");
    expect(t).toMatchObject({
      kind: "observed",
      km: 45.2,
      chordKm: 45.2,
      points: 38,
    });
  });
});

describe("fmtDateRange", () => {
  it.each([
    ["2026-05-22T15:10:00Z", "2026-05-22T18:00:00Z", "22 May 2026"],
    ["2026-05-22T15:10:00Z", "2026-05-24T13:01:00Z", "22–24 May 2026"],
    ["2026-05-28T00:00:00Z", "2026-06-02T00:00:00Z", "28 May – 2 June 2026"],
    [
      "2026-12-28T00:00:00Z",
      "2027-01-02T00:00:00Z",
      "28 December 2026 – 2 January 2027",
    ],
  ])("%s → %s → %s", (a, b, want) => {
    expect(fmtDateRange(a, b)).toBe(want);
  });
});

describe("dayLabel", () => {
  it.each([
    ["2026-05-22", "Friday 22 May"],
    ["2026-01-01", "Thursday 1 January"],
    ["2026-12-31", "Thursday 31 December"],
    ["2024-02-29", "Thursday 29 February"],
  ])("%s → %s", (date, label) => {
    expect(dayLabel(date)).toBe(label);
  });
});
