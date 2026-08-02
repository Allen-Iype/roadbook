"use client";

// The map island — the one place React hands a DOM node to an imperative
// library and steps back (BRIEF §1.1). MapLibre owns the div below and its
// WebGL context; React owns everything around it. The journey and style URL
// arrive as server-component props and never change while the map is mounted,
// so the lifecycle is create-once/destroy-once: the effect builds the map,
// the cleanup releases it, and StrictMode's dev-mode double-mount
// (create → destroy → create) is the framework proving the cleanup correct.
import { useEffect, useRef } from "react";
// MapLibre 6 is ESM-only with named exports (no default export); MapLibreMap
// is the library's own alias for its Map class, avoiding the clash with the
// JS built-in.
import {
  LngLatBounds,
  MapLibreMap,
  NavigationControl,
  setWorkerUrl,
} from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

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

export function RouteMap({
  journey,
  styleUrl,
}: {
  journey: Journey;
  styleUrl: string;
}) {
  // A ref, not state: the container node and the map instance are not
  // renderable data, and mutating a ref does not schedule a re-render.
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const map = new MapLibreMap({
      container,
      style: styleUrl,
      // Real position comes from fitBounds below; without a center MapLibre
      // would flash null island first.
      bounds: boundsOf(journey),
      fitBoundsOptions: { padding: 48 },
    });
    map.addControl(new NavigationControl({ showCompass: false }));

    // Sources and layers only after the style has loaded — adding them
    // synchronously after the constructor is the classic first bug.
    map.on("load", () => {
      map.addSource("route", { type: "geojson", data: toGeoJSON(journey) });

      // Layer order is paint order (BRIEF §1.2): gaps underneath, observed
      // above them, stop markers on top. The visual channel (§3A): observed
      // is solid, saturated, wide; a gap is dashed, thin, muted — sketched
      // in, not asserted. Hue stays in reserve for phase 3's road and air
      // classes. MapLibre 6 types line-dasharray as data-drivable, but
      // layer-per-class keeps the z-order between classes for free.
      map.addLayer({
        id: "gap-legs",
        type: "line",
        source: "route",
        filter: ["==", ["get", "kind"], "gap"],
        paint: {
          "line-color": "#a3a3a3",
          "line-width": 2,
          "line-dasharray": [2, 3],
        },
      });
      map.addLayer({
        id: "observed-legs",
        type: "line",
        source: "route",
        filter: ["==", ["get", "kind"], "observed"],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#10b981",
          "line-width": 3.5,
        },
      });
      map.addLayer({
        id: "stops",
        type: "circle",
        source: "route",
        filter: ["==", ["get", "kind"], "stop"],
        paint: {
          "circle-radius": 5,
          "circle-color": "#fbbf24",
          "circle-stroke-color": "#171717",
          "circle-stroke-width": 1.5,
        },
      });
    });

    // The cleanup releases the WebGL context; browsers cap live contexts,
    // and a leaked map only surfaces navigations later.
    return () => {
      map.remove();
    };
  }, [journey, styleUrl]);

  return (
    <div
      ref={containerRef}
      className="mt-6 h-[28rem] w-full rounded-md border border-neutral-800"
    />
  );
}

// The one [lat, lon] → [lon, lat] conversion in the frontend (BRIEF §1.2):
// the API speaks {lat, lon}; GeoJSON demands [longitude, latitude]. Every
// coordinate on the map passes through these two functions.
function lngLat(p: { lat: number; lon: number }): [number, number] {
  return [p.lon, p.lat];
}

function toGeoJSON(journey: Journey): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = journey.legs.map((leg) => ({
    type: "Feature",
    properties: { kind: leg.kind, gap_kind: leg.gap_kind ?? null },
    geometry: {
      type: "LineString",
      coordinates: leg.points.map(lngLat),
    },
  }));
  for (const stop of journey.stops) {
    features.push({
      type: "Feature",
      properties: { kind: "stop" },
      geometry: { type: "Point", coordinates: lngLat(stop.loc) },
    });
  }
  return { type: "FeatureCollection", features };
}

function boundsOf(journey: Journey): LngLatBounds {
  const bounds = new LngLatBounds();
  for (const leg of journey.legs) {
    for (const p of leg.points) {
      bounds.extend(lngLat(p));
    }
  }
  return bounds;
}
