"use client";

// The adventure plate below its masthead: cover, the map plate, day
// narrative and photos — one client component because the selected day is
// shared state between the narrative (which sets it) and the map (which
// dims everything else). Everything it renders comes from props the server
// component already fetched; no data logic lives here.
//
// Two consumers (phase 13 CP4): the owner's page (app shell) and the
// read-only shared view (public shell). The plate is the same — a stranger
// sees exactly the legs, kinds, and photos the owner sees — and the two
// differ only in what surrounds it, which arrives as props: the owner's
// islands (photo upload, share controls) as slots, and `shared` carrying
// the cover facts the shared read provides instead of a candidate.
//
// Phase 16 CP1 (BRIEF §4, revised at its STOP): the page is the account
// of the trip, and the account of how the route was drawn is one tap away
// at the foot. The cover leads with the trip block; each day is a heading,
// a line and its stays; the reconstruction figures sit in a <details>
// after everything else; the floor — legend, provenance bar, divergence
// flag, truncation marks — stays visible. On a phone the map plate comes
// right after the cover, before the narrative: the thing a person came
// for, before the words about it. On a wide screen the plate keeps its
// sticky right column.
import { useMemo, useState } from "react";

import { FixDot, LegKindLegend, StopDot } from "@/components/legend";
import { ProvenanceBar } from "@/components/provenance-bar";
import { Term } from "@/components/term";
import { fmtHours, fmtMode, roman } from "@/lib/format";
import { fmtDateRange, isFixLeg, sliceDays } from "@/lib/slice-days";
import { DayNarrative } from "./day-narrative";
import { ImportPhotosStrip, PhotoStrip } from "./photo-strip";
import { PlateExport } from "./plate-export";
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
  // One name for the cover and the exported image (phase 14 CP3): the
  // decision's, or the date fallback the confirm-all sweep also uses.
  const name = adventureName(journey, candidate, shared);
  const truncation = shared ?? candidate;

  // Three grid children in reading order — cover, plate, the rest — so a
  // phone stacks them as DOM order says. From lg: up the plate takes the
  // right column across both rows and sticks; the cover and the rest share
  // the left column. items-start keeps the sticky aside's area taller than
  // the aside itself, which is what lets it stick.
  return (
    <article className="mt-6 lg:grid lg:grid-cols-[minmax(24rem,32rem)_minmax(0,1fr)] lg:grid-rows-[auto_1fr] lg:items-start lg:gap-x-10">
      <Cover
        journey={journey}
        candidate={candidate}
        plate={plate}
        shared={shared}
        name={name}
      />

      {journey.legs.length > 0 && (
        <aside className="mt-6 lg:sticky lg:top-6 lg:col-start-2 lg:row-start-1 lg:row-span-2 lg:mt-0">
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
          <div className="border border-t-0 border-ink bg-paper px-4 py-2.5 [outline:1px_solid_var(--color-ink)] [outline-offset:3px]">
            <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
              <span className="font-display text-[11.5px] font-semibold tracking-[0.22em]">
                {selected !== null ? (
                  `DAY ${selected} HIGHLIGHTED`
                ) : plate !== null ? (
                  <>
                    <Term id="plate">PLATE {roman(plate)}</Term> · FULL ROUTE
                  </>
                ) : (
                  "FULL ROUTE"
                )}
              </span>
              <span className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink">
                {journey.legs.some(isFixLeg) && (
                  <span className="inline-flex items-center gap-2">
                    <FixDot />
                    <span className="font-semibold tracking-wide">Fix</span>
                    <span className="text-ink-2">one recorded position</span>
                  </span>
                )}
                {journey.stops.some((s) => !(s.loc.lat === 0 && s.loc.lon === 0)) && (
                  <span className="inline-flex items-center gap-2">
                    <StopDot />
                    <span className="font-semibold tracking-wide">Stop</span>
                  </span>
                )}
              </span>
            </div>
            <div className="mt-2.5">
              <LegKindLegend />
            </div>
            <div className="mt-2.5">
              {/* The image export (phase 14 CP3) — a second margin row, on
                  the owner's page and the shared view alike. It renders what
                  this plate shows: a highlighted day exports highlighted. */}
              <PlateExport
                journey={journey}
                days={days}
                selectedDay={selected}
                styleUrl={styleUrl}
                name={name}
                plate={plate}
                startTruncated={truncation?.start_truncated ?? false}
                endTruncated={truncation?.end_truncated ?? false}
              />
            </div>
          </div>
        </aside>
      )}

      <div className="lg:col-start-1 lg:row-start-2">
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
        <DrawingDisclosure journey={journey} />
      </div>
    </article>
  );
}

