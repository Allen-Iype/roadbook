// The exported plate's regression table (phase 14 CP3, BRIEF §6). The
// invariant-8 statement for an image that travels: whatever the input, the
// op-list carries the legend in the fixed wording. Plus the licence line
// (attribution present whenever a loaded source declares one), the
// dateline's region truncation, the scale-bar formula, the truncation
// words, and the rule that every figure is the served one.
import { describe, expect, it } from "vitest";

import { LEGEND_ENTRIES } from "@/lib/legend";
import {
  MAP_RECT,
  OVERLAY_DIM,
  OVERLAY_H,
  OVERLAY_W,
  PLATE_H,
  PLATE_SCALE,
  PLATE_W,
  ROUTE_BOX,
  attributionText,
  datelineText,
  joinAttributions,
  layoutStrings,
  metresPerPixel,
  overlayFilename,
  overlayLayout,
  plateFilename,
  plateLabel,
  plateLayout,
  plateSlug,
  scaleBar,
  truncationText,
  type PlateInput,
  type PlateJourney,
} from "@/lib/plate-image";
import { routeFeatures } from "@/lib/route-layers";
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];
type Stop = components["schemas"]["Stop"];

const t = "2026-04-24T09:54:05Z";
const pt = (lat: number, lon: number) => ({ t, lat, lon });

// A journey shaped like the demo's South coast drive, with figures chosen
// so the strings are unambiguous.
const journey: PlateJourney = {
  window_start: "2026-04-24T09:54:05Z",
  window_end: "2026-04-26T17:40:00Z",
  total_km: 699.6,
  observed_km: 699.6,
  routed_km: 0,
  unknown_km: 0,
  air_km: 0,
  countries: [{ iso_code: "IS", name: "Iceland" }],
  states: [
    { code: "ISL-1", name: "Suðurland", country_code: "IS" },
    { code: "ISL-2", name: "Austurland", country_code: "IS" },
  ],
  summary: { span_hours: 55.8, civil_days: 3, stops: 7, dwell_hours: 40.7 },
  legs: [
    { kind: "observed", points: [pt(64, -21), pt(64, -20)], distance_km: 84 },
    { kind: "gap", points: [pt(64, -20), pt(64, -19)], distance_km: 10 },
  ],
  stops: [{ loc: { lat: 64, lon: -20 } }, { loc: { lat: 0, lon: 0 } }],
};

const base: PlateInput = {
  journey,
  name: "South coast drive",
  plate: 1,
  startTruncated: false,
  endTruncated: false,
  selectedDay: null,
  attribution: "OpenFreeMap © OpenMapTiles Data from OpenStreetMap",
  mapView: { west: -22, east: -14, centerLat: 64 },
};

const legendWords = LEGEND_ENTRIES.flatMap((e) => [e.label, `— ${e.desc}`]);

describe("plateLayout — the legend can never be omitted (invariant 8)", () => {
  const variants: [string, PlateInput][] = [
    ["full input", base],
    ["shared view: no plate number", { ...base, plate: null }],
    ["highlighted day", { ...base, selectedDay: 2 }],
    ["no attribution", { ...base, attribution: null }],
    ["no map view", { ...base, mapView: null }],
    ["both truncations", { ...base, startTruncated: true, endTruncated: true }],
    [
      "empty journey",
      {
        ...base,
        journey: {
          ...journey,
          total_km: 0,
          observed_km: 0,
          legs: [],
          stops: [],
          countries: [],
          states: [],
        },
      },
    ],
  ];
  for (const [label, input] of variants) {
    it(`carries the fixed wording: ${label}`, () => {
      const strings = layoutStrings(plateLayout(input));
      for (const w of legendWords) expect(strings).toContain(w);
      // The samples ride with the words: one drawn sample per kind.
      const legendRow = plateLayout(input).ops.find(
        (op) => op.op === "row" && op.items.some((i) => i.item === "sample"),
      );
      expect(legendRow && legendRow.op === "row").toBe(true);
      if (legendRow && legendRow.op === "row") {
        const kinds = legendRow.items.flatMap((i) => (i.item === "sample" ? [i.kind] : []));
        expect(kinds).toEqual(["observed", "routed", "unknown", "air"]);
      }
    });
  }
});

