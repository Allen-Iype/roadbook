// The exported plate's layout (phase 14 CP3, BRIEF §4): a pure builder that
// turns a served Journey plus a few facts about the page into an op-list —
// fills, rules, text, the map slot, the provenance bar, the legend row —
// which lib/plate-export.ts then paints onto a 2D canvas. Splitting layout
// from painting is what makes the image testable without a browser:
// vitest runs this module in node and asserts what the image WILL say,
// not what pixels it happened to produce.
//
// Two rules the builder enforces by construction (BRIEF §1, §5):
//   - Every figure on the image is copied from the served Journey — the
//     same strings the cover prints, the same values `roadbook journey
//     -candidate N` prints (invariant 13). Nothing here computes a figure
//     of the journey's; the scale bar and the truncation of long lines are
//     rendering, not claims.
//   - The legend is unconditional (invariant 8): an image travels without
//     the page that would have explained its four inks, so the op-list
//     always carries the fixed wording from lib/legend.ts. There is no
//     input that omits it, and plate-image.test.ts says so.
//
// Geometry is in logical pixels on a 1200×800 plate; the painter scales by
// PLATE_SCALE for the 2400×1600 file. Positions are fixed by row, not
// measured: the painter measures text where flow matters (rows) and
// shrinks the one line that can overflow (the name).
//
// Two formats since CP4 (BRIEF §9): the PLATE, a picture with a basemap
// slot and a licence line; and the OVERLAY, a transparent story-portrait
// image with the route drawn tile-free from the same features both maps
// draw, the same figures, and its own ground under every mark (a halo
// under the route, paper casing around every glyph). `layoutFor` picks.
import { fmtDateRange } from "@/lib/slice-days";
import { roman } from "@/lib/format";
import { LEGEND_ENTRIES, type LegendKind } from "@/lib/legend";
import { fitProjection } from "@/lib/route-thumb";
import { INK } from "@/lib/tokens";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];

/** The subset of a Journey the layout reads (structural, so tests can
 * build minimal inputs). */
export type PlateJourney = Pick<
  Journey,
  | "window_start"
  | "window_end"
  | "total_km"
  | "observed_km"
  | "routed_km"
  | "unknown_km"
  | "air_km"
  | "countries"
  | "states"
  | "summary"
> & {
  legs: Pick<Journey["legs"][number], "kind" | "points" | "distance_km">[];
  stops: Pick<Journey["stops"][number], "loc">[];
};

export type PlateInput = {
  journey: PlateJourney;
  /** The cover's name: the decision name, or "Journey of <date>". */
  name: string;
  /** Position among confirmed adventures in date order; null on the shared
   * view and for unconfirmed candidates. */
  plate: number | null;
  startTruncated: boolean;
  endTruncated: boolean;
  /** 1-based highlighted day, or null for the full route. */
  selectedDay: number | null;
  /** Plain-text attribution of the loaded basemap (attributionText), or
   * null when no loaded source declared one. */
  attribution: string | null;
  /** The offscreen map's view after fitting — for the scale bar only. */
  mapView: { west: number; east: number; centerLat: number } | null;
};

// One format (BRIEF §2b): a 1200×800 plate at pixel ratio 2.
export const PLATE_W = 1200;
export const PLATE_H = 800;
export const PLATE_SCALE = 2;
/** The map's slot on the plate; the offscreen map is created at exactly
 * this logical size so its pixels land 1:1. */
export const MAP_RECT = { x: 32, y: 32, w: 1136, h: 528 } as const;
/** The double rule: an inner hairline on the map's edge and an outer one
 * this far out — the on-screen plate's border + outline-offset. */
export const FRAME_OFFSET = 3;
/** The scale bar's longest allowed length, logical px. */
export const SCALE_MAX_PX = 140;

export type PlateFormat = "plate" | "overlay";

