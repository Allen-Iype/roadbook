"use client";

// The life map island — the full-viewport map that IS the home page
// (DESIGN: "map is the navigation"). Same create-once/destroy-once
// lifecycle as the adventure page's route-map: MapLibre owns the div and
// its WebGL context, React owns everything floating above it.
//
// Interaction contract (BRIEF §0 "map-as-navigation mechanics"):
//   - every leg feature carries adventure_id; promoteId lifts it to the
//     feature id, so feature-state addresses a whole adventure at once;
//   - hover = queryRenderedFeatures against the invisible 16 px hit layer,
//     feature-state raises the ink weight, a card names the adventure;
//   - click navigates to the adventure page;
//   - the canvas is aria-hidden: this surface is pointer-only, and the
//     summoned list is the accessible enumeration (DESIGN §5). That is why
//     nothing here is focusable and no keyboard handler exists on the map.
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  LngLatBounds,
  MapLibreMap,
  NavigationControl,
  setWorkerUrl,
} from "maplibre-gl";
import type { MapMouseEvent } from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

import { bboxOf, legFeatures } from "@/lib/geo";
import { HIT_LAYER, LIFE_SOURCE, lifeMapLayers } from "@/lib/life-map-layers";
import { ProvenanceBar } from "@/components/provenance-bar";
import type { components } from "@/lib/api/schema";

// Same Turbopack worker-URL trap as route-map.tsx — see the comment there.
setWorkerUrl("/maplibre/maplibre-gl-worker.mjs");

type Journey = components["schemas"]["Journey"];

// What the home server component ships per confirmed adventure: the
// candidate row's identity and the journey's legs and honest distances.
// Legs come through as the API sent them (the charter forbids slim-array
// scale machinery — tens of journeys, not thousands).
export type Adventure = {
  id: number;
  name: string;
  /** Civil dates as the traveller experienced them (span offsets kept). */
  start: string;
  end: string;
  journey: Journey;
};

