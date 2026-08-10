// Day slicing for the adventure narrative (phase 6 BRIEF §0, §3A). One pure
// function cuts a journey's legs and stops into civil days — in the
// journey's own recorded UTC offsets, never the viewer's timezone — and
// returns day headings, events, and per-day distance. The narrative sections
// and the map highlight both consume this output, so they cannot disagree
// about which day owns what.
//
// The midnight rule, as approved: an event belongs to the civil day it
// starts in. A transit crossing midnight appears once, under its start day,
// with a note naming the day it ends; per-day distances sum drawn leg
// distances by start day; nothing is ever split at midnight — a split would
// manufacture a synthetic position and a proportional km figure inside
// inferred geometry.
//
// Civil dates and clocks are read straight off the RFC 3339 strings
// (`iso.slice(0, 10)` / `iso.slice(11, 16)`)) — the API preserves each
// timestamp's recorded offset, so the string IS the traveller's civil time.
// Durations, by contrast, are differences between instants (Date.parse),
// which is what keeps them correct across offset changes mid-journey.
import type { components } from "@/lib/api/schema";

type Leg = components["schemas"]["Leg"];
type Stop = components["schemas"]["Stop"];

/** The subset of a Journey that slicing reads (structural, so tests can
 * build minimal inputs). */
export type SliceInput = {
  window_start: string;
  window_end: string;
  legs: Leg[];
  stops: Stop[];
};

export type TransitKind = "observed" | "road" | "unknown" | "air";

export type DayEvent =
  // A stationary observed leg — a recorded position and nothing more. The
  // sparse journeys' single-point legs (phase 3: the airport gate renders
  // honestly) are exactly these.
  | {
      type: "fix";
      legIndex: number;
      start: string;
      end: string;
      points: number;
      lat: number;
      lon: number;
    }
  // A moving leg. km is the drawn distance (routed figure for road gaps),
  // chordKm the straight-line figure; overnight names the 1-based day the
  // leg ends on when that is a different civil day, else null.
  | {
      type: "transit";
      legIndex: number;
      kind: TransitKind;
      start: string;
      end: string;
      km: number;
      chordKm: number;
      points: number;
      overnight: number | null;
    }
  // A dwell contained in one civil day.
  | {
      type: "dwell";
      stopIndex: number;
      start: string;
      end: string;
      minutes: number;
    }
  // A multi-day dwell: it begins under its start day (naming the day it
  // ends), fills each intermediate day, and ends under its end day with the
  // true elapsed duration.
  | { type: "dwellBegin"; stopIndex: number; at: string; endsDay: number }
  | { type: "dwellDay"; stopIndex: number }
  | { type: "dwellEnd"; stopIndex: number; at: string; minutes: number }
  // A day fully inside a multi-day transit with no dwell to explain it —
  // emitted only when the day would otherwise be blank.
  | { type: "transitDay"; legIndex: number; kind: TransitKind }
  // The detection window's edges, anchoring the first and last day.
  | { type: "windowEdge"; edge: "start" | "end"; at: string };

export type Day = {
  /** 1-based, in calendar order. */
  index: number;
  /** Civil date, "2026-05-22", in the journey's own offsets. */
  date: string;
  /** "Friday 22 May" — the year lives on the cover's dateline. */
  label: string;
  /** Drawn km of legs starting this day (the midnight rule's sum). */
  km: number;
  /** Legs assigned to this day — by start day, the same rule as km. */
  legIndices: number[];
  /** Stops overlapping this civil day (a dwell spans every day it covers). */
  stopIndices: number[];
  /** Time-ordered events. Empty means: nothing recorded this day. */
  events: DayEvent[];
};

const WEEKDAYS = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

const DAY_MS = 86_400_000;

function dateOf(iso: string): string {
  return iso.slice(0, 10);
}

/** Traveller-local clock, "15:10" — the page-wide convention. */
export function clockOf(iso: string): string {
  return iso.slice(11, 16);
}

// Calendar arithmetic runs on Date.UTC over the civil date's own fields —
// never on the full timestamp, whose offset would shift the date.
function utcMidnight(date: string): number {
  const [y, m, d] = date.split("-").map(Number);
  return Date.UTC(y, m - 1, d);
}

export function dayLabel(date: string): string {
  const [, m, d] = date.split("-").map(Number);
  const weekday = WEEKDAYS[new Date(utcMidnight(date)).getUTCDay()];
  return `${weekday} ${d} ${MONTHS[m - 1]}`;
}

/** The cover's dateline range, from civil dates in the journey's own
 * offsets: "22–24 May 2026", "28 May – 2 June 2026", or the full form
 * across a year boundary. */
export function fmtDateRange(startIso: string, endIso: string): string {
  const [ay, am, ad] = dateOf(startIso).split("-").map(Number);
  const [by, bm, bd] = dateOf(endIso).split("-").map(Number);
  if (ay === by && am === bm && ad === bd) return `${ad} ${MONTHS[am - 1]} ${ay}`;
  if (ay === by && am === bm) return `${ad}–${bd} ${MONTHS[am - 1]} ${ay}`;
  if (ay === by)
    return `${ad} ${MONTHS[am - 1]} – ${bd} ${MONTHS[bm - 1]} ${ay}`;
  return `${ad} ${MONTHS[am - 1]} ${ay} – ${bd} ${MONTHS[bm - 1]} ${by}`;
}

/** The distance a leg is drawn with: the routed figure where road geometry
 * exists, the chord (observed sum or gap chord) otherwise. */