// The overlay: story portrait, 1080×1920 at pixel ratio 2 (BRIEF §9B). The
// route takes the upper part of the canvas; the figures the lower part,
// clear of a story UI's top and bottom bands.
export const OVERLAY_W = 540;
export const OVERLAY_H = 960;
/** Where the route is fitted on the overlay (pad inside it on every side). */
export const ROUTE_BOX = { x: 40, y: 110, w: 460, h: 440, pad: 24 } as const;
/** Everything outside a highlighted day at this alpha — the plate's dim. */
export const OVERLAY_DIM = 0.15;

// Grounds and inks beyond the leg inks — the DOM reads them from the theme
// (globals.css); the canvas cannot, so they are named here once, matching.
export const PLATE_COLORS = {
  paper: INK.paper,
  ink: INK.ink,
  ink2: "#6e6a5e",
  rule: "#c9c3b2",
  flag: INK.flag,
} as const;

export type FontRole = "display" | "mono" | "sans";

export type TextStyle = {
  role: FontRole;
  size: number;
  weight: 400 | 500 | 600 | 700;
  color: string;
  /** Letter-spacing in em (the tracked capitals of the plate label). */
  tracking?: number;
  /** Paper casing around the glyphs — the overlay's own ground for text. */
  halo?: boolean;
};

export type RowItem =
  | ({ item: "text"; text: string } & TextStyle)
  | { item: "sample"; kind: LegendKind }
  | { item: "dot"; kind: "fix" | "stop" }
  | { item: "space"; px: number };

export type Op =
  | { op: "fill"; x: number; y: number; w: number; h: number; color: string }
  | { op: "frame"; x: number; y: number; w: number; h: number; color: string; offset: number }
  | { op: "map"; x: number; y: number; w: number; h: number }
  | ({
      op: "text";
      x: number;
      y: number;
      text: string;
      align: "left" | "right";
      /** The painter shrinks the size until the text fits this width. */
      maxWidth?: number;
    } & TextStyle)
  | {
      op: "row";
      x: number;
      y: number;
      items: RowItem[];
      /** When set, items flow onto further lines within maxWidth. */
      wrap?: { maxWidth: number; lineHeight: number };
    }
  // The overlay's route: one path per leg in its kind, projected into the
  // route box; `alpha` is 1 or the dim for a highlighted day.
  | { op: "path"; kind: LegendKind; points: [number, number][]; alpha: number }
  | { op: "point"; kind: "fix" | "stop"; x: number; y: number; alpha: number }
  | {
      op: "bar";
      x: number;
      y: number;
      w: number;
      h: number;
      track: string;
      segments: { kind: LegendKind; fraction: number }[];
    }
  | { op: "scale"; x: number; y: number; px: number; label: string };

export type PlateLayout = {
  width: number;
  height: number;
  scale: number;
  ops: Op[];
};

// ---- text helpers, each pure and pinned by a test ----

/** MapLibre serves a source's attribution as HTML (anchors around each
 * credit). The canvas draws text, so the tags go and the entities decode:
 * "OpenFreeMap © OpenMapTiles Data from OpenStreetMap". */
export function attributionText(html: string): string {
  const named: Record<string, string> = {
    amp: "&",
    lt: "<",
    gt: ">",
    quot: '"',
    apos: "'",
    nbsp: " ",
    copy: "©",
  };
  return html
    .replace(/<[^>]*>/g, " ")
    .replace(/&#x([0-9a-f]+);/gi, (_m, hex: string) =>
      String.fromCodePoint(parseInt(hex, 16)),
    )
    .replace(/&#(\d+);/g, (_m, dec: string) =>
      String.fromCodePoint(parseInt(dec, 10)),
    )
    .replace(/&([a-z]+);/gi, (m, name: string) => named[name.toLowerCase()] ?? m)
    .replace(/\s+/g, " ")
    .trim();
}

/** Distinct attributions of every loaded source, in source order, joined —
 * null when none declared one (then the foot carries no credit line). */
export function joinAttributions(htmls: (string | undefined)[]): string | null {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const h of htmls) {
    if (!h) continue;
    const t = attributionText(h);
    if (t === "" || seen.has(t)) continue;
    seen.add(t);
    out.push(t);
  }
  return out.length === 0 ? null : out.join(" · ");
}