export function LifeMap({
  adventures,
  styleUrl,
}: {
  adventures: Adventure[];
  styleUrl: string;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  // The hovered adventure and the pointer position, in container pixels.
  // State, not a ref: the hover card is rendered by React.
  const [hover, setHover] = useState<{
    id: number;
    x: number;
    y: number;
  } | null>(null);
  const router = useRouter();

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const features = adventures.flatMap((a) =>
      // adventure_id stamped on every leg feature — the hover/click key.
      legFeatures(a.journey.legs, { adventure_id: a.id }),
    );
    const box = bboxOf(features);
    // Nothing with a position (cannot happen for a confirmed journey, but
    // the type allows it): render no map rather than a null-island view.
    if (!box) return;

    const map = new MapLibreMap({
      container,
      style: styleUrl,
      bounds: new LngLatBounds(box[0], box[1]),
      // Generous padding: the corner chrome floats over the canvas, and a
      // route ending under the legend would be a route hidden.
      fitBoundsOptions: { padding: 96 },
      attributionControl: { compact: false },
    });
    map.addControl(new NavigationControl({ showCompass: false }), "bottom-right");

    // The container is aria-hidden, and aria-hidden with focusable children
    // is worse than either alone: keyboard focus lands on elements assistive
    // tech cannot see. MapLibre ships its canvas, zoom buttons, and
    // attribution links focusable; on this pointer-only surface (DESIGN §5)
    // none of them may take focus — keyboard users reach every adventure
    // through the summoned list, never through the canvas. Run after
    // construction and again once the first full render settles, when the
    // style-driven attribution links exist.
    const stripFocus = () =>
      container
        .querySelectorAll<HTMLElement>("a, button, canvas, [tabindex]")
        .forEach((el) => (el.tabIndex = -1));
    stripFocus();
    map.once("idle", stripFocus);

    map.on("load", () => {
      map.addSource(LIFE_SOURCE, {
        type: "geojson",
        data: { type: "FeatureCollection", features },
        // promoteId: the adventure_id property becomes the feature id,
        // which is what setFeatureState addresses. Every leg of an
        // adventure shares the id, so one call highlights the whole route.
        promoteId: "adventure_id",
      });
      for (const layer of lifeMapLayers()) map.addLayer(layer);
    });

    // Which adventure the pointer is over — resolved against the hit layer
    // only. A tolerance box around the cursor would double the hit area the
    // hit layer already provides; the layer's 16 px width is the tolerance.
    const idAt = (e: MapMouseEvent): number | null => {
      if (!map.getLayer(HIT_LAYER)) return null; // style still loading
      const hits = map.queryRenderedFeatures(e.point, { layers: [HIT_LAYER] });
      return hits.length > 0 ? (hits[0].id as number) : null;
    };

    // hoveredId lives outside React state: feature-state updates are
    // MapLibre's own repaint path and must not wait for a render pass.
    let hoveredId: number | null = null;
    map.on("mousemove", (e) => {
      const id = idAt(e);
      if (id !== hoveredId) {
        if (hoveredId !== null)
          map.setFeatureState({ source: LIFE_SOURCE, id: hoveredId }, { hover: false });
        if (id !== null)
          map.setFeatureState({ source: LIFE_SOURCE, id }, { hover: true });
        hoveredId = id;
        map.getCanvas().style.cursor = id === null ? "" : "pointer";
      }
      setHover(id === null ? null : { id, x: e.point.x, y: e.point.y });
    });
    map.on("mouseout", () => {
      if (hoveredId !== null)
        map.setFeatureState({ source: LIFE_SOURCE, id: hoveredId }, { hover: false });
      hoveredId = null;
      setHover(null);
    });
    map.on("click", (e) => {
      const id = idAt(e);
      if (id !== null) router.push(`/adventure/${id}`);
    });

    return () => {
      map.remove();
      setHover(null);
    };
  }, [adventures, styleUrl, router]);

  const hovered = hover && adventures.find((a) => a.id === hover.id);

  return (
    <div className="absolute inset-0">
      {/* aria-hidden: for assistive tech this canvas is decoration; the
          summoned list carries the same adventures as real links. */}
      <div ref={containerRef} className="h-full w-full" aria-hidden />
      {hovered && <HoverCard adventure={hovered} x={hover.x} y={hover.y} />}
    </div>
  );
}

// The hover card: name, dates, honest distance with its provenance bar
// (invariant 8 — even a tooltip never states a distance without showing how
// much of it is measurement). Positioned near the pointer, flipped when the
// pointer is in the right or bottom third so it stays on screen.
function HoverCard({
  adventure,
  x,
  y,
}: {
  adventure: Adventure;
  x: number;
  y: number;
}) {
  const j = adventure.journey;
  const flipX = typeof window !== "undefined" && x > window.innerWidth * 0.62;
  const flipY = typeof window !== "undefined" && y > window.innerHeight * 0.7;
  return (
    <div
      aria-hidden
      className="pointer-events-none absolute z-10 w-64 border border-rule bg-paper px-4 py-3 shadow-sm"
      style={{
        left: x + (flipX ? -16 : 16),
        top: y + (flipY ? -12 : 12),
        transform: `translate(${flipX ? "-100%" : "0"}, ${flipY ? "-100%" : "0"})`,
      }}
    >
      <p className="font-display text-base font-semibold leading-snug">
        {adventure.name}
      </p>
      <p className="mt-0.5 font-mono text-xs text-ink-2">
        {adventure.start.slice(0, 10)} → {adventure.end.slice(0, 10)}
      </p>
      <p className="mt-2 font-mono text-sm">
        {j.total_km.toFixed(0)} km
        <span className="ml-1 text-xs text-ink-2">
          — {j.observed_km.toFixed(0)} observed
        </span>
      </p>
      <ProvenanceBar
        observed={j.observed_km}
        routed={j.routed_km}
        unknown={j.unknown_km}
        air={j.air_km}
        className="mt-1.5"
      />
      <p className="mt-2 text-xs text-ink-2">Click to open</p>
    </div>
  );
}
