// The adventure plate's route layers — one builder for the on-screen map
// and the offscreen export map (phase 14 CP3). Extracted from the route-map
// island the moment a second MapLibre instance needed the same layers: the
// exported image must draw exactly what the plate draws, and two copies of
// the layer list is how they would eventually drift (the same reason
// lib/geo.ts and lib/life-map-layers.ts exist).
//
// Pure module: no MapLibre import at runtime (the type imports are erased),
// no DOM, no map instance. Both islands feed these objects to addSource /
// addLayer / setPaintProperty unchanged, and route-layers.test.ts asserts
// over exactly the same objects — what is tested is what is drawn, on the
// page and in the file.
import type {
  ExpressionSpecification,
  LayerSpecification,
} from "maplibre-gl";

import { legFeatures, lngLat, stopFeatures } from "@/lib/geo";
import { isFixLeg, type Day } from "@/lib/slice-days";
import { INK } from "@/lib/tokens";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];

export const ROUTE_SOURCE = "route";

/** The padding fitBounds applies on the plate — the export map fits the
 * same bounds with the same padding, so the two frame the route alike. */
export const FIT_PADDING = 48;

// Everything outside the selected day fades to this opacity — dimmed, never
// hidden: geometry that vanished would misstate what the journey contains.
export const DIM_OPACITY = 0.15;

// One FeatureCollection feeds both the source and the fitted bounds, so an
// air leg's arc — which bulges away from the straight line between its
// endpoints — can never fall outside the view. Legs are converted one at a
// time because each carries its own day; a fix (a stationary observed leg)
// additionally becomes a Point feature the "fixes" layer can draw — a
// LineString with nowhere to go paints nothing, and observed evidence would
// render as absence (invariant 8, found in phase 9 CP2). Same predicate as
// the narrative's fix events, so the map dot and the "Fix —" line always
// agree. Stops carry the list of civil days they overlap.
export function routeFeatures(
  journey: Pick<Journey, "legs" | "stops">,
  days: Day[] = [],
): GeoJSON.FeatureCollection {
  const { legDay, stopDays } = dayLookups(days);
  return {
    type: "FeatureCollection",
    features: [
      ...journey.legs.flatMap((leg, i) =>
        legFeatures([leg], { day: legDay.get(i) ?? 0 }),
      ),
      ...journey.legs.flatMap((leg, i) =>
        isFixLeg(leg)
          ? [
              {
                type: "Feature" as const,
                properties: { kind: "fix", day: legDay.get(i) ?? 0 },
                geometry: {
                  type: "Point" as const,
                  coordinates: lngLat(leg.points[0]),
                },
              },
            ]
          : [],
      ),
      ...stopFeatures(journey.stops, (_s, i) => ({
        days: stopDays.get(i) ?? [],
      })),
    ],
  };
}

/** Day assignment lookups from the sliceDays output: a leg's start day, and
 * every civil day a stop overlaps. Photo markers use the same lookups. */
export function dayLookups(days: Day[]): {
  legDay: Map<number, number>;
  stopDays: Map<number, number[]>;
} {
  const legDay = new Map<number, number>();
  const stopDays = new Map<number, number[]>();
  for (const d of days) {
    for (const li of d.legIndices) legDay.set(li, d.index);
    for (const si of d.stopIndices)
      stopDays.set(si, [...(stopDays.get(si) ?? []), d.index]);
  }
  return { legDay, stopDays };
}