/** Ground metres per screen pixel at the view's centre latitude. Web
 * Mercator is linear in longitude, so the view's longitude span over its
 * pixel width is the pixel's angular width; one degree of longitude on
 * the ground is 111,319.49 m at the equator, scaled by cos(latitude).
 * Independent of tile-size and zoom conventions on purpose. */
export function metresPerPixel(
  west: number,
  east: number,
  centerLat: number,
  widthPx: number,
): number {
  const degPerPx = (east - west) / widthPx;
  return degPerPx * 111_319.49 * Math.cos((centerLat * Math.PI) / 180);
}

/** A scale bar no longer than maxPx whose distance is a round number —
 * 1, 2, 3, or 5 times a power of ten (MapLibre's ScaleControl rounding), in
 * metres under a kilometre and kilometres from there. */
export function scaleBar(
  mpp: number,
  maxPx: number,
): { px: number; label: string } {
  const maxMetres = mpp * maxPx;
  const pow10 = Math.pow(10, Math.floor(Math.log10(maxMetres)));
  const d = maxMetres / pow10;
  const round = d >= 10 ? 10 : d >= 5 ? 5 : d >= 3 ? 3 : d >= 2 ? 2 : 1;
  const metres = round * pow10;
  const label = metres >= 1000 ? `${metres / 1000} km` : `${metres} m`;
  return { px: metres / mpp, label };
}

/** The dateline: dates, countries, regions, day count — the cover's line,
 * minus the fix count (an image has no room to explain it). Regions are
 * truncated to the character budget with "+ n more" (BRIEF §4): an image
 * has one line, the page has as many as it needs. Countries are never
 * truncated — there are a handful at most. */
export function datelineText(
  j: Pick<
    PlateJourney,
    "window_start" | "window_end" | "countries" | "states" | "summary"
  >,
  maxChars: number,
): string {
  const days = j.summary.civil_days;
  const head = [
    fmtDateRange(j.window_start, j.window_end),
    ...(j.countries.length > 0 ? [j.countries.map((c) => c.name).join(" · ")] : []),
  ];
  const tail = `${days} ${days === 1 ? "day" : "days"}`;
  const names = j.states.map((s) => s.name);
  const line = (shown: string[], more: number) =>
    [
      ...head,
      ...(shown.length > 0 || more > 0
        ? [
            [
              ...shown,
              ...(more > 0 ? [`+ ${more} more`] : []),
            ].join(", "),
          ]
        : []),
      tail,
    ].join(" · ");
  for (let n = names.length; n >= 0; n--) {
    const candidate = line(names.slice(0, n), names.length - n);
    if (candidate.length <= maxChars || n === 0) return candidate;
  }
  return line([], names.length);
}

/** The cover's truncation sentences, verbatim (bug 4 made visible). */
export function truncationText(start: boolean, end: boolean): string | null {
  const parts = [
    ...(start
      ? ["The record starts mid-journey — it began before the imported window."]
      : []),
    ...(end
      ? ["Still in progress at the window's edge — the end shown is the cut, not the return."]
      : []),
  ];
  return parts.length === 0 ? null : parts.join(" ");
}

/** The plate label, exactly as the on-screen margin prints it. */
export function plateLabel(plate: number | null, selectedDay: number | null): string {
  if (selectedDay !== null) return `DAY ${selectedDay} HIGHLIGHTED`;
  if (plate !== null) return `PLATE ${roman(plate)} · FULL ROUTE`;
  return "FULL ROUTE";
}

/** roadbook-plate-<slug>.png: the name lowercased to ASCII letters, digits
 * and hyphens, at most 60 characters; "adventure" when nothing survives. */
export function plateSlug(name: string): string {
  const slug = name
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60)
    .replace(/-+$/g, "");
  return slug === "" ? "adventure" : slug;
}

export function plateFilename(name: string): string {
  return `roadbook-plate-${plateSlug(name)}.png`;
}

