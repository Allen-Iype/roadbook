// The day-sliced narrative (phase 6 BRIEF §1, DESIGN §3): each civil day of
// the journey as a section — "Day 3 · Sunday 24 May · 80 km" — with what
// the person did that day. Rendered from sliceDays output; the same output
// stamps the map's day assignments, so heading, line, and highlight can
// never disagree.
//
// Phase 16 CP1, revised at the STOP (plan 2026-10-29): a day is a heading
// with its kilometres over a provenance bar, ONE line in the traveller's
// words, and a row per stay. The per-event rows (every transit and
// recorded position) left the page — they were the reconstruction's
// account, and `roadbook journey` prints every leg for anyone who wants
// it.
//
// Phase 18 "places" filled the two slots that revision reserved: the day
// line opens with the regions the day was in ("Suðurland to Austurland ·
// …") and a stay row ends with its nearest listed town ("· near Höfn").
// The words come from lib/day-places.ts; the lookups are the API's.
//
// Phase 19 "nights": the stay that held a night says so ("Stayed the
// night"), a night of recorded travel is said on the day line, and so is a
// night the record cannot place. Which night is which is the API's
// (`Journey.nights`); lib/nights.ts picks the row.
//
// No "use client" directive: this module renders inside the adventure view,
// which is already a client component — the directive marks entry points
// into the client graph, not every file in it.
import { StopDot } from "@/components/legend";
import { ProvenanceBar } from "@/components/provenance-bar";
import { Term } from "@/components/term";
import { dayRegions, regionsPhrase, stayPhrases } from "@/lib/day-places";
import { nightOf, stayNight } from "@/lib/nights";
import {
  clockOf,
  dayComposition,
  dayLabel,
  dayPaceKmh,
  type Day,
  type DayEvent,
} from "@/lib/slice-days";
import { fmtDuration } from "@/lib/format";
import type { DisplayPhoto } from "@/lib/photo-display";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];
type Candidate = components["schemas"]["Candidate"];
// The narrative needs exactly the truncation flags of the candidate — so
// the shared view (CP4), which carries no candidate at all, can pass just
// those (they ride the SharedAdventure schema).
export type Truncation = Pick<Candidate, "start_truncated" | "end_truncated">;

export function DayNarrative({
  days,
  journey,
  candidate,
  photos,
  selected,
  onSelect,
}: {
  days: Day[];
  journey: Journey;
  candidate?: Truncation;
  photos: DisplayPhoto[];
  selected: number | null;
  onSelect: (dayIndex: number) => void;
}) {
  const byLeg = new Map<number, DisplayPhoto[]>();
  const byStop = new Map<number, DisplayPhoto[]>();
  for (const p of photos) {
    if (p.leg_index !== undefined)
      byLeg.set(p.leg_index, [...(byLeg.get(p.leg_index) ?? []), p]);
    else if (p.stop_index !== undefined)
      byStop.set(p.stop_index, [...(byStop.get(p.stop_index) ?? []), p]);
  }

  return (
    <section aria-label="Days">
      {days.map((day) => (
        <DaySection
          key={day.date}
          day={day}
          journey={journey}
          candidate={candidate}
          byLeg={byLeg}
          byStop={byStop}
          active={selected === day.index}
          onSelect={() => onSelect(day.index)}
        />
      ))}
    </section>
  );
}

