// Shared display formatting — used by the map popup, the photo strip, and
// the day narrative, so no two surfaces can phrase the same fact
// differently.

import type { components } from "@/lib/api/schema";

type PlaceKind = NonNullable<components["schemas"]["Photo"]["place_kind"]>;

export function fmtDistanceM(m: number): string {
  return m < 1000 ? `${Math.round(m)} m` : `${(m / 1000).toFixed(1)} km`;
}

/** "45 min" · "38 h" · "2 h 30 min" — elapsed time, never clock time. */
export function fmtDuration(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = Math.round(minutes % 60);
  if (h === 0) return `${m} min`;
  if (m === 0) return `${h} h`;
  return `${h} h ${m} min`;
}

/** "1 d 14 h" · "9 h 40 min" · "35 min" — a span in hours, days first once
 * it is longer than one (the summary's "away" and dwell figures). Minutes
 * drop out above a day: a span that long is not a minutes-precision
 * claim. */
export function fmtHours(hours: number): string {
  const totalMin = Math.round(hours * 60);
  if (totalMin < 24 * 60) return fmtDuration(totalMin);
  const d = Math.floor(totalMin / (24 * 60));
  const h = Math.round((totalMin - d * 24 * 60) / 60);
  if (h === 24) return `${d + 1} d`;
  return h === 0 ? `${d} d` : `${d} d ${h} h`;
}

/** A named assembly parameter off the echoed params object, or undefined:
 * the object is typed open (additionalProperties), so the value is checked
 * before it is read as a number. */
export function numberParam(
  params: { [key: string]: unknown },
  key: string,
): number | undefined {
  const v = params[key];
  return typeof v === "number" ? v : undefined;
}

/**
 * "IN_PASSENGER_VEHICLE" → "passenger vehicle" — the source's mode label
 * made readable without editorialising: strip the IN_/ON_ prefix, lowercase,
 * underscores to spaces. Unknown labels degrade to the same treatment rather
 * than throwing (the source adds labels without announcement).
 */
export function fmtMode(mode: string): string {
  return mode
    .replace(/^(IN|ON)_/, "")
    .toLowerCase()
    .replace(/_/g, " ");
}

/** "65.71°N 21.67°W" — the atlas-margin coordinate style. */
export function fmtLatLon(lat: number, lon: number): string {
  const ns = lat < 0 ? "S" : "N";
  const ew = lon < 0 ? "W" : "E";
  return `${Math.abs(lat).toFixed(2)}°${ns} ${Math.abs(lon).toFixed(2)}°${ew}`;
}

// The distance statement names which drawn geometry it was measured against
// (BRIEF §3G): the flag and the map must read as one claim.
export function placeStatement(kind: PlaceKind): string {
  switch (kind) {
    case "observed":
      return "from the observed track at this time";
    case "road":
      return "from the routed road at this time";
    case "unknown":
      return "from the straight-line gap at this time";
    case "stop":
      return "from the stop at this time";
    case "air":
      return "over an air leg — not checked against the arc";
  }
}

// Plate numbers are roman numerals in date order — atlas convention. Tens of
// adventures at most (the charter's scale), so the compact form suffices.
// Shared by the cover, the plate margin, and the exported image's label.
export function roman(n: number): string {
  const table: [number, string][] = [
    [40, "XL"],
    [10, "X"],
    [9, "IX"],
    [5, "V"],
    [4, "IV"],
    [1, "I"],
  ];
  let out = "";
  let rest = n;
  for (const [value, glyph] of table) {
    while (rest >= value) {
      out += glyph;
      rest -= value;
    }
  }
  return out;
}
