// The day-sliced narrative (BRIEF §1, DESIGN §3): each civil day of the
// journey as a section — "Day 3 · Sunday 24 May · 80 km" — with its fixes,
// transits, and dwells as a timeline, and photos slotted where their
// placements put them. Rendered from sliceDays output; the same output
// stamps the map's day assignments, so heading, events, and highlight can
// never disagree.
//
// No "use client" directive: this module renders inside the adventure view,
// which is already a client component — the directive marks entry points
// into the client graph, not every file in it.
import { clockOf, dayLabel, type Day, type DayEvent } from "@/lib/slice-days";
import { fmtDuration, fmtLatLon } from "@/lib/format";
import type { components } from "@/lib/api/schema";

type Journey = components["schemas"]["Journey"];
type Candidate = components["schemas"]["Candidate"];
type Photo = components["schemas"]["Photo"];

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
  candidate?: Candidate;
  photos: Photo[];
  selected: number | null;
  onSelect: (dayIndex: number) => void;
}) {
  const byLeg = new Map<number, Photo[]>();
  const byStop = new Map<number, Photo[]>();
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
  candidate?: Candidate;
  byLeg: Map<number, Photo[]>;
  byStop: Map<number, Photo[]>;
  active: boolean;
  onSelect: () => void;
}) {
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
      {day.events.length === 0 ? (
        <p className="pb-3 text-sm italic text-ink-2">
          Nothing recorded this day.
        </p>
      ) : (
        <ol className="pb-2">
          {day.events.map((e, i) => (
            <EventRow
              key={i}
              event={e}
              day={day}
              journey={journey}
              candidate={candidate}
              byLeg={byLeg}
              byStop={byStop}
            />
          ))}
        </ol>
      )}
      <DwellDayNote day={day} journey={journey} />
    </section>
  );
}

// The honest note under a day whose whole content is an ongoing dwell (the
// Westfjords Day 2): where the traveller was dwelling — or that no position
// was observed — and exactly which hours the record cannot describe.
function DwellDayNote({ day, journey }: { day: Day; journey: Journey }) {
  const dwellDay = day.events.find((e) => e.type === "dwellDay");
  if (!dwellDay) return null;
  const s = journey.stops[dwellDay.stopIndex];
  const minutes = Math.round((Date.parse(s.end) - Date.parse(s.start)) / 60_000);
  const where =
    s.loc.lat === 0 && s.loc.lon === 0
      ? "with no observed position"
      : `near ${fmtLatLon(s.loc.lat, s.loc.lon)}`;
  return (
    <p className="max-w-[46ch] pb-4 text-[13.5px] italic text-ink-2">
      Dwelling {where}. No fixes between {dayLabel(s.start.slice(0, 10))}{" "}
      {clockOf(s.start)} and {dayLabel(s.end.slice(0, 10))} {clockOf(s.end)} —{" "}
      {fmtDuration(minutes)} the record cannot describe, so this page does not
      try.
    </p>
  );
}

// Chip styles per event class. Spelled out: Tailwind extracts class names
// statically, so `text-${kind}` would silently generate nothing.
const CHIP: Record<string, { label: string; cls: string }> = {
  fix: { label: "fix", cls: "text-observed border-observed" },
  observed: { label: "observed", cls: "text-observed border-observed" },
  road: { label: "routed", cls: "text-routed border-routed" },
  unknown: { label: "unknown", cls: "text-unknown border-unknown" },
  air: { label: "air", cls: "text-air border-air" },
  dwell: { label: "dwell", cls: "text-ink-2 border-rule" },
  window: { label: "window", cls: "text-ink-2 border-rule" },
};

function EventRow({
  event,
  day,
  journey,
  candidate,
  byLeg,
  byStop,
}: {
  event: DayEvent;
  day: Day;
  journey: Journey;
  candidate?: Candidate;
  byLeg: Map<number, Photo[]>;
  byStop: Map<number, Photo[]>;
}) {
  const { time, chip, body, photos } = eventBits(
    event,
    day,
    journey,
    candidate,
    byLeg,
    byStop,
  );
  const c = CHIP[chip];
  return (
    <li className="flex items-baseline gap-2.5 py-1 text-sm">
      <span className="min-w-[86px] shrink-0 font-mono text-xs text-ink-2">
        {time}
      </span>
      <span
        className={`shrink-0 border px-1.5 py-px text-[10px] uppercase tracking-[0.1em] ${c.cls}`}
      >
        {c.label}
      </span>
      <span>
        {body}
        {photos && photos.length > 0 && (
          <>
            {" "}
            <PhotoThumbs photos={photos} />
          </>
        )}
      </span>
    </li>
  );
}