function DaySection({
  day,
  journey,
  candidate,
  byLeg,
  byStop,
  active,
  onSelect,
}: {
  day: Day;
  journey: Journey;
  candidate?: Truncation;
  byLeg: Map<number, DisplayPhoto[]>;
  byStop: Map<number, DisplayPhoto[]>;
  active: boolean;
  onSelect: () => void;
}) {
  const composition = dayComposition(day, journey);
  // Photos placed on the day's legs sit under the day line: the transit
  // rows that used to hold them are gone; the day is their slot now.
  const legPhotos = day.legIndices.flatMap((i) => byLeg.get(i) ?? []);
  const stays = day.events.filter(isStay);
  // One phrase per stay row; a run of stays near the same town says so once.
  const places = stayPhrases(
    stays.map((e) => e.stopIndex),
    journey,
  );
  return (
    <section
      className={active ? "border-t-2 border-ink" : "border-t border-rule"}
      aria-current={active ? "true" : undefined}
    >
      <h2>
        {/* The heading is the day's map toggle: pressing it highlights the
            day's geometry and dims the rest; pressing again shows the whole
            route. A real button — keyboard and AT get the same control. */}
        <button
          type="button"
          onClick={onSelect}
          aria-pressed={active}
          title={
            active
              ? "Show the whole route again"
              : "Highlight this day on the map"
          }
          className="flex w-full items-baseline gap-4 py-3 text-left"
        >
          <span
            className={`font-display text-[13px] font-semibold tracking-[0.22em] ${
              active ? "border-b-2 border-ink pb-0.5" : ""
            }`}
          >
            DAY {day.index}
          </span>
          <span className="text-sm text-ink-2">{day.label}</span>
          <span className="ml-auto font-mono text-[13px]">
            {day.km > 0 ? `${day.km.toFixed(day.km < 10 ? 1 : 0)} km` : "—"}
          </span>
        </button>
      </h2>
      {/* The day's kilometres carry their provenance like every headline
          distance (RESEARCH §5 principle 2); a day with none shows none. */}
      {day.km > 0 && (
        <ProvenanceBar
          observed={composition.observed}
          routed={composition.routed}
          unknown={composition.unknown}
          air={composition.air}
          className="-mt-1 mb-2.5"
        />
      )}
      <p data-day-line className="pb-2 text-sm">
        <DayLine day={day} journey={journey} candidate={candidate} />
        {legPhotos.length > 0 && (
          <>
            {" "}
            <PhotoThumbs photos={legPhotos} />
          </>
        )}
      </p>
      {stays.length > 0 && (
        <ol className="pb-2">
          {stays.map((e, i) => (
            <StayRow
              key={i}
              event={e}
              day={day}
              journey={journey}
              byStop={byStop}
              place={places[i]}
            />
          ))}
        </ol>
      )}
    </section>
  );
}

type Stay = Extract<DayEvent, { type: "dwell" | "dwellBegin" | "dwellDay" | "dwellEnd" }>;

function isStay(e: DayEvent): e is Stay {
  return (
    e.type === "dwell" ||
    e.type === "dwellBegin" ||
    e.type === "dwellDay" ||
    e.type === "dwellEnd"
  );
}

const muted = (text: string) => <span className="text-ink-2"> · {text}</span>;
const flag = (
  <span className="font-bold text-flag" aria-hidden>
    {" "}
    ⚑
  </span>
);

