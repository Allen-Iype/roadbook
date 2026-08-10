// The life map's style layers — T2, the zoom-graduated honesty channel
// (DESIGN §2). This module is pure data: no MapLibre import at runtime (the
// type import below is erased), no DOM, no map instance. The island feeds
// these objects to map.addLayer unchanged, and the invariant-8 unit test
// asserts over exactly the same objects — what is tested is what is drawn.
//
// The channel, stated once:
//   - At overview zooms, observed and routed merge into one "known ground"
//     ink. The two distinctions that survive at country scale — known vs
//     unknown, ground vs air — are exactly the ones a reader can still see.
//   - Unknown stays dashed grey and air stays a dotted arc at EVERY zoom.
//     Their paint carries no zoom expression at all: the honest classes are
//     constants, and only the two measured-or-inferred-along-roads classes
//     graduate. That asymmetry IS invariant 8 here (BRIEF §0), and the test
//     in life-map-layers.test.ts is its regression statement: at no zoom
//     does an unknown leg share the known-ground style.
import type {
  ExpressionSpecification,
  LayerSpecification,
} from "maplibre-gl";

import { INK } from "@/lib/tokens";

// The T2 thresholds, named (invariant 3 applied to styling): merged at and
// below MERGE_MAX, fully split at and above SPLIT. Between them MapLibre
// interpolates — a gradual reveal, not a pop. The demo life map fits Iceland
// around zoom 5–6 (merged); an adventure's own extent sits around 8–9
// (split) — the split completes as the reader crosses from "where did my
// life happen" to "what exactly happened here".
export const T2_MERGE_MAX_ZOOM = 7;
export const T2_SPLIT_ZOOM = 9;

// One GeoJSON source holds every confirmed adventure's legs; promoteId
// lifts the adventure_id property into the feature id so feature-state
// (hover) addresses a whole adventure at once — every leg of it — without
// per-feature bookkeeping.
export const LIFE_SOURCE = "adventures";

// The invisible pointer-target layer. 2–3 px inks are hopeless hover
// targets; this transparent copy of every leg, 16 px wide, is what
// queryRenderedFeatures hits. Opacity 0 keeps it out of the visual channel
// entirely (the test asserts that), and MapLibre still reports its features
// — querying reads geometry buckets, not pixels.
export const HIT_LAYER = "life-hit-area";

// Hover thickens the whole adventure: the multiplier applies to every leg
// kind alike (width is not part of the honesty channel; dash and hue are).
const hoverBoost: ExpressionSpecification = [
  "case",
  ["boolean", ["feature-state", "hover"], false],
  1.7,
  1,
];

// MapLibre requires ["zoom"] to feed a top-level interpolate, so the hover
// multiplier lives inside each stop's output (the documented
// "zoom-and-feature-state" composition) rather than wrapping the whole
// expression.
const width = (base: number, split: number): ExpressionSpecification => [
  "interpolate",
  ["linear"],
  ["zoom"],
  T2_MERGE_MAX_ZOOM,
  ["*", base, hoverBoost],
  T2_SPLIT_ZOOM,
  ["*", split, hoverBoost],
];

// The T2 color graduation for the two known-ground classes only: one ink
// below the merge zoom, the class's own ink at the split zoom.
const splitColor = (ink: string): ExpressionSpecification => [
  "interpolate",
  ["linear"],
  ["zoom"],
  T2_MERGE_MAX_ZOOM,
  INK.known,
  T2_SPLIT_ZOOM,
  ink,
];

// Layer order is paint order, bottom to top: unknown under everything (a
// dashed grey guess never covers a measurement), then air, then routed over
// its casing, observed on top, and the transparent hit area above all.
export function lifeMapLayers(): LayerSpecification[] {
  return [
    {
      id: "life-unknown",
      type: "line",
      source: LIFE_SOURCE,
      filter: ["==", ["get", "gap_kind"], "unknown"],
      paint: {
        // Constants, deliberately: no zoom expression may ever appear on
        // this layer's color or dash (invariant 8; the test enforces it).
        "line-color": INK.unknown,
        "line-width": ["*", 2, hoverBoost],
        "line-dasharray": [2, 3],
      },
    },
    {
      id: "life-air",
      type: "line",
      source: LIFE_SOURCE,
      filter: ["==", ["get", "gap_kind"], "air"],
      layout: { "line-cap": "round" },
      paint: {
        // Same rule as unknown: constant color, constant dot pattern.
        "line-color": INK.air,
        "line-width": ["*", 2.2, hoverBoost],
        "line-dasharray": [0.1, 2],
      },
    },
    {
      id: "life-routed-casing",
      type: "line",
      source: LIFE_SOURCE,
      filter: ["==", ["get", "gap_kind"], "road"],
      layout: { "line-cap": "round", "line-join": "round" },
      paint: {
        "line-color": INK.paper,
        "line-width": ["*", 5.5, hoverBoost],
        // The casing is part of the split identity of "routed", so it fades
        // in with the split; while merged, observed and routed are one
        // indistinguishable known-ground ink — no casing betraying which is
        // which at a zoom where the pair reads as a single fact.
        "line-opacity": [
          "interpolate",
          ["linear"],
          ["zoom"],
          T2_MERGE_MAX_ZOOM,
          0,
          T2_SPLIT_ZOOM,
          1,
        ],
      },
    },
    {
      id: "life-routed",
      type: "line",
      source: LIFE_SOURCE,
      filter: ["==", ["get", "gap_kind"], "road"],
      layout: { "line-cap": "round", "line-join": "round" },
      paint: {
        "line-color": splitColor(INK.routed),
        "line-width": width(2.5, 2.5),
      },
    },
    {
      id: "life-observed",
      type: "line",
      source: LIFE_SOURCE,
      filter: ["==", ["get", "kind"], "observed"],
      layout: { "line-cap": "round", "line-join": "round" },
      paint: {
        "line-color": splitColor(INK.observed),
        // Widths are equal while merged (one ink, one weight — nothing
        // reveals the split early) and observed takes its wider detail
        // weight as the split completes.
        "line-width": width(2.5, 3.5),
      },
    },
    {
      id: HIT_LAYER,
      type: "line",
      source: LIFE_SOURCE,
      paint: {
        "line-color": INK.ink,
        "line-opacity": 0,
        "line-width": 16,
      },
    },
  ];
}
