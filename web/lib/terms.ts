// Teaching notes for the load-bearing terms (phase 16 BRIEF §1 "teaching in
// place", §5D): one or two sentences each, no figures, opened by the Term
// control on the term itself where it is first met. The notes live here,
// beside the legend's fixed wording, so that the page, the docs and the
// CLI's help cannot drift apart — there is one place to change a note.
//
// The vocabulary is the one RESEARCH §4 marked load-bearing: the plate and
// its number, the provenance bar, the four leg kinds, the candidate — and
// (CP3) the triage score, which DESIGN §4 keeps on the candidate card and
// which a phone could not otherwise learn about. The
// pipeline's own terms (fix, dwell, window, trace, raw, divergence) are not
// taught; the pages stopped using them instead.
//
// Pure module (no React) so vitest can read it.
export const TERMS = {
  plate: {
    term: "Plate",
    note:
      "Your kept trips are numbered in the order you travelled, the oldest first.",
  },
  observed: {
    term: "Observed",
    note:
      "The phone recorded positions along this stretch. The line joins them exactly as recorded; nothing is added.",
  },
  routed: {
    term: "Routed",
    note:
      "Nothing was recorded between two positions, so the line follows the roads between them. The distance is the road's, not a measurement.",
  },
  unknown: {
    term: "Unknown",
    note:
      "Nothing was recorded here and no road was looked up. A dashed straight line marks the gap, and the distance is that line's.",
  },
  air: {
    term: "Air",
    note:
      "A flight, told by its speed. The dotted arc is the shortest path between where it began and ended, not where the aircraft flew.",
  },
  originals: {
    term: "Original photos",
    note:
      "A photo straight from the camera carries the time and place it was taken, so Roadbook can put it on the route. A copy sent through WhatsApp or most other messengers has that stripped out, and cannot be placed.",
  },
  score: {
    term: "Score",
    note:
      "How much this looks like a trip, from 0 to 100: how far from home, how long you stayed, how much your phone recorded. It only sets the order; you decide.",
  },
} as const;

export type TermKey = keyof typeof TERMS;