// The one line: what the day was, from its events and nothing more. The
// stays get their own rows below, so the line counts them and says how
// the day began and ended — the record's edges, an overnight transit, a
// day spent travelling or staying put. An empty day says so.
function DayLine({
  day,
  journey,
  candidate,
}: {
  day: Day;
  journey: Journey;
  candidate?: Truncation;
}) {
  if (day.events.length === 0) {
    return <span className="italic text-ink-2">Nothing recorded this day.</span>;
  }
  const parts: React.ReactNode[] = [];
  // Where the day was, first: the regions of its legs and stays, from the
  // recorded points only. A day with none (off a coastline, or nothing
  // recorded while moving) simply starts with the next part.
  const regions = regionsPhrase(dayRegions(day, journey));
  if (regions !== undefined) {
    parts.push(<span data-day-regions>{regions}</span>);
  }
  const start = day.events.find((e) => e.type === "windowEdge" && e.edge === "start");
  const end = day.events.find((e) => e.type === "windowEdge" && e.edge === "end");
  if (start) {
    parts.push(
      candidate?.start_truncated ? (
        <>
          Trip begins{flag}
          {muted("already under way when your file starts")}
        </>
      ) : (
        <>Trip begins</>
      ),
    );
  }
  const stops = new Set(
    day.events.flatMap((e) => ("stopIndex" in e ? [e.stopIndex] : [])),
  ).size;
  const dwellDay = day.events.find((e) => e.type === "dwellDay");
  const transitDay = day.events.find((e) => e.type === "transitDay");
  const overnight = day.events
    .filter((e) => e.type === "transit" && e.overnight !== null)
    .map((e) => (e.type === "transit" ? e.overnight : null))
    .at(-1);
  if (dwellDay) {
    // The whole day inside one stay: which hours the record cannot
    // describe, in the traveller's own clock (the honest Westfjords Day 2).
    const s = journey.stops[dwellDay.stopIndex];
    // The row below says "Stayed all day"; the line says what that means.
    parts.push(
      <>
        Nothing recorded from {dayLabel(s.start.slice(0, 10))}{" "}
        {clockOf(s.start)} to {dayLabel(s.end.slice(0, 10))} {clockOf(s.end)}
      </>,
    );
  } else if (transitDay) {
    parts.push(<>Travelling all day</>);
  } else if (stops > 0) {
    parts.push(<>{stops === 1 ? "1 stop" : `${stops} stops`}</>);
  } else if (day.km > 0) {
    parts.push(<>On the move</>);
  }
  // The day's average pace over its recorded stretches (the cover's rule,
  // per day) — absent when nothing moving was recorded that day.
  const pace = dayPaceKmh(day, journey);
  if (pace !== undefined) {
    parts.push(
      <>
        <span className="font-mono">{pace.toFixed(0)} km/h</span>
        <span className="text-ink-2"> average where recorded</span>
      </>,
    );
  }
  // The night this day ran into (phase 19), said once, where it belongs:
  // recorded travel through it and a night the record cannot place are
  // said here; a night at a stop is said on the stay's own row. The older
  // "overnight into Day N" only knew that some stretch crossed midnight —
  // usually the gap around an overnight stay — so it remains only for a
  // day with no night of its own to describe (a stretch that runs past
  // the trip's last civil date).
  const night = nightOf(journey, day.date);
  if (night?.kind === "moving") {
    parts.push(<span data-night="moving">travelling through the night</span>);
  } else if (night?.kind === "unknown") {
    parts.push(
      <span className="text-ink-2" data-night="unknown">
        night not recorded
      </span>,
    );
  } else if (night === undefined && overnight !== undefined && overnight !== null) {
    parts.push(<>overnight into Day {overnight}</>);
  }
  if (end) {
    parts.push(
      candidate?.end_truncated ? (
        <>
          Trip ends{flag}
          {muted("still under way when your file stops")}
        </>
      ) : (
        <>Trip ends</>
      ),
    );
  }
  return (
    <>
      {parts.map((p, i) => (
        <span key={i}>
          {i > 0 && " · "}
          {p}
        </span>
      ))}
    </>
  );
}