function eventBits(
  e: DayEvent,
  day: Day,
  journey: Journey,
  candidate: Candidate | undefined,
  byLeg: Map<number, Photo[]>,
  byStop: Map<number, Photo[]>,
): { time: string; chip: string; body: React.ReactNode; photos?: Photo[] } {
  // A dwell photo appears on the day it was taken — the one dwell event
  // instance (begin, intermediate day, end) whose civil date matches.
  const stopPhotos = (stopIndex: number) =>
    (byStop.get(stopIndex) ?? []).filter(
      (p) => p.taken_at && p.taken_at.slice(0, 10) === day.date,
    );
  const muted = (text: string) => (
    <span className="text-ink-2"> {text}</span>
  );

  switch (e.type) {
    case "fix":
      return {
        time:
          clockOf(e.start) === clockOf(e.end)
            ? clockOf(e.start)
            : `${clockOf(e.start)}–${clockOf(e.end)}`,
        chip: "fix",
        body: (
          <>
            Fix — <span className="font-mono">{fmtLatLon(e.lat, e.lon)}</span>
            {e.points > 1 && muted(`(${e.points} fixes)`)}
          </>
        ),
        photos: byLeg.get(e.legIndex),
      };
    case "transit": {
      const km = e.km.toFixed(1);
      const overnight =
        e.overnight !== null ? muted(`· overnight, ends Day ${e.overnight}`) : null;
      const bodyByKind: Record<typeof e.kind, React.ReactNode> = {
        observed: (
          <>
            Observed transit — <b className="font-mono">{km} km</b> ·{" "}
            {e.points} fixes{overnight}
          </>
        ),
        road: (
          <>
            Routed transit — <b className="font-mono">{km} km</b> along roads
            {muted(`(straight line ${e.chordKm.toFixed(1)} km)`)}
            {overnight}
          </>
        ),
        unknown: (
          <>
            Unobserved gap — straight line{" "}
            <b className="font-mono">{km} km</b>, nothing inferred{overnight}
          </>
        ),
        air: (
          <>
            Flight — <b className="font-mono">{km} km</b> great-circle arc
            {overnight}
          </>
        ),
      };
      return {
        time: `${clockOf(e.start)}–${clockOf(e.end)}`,
        chip: e.kind,
        body: bodyByKind[e.kind],
        photos: byLeg.get(e.legIndex),
      };
    }
    case "dwell": {
      const s = journey.stops[e.stopIndex];
      const where =
        s.loc.lat === 0 && s.loc.lon === 0
          ? muted("· position not observed")
          : muted(`· near ${fmtLatLon(s.loc.lat, s.loc.lon)}`);
      return {
        time: `${clockOf(e.start)}–${clockOf(e.end)}`,
        chip: "dwell",
        body: (
          <>
            Dwell — {fmtDuration(e.minutes)}
            {where}
          </>
        ),
        photos: stopPhotos(e.stopIndex),
      };
    }
    case "dwellBegin":
      return {
        time: clockOf(e.at),
        chip: "dwell",
        body: <>Dwell begins{muted(`· ends Day ${e.endsDay}`)}</>,
        photos: stopPhotos(e.stopIndex),
      };
    case "dwellDay":
      return {
        time: "all day",
        chip: "dwell",
        body: <>Dwelling — no movement observed</>,
        photos: stopPhotos(e.stopIndex),
      };
    case "dwellEnd":
      return {
        time: clockOf(e.at),
        chip: "dwell",
        body: <>Dwell ends ({fmtDuration(e.minutes)})</>,
        photos: stopPhotos(e.stopIndex),
      };
    case "transitDay": {
      const wording: Record<typeof e.kind, string> = {
        observed: "observed",
        road: "inferred along roads",
        unknown: "unobserved",
        air: "in the air",
      };
      return {
        time: "all day",
        chip: e.kind,
        body: <>Transit under way all day — {wording[e.kind]}</>,
      };
    }
    case "windowEdge":
      return {
        time: clockOf(e.at),
        chip: "window",
        body:
          e.edge === "start" ? (
            candidate?.start_truncated ? (
              <>
                Window opens — the journey was already under way; it began
                before the imported window
              </>
            ) : (
              <>Window opens — beyond home range</>
            )
          ) : candidate?.end_truncated ? (
            <>
              Window closes — still in progress at the window&apos;s edge
            </>
          ) : (
            <>Window closes — back within home range</>
          ),
      };
  }
}

// Photos slot into the timeline by placement (BRIEF §3G): each placed photo
// appears inline on the leg or dwell whose span held its instant —
// thumbnails where the journey says they happened.
function PhotoThumbs({ photos }: { photos: Photo[] }) {
  return (
    <span className="inline-flex gap-1 align-middle">
      {photos.map((p) => (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          key={p.id}
          src={`/api/photos/${p.id}/thumb`}
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
