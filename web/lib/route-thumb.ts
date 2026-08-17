// The route thumbnail (phase 9 CP2): a journey's shape as plain SVG paths,
// for surfaces that want a small picture of the route without a map — the
// life-map hover card now, the adventure grid's plate covers in CP4. A
// MapLibre instance per thumbnail would be the wrong tool: browsers cap
// live WebGL contexts (~8–16), and a grid of tiles would exhaust them; the
// `/welcome` plate already proved routes read fine as bare SVG strokes.
//
// Pure module — no DOM, no React — so vitest can pin the projection and
// the invariant-8 property: every path carries its leg kind, and kinds are
// never merged. The four-way encoding survives at thumbnail size through
// the same non-color channels as everywhere else (solid / cased / dashed /
// dotted — the component maps kinds to the reserved inks and dash
// patterns).
import { greatCircleArc, lngLat } from "@/lib/geo";
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];

export type ThumbKind = "observed" | "road" | "unknown" | "air";

export type ThumbPath = { kind: ThumbKind; d: string };

export type Thumb = {
  /** "0 0 w h" for the <svg> element. */
  viewBox: string;
  width: number;
  height: number;
  /** In paint order: unknown first (a guess never covers a measurement),
      then air, road, observed on top — same z-order as the maps. */
  paths: ThumbPath[];
};

const PAINT_ORDER: ThumbKind[] = ["unknown", "air", "road", "observed"];

function kindOf(leg: Leg): ThumbKind {
  if (leg.kind === "observed") return "observed";
  if (leg.gap_kind === "road") return "road";
  if (leg.gap_kind === "air") return "air";
  return "unknown";
}

// The drawn geometry per kind mirrors lib/geo.legFeatures: air bulges into
// its great-circle arc (fewer segments — thumbnail pixels cannot show 64),
// road rides its routed polyline, everything else its own points.
function lineOf(leg: Leg): [number, number][] {
  if (leg.gap_kind === "air")
    return greatCircleArc(lngLat(leg.points[0]), lngLat(leg.points[1]), 24);
  if (leg.gap_kind === "road" && leg.routed_points)
    return leg.routed_points.map(lngLat);
  return leg.points.map(lngLat);
}

export function routeThumb(
  legs: Leg[],
  width = 224,
  height = 120,
  pad = 8,
): Thumb | null {
  const lines = legs
    .map((leg) => ({ kind: kindOf(leg), coords: lineOf(leg) }))
    .filter((l) => l.coords.length >= 2);
  if (lines.length === 0) return null;

  // Equirectangular with the longitude axis compressed by cos(mid-lat) —
  // the same first-order flattening every small-extent web map effectively
  // shows, so the thumbnail's shape matches what the map will show. Y is
  // negated: SVG y grows downward, latitude grows upward.
  let minLat = Infinity, maxLat = -Infinity, minLon = Infinity, maxLon = -Infinity;
  for (const l of lines)
    for (const [lon, lat] of l.coords) {
      if (lat < minLat) minLat = lat;
      if (lat > maxLat) maxLat = lat;
      if (lon < minLon) minLon = lon;
      if (lon > maxLon) maxLon = lon;
    }
  const k = Math.cos((((minLat + maxLat) / 2) * Math.PI) / 180);
  const spanX = (maxLon - minLon) * k;
  const spanY = maxLat - minLat;
  // A journey is never a point, but a degenerate input must not divide by
  // zero — it draws centered instead.
  const scale =
    spanX === 0 && spanY === 0
      ? 1
      : Math.min((width - 2 * pad) / (spanX || 1e-9), (height - 2 * pad) / (spanY || 1e-9));
  const offX = (width - spanX * scale) / 2;
  const offY = (height - spanY * scale) / 2;

  const px = (lon: number, lat: number): [number, number] => [
    offX + (lon - minLon) * k * scale,
    offY + (maxLat - lat) * scale,
  ];

  const paths = lines
    .map(({ kind, coords }) => ({
      kind,
      d: coords
        .map(([lon, lat], i) => {
          const [x, y] = px(lon, lat);
          return `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`;
        })
        .join(""),
    }))
    .sort((a, b) => PAINT_ORDER.indexOf(a.kind) - PAINT_ORDER.indexOf(b.kind));

  return { viewBox: `0 0 ${width} ${height}`, width, height, paths };
}
