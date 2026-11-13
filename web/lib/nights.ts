// Nights (phase 19): the words the trip page prints from `Journey.nights`.
// The API decides which class each night is in and which stop held it
// (internal/journey/nights.go); a night at a stop takes its town from the
// stop (phase 18). Nothing here measures anything — this module groups
// those answers and words them.
//
// Pure (no React) so vitest can read it.
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];
type Night = components["schemas"]["Night"];

/** Just the fields this module reads, so tests can build small inputs. */
export type NightsInput = {
  nights: Night[];
  stops: Pick<Journey["stops"][number], "place" | "start">[];
};

/** How many town groups the summary names before the rest become
 * "elsewhere". A long tour has dozens of night halts; the trip block is
 * one line. */
export const NIGHT_PLACES_SHOWN = 3;

export type NightsSummary = {
  count: number;
  /** "near Trimbak (2)", "on the move", "not recorded (3)" — in the order
   * printed. The number follows in brackets, and only when it is more
   * than one; a single part never carries one, the count beside it says
   * it. (The number trails because a far town's wording starts with one:
   * "13 km from Barshi (2)".) */
  parts: string[];
};

/** The Nights row: the count and where they were spent.
 *
 * Nights at a stop are grouped by the stop's town — consecutive nights in
 * one town are often different stops — and listed most nights first (ties
 * in the order the trip reached them), the first NIGHT_PLACES_SHOWN by
 * name and the remainder as "elsewhere". Stops with no town in range are
 * one group, said as such. Then nights on the move, then nights the record
 * cannot place. The numbers always add up to the count.
 */
export function nightsSummary(journey: NightsInput): NightsSummary {
  const count = journey.nights.length;
  const groups = new Map<string, { n: number; near: boolean; km: number }>();
  let unnamed = 0;
  let moving = 0;
  let unknown = 0;
  for (const night of journey.nights) {
    if (night.kind === "moving") moving++;
    else if (night.kind === "unknown") unknown++;
    else {
      const place =
        night.stop_index === undefined
          ? undefined
          : journey.stops[night.stop_index]?.place;
      if (!place) {
        unnamed++;
        continue;
      }
      const g = groups.get(place.name) ?? { n: 0, near: false, km: Infinity };
      g.n++;
      g.near ||= place.near;
      g.km = Math.min(g.km, place.distance_km);
      groups.set(place.name, g);
    }
  }
  // Map order is first-reached order; a stable sort keeps it among ties.
  const ranked = [...groups.entries()].sort((a, b) => b[1].n - a[1].n);
  const items: [number, string][] = [];
  for (const [name, g] of ranked.slice(0, NIGHT_PLACES_SHOWN)) {
    // One group, one wording: "near" if any of its nights was within the
    // near distance, else the closest of them in whole kilometres.
    items.push([g.n, g.near ? `near ${name}` : `${Math.round(g.km)} km from ${name}`]);
  }
  const elsewhere = ranked.slice(NIGHT_PLACES_SHOWN).reduce((a, [, g]) => a + g.n, 0);
  if (elsewhere > 0) items.push([elsewhere, "elsewhere"]);
  if (unnamed > 0) items.push([unnamed, "no town nearby"]);
  if (moving > 0) items.push([moving, "on the move"]);
  if (unknown > 0) items.push([unknown, "not recorded"]);
  const parts = items.map(([n, words]) =>
    items.length > 1 && n > 1 ? `${words} (${n})` : words,
  );
  return { count, parts };
}

/** The night that began on this civil date, if the trip has one. */
export function nightOf(journey: Pick<NightsInput, "nights">, date: string): Night | undefined {
  return journey.nights.find((n) => n.date === date);
}

/** The civil date before `date` ("2026-05-23" → "2026-05-22"). Dates only:
 * no clock and no zone can shift it. */
export function dateBefore(date: string): string {
  const d = new Date(`${date}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() - 1);
  return d.toISOString().slice(0, 10);
}

/** Whether a stay's row on a given day is the row its night belongs on.
 *
 * "tonight": the stop holds the night that began this day — the usual
 * case, a stay from the evening on. "lastNight": the stop holds the night
 * that began the day before and only started after midnight — a late
 * arrival — so it has no row on that earlier day to say it on.
 */
export function stayNight(
  journey: NightsInput,
  stopIndex: number,
  dayDate: string,
): "tonight" | "lastNight" | undefined {
  const tonight = nightOf(journey, dayDate);
  if (tonight?.kind === "stop" && tonight.stop_index === stopIndex) return "tonight";
  const last = nightOf(journey, dateBefore(dayDate));
  if (
    last?.kind === "stop" &&
    last.stop_index === stopIndex &&
    journey.stops[stopIndex].start.slice(0, 10) === dayDate
  ) {
    return "lastNight";
  }
  return undefined;
}