// The cover (DESIGN §4, reorganised in phase 16 CP1): eyebrow, name, the
// dateline with the places passed through; then the trip block — days and
// the headline kilometres over their provenance bar, time away, time on
// the move, stops, distance from home. The account of the drawing (kind
// kilometres, point counts, Google's figure, modes, the divergence check)
// is one <details> at the foot of the page (DrawingDisclosure), closed by
// default, with a flagged divergence's one line above it.
// No score: ranking is triage machinery, and it stays in the candidates
// gallery and the confirm cell.
function Cover({
  journey,
  candidate,
  plate,
  shared,
  name,
}: {
  journey: Journey;
  candidate?: Candidate;
  plate: number | null;
  shared?: SharedCover;
  name: string;
}) {
  // The day count is the SERVED figure (phase 14 CP2): the same rule the
  // narrative's sliceDays applies, computed in Go and printed by the CLI.
  // A vitest asserts the two agree on real journeys, so the cover can
  // never say three days over a narrative of four.
  const dayCount = journey.summary.civil_days;
  const decision = candidate?.decision;
  const confirmed = decision?.action === "confirmed";
  const truncation = shared ?? candidate;

  return (
    <header className="lg:col-start-1 lg:row-start-1">
      {/* The shared eyebrow names what a stranger is looking at; the plate
          number is the owner's atlas register and stays theirs. */}
      <p className="text-[11.5px] tracking-[0.24em] text-ink-2">
        {shared ? (
          "SHARED ADVENTURE"
        ) : confirmed ? (
          <>
            CONFIRMED ADVENTURE
            {plate !== null && (
              <>
                {" · "}
                <Term id="plate">PLATE {roman(plate)}</Term>
              </>
            )}
          </>
        ) : decision?.action === "dismissed" ? (
          "DISMISSED CANDIDATE"
        ) : (
          <Term id="candidate">CANDIDATE</Term>
        )}
      </p>
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
        )}
      </p>
      {(truncation?.start_truncated || truncation?.end_truncated) && (
        // Truncation as words (bug 4 made visible). The amber stays on the
        // flag glyph: the token passes contrast as a mark but not as
        // sentence text (the phase 6 CP4 a11y pass), so the words are ink.
        <p className="mt-2 max-w-[52ch] text-[13px]">
          <span className="font-bold text-flag" aria-hidden>
            ⚑
          </span>{" "}
          {truncation.start_truncated &&
            "Already under way when the record begins. "}
          {truncation.end_truncated &&
            "Still under way when the record ends."}
        </p>
      )}

      <TripBlock
        journey={journey}
        dayCount={dayCount}
        candidate={shared ? undefined : candidate}
      />

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
            <>Undecided — confirm or dismiss in the candidates gallery</>
          )}
        </p>
      )}
    </header>
  );
}

