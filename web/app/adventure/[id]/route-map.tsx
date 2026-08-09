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
    const data = toGeoJSON(journey);

    // Placed photos join the map (BRIEF §3G). Only placed ones: a photo
    // with a position but no placeable instant stays in the strip, marked
    // unplaced — absence rendered as absence.
    const placed = photos.filter((p) => p.pos && p.place_kind);

    const bounds = boundsOf(data);
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
          "line-color": "#8a8375",
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
          "line-color": "#3f7069",
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
          "line-color": "#2a5da8",
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
          "line-color": "#a81e22",
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
          "circle-color": "#26251f",
          "circle-stroke-color": "#f5f2e8",
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
      // An air leg draws as a great-circle arc between its two endpoints.
      // The arc is generated here, client-side, because it is presentation:
      // the API reports exactly the two timestamped points it measured, and
      // fabricated intermediate coordinates must not enter a response that
      // otherwise contains only measurements (phase 3 BRIEF §3F). A road
      // leg draws its cached routed geometry, which rides alongside the two
      // timestamped endpoints in routed_points.
      coordinates:
        leg.gap_kind === "air"
          ? greatCircleArc(lngLat(leg.points[0]), lngLat(leg.points[1]))
          : leg.gap_kind === "road" && leg.routed_points
            ? leg.routed_points.map(lngLat)
            : leg.points.map(lngLat),
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

// greatCircleArc interpolates the shortest path on the sphere between two
// [lon, lat] points ("slerp": treat each point as a unit vector from Earth's
// centre, blend the two vectors so every intermediate stays on the sphere,
// convert back). A straight line on the map is a straight line in projected
// coordinates — visibly wrong for a flight; the arc is what the aircraft
// approximately flew, and its length is the endpoint chord the API already
// reports as the leg's distance.
function greatCircleArc(
  a: [number, number],
  b: [number, number],
  segments = 64,
): [number, number][] {
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
  const coords: [number, number][] = [];
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

function boundsOf(data: GeoJSON.FeatureCollection): LngLatBounds {
  const bounds = new LngLatBounds();
  for (const f of data.features) {
    if (f.geometry.type === "LineString") {
      for (const c of f.geometry.coordinates) {
        bounds.extend(c as [number, number]);
      }
    } else if (f.geometry.type === "Point") {
      bounds.extend(f.geometry.coordinates as [number, number]);
    }
  }
  return bounds;
}