describe("plateLayout — what else the margin says", () => {
  it("is one format: 1200×800 at scale 2, with the map in its slot", () => {
    const l = plateLayout(base);
    expect([l.width, l.height, l.scale]).toEqual([PLATE_W, PLATE_H, PLATE_SCALE]);
    expect(l.ops.find((op) => op.op === "map")).toEqual({ op: "map", ...MAP_RECT });
  });

  it("prints the served figures verbatim, as the cover does", () => {
    const s = layoutStrings(plateLayout(base));
    expect(s).toContain("South coast drive");
    expect(s).toContain("700 km");
    expect(s).toContain("drawn — 100% of it measured");
    expect(s).toContain(
      "observed 699.6 km · routed 0.0 km · unknown 0.0 km · air 0.0 km",
    );
    expect(s).toContain(
      "24–26 April 2026 · Iceland · Suðurland, Austurland · 3 days",
    );
    expect(s).toContain("PLATE I · FULL ROUTE");
    expect(s).toContain("ROADBOOK");
  });

  it("carries the attribution line whenever a source declared one", () => {
    expect(layoutStrings(plateLayout(base))).toContain(
      "OpenFreeMap © OpenMapTiles Data from OpenStreetMap",
    );
    const without = layoutStrings(plateLayout({ ...base, attribution: null }));
    expect(without.some((x) => x.includes("OpenStreetMap"))).toBe(false);
    expect(without).toContain("ROADBOOK");
  });

  it("labels a highlighted day and drops the plate number then", () => {
    const s = layoutStrings(plateLayout({ ...base, selectedDay: 2 }));
    expect(s).toContain("DAY 2 HIGHLIGHTED");
    expect(s.some((x) => x.startsWith("PLATE"))).toBe(false);
  });

  it("reads FULL ROUTE without a plate number (the shared view)", () => {
    expect(plateLabel(null, null)).toBe("FULL ROUTE");
    expect(plateLabel(4, null)).toBe("PLATE IV · FULL ROUTE");
    expect(plateLabel(4, 3)).toBe("DAY 3 HIGHLIGHTED");
  });

  it("prints the truncation words with the flag glyph when the window is cut", () => {
    const none = layoutStrings(plateLayout(base));
    expect(none.some((x) => x.includes("mid-journey"))).toBe(false);
    const both = layoutStrings(
      plateLayout({ ...base, startTruncated: true, endTruncated: true }),
    );
    expect(both).toContain("⚑");
    expect(both).toContain(
      "The record starts mid-journey — it began before the imported window. " +
        "Still in progress at the window's edge — the end shown is the cut, not the return.",
    );
    expect(truncationText(false, true)).toBe(
      "Still in progress at the window's edge — the end shown is the cut, not the return.",
    );
    expect(truncationText(false, false)).toBeNull();
  });

  it("adds Fix and Stop legend entries exactly when the plate draws them", () => {
    const fixAndStop = layoutStrings(
      plateLayout({
        ...base,
        journey: {
          ...journey,
          legs: [{ kind: "observed", points: [pt(64, -21)], distance_km: 0 }],
        },
      }),
    );
    expect(fixAndStop).toContain("Fix");
    expect(fixAndStop).toContain("Stop");
    const neither = layoutStrings(
      plateLayout({
        ...base,
        journey: { ...journey, stops: [{ loc: { lat: 0, lon: 0 } }] },
      }),
    );
    expect(neither).not.toContain("Fix");
    expect(neither).not.toContain("Stop"); // a (0,0) dwell is not drawn
  });

  it("draws the provenance bar from the served split", () => {
    const bar = plateLayout({
      ...base,
      journey: { ...journey, observed_km: 50, routed_km: 30, unknown_km: 20, air_km: 0, total_km: 100 },
    }).ops.find((op) => op.op === "bar");
    expect(bar && bar.op === "bar" && bar.segments).toEqual([
      { kind: "observed", fraction: 0.5 },
      { kind: "routed", fraction: 0.3 },
      { kind: "unknown", fraction: 0.2 },
    ]);
  });

  it("emits a scale bar only when it knows the view", () => {
    expect(plateLayout(base).ops.some((op) => op.op === "scale")).toBe(true);
    expect(plateLayout({ ...base, mapView: null }).ops.some((op) => op.op === "scale")).toBe(false);
  });
});

