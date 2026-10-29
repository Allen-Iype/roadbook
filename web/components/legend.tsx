// The leg-kind legend as a permanent fixture (DESIGN §6, treatment T3), with
// the fixed wording used everywhere a legend appears (lib/legend.ts — shared
// with the exported image's margin since phase 14 CP3). The samples draw the
// actual channel: observed solid, routed solid over its casing, unknown
// dashed, air round-dotted — so the legend teaches the map's language, not
// just its colors.
//
// Phase 16 CP1: the descriptions are visible at every width (they used to
// yield below sm:, which left a phone reader with four names and no
// explanation — RESEARCH §2). On a phone the legend sets as two columns,
// each entry's description under its name; from sm: up it flows as one
// line. Each name is a Term: a tap opens the longer note from lib/terms.ts.
import { LEGEND_ENTRIES as ENTRIES } from "@/lib/legend";
import { Term } from "@/components/term";

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

export function LegKindLegend() {
  return (
    <ul
      aria-label="Legend"
      className="grid list-none grid-cols-2 gap-x-4 gap-y-2.5 text-xs text-ink sm:flex sm:flex-wrap sm:gap-x-5 sm:gap-y-1"
    >
      {ENTRIES.map((e) => (
        <li key={e.key} className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
          <LegSample kind={e.key} className="h-2 w-9 shrink-0 self-center sm:w-11" />
          <Term id={e.key}>
            <span className="font-semibold tracking-wide">{e.label}</span>
          </Term>
          <span data-legend-desc className="basis-full text-ink-2 sm:basis-auto">
            <span className="hidden sm:inline">— </span>
            {e.desc}
          </span>
        </li>
      ))}
    </ul>
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
