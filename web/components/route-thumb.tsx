// Renders a lib/route-thumb.ts Thumb as an inline SVG in the Atlas
// encoding: the reserved leg inks with their non-color channels (observed
// solid and widest, routed solid over a paper casing, unknown dashed, air
// round-dotted) — invariant 8 at thumbnail size. Stroke classes are spelled
// out because Tailwind extracts utilities statically.
import { routeThumb, type ThumbKind } from "@/lib/route-thumb";
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];

const STROKE: Record<ThumbKind, string> = {
  observed: "stroke-observed",
  road: "stroke-routed",
  unknown: "stroke-unknown",
  air: "stroke-air",
};

const DASH: Record<ThumbKind, string | undefined> = {
  observed: undefined,
  road: undefined,
  unknown: "3 3",
  air: "0.1 3.4",
};

export function RouteThumb({
  legs,
  className,
}: {
  legs: Leg[];
  className?: string;
}) {
  const thumb = routeThumb(legs);
  if (!thumb) return null;
  return (
    <svg viewBox={thumb.viewBox} className={className} aria-hidden>
      {thumb.paths.map((p, i) => (
        // Routed rides on a paper casing, exactly like the map layers; the
        // casing renders immediately under its ink so route crossings keep
        // the z-order the paint order established.
        <g key={i}>
          {p.kind === "road" && (
            <path
              d={p.d}
              fill="none"
              strokeWidth={3.5}
              strokeLinecap="round"
              strokeLinejoin="round"
              className="stroke-paper"
            />
          )}
          <path
            d={p.d}
            fill="none"
            strokeWidth={p.kind === "observed" ? 2.25 : 1.6}
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeDasharray={DASH[p.kind]}
            className={STROKE[p.kind]}
          />
        </g>
      ))}
    </svg>
  );
}