// Layer order is paint order, bottom to top: unknown gaps underneath (a
// dashed grey guess never covers a measurement), then air, then routed over
// its paper casing, observed on top, fix dots, stop markers above all. The
// channel is the Atlas plate encoding (DESIGN §6): kind is never hue alone —
// observed is solid and widest and uncased, routed is solid over its
// casing, unknown is dashed, air is round-dotted. Unlike the life map there
// is no zoom graduation here: the detail plate keeps the full four-way
// split at every zoom (DESIGN §2), and so does the exported image.
export const ROUTE_LAYERS: LayerSpecification[] = [
  {
    id: "unknown-legs",
    type: "line",
    source: ROUTE_SOURCE,
    filter: [
      "all",
      ["==", ["get", "kind"], "gap"],
      ["!=", ["get", "gap_kind"], "air"],
      ["!=", ["get", "gap_kind"], "road"],
    ],
    paint: {
      "line-color": INK.unknown,
      "line-width": 2,
      "line-dasharray": [2, 3],
    },
  },
  {
    id: "air-legs",
    type: "line",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "gap_kind"], "air"],
    layout: { "line-cap": "round" },
    paint: {
      "line-color": INK.air,
      "line-width": 2.2,
      "line-dasharray": [0.1, 2],
    },
  },
  {
    id: "road-casing",
    type: "line",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "gap_kind"], "road"],
    layout: { "line-cap": "round", "line-join": "round" },
    paint: {
      "line-color": INK.paper,
      "line-width": 5.5,
    },
  },
  {
    id: "road-legs",
    type: "line",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "gap_kind"], "road"],
    layout: { "line-cap": "round", "line-join": "round" },
    paint: {
      "line-color": INK.routed,
      "line-width": 2.5,
    },
  },
  {
    id: "observed-legs",
    type: "line",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "kind"], "observed"],
    layout: { "line-cap": "round", "line-join": "round" },
    paint: {
      "line-color": INK.observed,
      "line-width": 3.5,
    },
  },
  // Fixes wear the observed ink — they ARE measurements — at a smaller
  // radius than stops, whose black dots stay the page's "you stayed here"
  // marks. Below stops in paint order: where a dwell and its gate fix
  // coincide, the stop reads on top.
  {
    id: "fixes",
    type: "circle",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "kind"], "fix"],
    paint: {
      "circle-radius": 4,
      "circle-color": INK.observed,
      "circle-stroke-color": INK.paper,
      "circle-stroke-width": 1.25,
    },
  },
  {
    id: "stops",
    type: "circle",
    source: ROUTE_SOURCE,
    filter: ["==", ["get", "kind"], "stop"],
    paint: {
      "circle-radius": 5,
      "circle-color": INK.ink,
      "circle-stroke-color": INK.paper,
      "circle-stroke-width": 1.5,
    },
  },
];

const LINE_LAYERS = [
  "unknown-legs",
  "air-legs",
  "road-casing",
  "road-legs",
  "observed-legs",
] as const;

// Day-highlight opacity: legs carry their start day (`day`), stops the list
// of civil days they overlap (`days`) — both stamped from the same sliceDays
// output that drives the narrative, so the two cannot disagree. Fix dots
// carry a leg's start day, so they dim on the leg rule.
const legOpacity = (sel: number | null): number | ExpressionSpecification =>
  sel === null ? 1 : ["case", ["==", ["get", "day"], sel], 1, DIM_OPACITY];
const stopOpacity = (sel: number | null): number | ExpressionSpecification =>
  sel === null ? 1 : ["case", ["in", sel, ["get", "days"]], 1, DIM_OPACITY];

export type PaintChange = {
  layer: string;
  property: "line-opacity" | "circle-opacity" | "circle-stroke-opacity";
  value: number | ExpressionSpecification;
};

/** The paint changes that highlight one day (or restore the whole route
 * when `sel` is null) — applied with setPaintProperty by both maps. */
export function highlightPaint(sel: number | null): PaintChange[] {
  return [
    ...LINE_LAYERS.map((layer): PaintChange => ({
      layer,
      property: "line-opacity",
      value: legOpacity(sel),
    })),
    { layer: "stops", property: "circle-opacity", value: stopOpacity(sel) },
    { layer: "stops", property: "circle-stroke-opacity", value: stopOpacity(sel) },
    { layer: "fixes", property: "circle-opacity", value: legOpacity(sel) },
    { layer: "fixes", property: "circle-stroke-opacity", value: legOpacity(sel) },
  ];
}
