import { describe, expect, it } from "vitest";

import {
  REJECTION_REDIRECTS,
  WELCOME_SECTIONS,
  redirectFor,
} from "./rejection-anchors";

// The no-dead-ends regression (phase 7 BRIEF §7.3: "every sniffer label
// routes to a sensible anchor — walked for all thirteen"). This list mirrors
// the taxonomy in internal/timeline/parse.go: the thirteen labels Sniff can
// return synchronously, plus the two Parse discovers later on a failed
// imports row. If the Go taxonomy grows, this test is the visible failure
// that says the web map fell behind.
const SNIFF_LABELS = [
  "empty",
  "gzip",
  "zip",
  "pdf",
  "image",
  "html",
  "kml",
  "xml",
  "binary",
  "not-json",
  "semantic-history",
  "records-json",
  "my-activity",
] as const;

const PARSE_LABELS = ["truncated", "json-unrecognised"] as const;

const PAGE_ANCHORS = new Set<string>(Object.values(WELCOME_SECTIONS));

describe("rejection redirection", () => {
  it.each([...SNIFF_LABELS, ...PARSE_LABELS])(
    "label %s maps to an existing /welcome section",
    (label) => {
      // Explicitly in the map — redirectFor's fallback must never be doing
      // the work for a label we know about.
      expect(REJECTION_REDIRECTS[label]).toBeDefined();
      const r = redirectFor(label);
      expect(PAGE_ANCHORS.has(r.anchor)).toBe(true);
      expect(r.link.length).toBeGreaterThan(0);
    },
  );

  it("has no stale entries for labels the taxonomy does not produce", () => {
    const known = new Set<string>([...SNIFF_LABELS, ...PARSE_LABELS]);
    for (const label of Object.keys(REJECTION_REDIRECTS)) {
      expect(known.has(label), `stale map entry: ${label}`).toBe(true);
    }
  });

  it("unknown and absent labels fall back to a real section", () => {
    for (const format of [undefined, "some-future-label"]) {
      const r = redirectFor(format);
      expect(PAGE_ANCHORS.has(r.anchor)).toBe(true);
    }
  });
});