describe("attributionText", () => {
  it("strips OpenFreeMap's anchors and decodes the entity", () => {
    expect(
      attributionText(
        '<a href="https://openfreemap.org" target="_blank">OpenFreeMap</a> ' +
          '<a href="https://www.openmaptiles.org/" target="_blank">&copy; OpenMapTiles</a> ' +
          'Data from <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a>',
      ),
    ).toBe("OpenFreeMap © OpenMapTiles Data from OpenStreetMap");
  });
  it("decodes numeric entities and leaves plain text alone", () => {
    expect(attributionText("&#169; Tiles &amp; data &#x00A9; me")).toBe("© Tiles & data © me");
    expect(attributionText("Plain credit")).toBe("Plain credit");
  });
  it("joins distinct sources and answers null for none", () => {
    expect(joinAttributions([undefined, "", "<b>A</b>", "A", "B &copy;"])).toBe("A · B ©");
    expect(joinAttributions([undefined, ""])).toBeNull();
  });
});

describe("datelineText — regions truncate to the line with '+ n more'", () => {
  const many = {
    ...journey,
    states: ["Höfuðborgarsvæði", "Vesturland", "Vestfirðir", "Norðurland vestra", "Norðurland eystra", "Austurland", "Suðurland"].map(
      (name, i) => ({ code: `ISL-${i}`, name, country_code: "IS" }),
    ),
  };
  it("lists every region when it fits", () => {
    expect(datelineText(many, 200)).toBe(
      "24–26 April 2026 · Iceland · Höfuðborgarsvæði, Vesturland, Vestfirðir, Norðurland vestra, Norðurland eystra, Austurland, Suðurland · 3 days",
    );
  });
  it("drops regions from the end and counts them", () => {
    const line = datelineText(many, 90);
    expect(line.length).toBeLessThanOrEqual(90);
    expect(line).toBe(
      "24–26 April 2026 · Iceland · Höfuðborgarsvæði, Vesturland, Vestfirðir, + 4 more · 3 days",
    );
  });
  it("never drops the dates, countries, or day count", () => {
    const line = datelineText(many, 10);
    expect(line).toBe("24–26 April 2026 · Iceland · + 7 more · 3 days");
  });
  it("omits the region part when there are none, and says '1 day'", () => {
    expect(
      datelineText(
        { ...journey, states: [], countries: [], summary: { ...journey.summary, civil_days: 1 } },
        200,
      ),
    ).toBe("24–26 April 2026 · 1 day");
  });
});

describe("scale bar", () => {
  it("metres per pixel: the view's longitude span, scaled by cos(latitude)", () => {
    // One degree of longitude across 1000 px at the equator.
    expect(metresPerPixel(0, 1, 0, 1000)).toBeCloseTo(111.31949, 3);
    // At 60°N the same span covers half the ground.
    expect(metresPerPixel(0, 1, 60, 1000)).toBeCloseTo(55.659745, 3);
  });
  it("rounds to 1/2/3/5 × 10ⁿ and never exceeds the budget", () => {
    expect(scaleBar(100, 140)).toEqual({ px: 100, label: "10 km" }); // 14 km → 10 km
    expect(scaleBar(30, 140)).toEqual({ px: 100, label: "3 km" }); // 4.2 km → 3 km
    expect(scaleBar(4, 140)).toEqual({ px: 125, label: "500 m" }); // 560 m → 500 m
    expect(scaleBar(0.5, 140)).toEqual({ px: 100, label: "50 m" }); // 70 m → 50 m
    for (const mpp of [0.3, 1, 7, 55, 120, 900]) {
      expect(scaleBar(mpp, 140).px).toBeLessThanOrEqual(140);
      expect(scaleBar(mpp, 140).px).toBeGreaterThan(28); // always ≥ a fifth of the budget
    }
  });
});

