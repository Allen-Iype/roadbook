"use client";

// The adventure plate below its masthead: cover, day narrative, photos, and
// the map — one client component because the selected day is shared state
// between the narrative (which sets it) and the map (which dims everything
// else). Everything it renders comes from props the server component
// already fetched; no data logic lives here.
//
// Two consumers (phase 13 CP4): the owner's page (app shell) and the
// read-only shared view (public shell). The plate is the same — a stranger
// sees exactly the legs, kinds, and photos the owner sees — and the two
// differ only in what surrounds it, which arrives as props: the owner's
// islands (photo upload, share controls) as slots, and `shared` carrying
// the cover facts the shared read provides instead of a candidate.
import { useMemo, useState } from "react";

import { LegKindLegend } from "@/components/legend";
import { ProvenanceBar } from "@/components/provenance-bar";
import { fmtMode } from "@/lib/format";
import { fmtDateRange, isFixLeg, sliceDays } from "@/lib/slice-days";
import { DayNarrative } from "./day-narrative";
import { ImportPhotosStrip, PhotoStrip } from "./photo-strip";
import { RouteMap } from "./route-map";
import {
  OWNER_THUMBS,
  displayAttached,
  displayImported,
  type ThumbRoots,
} from "@/lib/photo-display";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];
type Candidate = components["schemas"]["Candidate"];
type Photo = components["schemas"]["Photo"];
type ImportPhoto = components["schemas"]["ImportPhoto"];

/** What the shared view knows about the adventure instead of a candidate. */
export type SharedCover = {
  name: string;
  start_truncated: boolean;
  end_truncated: boolean;
};

export function AdventureView({
  journey,
  candidate,
  photos,
  importPhotos,
  styleUrl,
  plate,
  shared,
  thumbs = OWNER_THUMBS,
  photosSection,
  shareControls,
}: {
  journey: Journey;
  candidate?: Candidate;
  /** null when the candidate is not confirmed — photos do not exist then. */
  photos: Photo[] | null;
  /** Records span-joined to this candidate (CP4) — any candidate, read-time. */
  importPhotos: ImportPhoto[];
  styleUrl: string;
  /** Position among confirmed adventures in date order; null if unconfirmed. */
  plate: number | null;
  /** Set on the shared view: the cover facts, and read-only everywhere. */
  shared?: SharedCover;
  /** Which thumbnail proxies the tiles and markers load through. */
  thumbs?: ThumbRoots;
  /** The owner's photo upload/delete island; absent on the shared view. */
  photosSection?: React.ReactNode;
  /** The owner's share-link controls; absent on the shared view. */
  shareControls?: React.ReactNode;
}) {
  // Memoised: these feed effect dependencies in RouteMap, and a fresh array
  // identity per render would tear the map down on every day selection.
  const days = useMemo(() => sliceDays(journey), [journey]);
  // Both provenances flatten to one display list for the map and the
  // narrative — the strips below keep them apart, where capability differs.
  const attachedList = useMemo(
    () => (photos ?? []).map((p) => displayAttached(p, thumbs)),
    [photos, thumbs],
  );
  const importedList = useMemo(
    () => importPhotos.map((p) => displayImported(p, thumbs)),
    [importPhotos, thumbs],
  );
  const photoList = useMemo(
    () => [...attachedList, ...importedList],
    [attachedList, importedList],
  );
  const [selected, setSelected] = useState<number | null>(null);

  return (
    <div className="mt-6 grid items-start gap-10 lg:grid-cols-[minmax(24rem,32rem)_minmax(0,1fr)]">
      <article>
        <Cover
          journey={journey}
          candidate={candidate}
          dayCount={days.length}
          plate={plate}
          shared={shared}
        />
        {shareControls}
        <DayNarrative
          days={days}
          journey={journey}
          candidate={shared ?? candidate}
          photos={photoList}
          selected={selected}
          onSelect={(i) => setSelected((cur) => (cur === i ? null : i))}
        />
        {photosSection}
        {shared ? (
          <>
            <PhotoStrip
              heading="Photos"
              intro={
                <>
                  {attachedList.length === 1
                    ? "One photo"
                    : `${attachedList.length} photos`}{" "}
                  the owner added to this adventure, placed by capture time
                  against the drawn route.
                </>
              }
              photos={attachedList}
            />
            <PhotoStrip
              heading="From the owner's photo imports"
              intro={
                <>
                  {importedList.length === 1
                    ? "One photo"
                    : `${importedList.length} photos`}{" "}
                  from the owner&apos;s photo imports{" "}
                  {importedList.length === 1 ? "was" : "were"} taken inside
                  this journey&apos;s window — placed by capture time against
                  the drawn route.
                </>
              }
              photos={importedList}
            />
          </>
        ) : (
          <ImportPhotosStrip photos={importedList} />
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
            {journey.legs.some(isFixLeg) && (
              <span className="inline-flex items-center gap-2 text-xs text-ink">
                <span className="inline-block h-2 w-2 rounded-full border border-paper bg-observed shadow-[0_0_0_1px_var(--color-observed)]" />
                <span className="font-semibold tracking-wide">Fix</span>
              </span>
            )}
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
  shared,
}: {
  journey: Journey;
  candidate?: Candidate;
  dayCount: number;
  plate: number | null;
  shared?: SharedCover;
}) {
  const decision = candidate?.decision;
  const confirmed = decision?.action === "confirmed";
  const name = shared
    ? shared.name
    : confirmed && decision.name
      ? decision.name
      : `Journey of ${journey.window_start.slice(0, 10)}`;
  // The shared eyebrow names what a stranger is looking at; the plate
  // number is the owner's atlas register and stays theirs.
  const eyebrow = shared
    ? "SHARED ADVENTURE"
    : confirmed
      ? `CONFIRMED ADVENTURE${plate !== null ? ` · PLATE ${roman(plate)}` : ""}`
      : decision?.action === "dismissed"
        ? "DISMISSED CANDIDATE"
        : "CANDIDATE";
  const truncation = shared ?? candidate;
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
        )}
        {/* Regions after their countries, comma-joined so the two levels
            read apart: "Iceland · Vestfirðir, Suðurland" (phase 14 CP1).
            Local names with their diacritics — what the road sign said. */}
        {journey.states.length > 0 && (
          <> · {journey.states.map((s) => s.name).join(", ")}</>
        )}{" "}
        · {dayCount} {dayCount === 1 ? "day" : "days"} · {journey.merged_points}{" "}
        fixes
      </p>
      {(truncation?.start_truncated || truncation?.end_truncated) && (
        // Truncation as words (bug 4 made visible), not markers. The amber
        // stays on the flag glyph: the token passes contrast as a mark but
        // not as sentence text (the CP4 a11y pass), so words are ink.
        <p className="mt-2 max-w-[52ch] text-[13px]">
          <span className="font-bold text-flag" aria-hidden>
            ⚑
          </span>{" "}
          {truncation.start_truncated &&
            "The record starts mid-journey — it began before the imported window. "}
          {truncation.end_truncated &&
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
            <>
              {" · "}
              {journey.states.length > 0
                ? "countries and regions derived from route points"
                : "countries derived from route points"}
            </>
          )}
        </p>
        <ModeLine journey={journey} />
      </div>

      <Divergence journey={journey} />

      {shared ? (
        <p className="mb-8 mt-4 text-[11.5px] uppercase tracking-[0.18em] text-ink-2">
          Read-only · shared by its owner
        </p>
      ) : (
        <p className="mb-8 mt-4 text-[11.5px] uppercase tracking-[0.18em] text-ink-2">
          {decision ? (
            <>
              {decision.action} {shortDate(decision.updated_at)}
            </>
          ) : (
            <>Undecided — confirm or dismiss in the candidates table</>
          )}
        </p>
      )}
    </header>
  );
}

