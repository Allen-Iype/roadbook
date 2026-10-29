// The leg-kind legend's fixed wording (DESIGN §6) — one constant, every
// legend reads it: the on-page legend component, the plate margin, and the
// exported image's margin (phase 14 CP3). An image travels without the page
// that would have explained its four inks, so the image's legend must be
// THIS wording, and a test pins that the image builder cannot omit it.
// Pure module (no React) so vitest can read it.
//
// Phase 16 CP1 (BRIEF §5C, decided at the CP1 STOP, 2026-10-29): the
// descriptions say in plain words what the pipeline terms said —
// "recorded fixes" and "great-circle arc" were on the research's leakage
// list. "The phone", not "your phone": the same legend sits on a shared
// view and travels on an exported image, where the reader is not the
// phone's owner. The four names, the order, the samples and the legend's
// presence everywhere are unchanged.
// The longer teaching notes behind each name live in lib/terms.ts.
export const LEGEND_ENTRIES = [
  { key: "observed", label: "Observed", desc: "the phone recorded it" },
  { key: "routed", label: "Routed", desc: "drawn along roads" },
  { key: "unknown", label: "Unknown", desc: "nothing recorded, straight line" },
  { key: "air", label: "Air", desc: "a flight" },
] as const;

export type LegendKind = (typeof LEGEND_ENTRIES)[number]["key"];