describe("filename", () => {
  it("slugs the name to ASCII and falls back to 'adventure'", () => {
    expect(plateSlug("Westfjords loop")).toBe("westfjords-loop");
    expect(plateSlug("Höfn — Stokksnes, Day 2!")).toBe("hofn-stokksnes-day-2");
    expect(plateSlug("Journey of 2026-05-22")).toBe("journey-of-2026-05-22");
    expect(plateSlug("   ")).toBe("adventure");
    expect(plateSlug("東京")).toBe("adventure");
    expect(plateSlug("x".repeat(80))).toHaveLength(60);
    expect(plateFilename("Akureyri weekend")).toBe("roadbook-plate-akureyri-weekend.png");
  });
});

// ---- the overlay (CP4, BRIEF §9) ----

const t2 = "2026-04-24T12:00:00Z";
const legs: Leg[] = [
  { kind: "observed", points: [pt(64.0, -21.0), pt(64.1, -20.5), pt(64.2, -20.0)], distance_km: 84, start: t, end: t2 },
  { kind: "gap", gap_kind: "road", points: [pt(64.2, -20.0), pt(64.3, -19.0)], distance_km: 50, routed_points: [{ lat: 64.2, lon: -20.0 }, { lat: 64.25, lon: -19.5 }, { lat: 64.3, lon: -19.0 }], routed_km: 55, start: t2, end: t2 },
  { kind: "gap", gap_kind: "unknown", points: [pt(64.3, -19.0), pt(64.4, -18.0)], distance_km: 40, start: t2, end: t2 },
  { kind: "gap", gap_kind: "air", points: [pt(64.4, -18.0), pt(65.6, -18.1)], distance_km: 130, start: t2, end: t2 },
  { kind: "observed", points: [pt(65.6, -18.1)], distance_km: 0, start: t2, end: t2 },
];
const stops: Stop[] = [
  { start: t, end: t2, loc: { lat: 64.2, lon: -20.0 }, points: 3, displacement_km: 0 },
  { start: t, end: t2, loc: { lat: 0, lon: 0 }, points: 0, displacement_km: 0 },
];
const overlayJourney: PlateJourney = { ...journey, legs, stops };
const days = [
  { index: 1, date: "2026-04-24", label: "", km: 0, legIndices: [0, 1], stopIndices: [0], events: [] },
  { index: 2, date: "2026-04-25", label: "", km: 0, legIndices: [2, 3, 4], stopIndices: [1], events: [] },
];
const overlayBase = {
  journey: overlayJourney,
  name: "South coast drive",
  plate: 1,
  startTruncated: false,
  endTruncated: false,
  selectedDay: null as number | null,
};
const overlay = (sel: number | null = null) =>
  overlayLayout({ ...overlayBase, selectedDay: sel }, routeFeatures({ legs, stops }, days));

