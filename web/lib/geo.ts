// Shared map geometry (phase 6 checkpoint 2). Extracted from the adventure
// page's route-map island the moment a second map — the life map — needed the
// same conversions: two copies of coordinate handling is how the two maps
// would eventually disagree about what a leg looks like.
//
// This module is pure: no MapLibre import, no DOM, no I/O. That is what lets
// the invariant-8 style test exercise map code in vitest without a browser.
// (The `import type` below is erased at compile time — types are not code.)
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];
type Stop = components["schemas"]["Stop"];

/** GeoJSON positions are [longitude, latitude]. */
export type LonLat = [number, number];

// The one {lat, lon} → [lon, lat] conversion in the frontend (phase 2 BRIEF
// §1.2): the API speaks {lat, lon}; GeoJSON demands [longitude, latitude].
// Every coordinate on every map passes through this function.
export function lngLat(p: { lat: number; lon: number }): LonLat {
  return [p.lon, p.lat];
}

// The line one leg is drawn with, per kind:
//   air  — a great-circle arc between the two endpoints, generated here
//          because it is presentation: the API reports exactly the two
//          timestamped points it measured, and fabricated intermediate
//          coordinates must not enter a response that otherwise contains
//          only measurements (phase 3 BRIEF §3F).
//   road — the cached routed geometry, riding alongside the two timestamped
//          endpoints in routed_points.
//   a recorded stretch with a road (phase 23) — snapped_points, the road
//          its points lie on, joined to its own first and last recorded
//          points. The join matters: the matcher puts each end on the
//          nearest road, up to match_params.radius_m from where it was
//          recorded, while the neighbouring legs start and end at the
//          recorded point itself; without it the drawn line would break
//          at every seam when zoomed in. The stretch is still "observed"
//          — the kind is not read from the geometry (invariant 6).
//   else — the leg's own points (a recorded stretch point to point, or an
//          unknown gap's chord).
// The one place this choice is made: the maps, the cover thumbnails and the
// exported image all draw from it, so they cannot disagree about a leg.
export function legLine(leg: Leg, arcSegments?: number): LonLat[] {
  if (leg.gap_kind === "air")
    return greatCircleArc(lngLat(leg.points[0]), lngLat(leg.points[1]), arcSegments);
  if (leg.gap_kind === "road" && leg.routed_points)
    return leg.routed_points.map(lngLat);
  if (leg.kind === "observed" && leg.snapped_points && leg.snapped_points.length >= 2) {
    const line = leg.snapped_points.map(lngLat);
    const first = lngLat(leg.points[0]);
    const last = lngLat(leg.points[leg.points.length - 1]);
    if (!samePosition(first, line[0])) line.unshift(first);
    if (!samePosition(last, line[line.length - 1])) line.push(last);
    return line;
  }
  return leg.points.map(lngLat);
}

function samePosition(a: LonLat, b: LonLat): boolean {
  return a[0] === b[0] && a[1] === b[1];
}

// One leg → one LineString feature, drawn with legLine.
// `extraProps` lets a caller stamp every feature — the life map stamps
// adventure_id so hover and click can resolve which adventure a line is.
export function legFeatures(
  legs: Leg[],
  extraProps: Record<string, string | number> = {},
): GeoJSON.Feature[] {
  return legs.map((leg) => ({
    type: "Feature",
    properties: { ...extraProps, kind: leg.kind, gap_kind: leg.gap_kind ?? null },
    geometry: { type: "LineString", coordinates: legLine(leg) },
  }));
}

// Stops become Point features — except dwells with no observed position. The
// API's zero value means "absent" ((0,0) = absent, the same convention as
// photo positions). Drawing one would put a confident dot on null island off
// West Africa — and drag fitted bounds out until the real route is a speck.
// `propsFor` receives the stop's original index (filtering must not renumber
// it — indices are how photos and day slices address stops).
export function stopFeatures(
  stops: Stop[],
  propsFor?: (stop: Stop, index: number) => Record<string, unknown>,
): GeoJSON.Feature[] {
  return stops.flatMap((s, i) =>
    s.loc.lat === 0 && s.loc.lon === 0
      ? []
      : [
          {
            type: "Feature" as const,
            properties: { kind: "stop", ...(propsFor?.(s, i) ?? {}) },
            geometry: {
              type: "Point" as const,
              coordinates: lngLat(s.loc),
            },
          },
        ],
  );
}

// greatCircleArc interpolates the shortest path on the sphere between two
// [lon, lat] points ("slerp": treat each point as a unit vector from Earth's
// centre, blend the two vectors so every intermediate stays on the sphere,
// convert back). A straight line on the map is a straight line in projected
// coordinates — visibly wrong for a flight; the arc is what the aircraft
// approximately flew, and its length is the endpoint chord the API already
// reports as the leg's distance.
export function greatCircleArc(
  a: LonLat,
  b: LonLat,
  segments = 64,
): LonLat[] {
  const rad = Math.PI / 180;
  const [λ1, φ1] = [a[0] * rad, a[1] * rad];
  const [λ2, φ2] = [b[0] * rad, b[1] * rad];
  // Angular distance between the endpoints (haversine, on the unit sphere).
  const d =
    2 *
    Math.asin(
      Math.sqrt(
        Math.sin((φ2 - φ1) / 2) ** 2 +
          Math.cos(φ1) * Math.cos(φ2) * Math.sin((λ2 - λ1) / 2) ** 2,
      ),
    );
  if (d < 1e-9) return [a, b];
  const coords: LonLat[] = [];
  for (let i = 0; i <= segments; i++) {
    const f = i / segments;
    const A = Math.sin((1 - f) * d) / Math.sin(d);
    const B = Math.sin(f * d) / Math.sin(d);
    const x = A * Math.cos(φ1) * Math.cos(λ1) + B * Math.cos(φ2) * Math.cos(λ2);
    const y = A * Math.cos(φ1) * Math.sin(λ1) + B * Math.cos(φ2) * Math.sin(λ2);
    const z = A * Math.sin(φ1) + B * Math.sin(φ2);
    coords.push([
      Math.atan2(y, x) / rad,
      Math.atan2(z, Math.sqrt(x * x + y * y)) / rad,
    ]);
  }
  return coords;
}

// The southwest/northeast corners of everything drawn, as plain positions —
// pure data, so this stays testable; each island turns it into a MapLibre
// LngLatBounds. Null when there is nothing with a position.
export function bboxOf(
  features: GeoJSON.Feature[],
): [LonLat, LonLat] | null {
  let w = Infinity, s = Infinity, e = -Infinity, n = -Infinity;
  const extend = (c: GeoJSON.Position) => {
    if (c[0] < w) w = c[0];
    if (c[0] > e) e = c[0];
    if (c[1] < s) s = c[1];
    if (c[1] > n) n = c[1];
  };
  for (const f of features) {
    if (f.geometry.type === "LineString") f.geometry.coordinates.forEach(extend);
    else if (f.geometry.type === "Point") extend(f.geometry.coordinates);
  }
  return w === Infinity ? null : [[w, s], [e, n]];
}