// The trip block (phase 14's summary, made the hero in phase 16 CP1): what
// the trip was, in the traveller's words. The headline kilometres carry
// their provenance bar and its caption — the one place the bar is taught
// (a Term) — then time away, time on the move, stops, distance from home.
// Every figure is served by the API and printed identically by
// `roadbook journey -candidate N`. "On the move" is the source's own
// transit time — the sum of its activity durations, the same claim as the
// mode line in the disclosure — chosen at the phase 14 review over
// span-minus-dwell; the average speed beside it is measured, over recorded
// driving only. The one home-relative figure appears for the owner only:
// the shared view never names home (phase 14 BRIEF §3E), so `candidate`
// is withheld there. Labels quiet sans, figures mono (DESIGN §6).
function TripBlock({
  journey,
  dayCount,
  candidate,
}: {
  journey: Journey;
  dayCount: number;
  candidate?: Candidate;
}) {
  const s = journey.summary;
  const bd = journey.mode_breakdown;
  const transitHours =
    bd === undefined ? undefined : bd.reduce((acc, m) => acc + m.hours, 0);
  return (
    <section
      aria-label="The trip"
      className="mt-5 border-t border-t-ink py-4"
    >
      <p className="font-display text-4xl font-semibold leading-none sm:text-5xl">
        {journey.total_km.toFixed(0)} km
        <span className="ml-2.5 font-sans text-base font-normal text-ink-2">
          {dayCount === 1 ? "in one day" : `over ${dayCount} days`}
        </span>
      </p>
      <ProvenanceBar
        observed={journey.observed_km}
        routed={journey.routed_km}
        unknown={journey.unknown_km}
        air={journey.air_km}
        className="mb-2 mt-3.5"
      />
      <p className="text-xs text-ink-2">
        <Term id="provenance">How much of this was recorded, how much drawn in</Term>
      </p>

      <dl className="mt-5 grid grid-cols-[auto_minmax(0,1fr)] gap-x-5 gap-y-2 text-[13px] leading-snug">
        <dt className="text-ink-2">Time away</dt>
        <dd>
          <span className="font-mono">{fmtHours(s.span_hours)}</span>
        </dd>
        <dt className="text-ink-2">On the move</dt>
        <dd>
          {transitHours === undefined ? (
            <span className="text-ink-2">no record of it</span>
          ) : (
            <>
              <span className="font-mono">{fmtHours(transitHours)}</span>
              <span className="text-ink-2"> by Google&apos;s count</span>
              {s.observed_pace_kmh !== undefined && (
                <>
                  {" · "}
                  <span className="font-mono">
                    {s.observed_pace_kmh.toFixed(0)} km/h
                  </span>
                  <span className="text-ink-2"> average where recorded</span>
                </>
              )}
            </>
          )}
        </dd>
        <dt className="text-ink-2">Stopped</dt>
        <dd>
          {s.stops === 0 ? (
            <span className="text-ink-2">no stops recorded</span>
          ) : (
            <>
              <span className="font-mono">{s.stops}</span>
              {s.stops === 1 ? " stop · " : " stops · "}
              <span className="font-mono">{fmtHours(s.dwell_hours)}</span>
            </>
          )}
        </dd>
        {candidate && (
          <>
            <dt className="text-ink-2">Farthest</dt>
            <dd>
              <span className="font-mono">{candidate.dest_km} km</span>
              {" from home"}
            </dd>
          </>
        )}
      </dl>
    </section>
  );
}

// The disclosure (BRIEF §5B), at the FOOT of the page since the CP1 STOP:
// the last thing on a page about the trip, reachable, closed by default.
// The cover's flagged distance check still points here ("below").
function DrawingDisclosure({ journey }: { journey: Journey }) {
  return (
    <>
      <DivergenceFlag journey={journey} />
      <details className="mt-3 border-t border-b border-rule text-xs [&[open]>summary>span]:rotate-90">
        {/* py-3 on a 20px line = 44px: a thumb-sized disclosure control. */}
        <summary className="cursor-pointer list-none py-3 text-sm text-ink-2 marker:content-none [&::-webkit-details-marker]:hidden">
          <span className="mr-2 inline-block w-3 text-center transition-transform" aria-hidden>
            ▸
          </span>
          How this route was drawn
        </summary>
        <DrawingAccount journey={journey} />
      </details>
    </>
  );
}

// The account of the drawing, behind the disclosure (BRIEF §5B): the kind
// kilometres, the point counts, Google's own figure, the modes as the
// source labelled them, and the divergence check in full. These are the
// figures `roadbook journey -candidate N` prints (invariant 13) — kept,
// reachable, out of the way.
function DrawingAccount({ journey }: { journey: Journey }) {
  return (
    <div className="space-y-2 pb-4 text-ink-2">
      <p className="font-mono">
        observed {journey.observed_km.toFixed(1)} km · routed{" "}
        {journey.routed_km.toFixed(1)} km · unknown{" "}
        {journey.unknown_km.toFixed(1)} km · air {journey.air_km.toFixed(1)}{" "}
        km
      </p>
      <p>
        {journey.merged_points} recorded points ({journey.trace_points_kept}{" "}
        from the timeline + {journey.raw_points_kept} raw)
        {journey.google_km > 0 && (
          <>
            {" · Google's own distance "}
            <span className="font-mono">{journey.google_km.toFixed(1)} km</span>
          </>
        )}
        {journey.countries.length > 0 && (
          <>
            {" · "}
            {journey.states.length > 0
              ? "countries and regions from the route's recorded points"
              : "countries from the route's recorded points"}
          </>
        )}
      </p>
      <ModeLine journey={journey} />
      <DivergenceFigures journey={journey} />
    </div>
  );
}