describe("overlayLayout — the transparent format", () => {
  it("is story portrait at scale 2, with no ground, no map, no scale bar, no credit", () => {
    const l = overlay();
    expect([l.width, l.height, l.scale]).toEqual([OVERLAY_W, OVERLAY_H, PLATE_SCALE]);
    for (const op of l.ops) expect(["fill", "map", "scale", "frame"]).not.toContain(op.op);
    expect(layoutStrings(l).some((x) => x.includes("OpenStreetMap"))).toBe(false);
  });

  it("carries the legend in the fixed wording, unconditionally", () => {
    for (const l of [overlay(), overlay(2), overlayLayout({ ...overlayBase, journey: { ...overlayJourney, legs: [], stops: [] } }, routeFeatures({ legs: [], stops: [] }))]) {
      const strings = layoutStrings(l);
      for (const w of legendWords) expect(strings).toContain(w);
    }
  });

  it("prints the plate's figures verbatim and the wordmark", () => {
    const s = layoutStrings(overlay());
    const p = layoutStrings(plateLayout({ ...base, journey: overlayJourney }));
    for (const x of ["South coast drive", "700 km", "drawn — 100% of it measured",
      "observed 699.6 km · routed 0.0 km · unknown 0.0 km · air 0.0 km",
      "24–26 April 2026 · Iceland · Suðurland, Austurland · 3 days", "PLATE I · FULL ROUTE", "ROADBOOK"]) {
      expect(s).toContain(x);
      expect(p).toContain(x);
    }
    expect(overlayFilename("South coast drive")).toBe("roadbook-overlay-south-coast-drive.png");
  });

  it("draws every leg in its kind, in paint order, plus the fix and the drawn stop", () => {
    const paths = overlay().ops.flatMap((op) => (op.op === "path" ? [op] : []));
    expect(paths.map((p) => p.kind)).toEqual(["unknown", "air", "routed", "observed"]);
    // Routed rides its routed polyline (3 vertices), air its arc (65).
    expect(paths.find((p) => p.kind === "routed")!.points).toHaveLength(3);
    expect(paths.find((p) => p.kind === "air")!.points).toHaveLength(65);
    const points = overlay().ops.flatMap((op) => (op.op === "point" ? [op] : []));
    expect(points.map((p) => p.kind)).toEqual(["fix", "stop"]); // the (0,0) stop is absent
  });

  it("keeps the whole route inside the route box", () => {
    const inside = (x: number, y: number) =>
      x >= ROUTE_BOX.x + ROUTE_BOX.pad - 1e-6 && x <= ROUTE_BOX.x + ROUTE_BOX.w - ROUTE_BOX.pad + 1e-6 &&
      y >= ROUTE_BOX.y + ROUTE_BOX.pad - 1e-6 && y <= ROUTE_BOX.y + ROUTE_BOX.h - ROUTE_BOX.pad + 1e-6;
    for (const op of overlay().ops) {
      if (op.op === "path") for (const [x, y] of op.points) expect(inside(x, y)).toBe(true);
      if (op.op === "point") expect(inside(op.x, op.y)).toBe(true);
    }
  });

  it("a highlighted day dims the rest to the plate's opacity and says so", () => {
    const l = overlay(2);
    const alphaOf = (kind: string) => l.ops.find((op) => op.op === "path" && op.kind === kind)!;
    expect((alphaOf("observed") as { alpha: number }).alpha).toBe(OVERLAY_DIM); // day 1
    expect((alphaOf("unknown") as { alpha: number }).alpha).toBe(1); // day 2
    const pts = l.ops.flatMap((op) => (op.op === "point" ? [op] : []));
    expect(pts.find((p) => p.kind === "fix")!.alpha).toBe(1); // leg 4, day 2
    expect(pts.find((p) => p.kind === "stop")!.alpha).toBe(OVERLAY_DIM); // stop 0, day 1
    expect(layoutStrings(l)).toContain("DAY 2 HIGHLIGHTED");
    for (const op of overlay().ops) if (op.op === "path" || op.op === "point") expect(op.alpha).toBe(1);
  });

  it("every glyph carries its paper casing — the overlay's ground for text", () => {
    for (const op of overlay().ops) {
      if (op.op === "text") expect(op.halo).toBe(true);
      if (op.op === "row") for (const it of op.items) if (it.item === "text") expect(it.halo).toBe(true);
    }
    // The plate's text never does: it has paper under it already.
    for (const op of plateLayout(base).ops) if (op.op === "text") expect(op.halo).toBeUndefined();
  });

  it("wraps the legend and the truncation line to the overlay's width", () => {
    const l = overlayLayout({ ...overlayBase, startTruncated: true, endTruncated: true }, routeFeatures({ legs, stops }, days));
    const wrapped = l.ops.filter((op) => op.op === "row" && op.wrap);
    expect(wrapped).toHaveLength(2);
    for (const op of wrapped) if (op.op === "row" && op.wrap) expect(op.wrap.maxWidth).toBe(OVERLAY_W - 80);
  });
});