// The source-asserted mode line (phase 11 §6.2): Google's own labels and
// distances, visually subordinate to the measured figures above it and never
// summed with them — the source's guesses have recorded failures (a 1,023 km
// "motorcycling" relocation; probability-0.00 modes), and the pipeline
// itself trusts speed over mode for flight classification. Absent breakdown
// (a photo-sourced journey has no activities) states the absence, never
// zeros. Reproduction: `roadbook journey -candidate N` prints the same line.
function ModeLine({ journey }: { journey: Journey }) {
  const bd = journey.mode_breakdown;
  if (bd === undefined) {
    return (
      <p className="mt-1.5 text-xs text-ink-2">
        No mode record — this journey&apos;s evidence carries no activity
        data.
      </p>
    );
  }
  if (bd.length === 0) return null;
  return (
    <p className="mt-1.5 text-xs text-ink-2">
      By mode, as the source recorded it (modes are guesses):{" "}
      {bd.map((m, i) => (
        <span key={m.mode}>
          {i > 0 && " · "}
          {fmtMode(m.mode)}{" "}
          <span className="font-mono">{m.km.toFixed(1)} km</span>
        </span>
      ))}
    </p>
  );
}

// The phase 3 validation line: the road-comparable reconstruction against
// Google's own ground figure — air excluded from both sides. Flagged, it
// becomes the cover's one amber note; unflagged, a quiet line. Divergence is
// a conversation starter, never a gate: the unknown and routed legs
// explaining it are right there on the map.
function Divergence({ journey }: { journey: Journey }) {
  if (journey.divergence_pct === undefined) return null;
  // Spacing rides in {" "} expressions throughout: SWC drops inter-tag
  // whitespace in some constructs (the phase 7 trap — this very line
  // rendered "62.7 km· Google's" and "172.0 km(-63.5%)" until phase 9's
  // screenshot review caught it).
  const figures = (
    <>
      ground reconstruction{" "}
      <span className="font-mono">{journey.ground_km.toFixed(1)} km</span>
      {" · Google's ground figure "}
      <span className="font-mono">{journey.google_ground_km.toFixed(1)} km</span>
      {" ("}
      {journey.divergence_pct >= 0 ? "+" : ""}
      {journey.divergence_pct.toFixed(1)}
      {"%)"}
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
