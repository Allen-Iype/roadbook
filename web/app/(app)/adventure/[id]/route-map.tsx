"use client";

// The map island — the one place React hands a DOM node to an imperative
// library and steps back (BRIEF §1.1). MapLibre owns the div below and its
// WebGL context; React owns everything around it. The journey and style URL
// arrive as server-component props and never change while the map is mounted,
// so the map itself is create-once/destroy-once — but the *selected day* is
// live client state, and rebuilding a WebGL map per click would flash and
// refetch tiles. Hence two effects: the main one builds the map; a second,
// cheap one repaints opacity when the selection changes, addressing the map
// through a ref. This is the standard split for imperative libraries under
// React: reconstruction for identity changes, mutation for style changes.
import { useEffect, useRef } from "react";
// MapLibre 6 is ESM-only with named exports (no default export); MapLibreMap
// is the library's own alias for its Map class, avoiding the clash with the
// JS built-in.
import {
  LngLatBounds,
  MapLibreMap,
  Marker,
  NavigationControl,
  Popup,
  ScaleControl,
  setWorkerUrl,
} from "maplibre-gl";
import type { ExpressionSpecification } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

import { fmtDistanceM, placeStatement } from "@/lib/format";
// Geometry conversions shared with the life map (extracted checkpoint 2):
// one implementation of leg→GeoJSON, arcs, and bounds for every map.
import { bboxOf, legFeatures, lngLat, stopFeatures } from "@/lib/geo";
import { INK } from "@/lib/tokens";
import { isFixLeg, type Day } from "@/lib/slice-days";
import type { DisplayPhoto } from "@/lib/photo-display";
import type { components } from "@/lib/api/schema";

// MapLibre parses tiles in a Web Worker whose URL it derives from its own
// import.meta.url — which, once Turbopack has bundled the library, points at
// a chunk directory where the worker file does not exist. The failure is
// silent: the style never finishes loading, 'load' never fires, the map stays
// blank. The library's escape hatch is setWorkerUrl; the worker and its one
// import are copied into public/maplibre/ by the copy:maplibre npm script
// (predev/prebuild), so they are always the pinned package's own files.
setWorkerUrl("/maplibre/maplibre-gl-worker.mjs");

type Journey = components["schemas"]["Journey"];

// Everything outside the selected day fades to this opacity — dimmed, never
// hidden: geometry that vanished would misstate what the journey contains.
const DIM_OPACITY = 0.15;

const LINE_LAYERS = [
  "unknown-legs",
  "air-legs",
  "road-casing",
  "road-legs",
  "observed-legs",
] as const;

// Day-highlight opacity: legs carry their start day (`day`), stops the list
// of civil days they overlap (`days`) — both stamped from the same sliceDays
// output that drives the narrative, so the two cannot disagree.
const legOpacity = (sel: number | null): number | ExpressionSpecification =>
  sel === null ? 1 : ["case", ["==", ["get", "day"], sel], 1, DIM_OPACITY];
const stopOpacity = (sel: number | null): number | ExpressionSpecification =>
  sel === null ? 1 : ["case", ["in", sel, ["get", "days"]], 1, DIM_OPACITY];

type MarkerInfo = { el: HTMLElement; days: number[] };

function applyHighlight(
  map: MapLibreMap,
  markers: MarkerInfo[],
  sel: number | null,
) {
  for (const id of LINE_LAYERS) {
    if (map.getLayer(id)) map.setPaintProperty(id, "line-opacity", legOpacity(sel));
  }
  if (map.getLayer("stops")) {
    map.setPaintProperty("stops", "circle-opacity", stopOpacity(sel));
    map.setPaintProperty("stops", "circle-stroke-opacity", stopOpacity(sel));
  }
  // Fix dots carry a leg's start day, so they dim on the leg rule.
  if (map.getLayer("fixes")) {
    map.setPaintProperty("fixes", "circle-opacity", legOpacity(sel));
    map.setPaintProperty("fixes", "circle-stroke-opacity", legOpacity(sel));
  }
  for (const m of markers) {
    m.el.style.opacity =
      sel === null || m.days.includes(sel) ? "1" : String(DIM_OPACITY);
  }
}

