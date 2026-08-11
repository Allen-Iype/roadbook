"use client";

// The adventure page below its masthead: cover, day narrative, photos, and
// the map plate — one client component because the selected day is shared
// state between the narrative (which sets it) and the map (which dims
// everything else). Everything it renders comes from props the server
// component already fetched; no data logic lives here.
import { useMemo, useState } from "react";

import { LegKindLegend } from "@/components/legend";
import { ProvenanceBar } from "@/components/provenance-bar";
import { fmtDateRange, sliceDays } from "@/lib/slice-days";
import { DayNarrative } from "./day-narrative";
import { PhotosSection } from "./photos-section";
import { RouteMap } from "./route-map";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];
type Candidate = components["schemas"]["Candidate"];
type Photo = components["schemas"]["Photo"];

export function AdventureView({
  journey,
  candidate,
  photos,
  styleUrl,
  plate,
}: {
  journey: Journey;
  candidate?: Candidate;
  /** null when the candidate is not confirmed — photos do not exist then. */
  photos: Photo[] | null;
  styleUrl: string;
  /** Position among confirmed adventures in date order; null if unconfirmed. */
  plate: number | null;
}) {
  // Memoised: these feed effect dependencies in RouteMap, and a fresh array
  // identity per render would tear the map down on every day selection.
  const days = useMemo(() => sliceDays(journey), [journey]);
  const photoList = useMemo(() => photos ?? [], [photos]);
  const [selected, setSelected] = useState<number | null>(null);

  return (
    <div className="mt-6 grid items-start gap-10 lg:grid-cols-[minmax(24rem,32rem)_minmax(0,1fr)]">
      <article>
        <Cover
          journey={journey}
          candidate={candidate}
          dayCount={days.length}
          plate={plate}
        />
        <DayNarrative
          days={days}
          journey={journey}
          candidate={candidate}
          photos={photoList}
          selected={selected}
          onSelect={(i) => setSelected((cur) => (cur === i ? null : i))}
        />
        {photos !== null && candidate && (
          <PhotosSection candidateId={candidate.id} photos={photos} />
        )}
      </article>

      {journey.legs.length > 0 && (
        <aside className="lg:sticky lg:top-6">
          {/* The plate: map framed with a double rule, its margin printed
              below — legend, scale bar (on the canvas), and plate label as
              marginalia (DESIGN §6). */}
          <div className="border border-ink [outline:1px_solid_var(--color-ink)] [outline-offset:3px]">
            <RouteMap
              journey={journey}
              styleUrl={styleUrl}
              photos={photoList}
              days={days}
              selectedDay={selected}
              className="h-[26rem] w-full lg:h-[min(76vh,52rem)]"
            />
          </div>
          <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border border-t-0 border-ink bg-paper px-4 py-2.5 [outline:1px_solid_var(--color-ink)] [outline-offset:3px]">
            <span className="font-display text-[11.5px] font-semibold tracking-[0.22em]">
              {selected !== null
                ? `DAY ${selected} HIGHLIGHTED`
                : plate !== null
                  ? `PLATE ${roman(plate)} · FULL ROUTE`
                  : "FULL ROUTE"}
            </span>
            <LegKindLegend wordy={false} />
            {journey.stops.some((s) => !(s.loc.lat === 0 && s.loc.lon === 0)) && (
              <span className="inline-flex items-center gap-2 text-xs text-ink">
                <span className="inline-block h-2.5 w-2.5 rounded-full border border-paper bg-ink shadow-[0_0_0_1px_var(--color-ink)]" />
                <span className="font-semibold tracking-wide">Stop</span>
              </span>
            )}
          </div>
        </aside>
      )}
    </div>
  );
}

