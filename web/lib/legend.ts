// The leg-kind legend's fixed wording (DESIGN §6) — one constant, every
// legend reads it: the on-page legend component, the plate margin, and the
// exported image's margin (phase 14 CP3). An image travels without the page
// that would have explained its four inks, so the image's legend must be
// THIS wording, and a test pins that the image builder cannot omit it.
// Pure module (no React) so vitest can read it.
export const LEGEND_ENTRIES = [
  { key: "observed", label: "Observed", desc: "recorded fixes" },
  { key: "routed", label: "Routed", desc: "inferred along roads" },
  { key: "unknown", label: "Unknown", desc: "straight line, nothing inferred" },
  { key: "air", label: "Air", desc: "great-circle arc" },
] as const;

export type LegendKind = (typeof LEGEND_ENTRIES)[number]["key"];
