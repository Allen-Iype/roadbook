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
  Marker,
  NavigationControl,
  Popup,
  setWorkerUrl,
} from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

import { fmtDistanceM, placeStatement } from "@/lib/format";
// Geometry conversions shared with the life map (extracted checkpoint 2):
// one implementation of leg→GeoJSON, arcs, and bounds for every map.
import { bboxOf, legFeatures, lngLat, stopFeatures } from "@/lib/geo";
import { INK } from "@/lib/tokens";
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
type Photo = components["schemas"]["Photo"];

export function RouteMap({
  journey,
  styleUrl,
  photos = [],
}: {
  journey: Journey;
  styleUrl: string;
  photos?: Photo[];
}) {
  // A ref, not state: the container node and the map instance are not
  // renderable data, and mutating a ref does not schedule a re-render.
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // Built once per mount: the same FeatureCollection feeds both the source
    // and the bounds, so an air leg's arc — which bulges away from the
    // straight line between its endpoints — can never fall outside the view.
    const data: GeoJSON.FeatureCollection = {
      type: "FeatureCollection",
      features: [...legFeatures(journey.legs), ...stopFeatures(journey.stops)],
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

    // Photo markers are DOM overlays (Marker), not style layers: there are
    // tens of them at most, each is a thumbnail image, and DOM markers need
    // no sprite loading. The marker is the measurement — solid, confident;
    // the amber ring is the far flag: the photo disagreeing with the drawn
    // route, not a doubt about the photo (invariants 5 and 8).
    const markers = placed.map((p) => {
      const el = document.createElement("img");
      el.src = `/api/photos/${p.id}/thumb`;
      el.alt = p.original_name;
      el.style.cssText =
        "width:34px;height:34px;object-fit:cover;border-radius:6px;cursor:pointer;" +
        (p.far_flagged
          ? "border:2.5px solid #b97f10;box-shadow:0 0 0 2px #f5f2e8;"
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
        dist.textContent = `${fmtDistanceM(p.distance_from_route_m)} ${placeStatement(p.place_kind)}`;
        if (p.far_flagged) dist.style.color = "#b97f10";
        body.appendChild(dist);
      }
      popup.setDOMContent(body);

      return new Marker({ element: el })
        .setLngLat(lngLat(p.pos!))
        .setPopup(popup)
        .addTo(map);
    });

    // Sources and layers only after the style has loaded — adding them
    // synchronously after the constructor is the classic first bug.
    map.on("load", () => {
      map.addSource("route", { type: "geojson", data });

      // Layer order is paint order (BRIEF §1.2): unknown gaps underneath,
      // then air, then observed, stop markers on top (phase 3 BRIEF §3F).
      // The visual channel: dashes are the one non-negotiable inference
      // marker — every non-observed class is dashed — and hue, held in
      // reserve through phase 2, now distinguishes *kinds* of inference.
      // MapLibre 6 types line-dasharray as data-drivable, but
      // layer-per-class keeps the z-order between classes for free.
      map.addLayer({
        id: "gap-legs",
        type: "line",
        source: "route",
        filter: [
          "all",
          ["==", ["get", "kind"], "gap"],
          ["!=", ["get", "gap_kind"], "air"],
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
        paint: {
          "line-color": INK.air,
          "line-width": 2,
          "line-dasharray": [2, 3],
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
          "line-color": INK.observed,
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
          "circle-color": INK.ink,
          "circle-stroke-color": INK.paper,
          "circle-stroke-width": 1.5,
        },
      });
    });

    // The cleanup releases the WebGL context; browsers cap live contexts,
    // and a leaked map only surfaces navigations later. Markers are removed
    // first — map.remove() would orphan their DOM elements otherwise.
    return () => {
      markers.forEach((m) => m.remove());
      map.remove();
    };
  }, [journey, styleUrl, photos]);

  return (
    <div
      ref={containerRef}
      className="mt-6 h-[28rem] w-full rounded-md border border-rule"
    />
  );
}