// The cover (BRIEF §1, DESIGN §4): the honest figures only — distance with
// its provenance bar, dates, days, fixes, countries, truncation and
// divergence as words. No score: ranking is triage machinery, and it stays
// in the candidates table and the confirm cell.
function Cover({
  journey,
  candidate,
  dayCount,
  plate,
}: {
  journey: Journey;
  candidate?: Candidate;
  dayCount: number;
  plate: number | null;
}) {
  const decision = candidate?.decision;
  const confirmed = decision?.action === "confirmed";
  const name =
    confirmed && decision.name
      ? decision.name
      : `Journey of ${journey.window_start.slice(0, 10)}`;
  const eyebrow = confirmed
    ? `CONFIRMED ADVENTURE${plate !== null ? ` · PLATE ${roman(plate)}` : ""}`
    : decision?.action === "dismissed"
      ? "DISMISSED CANDIDATE"
      : "CANDIDATE";
  const pctObserved =
    journey.total_km > 0
      ? Math.round((journey.observed_km / journey.total_km) * 100)
      : 0;

  return (
    <header>
      <p className="text-[11.5px] tracking-[0.24em] text-ink-2">{eyebrow}</p>
      <h1 className="mt-2 font-display text-4xl font-semibold leading-[1.05] sm:text-5xl">
        {name}
      </h1>
      <p className="mt-3 font-mono text-[13px] text-ink-2">
        {fmtDateRange(journey.window_start, journey.window_end)}
        {journey.countries.length > 0 && (
          <> · {journey.countries.map((c) => c.name).join(" · ")}</>
        )}{" "}
        · {dayCount} {dayCount === 1 ? "day" : "days"} · {journey.merged_points}{" "}
        fixes
      </p>
      {(candidate?.start_truncated || candidate?.end_truncated) && (
        // Truncation as words (bug 4 made visible), not markers. The amber
        // stays on the flag glyph: the token passes contrast as a mark but
        // not as sentence text (the CP4 a11y pass), so words are ink.
        <p className="mt-2 max-w-[52ch] text-[13px]">
          <span className="font-bold text-flag" aria-hidden>
            ⚑
          </span>{" "}
          {candidate.start_truncated &&
            "The record starts mid-journey — it began before the imported window. "}
          {candidate.end_truncated &&
            "Still in progress at the window's edge — the end shown is the cut, not the return."}
        </p>
      )}

      <div className="mt-5 border-b border-rule border-t border-t-ink py-4">
        <p className="font-display text-3xl font-semibold leading-none sm:text-4xl">
          {journey.total_km.toFixed(0)} km
          <span className="ml-2 font-sans text-base font-normal text-ink-2">
            drawn — {pctObserved}% of it measured
          </span>
        </p>
        <ProvenanceBar
          observed={journey.observed_km}
          routed={journey.routed_km}
          unknown={journey.unknown_km}
          air={journey.air_km}
          className="mb-2 mt-3"
        />
        <p className="font-mono text-xs text-ink-2">
          observed {journey.observed_km.toFixed(1)} km · routed{" "}
          {journey.routed_km.toFixed(1)} km · unknown{" "}
          {journey.unknown_km.toFixed(1)} km · air {journey.air_km.toFixed(1)}{" "}
          km
        </p>
        <p className="mt-1.5 text-xs text-ink-2">
          {journey.merged_points} points ({journey.trace_points_kept} trace +{" "}
          {journey.raw_points_kept} raw)
          {journey.google_km > 0 && (
            <> · Google&apos;s own figure {journey.google_km.toFixed(1)} km</>
          )}
          {journey.countries.length > 0 && (
            <> · countries derived from route points</>
          )}
        </p>
      </div>

      <Divergence journey={journey} />

      <p className="mb-8 mt-4 text-[11.5px] uppercase tracking-[0.18em] text-ink-2">
        {decision ? (
          <>
            {decision.action} {shortDate(decision.updated_at)}
          </>
        ) : (
          <>Undecided — confirm or dismiss in the candidates table</>
        )}
      </p>
    </header>
  );
}

// The phase 3 validation line: the road-comparable reconstruction against
// Google's own ground figure — air excluded from both sides. Flagged, it
// becomes the cover's one amber note; unflagged, a quiet line. Divergence is
// a conversation starter, never a gate: the unknown and routed legs
// explaining it are right there on the map.
function Divergence({ journey }: { journey: Journey }) {
  if (journey.divergence_pct === undefined) return null;
  const figures = (
    <>
      ground reconstruction{" "}
      <span className="font-mono">{journey.ground_km.toFixed(1)} km</span> ·
      Google&apos;s ground figure{" "}
      <span className="font-mono">{journey.google_ground_km.toFixed(1)} km</span>{" "}
      ({journey.divergence_pct >= 0 ? "+" : ""}
      {journey.divergence_pct.toFixed(1)}%)
    </>
  );
  if (!journey.divergence_flagged) {
    return <p className="mt-3 text-xs text-ink-2">{figures}</p>;
  }
  return (
    <p className="mt-4 flex gap-2.5 border-l-[3px] border-flag bg-land px-3 py-2.5 text-[13px]">
      <span className="font-bold text-flag" aria-hidden>
        ⚑
      </span>
      <span>
        Distance check flagged: {figures}. Unroutable gaps under-count, road
        detours over-count, and a truncated window compares against
        door-to-door figures — the legs explaining it are on the map.
      </span>
    </p>
  );
}

// "6 AUG 2026" — marginalia date for the decision line.
const SHORT_MONTHS = [
  "JAN",
  "FEB",
  "MAR",
  "APR",
  "MAY",
  "JUN",
  "JUL",
  "AUG",
  "SEP",
  "OCT",
  "NOV",
  "DEC",
];
function shortDate(iso: string): string {
  const [y, m, d] = iso.slice(0, 10).split("-").map(Number);
  return `${d} ${SHORT_MONTHS[m - 1]} ${y}`;
}

// Plate numbers are roman numerals in date order — atlas convention. Tens of
// adventures at most (the charter's scale), so the compact form suffices.
function roman(n: number): string {
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