// The source-asserted mode line (phase 11 §6.2): Google's own labels and
// distances, never summed with the measured figures — the source's guesses
// have recorded failures (a 1,023 km "motorcycling" relocation;
// probability-0.00 modes), and the pipeline itself trusts speed over mode
// for flight classification. Absent breakdown (a photo-sourced journey has
// no activities) states the absence, never zeros. Reproduction:
// `roadbook journey -candidate N` prints the same line.
function ModeLine({ journey }: { journey: Journey }) {
  const bd = journey.mode_breakdown;
  if (bd === undefined) {
    return <p>No mode record — this journey&apos;s evidence carries no activity data.</p>;
  }
  if (bd.length === 0) return null;
  // Per-mode hours (phase 14 CP2) ride beside the km — the same claim, the
  // source's own; their sum is the trip block's "on the move" row.
  return (
    <p>
      By mode, as Google labelled it (modes are guesses):{" "}
      {bd.map((m, i) => (
        <span key={m.mode}>
          {i > 0 && " · "}
          {fmtMode(m.mode)}{" "}
          <span className="font-mono">
            {m.km.toFixed(1)} km, {fmtHours(m.hours)}
          </span>
        </span>
      ))}
    </p>
  );
}

// The phase 3 validation check, in full, inside the disclosure: the
// road-comparable reconstruction against Google's own ground figure — air
// excluded from both sides. Divergence is a conversation starter, never a
// gate: the unknown and routed legs explaining it are right there on the
// map. Spacing rides in {" "} expressions throughout: SWC drops inter-tag
// whitespace in some constructs (the phase 7 trap — this very line
// rendered "62.7 km· Google's" until phase 9's screenshot review).
function DivergenceFigures({ journey }: { journey: Journey }) {
  if (journey.divergence_pct === undefined) return null;
  return (
    <p>
      Distance check: drawn on the ground{" "}
      <span className="font-mono">{journey.ground_km.toFixed(1)} km</span>
      {" · Google's ground figure "}
      <span className="font-mono">{journey.google_ground_km.toFixed(1)} km</span>
      {" ("}
      {journey.divergence_pct >= 0 ? "+" : ""}
      {journey.divergence_pct.toFixed(1)}
      {"%)"}
      {journey.divergence_flagged &&
        ". Gaps with no road under-count, detours over-count, and a record cut short compares against a door-to-door figure — the legs explaining it are on the map."}
    </p>
  );
}

// The floor (BRIEF §1): a flagged divergence keeps its flag and one line
// outside the disclosure, visible with it closed — at the foot of the
// page since the review (maintainer: not a cover fact), directly above
// the account that explains it.
function DivergenceFlag({ journey }: { journey: Journey }) {
  if (journey.divergence_pct === undefined || !journey.divergence_flagged) return null;
  const pct = Math.abs(Math.round(journey.divergence_pct));
  return (
    <p className="mt-4 flex gap-2.5 border-l-[3px] border-flag bg-land px-3 py-2.5 text-[13px]">
      <span className="font-bold text-flag" aria-hidden>
        ⚑
      </span>
      <span>
        The drawn distance and Google&apos;s differ by{" "}
        <span className="font-mono">{pct}%</span> — the account below says
        why.
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

/** The cover's name: the decision's own, else the date fallback — the same
 * string the confirm-all sweep stores, so an unnamed adventure reads the
 * same everywhere. The shared view carries its name ready-made. */
function adventureName(
  journey: Journey,
  candidate: Candidate | undefined,
  shared: SharedCover | undefined,
): string {
  if (shared) return shared.name;
  const decision = candidate?.decision;
  return decision?.action === "confirmed" && decision.name
    ? decision.name
    : `Journey of ${journey.window_start.slice(0, 10)}`;
}
