// Where a day was (phase 18 "places"): the words the day narrative prints
// from what the API serves — every leg with the regions it crossed, every
// stop with its region and its nearest listed town. Nothing here looks
// anything up or measures anything: which town, how far, and whether that
// counts as "near" are decided once in Go (`Stop.place.near`), so the page,
// the CLI and any later surface word the same stop the same way. This
// module only arranges those answers into a line.
//
// Pure (no React) so vitest can read it.
import type { components } from "@/lib/api/schema";

import type { Day } from "@/lib/slice-days";

type Journey = components["schemas"]["Journey"];
type StopPlace = components["schemas"]["StopPlace"];

/** Just the fields this module reads, so tests can build small inputs. */
export type PlacesInput = {
  legs: Pick<Journey["legs"][number], "start" | "states">[];
  stops: Pick<Journey["stops"][number], "start" | "state" | "place">[];
};

export type DayRegions = {
  /** Region names in the order the day first reached them, each once. */
  names: string[];
  /** The day ended in the region it started in, after leaving it. */
  back: boolean;
};

/** The regions of a day's legs and stops, in time order.
 *
 * Each region is listed once, where the day first reached it. A day that
 * went out and came back would otherwise read "A, B, A", and a road that
 * runs along a border would read "A, B, A, B": the first is said with
 * `back`, the second is not worth saying. A leg or stop with no region (a
 * point off a simplified coastline, an unrecorded stretch) adds nothing.
 */
export function dayRegions(
  day: Pick<Day, "legIndices" | "stopIndices">,
  journey: PlacesInput,
): DayRegions {
  const pieces: { start: string; names: string[] }[] = [];
  for (const i of day.legIndices) {
    const leg = journey.legs[i];
    pieces.push({ start: leg.start, names: leg.states.map((s) => s.name) });
  }
  for (const i of day.stopIndices) {
    const stop = journey.stops[i];
    if (stop.state) pieces.push({ start: stop.start, names: [stop.state.name] });
  }
  // Timestamps carry their own offsets, so they are compared as instants.
  pieces.sort((a, b) => Date.parse(a.start) - Date.parse(b.start));
  const sequence = pieces.flatMap((p) => p.names);
  const names = [...new Set(sequence)];
  const back = names.length > 1 && sequence.at(-1) === names[0];
  return { names, back };
}

/** "Suðurland" · "Suðurland to Austurland" · "Suðurland to Austurland and
 * back" · "Telangana, Karnataka, Maharashtra". Undefined when the day has
 * no region to name. */
export function regionsPhrase(r: DayRegions): string | undefined {
  if (r.names.length === 0) return undefined;
  const list = r.names.length === 2 ? r.names.join(" to ") : r.names.join(", ");
  return r.back ? `${list} and back` : list;
}

/** "near Ísafjörður" within the near distance, "16 km from Ísafjörður"
 * beyond it. Never "in": the list is points, and the nearest point says
 * nothing about which town's boundary a stay was inside. */
export function placePhrase(place: StopPlace | undefined): string | undefined {
  if (!place) return undefined;
  if (place.near) return `near ${place.name}`;
  return `${Math.round(place.distance_km)} km from ${place.name}`;
}

/** The place phrase for each stay row of one day, in row order — printed
 * on the first row of a run of consecutive stays that share it and left
 * out on the rest, so a day spent around one town names it once. A stay
 * with no name ends the run. */
export function stayPhrases(
  stopIndices: number[],
  journey: Pick<PlacesInput, "stops">,
): (string | undefined)[] {
  let previous: string | undefined;
  return stopIndices.map((i) => {
    const phrase = placePhrase(journey.stops[i].place);
    const shown = phrase !== previous ? phrase : undefined;
    previous = phrase;
    return shown;
  });
}