// A fix on the image is what a fix on the map is: a stationary observed
// leg. Same predicate as lib/slice-days' isFixLeg, restated over the
// structural subset so the builder needs no full Leg.
function hasFix(j: PlateJourney): boolean {
  return j.legs.some(
    (l) => l.kind === "observed" && (l.points.length === 1 || l.distance_km === 0),
  );
}
function hasStop(j: PlateJourney): boolean {
  return j.stops.some((s) => !(s.loc.lat === 0 && s.loc.lon === 0));
}

// IBM Plex Mono advances 0.6 em per glyph — the one measurement the builder
// can make without a canvas, and the reason the dateline is mono: its
// character budget is exact.
const MONO_ADVANCE_EM = 0.6;

export function plateLayout(input: PlateInput): PlateLayout {
  const { journey: j, mapView } = input;
  const C = PLATE_COLORS;
  const left = MAP_RECT.x;
  const right = MAP_RECT.x + MAP_RECT.w;
  const width = MAP_RECT.w;
  const ops: Op[] = [];

  ops.push({ op: "fill", x: 0, y: 0, w: PLATE_W, h: PLATE_H, color: C.paper });
  ops.push({ op: "map", ...MAP_RECT });
  ops.push({ op: "frame", ...MAP_RECT, color: C.ink, offset: FRAME_OFFSET });

  // The scale bar sits inside the map frame, bottom-left, as the on-screen
  // ScaleControl does; drawn here because that control is DOM, not canvas.
  if (mapView) {
    const mpp = metresPerPixel(mapView.west, mapView.east, mapView.centerLat, MAP_RECT.w);
    const bar = scaleBar(mpp, SCALE_MAX_PX);
    ops.push({
      op: "scale",
      x: MAP_RECT.x + 14,
      y: MAP_RECT.y + MAP_RECT.h - 14,
      px: bar.px,
      label: bar.label,
    });
  }

  // The margin, row by row. y is a baseline; each row states its own lead.
  let y = MAP_RECT.y + MAP_RECT.h + FRAME_OFFSET;

  // Name and plate label share the first line: the name in the display
  // serif, shrunk if it would collide with the label's tracked capitals.
  y += 34;
  ops.push({
    op: "text",
    x: left,
    y,
    text: input.name,
    align: "left",
    role: "display",
    size: 38,
    weight: 600,
    color: C.ink,
    maxWidth: width - 280,
  });
  ops.push({
    op: "text",
    x: right,
    y,
    text: plateLabel(input.plate, input.selectedDay),
    align: "right",
    role: "display",
    size: 11.5,
    weight: 600,
    color: C.ink,
    tracking: 0.22,
  });

  // The dateline, in mono at a known advance — its budget is exact.
  y += 26;
  const datelineSize = 14;
  ops.push({
    op: "text",
    x: left,
    y,
    text: datelineText(j, Math.floor(width / (datelineSize * MONO_ADVANCE_EM))),
    align: "left",
    role: "mono",
    size: datelineSize,
    weight: 400,
    color: C.ink2,
  });

  // Truncation as words, with the amber flag glyph and ink words — the
  // cover's own treatment (the token passes contrast as a mark, not text).
  const trunc = truncationText(input.startTruncated, input.endTruncated);
  if (trunc) {
    y += 22;
    ops.push({
      op: "row",
      x: left,
      y,
      items: [
        { item: "text", text: "⚑", role: "sans", size: 12.5, weight: 700, color: C.flag },
        { item: "space", px: 6 },
        { item: "text", text: trunc, role: "sans", size: 12.5, weight: 400, color: C.ink },
      ],
    });
  }

  // The headline distance with its provenance bar and split — the cover's
  // three lines, figures verbatim (toFixed(0) / toFixed(1) as there).
  const pct = j.total_km > 0 ? Math.round((j.observed_km / j.total_km) * 100) : 0;
  y += 40;
  ops.push({
    op: "row",
    x: left,
    y,
    items: [
      { item: "text", text: `${j.total_km.toFixed(0)} km`, role: "display", size: 26, weight: 600, color: C.ink },
      { item: "space", px: 10 },
      { item: "text", text: `drawn — ${pct}% of it measured`, role: "sans", size: 14, weight: 400, color: C.ink2 },
    ],
  });
  y += 10;
  const total = j.observed_km + j.routed_km + j.unknown_km + j.air_km;
  ops.push({
    op: "bar",
    x: left,
    y,
    w: width,
    h: 3,
    track: C.rule,
    segments:
      total > 0
        ? (
            [
              ["observed", j.observed_km],
              ["routed", j.routed_km],
              ["unknown", j.unknown_km],
              ["air", j.air_km],
            ] as [LegendKind, number][]
          )
            .map(([kind, km]) => ({ kind, fraction: km / total }))
            .filter((s) => s.fraction > 0.002)
        : [],
  });
  y += 20;
  ops.push({
    op: "text",
    x: left,
    y,
    text:
      `observed ${j.observed_km.toFixed(1)} km · routed ${j.routed_km.toFixed(1)} km · ` +
      `unknown ${j.unknown_km.toFixed(1)} km · air ${j.air_km.toFixed(1)} km`,
    align: "left",
    role: "mono",
    size: 12,
    weight: 400,
    color: C.ink2,
  });

  // The legend, unconditional, in the fixed wording; Fix and Stop entries
  // join it exactly when the plate draws them.
  y += 32;
  const legend: RowItem[] = [];
  for (const e of LEGEND_ENTRIES) {
    if (legend.length > 0) legend.push({ item: "space", px: 22 });
    legend.push(
      { item: "sample", kind: e.key },
      { item: "space", px: 8 },
      { item: "text", text: e.label, role: "sans", size: 12, weight: 700, color: C.ink },
      { item: "space", px: 5 },
      { item: "text", text: `— ${e.desc}`, role: "sans", size: 12, weight: 400, color: C.ink2 },
    );
  }
  if (hasFix(j)) {
    legend.push(
      { item: "space", px: 22 },
      { item: "dot", kind: "fix" },
      { item: "space", px: 8 },
      { item: "text", text: "Fix", role: "sans", size: 12, weight: 700, color: C.ink },
    );
  }
  if (hasStop(j)) {
    legend.push(
      { item: "space", px: 22 },
      { item: "dot", kind: "stop" },
      { item: "space", px: 8 },
      { item: "text", text: "Stop", role: "sans", size: 12, weight: 700, color: C.ink },
    );
  }
  ops.push({ op: "row", x: left, y, items: legend });

  // The foot: the basemap's own credit (a licence condition, read from the
  // loaded style — never hardcoded) and the product's name.
  y += 30;
  if (input.attribution) {
    ops.push({
      op: "text",
      x: left,
      y,
      text: input.attribution,
      align: "left",
      role: "sans",
      size: 11,
      weight: 400,
      color: C.ink2,
      maxWidth: width - 140,
    });
  }
  ops.push({
    op: "text",
    x: right,
    y,
    text: "ROADBOOK",
    align: "right",
    role: "display",
    size: 12,
    weight: 600,
    color: C.ink,
    tracking: 0.28,
  });

  return { width: PLATE_W, height: PLATE_H, scale: PLATE_SCALE, ops };
}