// A stay: the map's stop marker, a time, and one line — "Stayed 38 h ·
// near Höfn". The place is the nearest listed town to the stay's position
// (never "in"). Where the phone recorded no position for the stay, the
// position is the one in Google's record of the visit; the name is then a
// Term, whose dotted underline marks it as resting on Google's word and
// whose note says so. The marker stays hollow: the map still draws no
// such stop.
// aria-hidden on the glyph span must be undefined, not false, when the
// label applies: React renders aria-hidden="false", which is honest but
// noisy; absent is the plain form.
function StayRow({
  event: e,
  day,
  journey,
  byStop,
  place,
}: {
  event: Stay;
  day: Day;
  journey: Journey;
  byStop: Map<number, DisplayPhoto[]>;
  /** "near Höfn" / "16 km from Höfn"; undefined when the stay has no name
   * or the row above already said it. */
  place?: string;
}) {
  // A stay's photo appears on the day it was taken — the one stay event
  // instance (begin, intermediate day, end) whose civil date matches.
  const photos = (byStop.get(e.stopIndex) ?? []).filter(
    (p) => p.taken_at && p.taken_at.slice(0, 10) === day.date,
  );
  const s = journey.stops[e.stopIndex];
  const unplaced = s.loc.lat === 0 && s.loc.lon === 0;
  // Whether this row is where one of the trip's nights is said: the stay
  // from the evening on ("tonight"), or a stay that began after midnight
  // and held the night before ("lastNight").
  const held = stayNight(journey, e.stopIndex, day.date);
  let time: string;
  let words: React.ReactNode;
  switch (e.type) {
    case "dwell":
      time = `${clockOf(e.start)}–${clockOf(e.end)}`;
      words = (
        <>
          Stayed <span className="font-mono">{fmtDuration(e.minutes)}</span>
          {held === "lastNight" && muted("overnight")}
        </>
      );
      break;
    case "dwellBegin":
      time = clockOf(e.at);
      // "the night" already says the stay runs into the next day; the
      // day it ends is added only when that is later still.
      words =
        held === "tonight" ? (
          <>
            Stayed the night
            {e.endsDay > day.index + 1 && muted(`through to Day ${e.endsDay}`)}
          </>
        ) : (
          <>
            Stayed{muted(`through to Day ${e.endsDay}`)}
            {held === "lastNight" && muted("overnight")}
          </>
        );
      break;
    case "dwellDay":
      time = "all day";
      words = held === "tonight" ? <>Stayed all day and the night</> : <>Stayed all day</>;
      break;
    case "dwellEnd":
      time = clockOf(e.at);
      words = (
        <>
          Moved on after{" "}
          <span className="font-mono">{fmtDuration(e.minutes)}</span>
        </>
      );
      break;
  }
  return (
    <li
      data-night-held={held}
      className="grid grid-cols-[2.25rem_5.5rem_minmax(0,1fr)] items-baseline gap-x-2.5 py-1.5 text-sm"
    >
      {/* A stay with no recorded position: hollow marker, the words on
          hover and for assistive tech — not in the row (maintainer's
          review: not a first-glance fact). The map draws no such stop. */}
      <span
        className="flex self-center"
        aria-hidden={unplaced ? undefined : true}
        aria-label={unplaced ? "position not recorded" : undefined}
        title={unplaced ? "Position not recorded — this stay is not on the map" : undefined}
      >
        <StopDot unplaced={unplaced} />
      </span>
      <span className="font-mono text-xs text-ink-2">{time}</span>
      <span>
        {words}
        {place !== undefined && (
          // inline-block: the dot and the phrase wrap together or not at
          // all, so a narrow row never ends on a dangling dot or splits
          // "near" from its town; max-w-full lets a very long name still
          // wrap inside the block instead of pushing the page sideways.
          // The space sits outside the block: leading white space inside
          // an inline-block is collapsed away.
          <>
            {" "}
            <span className="inline-block max-w-full text-ink-2" data-stay-place>
              {"· "}
              {s.named_from === "visit" ? (
                <Term id="visitPlace">{place}</Term>
              ) : (
                place
              )}
            </span>
          </>
        )}
        {photos.length > 0 && (
          <>
            {" "}
            <PhotoThumbs photos={photos} />
          </>
        )}
      </span>
    </li>
  );
}

// Photos slot into the narrative by placement (phase 4 BRIEF §3G): each
// placed photo appears inline on the day whose legs held its instant, or
// on the stay — thumbnails where the journey says they happened.
function PhotoThumbs({ photos }: { photos: DisplayPhoto[] }) {
  return (
    <span className="inline-flex gap-1 align-middle">
      {photos
        .filter((p) => p.thumb_url !== null)
        .map((p) => (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            key={p.key}
            src={p.thumb_url!}
            alt={p.original_name}
            title={p.original_name}
            className={`h-6 w-6 rounded object-cover ${
              p.far_flagged ? "ring-1 ring-flag" : ""
            }`}
          />
        ))}
    </span>
  );
}
