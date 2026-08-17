// The Atlas plate leg inks and grounds for map paint (DESIGN §6). MapLibre
// paints into a WebGL canvas and cannot read CSS custom properties, so the
// map needs these as JS values. globals.css `@theme` declares the same hexes
// for every DOM surface; the two files must agree — this one exists so that
// inside map code the value appears exactly once, named (invariant 3 applied
// to color, per the phase 6 BRIEF §0).
export const INK = {
  observed: "#a81e22", // the traveller's own red ink: solid, uncased
  routed: "#2a5da8", // the printed road: solid over a paper casing
  unknown: "#8a8375", // drawn, never asserted: dashed
  air: "#3f7069", // great-circle arc: round-dotted
  known: "#3a3a34", // T2 overview only: observed+routed merged
  ink: "#26251f",
  paper: "#f5f2e8",
  flag: "#8a5c0b", // honesty warnings only; ≥4.5:1 on both grounds (scripts/contrast.mjs)
} as const;