export function drawnKm(leg: Leg): number {
  return leg.gap_kind === "road" && leg.routed_km !== undefined
    ? leg.routed_km
    : leg.distance_km;
}

function minutesBetween(start: string, end: string): number {
  return Math.round((Date.parse(end) - Date.parse(start)) / 60_000);
}

export function sliceDays(journey: SliceInput): Day[] {
  // The day range: every civil date any timestamp touches, window edges
  // included. Lexical min/max is chronological for YYYY-MM-DD.
  let first = dateOf(journey.window_start);
  let last = dateOf(journey.window_end);
  const touch = (iso: string) => {
    const d = dateOf(iso);
    if (d < first) first = d;
    if (d > last) last = d;
  };
  for (const leg of journey.legs) {
    touch(leg.start);
    touch(leg.end);
  }
  for (const s of journey.stops) {
    touch(s.start);
    touch(s.end);
  }

  const days: Day[] = [];
  const byDate = new Map<string, Day>();
  for (let ms = utcMidnight(first); ms <= utcMidnight(last); ms += DAY_MS) {
    const date = new Date(ms).toISOString().slice(0, 10);
    const day: Day = {
      index: days.length + 1,
      date,
      label: dayLabel(date),
      km: 0,
      legIndices: [],
      stopIndices: [],
      events: [],
    };
    days.push(day);
    byDate.set(date, day);
  }

  // Legs: assigned to their start day, whole (the midnight rule).
  journey.legs.forEach((leg, legIndex) => {
    const day = byDate.get(dateOf(leg.start))!;
    day.legIndices.push(legIndex);
    day.km += drawnKm(leg);
    if (
      leg.kind === "observed" &&
      (leg.points.length === 1 || leg.distance_km === 0)
    ) {
      day.events.push({
        type: "fix",
        legIndex,
        start: leg.start,
        end: leg.end,
        points: leg.points.length,
        lat: leg.points[0].lat,
        lon: leg.points[0].lon,
      });
    } else {
      const endDate = dateOf(leg.end);
      day.events.push({
        type: "transit",
        legIndex,
        kind: leg.kind === "observed" ? "observed" : (leg.gap_kind ?? "unknown"),
        start: leg.start,
        end: leg.end,
        km: drawnKm(leg),
        chordKm: leg.distance_km,
        points: leg.points.length,
        overnight:
          endDate === dateOf(leg.start) ? null : byDate.get(endDate)!.index,
      });
    }
  });

  // Stops: a dwell belongs to every civil day it overlaps.
  journey.stops.forEach((s, stopIndex) => {
    const startDate = dateOf(s.start);
    const endDate = dateOf(s.end);
    for (const day of days) {
      if (day.date >= startDate && day.date <= endDate) {
        day.stopIndices.push(stopIndex);
      }
    }
    if (startDate === endDate) {
      byDate.get(startDate)!.events.push({
        type: "dwell",
        stopIndex,
        start: s.start,
        end: s.end,
        minutes: minutesBetween(s.start, s.end),
      });
    } else {
      byDate.get(startDate)!.events.push({
        type: "dwellBegin",
        stopIndex,
        at: s.start,
        endsDay: byDate.get(endDate)!.index,
      });
      byDate.get(endDate)!.events.push({
        type: "dwellEnd",
        stopIndex,
        at: s.end,
        minutes: minutesBetween(s.start, s.end),
      });
      for (
        let ms = utcMidnight(startDate) + DAY_MS;
        ms < utcMidnight(endDate);
        ms += DAY_MS
      ) {
        const date = new Date(ms).toISOString().slice(0, 10);
        byDate.get(date)?.events.push({ type: "dwellDay", stopIndex });
      }
    }
  });

  // A blank day fully inside a multi-day leg gets the leg named — a day the
  // reader cannot account for is worse than a quiet "still under way".
  for (const day of days) {
    if (day.events.length > 0) continue;
    const spanning = journey.legs.findIndex(
      (leg) => dateOf(leg.start) < day.date && dateOf(leg.end) > day.date,
    );
    if (spanning >= 0) {
      const leg = journey.legs[spanning];
      day.events.push({
        type: "transitDay",
        legIndex: spanning,
        kind: leg.kind === "observed" ? "observed" : (leg.gap_kind ?? "unknown"),
      });
    }
  }

  // Time order within each day. dwellDay/transitDay describe the whole day
  // and sort first; ties between instants keep construction order (legs in
  // journey order), which puts a fix before the transit departing from it —
  // except a dwell's end, which precedes anything sharing its instant: you
  // finish dwelling, then depart.
  const keyOf = (e: DayEvent): number => {
    switch (e.type) {
      case "dwellDay":
      case "transitDay":
        return -Infinity;
      case "dwellBegin":
      case "dwellEnd":
      case "windowEdge":
        return Date.parse(e.at);
      default:
        return Date.parse(e.start);
    }
  };
  const tieOf = (e: DayEvent): number => (e.type === "dwellEnd" ? 0 : 1);
  for (const day of days) {
    day.events.sort((a, b) => keyOf(a) - keyOf(b) || tieOf(a) - tieOf(b));
  }

  // The window's edges frame the story: first event of the first day, last
  // event of the last day — placed explicitly so no same-instant event ever
  // sorts around them.
  byDate.get(dateOf(journey.window_start))?.events.unshift({
    type: "windowEdge",
    edge: "start",
    at: journey.window_start,
  });
  byDate.get(dateOf(journey.window_end))?.events.push({
    type: "windowEdge",
    edge: "end",
    at: journey.window_end,
  });

  return days;
}
