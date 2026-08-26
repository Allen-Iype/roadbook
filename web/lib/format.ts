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
