// The legend's drawn pieces: the line sample per kind and the two markers,
// as the map draws them. The samples draw the actual channel: observed
// solid, routed solid over its casing, unknown dashed, air round-dotted.
// The legend itself (the list with the fixed wording from lib/legend.ts)
// is components/legend-info.tsx since the copy pass: one button per map
// (docs/product/words.md decision 2, 2026-11-02) instead of a printed
// fixture under every map (phase 16 CP1 / DESIGN §6 as it then stood).
import { LEGEND_ENTRIES as ENTRIES } from "@/lib/legend";

type Kind = (typeof ENTRIES)[number]["key"];

// Class names are spelled out per kind: Tailwind extracts utilities from
// source text statically, so a constructed `stroke-${kind}` would silently
// generate nothing.
const STROKE: Record<Kind, string> = {
  observed: "stroke-observed",
  routed: "stroke-routed",
  unknown: "stroke-unknown",
  air: "stroke-air",
};

/** The drawn line sample for one kind — the legend's, and (phase 16) the
 * narrative row's glyph, so a row and the legend show the same stroke. */
export function LegSample({
  kind,
  className = "h-2 w-11",
}: {
  kind: Kind;
  className?: string;
}) {
  return (
    <svg viewBox="0 0 44 8" className={className} aria-hidden>
      {kind === "routed" && (
        <line x1="2" y1="4" x2="42" y2="4" strokeWidth="7" className="stroke-paper" />
      )}
      <line
        x1="2"
        y1="4"
        x2="42"
        y2="4"
        strokeWidth={kind === "observed" ? 3 : 2.5}
        strokeLinecap="round"
        strokeDasharray={
          kind === "unknown" ? "6 4" : kind === "air" ? "0.1 5.5" : undefined
        }
        className={STROKE[kind]}
      />
    </svg>
  );
}

/** The map's fix marker, as the map draws it, for the plate margin and the narrative rows. */
export function FixDot() {
  return (
    <span className="inline-block h-2 w-2 rounded-full border border-paper bg-observed shadow-[0_0_0_1px_var(--color-observed)]" />
  );
}

/** The map's stop marker, as the map draws it, for the plate margin and the
 * narrative rows. `unplaced` draws it hollow: a stay the record holds no
 * position for, which the map therefore cannot show (phase 16 review —
 * the words "position not recorded" left the row for the glyph's title). */
export function StopDot({ unplaced = false }: { unplaced?: boolean }) {
  return unplaced ? (
    <span className="inline-block h-2.5 w-2.5 rounded-full border border-ink bg-paper shadow-[inset_0_0_0_1px_var(--color-paper)]" />
  ) : (
    <span className="inline-block h-2.5 w-2.5 rounded-full border border-paper bg-ink shadow-[0_0_0_1px_var(--color-ink)]" />
  );
}