/** Every string an op-list would draw, in order — what the tests read. */
export function layoutStrings(layout: PlateLayout): string[] {
  const out: string[] = [];
  for (const op of layout.ops) {
    if (op.op === "text") out.push(op.text);
    else if (op.op === "scale") out.push(op.label);
    else if (op.op === "row")
      for (const it of op.items) if (it.item === "text") out.push(it.text);
  }
  return out;
}

// ---- the overlay (CP4) ----

// The kind a drawn feature carries, in legend terms. Mirrors the map
// layers' filters: a gap that is neither air nor road is unknown.
function featureKind(props: GeoJSON.GeoJsonProperties): LegendKind | null {
  const kind = props?.kind;
  if (kind === "observed") return "observed";
  if (kind !== "gap") return null;
  const gap = props?.gap_kind;
  return gap === "air" ? "air" : gap === "road" ? "routed" : "unknown";
}

const PAINT_ORDER: LegendKind[] = ["unknown", "air", "routed", "observed"];

/** The overlay's op-list. `features` is the SAME collection the maps draw
 * (lib/route-layers routeFeatures) — legs stamped with their day, fix
 * points, stops with their days — so the overlay cannot draw a route the
 * plate does not. */
export function overlayLayout(
  input: Omit<PlateInput, "attribution" | "mapView">,
  features: GeoJSON.FeatureCollection,
): PlateLayout {
  const { journey: j } = input;
  const C = PLATE_COLORS;
  const left = 40;
  const right = OVERLAY_W - 40;
  const width = right - left;
  const ops: Op[] = [];
  const sel = input.selectedDay;

  // The route, projected by the thumbnail's own projection into the box.
  const coords: [number, number][] = [];
  for (const f of features.features) {
    if (f.geometry.type === "LineString")
      for (const c of f.geometry.coordinates) coords.push([c[0], c[1]]);
    else if (f.geometry.type === "Point")
      coords.push([f.geometry.coordinates[0], f.geometry.coordinates[1]]);
  }
  if (coords.length > 0) {
    const project = fitProjection(coords, ROUTE_BOX.w, ROUTE_BOX.h, ROUTE_BOX.pad);
    const px = (c: GeoJSON.Position): [number, number] => {
      const [x, y] = project(c[0], c[1]);
      return [ROUTE_BOX.x + x, ROUTE_BOX.y + y];
    };
    const legAlpha = (day: unknown) =>
      sel === null || day === sel ? 1 : OVERLAY_DIM;
    const paths: Extract<Op, { op: "path" }>[] = [];
    const points: Extract<Op, { op: "point" }>[] = [];
    for (const f of features.features) {
      const props = f.properties ?? {};
      if (f.geometry.type === "LineString") {
        const kind = featureKind(props);
        if (!kind || f.geometry.coordinates.length < 2) continue;
        paths.push({
          op: "path",
          kind,
          points: f.geometry.coordinates.map(px),
          alpha: legAlpha(props.day),
        });
      } else if (f.geometry.type === "Point") {
        const [x, y] = px(f.geometry.coordinates);
        if (props.kind === "fix")
          points.push({ op: "point", kind: "fix", x, y, alpha: legAlpha(props.day) });
        else if (props.kind === "stop") {
          const days = Array.isArray(props.days) ? (props.days as number[]) : [];
          points.push({
            op: "point",
            kind: "stop",
            x,
            y,
            alpha: sel === null || days.includes(sel) ? 1 : OVERLAY_DIM,
          });
        }
      }
    }
    // Paint order as on the maps: unknown under everything, observed on
    // top; fixes then stops above the lines.
    paths.sort((a, b) => PAINT_ORDER.indexOf(a.kind) - PAINT_ORDER.indexOf(b.kind));
    ops.push(...paths, ...points.filter((p) => p.kind === "fix"), ...points.filter((p) => p.kind === "stop"));
  }

  // The figures, in the plate's order, every glyph with its paper casing.
  let y = ROUTE_BOX.y + ROUTE_BOX.h + 64;
  ops.push({
    op: "text",
    x: left,
    y,
    text: plateLabel(input.plate, sel),
    align: "left",
    role: "display",
    size: 11,
    weight: 600,
    color: C.ink,
    tracking: 0.22,
    halo: true,
  });
  y += 36;
  ops.push({
    op: "text",
    x: left,
    y,
    text: input.name,
    align: "left",
    role: "display",
    size: 32,
    weight: 600,
    color: C.ink,
    maxWidth: width,
    halo: true,
  });
  y += 24;
  // 11.5 px mono over 460 px is a 66-character budget — the demo's two
  // regions fit; at 12 px they were one character over and truncated.
  const datelineSize = 11.5;
  ops.push({
    op: "text",
    x: left,
    y,
    text: datelineText(j, Math.floor(width / (datelineSize * MONO_ADVANCE_EM))),
    align: "left",
    role: "mono",
    size: datelineSize,
    weight: 500,
    color: C.ink,
    halo: true,
  });
  const trunc = truncationText(input.startTruncated, input.endTruncated);
  if (trunc) {
    y += 20;
    ops.push({
      op: "row",
      x: left,
      y,
      wrap: { maxWidth: width, lineHeight: 16 },
      items: [
        { item: "text", text: "⚑", role: "sans", size: 11.5, weight: 700, color: C.flag, halo: true },
        { item: "space", px: 5 },
        ...trunc.split(" ").flatMap((w, i): RowItem[] => [
          ...(i > 0 ? [{ item: "space" as const, px: 3.5 }] : []),
          { item: "text", text: w, role: "sans", size: 11.5, weight: 400, color: C.ink, halo: true },
        ]),
      ],
    });
    y += 4;
  }
  const pct = j.total_km > 0 ? Math.round((j.observed_km / j.total_km) * 100) : 0;
  y += 38;
  ops.push({
    op: "row",
    x: left,
    y,
    items: [
      { item: "text", text: `${j.total_km.toFixed(0)} km`, role: "display", size: 26, weight: 600, color: C.ink, halo: true },
      { item: "space", px: 8 },
      { item: "text", text: `drawn — ${pct}% of it measured`, role: "sans", size: 12.5, weight: 400, color: C.ink, halo: true },
    ],
  });
  y += 9;
  const total = j.observed_km + j.routed_km + j.unknown_km + j.air_km;
  ops.push({
    op: "bar",
    x: left,
    y,
    w: width,
    h: 3,
    track: C.rule,
    segments:
      total > 0
        ? (
            [
              ["observed", j.observed_km],
              ["routed", j.routed_km],
              ["unknown", j.unknown_km],
              ["air", j.air_km],
            ] as [LegendKind, number][]
          )
            .map(([kind, km]) => ({ kind, fraction: km / total }))
            .filter((s) => s.fraction > 0.002)
        : [],
  });
  y += 18;
  ops.push({
    op: "text",
    x: left,
    y,
    text:
      `observed ${j.observed_km.toFixed(1)} km · routed ${j.routed_km.toFixed(1)} km · ` +
      `unknown ${j.unknown_km.toFixed(1)} km · air ${j.air_km.toFixed(1)} km`,
    align: "left",
    role: "mono",
    size: 10.5,
    weight: 500,
    color: C.ink,
    halo: true,
  });

  // The legend, unconditional, full wording, wrapping to the width.
  y += 30;
  const legend: RowItem[] = [];
  for (const e of LEGEND_ENTRIES) {
    if (legend.length > 0) legend.push({ item: "space", px: 18 });
    legend.push(
      { item: "sample", kind: e.key },
      { item: "space", px: 7 },
      { item: "text", text: e.label, role: "sans", size: 11.5, weight: 700, color: C.ink, halo: true },
      { item: "space", px: 4 },
      { item: "text", text: `— ${e.desc}`, role: "sans", size: 11.5, weight: 400, color: C.ink, halo: true },
    );
  }
  if (hasFix(j)) {
    legend.push(
      { item: "space", px: 18 },
      { item: "dot", kind: "fix" },
      { item: "space", px: 7 },
      { item: "text", text: "Fix", role: "sans", size: 11.5, weight: 700, color: C.ink, halo: true },
    );
  }
  if (hasStop(j)) {
    legend.push(
      { item: "space", px: 18 },
      { item: "dot", kind: "stop" },
      { item: "space", px: 7 },
      { item: "text", text: "Stop", role: "sans", size: 11.5, weight: 700, color: C.ink, halo: true },
    );
  }
  ops.push({ op: "row", x: left, y, items: legend, wrap: { maxWidth: width, lineHeight: 22 } });

  // The foot: the wordmark, and nothing to credit — there is no basemap.
  ops.push({
    op: "text",
    x: right,
    y: OVERLAY_H - 56,
    text: "ROADBOOK",
    align: "right",
    role: "display",
    size: 12,
    weight: 600,
    color: C.ink,
    tracking: 0.28,
    halo: true,
  });

  return { width: OVERLAY_W, height: OVERLAY_H, scale: PLATE_SCALE, ops };
}

/** roadbook-overlay-<slug>.png */
export function overlayFilename(name: string): string {
  return `roadbook-overlay-${plateSlug(name)}.png`;
}