export function RouteMap({
  journey,
  styleUrl,
  photos = [],
  days = [],
  selectedDay = null,
  className = "h-[28rem] w-full",
}: {
  journey: Journey;
  styleUrl: string;
  photos?: DisplayPhoto[];
  /** sliceDays output; when present, features carry day assignments. */
  days?: Day[];
  /** 1-based day to highlight; null shows the whole route full-strength. */
  selectedDay?: number | null;
  className?: string;
}) {
  // Refs, not state: the container node, the map instance, and the marker
  // elements are not renderable data, and mutating a ref does not schedule
  // a re-render.
  const containerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MapLibreMap | null>(null);
  const markersRef = useRef<MarkerInfo[]>([]);
  // The selection the load handler reads — the style may still be loading
  // when the user first clicks a day.
  const selectedRef = useRef<number | null>(selectedDay);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // Day assignment lookups from the sliceDays output.
    const legDay = new Map<number, number>();
    const stopDays = new Map<number, number[]>();
    for (const d of days) {
      for (const li of d.legIndices) legDay.set(li, d.index);
      for (const si of d.stopIndices)
        stopDays.set(si, [...(stopDays.get(si) ?? []), d.index]);
    }

    // Built once per mount: the same FeatureCollection feeds both the source
    // and the bounds, so an air leg's arc — which bulges away from the
    // straight line between its endpoints — can never fall outside the view.
    // Legs are converted one at a time because each carries its own day.
    const data: GeoJSON.FeatureCollection = {
      type: "FeatureCollection",
      features: [
        ...journey.legs.flatMap((leg, i) =>
          legFeatures([leg], { day: legDay.get(i) ?? 0 }),
        ),
        // A fix — a stationary observed leg — is a LineString with nowhere
        // to go: one point (or two coincident ones) paints nothing, so
        // observed evidence would render as absence (invariant 8, found in
        // phase 9 CP2). Each fix additionally becomes a Point feature the
        // "fixes" layer can draw. Same predicate as the narrative's fix
        // events, so the map dot and the "Fix —" line always agree.
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

    // Placed photos join the map (BRIEF §3G). Only placed ones: a photo
    // with a position but no placeable instant stays in the strip, marked
    // unplaced — absence rendered as absence.
    const placed = photos.filter((p) => p.pos && p.place_kind);

    // This page renders the map only when legs exist, so the box is never
    // null here; the assertion documents that rather than papering over it.
    const box = bboxOf(data.features);
    if (!box) return;
    const bounds = new LngLatBounds(box[0], box[1]);
    // A flagged photo can sit far off the route; the bounds include it
    // because a disagreement pushed off-screen is a disagreement hidden.
    for (const p of placed) bounds.extend(lngLat(p.pos!));

    const map = new MapLibreMap({
      container,
      style: styleUrl,
      // Real position comes from fitBounds below; without a center MapLibre
      // would flash null island first.
      bounds,
      fitBoundsOptions: { padding: 48 },
    });
    map.addControl(new NavigationControl({ showCompass: false }));
    // The scale bar is plate marginalia the atlas frame promised (DESIGN
    // §6); MapLibre keeps it honest across zoom and latitude.
    map.addControl(new ScaleControl({ maxWidth: 120 }), "bottom-left");
    mapRef.current = map;

    // Photo markers are DOM overlays (Marker), not style layers: there are
    // tens of them at most, each is a thumbnail image, and DOM markers need
    // no sprite loading. The marker is the measurement — solid, confident;
    // the amber ring is the far flag: the photo disagreeing with the drawn
    // route, not a doubt about the photo (invariants 5 and 8).
    // Only thumbnail-bearing photos become markers: a record with no
    // thumbnail (HEIC) has nothing to show as one — its position is already
    // in the drawn geometry as a fix, and its facts live in the strip.
    const markers = placed.filter((p) => p.thumb_url !== null).map((p) => {
      const el = document.createElement("img");
      el.src = p.thumb_url!;
      el.alt = p.original_name;
      el.style.cssText =
        "width:34px;height:34px;object-fit:cover;border-radius:6px;cursor:pointer;" +
        "transition:opacity .3s;" +
        (p.far_flagged
          ? `border:2.5px solid ${INK.flag};box-shadow:0 0 0 2px #f5f2e8;`
          : "border:1.5px solid #f5f2e8;box-shadow:0 0 0 1px #26251f;");

      const popup = new Popup({ offset: 20, closeButton: false });
      const body = document.createElement("div");
      body.style.cssText = "font-size:12px;color:#26251f;max-width:220px;";
      const title = document.createElement("strong");
      title.textContent = p.original_name; // textContent: filenames are user input
      body.appendChild(title);
      const line = document.createElement("div");
      line.textContent = p.taken_at
        ? `${p.taken_at.slice(0, 10)} ${p.taken_at.slice(11, 16)} (${p.time_source})`
        : "no capture time";
      body.appendChild(line);
      if (p.distance_from_route_m !== undefined && p.place_kind) {
        const dist = document.createElement("div");
        if (p.far_flagged) {
          // Amber on the glyph only — the flag token fails contrast as
          // running text (CP4 a11y pass); the words stay ink.
          const flag = document.createElement("span");
          flag.textContent = "⚑ ";
          flag.style.cssText = `color:${INK.flag};font-weight:700;`;
          dist.appendChild(flag);
        }
        dist.appendChild(
          document.createTextNode(
            `${fmtDistanceM(p.distance_from_route_m)} ${placeStatement(p.place_kind)}`,
          ),
        );
        body.appendChild(dist);
      }
      popup.setDOMContent(body);

      const marker = new Marker({ element: el })
        .setLngLat(lngLat(p.pos!))
        .setPopup(popup)
        .addTo(map);

      // The photo dims with the day its placement belongs to: a leg photo
      // follows the leg's start day, a stop photo every day its dwell spans.
      const photoDays =
        p.leg_index !== undefined
          ? legDay.has(p.leg_index)
            ? [legDay.get(p.leg_index)!]
            : []
          : p.stop_index !== undefined
            ? (stopDays.get(p.stop_index) ?? [])
            : [];
      return { marker, info: { el, days: photoDays } };
    });
    markersRef.current = markers.map((m) => m.info);

    // Sources and layers only after the style has loaded — adding them
    // synchronously after the constructor is the classic first bug.
    map.on("load", () => {
      map.addSource("route", { type: "geojson", data });

      // Layer order is paint order, bottom to top: unknown gaps underneath
      // (a dashed grey guess never covers a measurement), then air, then
      // routed over its paper casing, observed on top, stop markers above
      // all. The channel is the Atlas plate encoding (DESIGN §6): kind is
      // never hue alone — observed is solid and widest and uncased, routed
      // is solid over its casing, unknown is dashed, air is round-dotted.
      // Unlike the life map there is no zoom graduation here: the detail
      // plate keeps the full four-way split at every zoom (DESIGN §2).
      map.addLayer({
        id: "unknown-legs",
        type: "line",
        source: "route",
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
      });
      map.addLayer({
        id: "air-legs",
        type: "line",
        source: "route",
        filter: ["==", ["get", "gap_kind"], "air"],
        layout: { "line-cap": "round" },
        paint: {
          "line-color": INK.air,
          "line-width": 2.2,
          "line-dasharray": [0.1, 2],
        },
      });
      map.addLayer({
        id: "road-casing",
        type: "line",
        source: "route",
        filter: ["==", ["get", "gap_kind"], "road"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": INK.paper,
          "line-width": 5.5,
        },
      });
      map.addLayer({
        id: "road-legs",
        type: "line",
        source: "route",
        filter: ["==", ["get", "gap_kind"], "road"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": INK.routed,
          "line-width": 2.5,
        },
      });
      map.addLayer({
        id: "observed-legs",
        type: "line",
        source: "route",
        filter: ["==", ["get", "kind"], "observed"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": INK.observed,
          "line-width": 3.5,
        },
      });
      // Fixes wear the observed ink — they ARE measurements — at a smaller
      // radius than stops, whose black dots stay the page's "you stayed
      // here" marks. Below stops in paint order: where a dwell and its
      // gate fix coincide, the stop reads on top.
      map.addLayer({
        id: "fixes",
        type: "circle",
        source: "route",
        filter: ["==", ["get", "kind"], "fix"],
        paint: {
          "circle-radius": 4,
          "circle-color": INK.observed,
          "circle-stroke-color": INK.paper,
          "circle-stroke-width": 1.25,
        },
      });
      map.addLayer({
        id: "stops",
        type: "circle",
        source: "route",
        filter: ["==", ["get", "kind"], "stop"],
        paint: {
          "circle-radius": 5,
          "circle-color": INK.ink,
          "circle-stroke-color": INK.paper,
          "circle-stroke-width": 1.5,
        },
      });

      // A day may already be selected while the style was loading.
      applyHighlight(map, markersRef.current, selectedRef.current);
    });

    // The cleanup releases the WebGL context; browsers cap live contexts,
    // and a leaked map only surfaces navigations later. Markers are removed
    // first — map.remove() would orphan their DOM elements otherwise.
    return () => {
      markers.forEach((m) => m.marker.remove());
      markersRef.current = [];
      mapRef.current = null;
      map.remove();
    };
  }, [journey, styleUrl, photos, days]);

  // Selection changes repaint opacity in place — never rebuild the map.
  useEffect(() => {
    selectedRef.current = selectedDay;
    const map = mapRef.current;
    if (map) applyHighlight(map, markersRef.current, selectedDay);
  }, [selectedDay]);

  return <div ref={containerRef} className={className} />;
}
