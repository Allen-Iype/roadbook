// The leg-kind legend as a permanent fixture (DESIGN §6, treatment T3), with
// the fixed wording used everywhere a legend appears. The samples draw the
// actual channel: observed solid, routed solid over its casing, unknown
// dashed, air round-dotted — so the legend teaches the map's language, not
// just its colors.
const ENTRIES = [
  { key: "observed", label: "Observed", desc: "recorded fixes" },
  { key: "routed", label: "Routed", desc: "inferred along roads" },
  { key: "unknown", label: "Unknown", desc: "straight line, nothing inferred" },
  { key: "air", label: "Air", desc: "great-circle arc" },
] as const;

// Class names are spelled out per kind: Tailwind extracts utilities from
// source text statically, so a constructed `stroke-${kind}` would silently
// generate nothing.
const STROKE: Record<(typeof ENTRIES)[number]["key"], string> = {
  observed: "stroke-observed",
  routed: "stroke-routed",
  unknown: "stroke-unknown",
  air: "stroke-air",
};

function Sample({ kind }: { kind: (typeof ENTRIES)[number]["key"] }) {
  return (
    <svg viewBox="0 0 44 8" className="h-2 w-11" aria-hidden>
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

export function LegKindLegend({ wordy = true }: { wordy?: boolean }) {
  return (
    <p className="flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-ink">
      {ENTRIES.map((e) => (
        <span key={e.key} className="inline-flex items-center gap-2">
          <Sample kind={e.key} />
          <span className="font-semibold tracking-wide">{e.label}</span>
          {wordy && <span className="text-ink-2">— {e.desc}</span>}
        </span>
      ))}
    </p>
  );
}
