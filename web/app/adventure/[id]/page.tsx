import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { PhotosSection } from "./photos-section";
import { RouteMap } from "./route-map";

// The basemap style URL is server-side config passed down as a prop — the
// phase 5 "configurable tile provider" seam, done cheap now (BRIEF §1.2).
// Tile requests are the map's one unavoidable third-party dependency; the
// default is OpenFreeMap, which needs no key and no account.
const MAP_STYLE_URL =
  process.env.ROADBOOK_MAP_STYLE ?? "https://tiles.openfreemap.org/styles/liberty";

// Server component for one adventure. Next 16 dynamic route: the [id] folder
// segment arrives via `params`, which is a Promise here (async request APIs,
// Next 15+) — it must be awaited before use. Everything on this page is
// server-rendered per request; the map island joins in checkpoint 3 as the
// page's one client component.
export const dynamic = "force-dynamic";

type Journey = components["schemas"]["Journey"];
type Leg = components["schemas"]["Leg"];
type Candidate = components["schemas"]["Candidate"];
type Photo = components["schemas"]["Photo"];

export default async function AdventurePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: rawId } = await params;
  const id = Number(rawId);
  if (!Number.isInteger(id)) notFound();

  // Two calls: the journey, and the candidate list for this candidate's own
  // row (dates, badges, decision name). At tens of candidates a filtered
  // list fetch costs nothing; a dedicated GET /candidates/{id} can join the
  // contract when something real needs it.
  const [journeyRes, listRes] = await Promise.all([
    api.GET("/candidates/{id}/journey", { params: { path: { id } } }),
    api.GET("/candidates"),
  ]);

  if (journeyRes.error || !journeyRes.data) {
    // The generated client types the 404 body; anything else is "API down".
    if (journeyRes.response?.status === 404) notFound();
    return (
      <main className="mx-auto w-full max-w-5xl px-6 py-10">
        <p className="text-red-400">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </main>
    );
  }
  const journey = journeyRes.data;
  const candidate = listRes.data?.candidates.find((c) => c.id === id);

  // Photos exist only for confirmed adventures (BRIEF §3A) — the list call
  // is skipped otherwise rather than made to 409.
  const confirmed = candidate?.decision?.action === "confirmed";
  const photos = confirmed
    ? (
        await api.GET("/candidates/{id}/photos", { params: { path: { id } } })
      ).data?.photos ?? []
    : null;

  return (
    <main className="mx-auto w-full max-w-5xl px-6 py-10">
      <p className="text-sm">
        <Link href="/" className="text-neutral-400 hover:text-neutral-200">
          ← All candidates
        </Link>
      </p>
      <Header journey={journey} candidate={candidate} />
      {journey.legs.length > 0 && (
        <>
          <RouteMap
            journey={journey}
            styleUrl={MAP_STYLE_URL}
            photos={photos ?? undefined}
          />
          <Legend
            hasAir={journey.air_km > 0}
            hasRoad={journey.routed_km > 0}
            hasPhotos={(photos ?? []).some((p) => p.place_kind)}
            hasFlagged={(photos ?? []).some((p) => p.far_flagged)}
          />
        </>
      )}
      <Provenance journey={journey} />
      {photos !== null && (
        <PhotosSection candidateId={id} photos={photos} />
      )}
      <LegTable legs={journey.legs} photos={photos ?? []} />
      <Stops journey={journey} photos={photos ?? []} />
    </main>
  );
}

// The legend states the visual channel in words (invariant 8: confidence is
// never hidden, and never left to interpretation). Entries for classes the
// journey does not contain are omitted — a legend describing lines that are
// not on the map would be noise, not honesty.
function Legend({
  hasAir,
  hasRoad,
  hasPhotos,
  hasFlagged,
}: {
  hasAir: boolean;
  hasRoad: boolean;
  hasPhotos: boolean;
  hasFlagged: boolean;
}) {
  return (
    <p className="mt-2 flex flex-wrap items-center gap-x-5 gap-y-1 text-xs text-neutral-400">
      <span className="inline-flex items-center gap-2">
        <span className="inline-block h-1 w-8 rounded bg-emerald-500" />
        observed — positions recorded along the way
      </span>
      <span className="inline-flex items-center gap-2">
        <svg width="32" height="4" aria-hidden>
          <line
            x1="0"
            y1="2"
            x2="32"
            y2="2"
            stroke="#a3a3a3"
            strokeWidth="2"
            strokeDasharray="4 5"
          />
        </svg>
        unknown gap — no observations; the straight line is not a route
      </span>
      {hasRoad && (
        <span className="inline-flex items-center gap-2">
          <svg width="32" height="4" aria-hidden>
            <line
              x1="0"
              y1="2"
              x2="32"
              y2="2"
              stroke="#38bdf8"
              strokeWidth="2.5"
              strokeDasharray="4 5"
            />
          </svg>
          routed road — inferred from the road network, not measured
        </span>
      )}
      {hasAir && (
        <span className="inline-flex items-center gap-2">
          <svg width="32" height="4" aria-hidden>
            <line
              x1="0"
              y1="2"
              x2="32"
              y2="2"
              stroke="#a78bfa"
              strokeWidth="2"
              strokeDasharray="4 5"
            />
          </svg>
          air — implied speed says flight; the arc is a great circle, not a
          track
        </span>
      )}
      <span className="inline-flex items-center gap-2">
        <span className="inline-block h-2.5 w-2.5 rounded-full border border-neutral-900 bg-amber-400" />
        stop
      </span>
      {hasPhotos && (
        <span className="inline-flex items-center gap-2">
          <span className="inline-block h-3 w-3 rounded border border-neutral-300 bg-neutral-600" />
          photo — positioned by its own metadata, a measurement
        </span>
      )}
      {hasFlagged && (
        <span className="inline-flex items-center gap-2">
          <span className="inline-block h-3 w-3 rounded border-2 border-amber-400 bg-neutral-600" />
          flagged photo — sits far from the route drawn for its time
        </span>
      )}
    </p>
  );
}

function Header({
  journey,
  candidate,
}: {
  journey: Journey;
  candidate?: Candidate;
}) {
  const name =
    candidate?.decision?.action === "confirmed" && candidate.decision.name
      ? candidate.decision.name
      : `Journey of ${journey.window_start.slice(0, 10)}`;
  const days = (
    (Date.parse(journey.window_end) - Date.parse(journey.window_start)) /
    86_400_000
  ).toFixed(1);
  return (
    <header className="mt-4">
      <h1 className="text-2xl font-semibold">{name}</h1>
      <p className="mt-1 text-sm text-neutral-400">
        {journey.window_start.slice(0, 10)} → {journey.window_end.slice(0, 10)}{" "}
        · {days} days
        {candidate && <> · {candidate.dest_km} km from home</>}
        {candidate?.start_truncated && (
          <span className="ml-1 text-amber-400" title="Began before the imported window.">
            ◂ truncated
          </span>
        )}
        {candidate?.end_truncated && (
          <span className="ml-1 text-amber-400" title="Still in progress at the window edge.">
            truncated ▸
          </span>
        )}
      </p>
      {journey.countries.length > 0 && (
        // Derived line (BRIEF §1.4): point-in-polygon against bundled Natural
        // Earth polygons, computed locally — the label says so because border
        // -adjacent points can misattribute at 1:110m resolution.
        <p className="mt-1 text-sm text-neutral-300">
          {journey.countries.map((c) => c.name).join(" · ")}{" "}
          <span className="text-xs text-neutral-500">
            — countries derived from route points
          </span>
        </p>
      )}
    </header>
  );
}

// The provenance line is the page's honesty statement (invariants 5 and 8):
// how much of the drawn journey is measurement, how much is inference.
function Provenance({ journey }: { journey: Journey }) {
  const pct = (part: number) =>
    journey.total_km > 0 ? ((part / journey.total_km) * 100).toFixed(1) : "0";
  return (
    <section className="mt-6 text-sm">
      <p>
        <span className="font-mono">{journey.total_km.toFixed(1)} km</span>{" "}
        reconstructed —{" "}
        <span className="text-emerald-400">
          {journey.observed_km.toFixed(1)} km observed ({pct(journey.observed_km)}%)
        </span>{" "}
        +{" "}
        <span className="text-neutral-400">
          {journey.inferred_km.toFixed(1)} km inferred ({pct(journey.inferred_km)}%)
        </span>
        {journey.air_km > 0 && (
          <>
            {" "}
            —{" "}
            <span className="text-violet-400">
              of which {journey.air_km.toFixed(1)} km air (great-circle)
            </span>
          </>
        )}
      </p>
      {journey.routed_km > 0 && (
        <p className="mt-1">
          <span className="text-sky-400">
            routed roads cover {journey.routed_km.toFixed(1)} km
          </span>{" "}
          <span className="text-neutral-400">
            (from the routing cache);{" "}
            {journey.unknown_km > 0
              ? `${journey.unknown_km.toFixed(1)} km of gaps remain unknown`
              : "no gaps remain unknown"}
          </span>
        </p>
      )}
      {journey.divergence_pct !== undefined && (
        // The phase 3 validation line (BRIEF §3E): the road-comparable
        // reconstruction against Google's own ground figure — air excluded
        // from both sides. The flag is computed server-side against the
        // named divergence_warn_pct parameter; the frontend only renders
        // it. Divergence is a conversation starter, never a gate: the
        // unknown and routed legs explaining it are right there on the map.
        <p className="mt-1">
          <span className="text-neutral-300">
            ground reconstruction{" "}
            <span className="font-mono">{journey.ground_km.toFixed(1)} km</span>{" "}
            · Google&apos;s ground figure{" "}
            <span className="font-mono">
              {journey.google_ground_km.toFixed(1)} km
            </span>
          </span>{" "}
          <span
            className={
              journey.divergence_flagged ? "text-amber-400" : "text-neutral-400"
            }
          >
            ({journey.divergence_pct >= 0 ? "+" : ""}
            {journey.divergence_pct.toFixed(1)}%
            {journey.divergence_flagged && " — diverges beyond the warning threshold"}
            )
          </span>
        </p>
      )}
      <p className="mt-1 text-xs text-neutral-500">
        {journey.merged_points} points ({journey.trace_points_kept} trace +{" "}
        {journey.raw_points_kept} raw)
        {journey.google_km > 0 && (
          <> · Google&apos;s own figure {journey.google_km.toFixed(1)} km total</>
        )}
      </p>
    </section>
  );
}

// Photos slot into the timeline by placement (BRIEF §3G): each placed photo
// appears inline on the leg or stop whose span held its instant — thumbnails
// where the journey says they happened.
function photoThumbs(photos: Photo[], flagged?: boolean) {
  if (photos.length === 0) return null;
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
            flagged || p.far_flagged ? "ring-1 ring-amber-400" : ""
          }`}
        />
      ))}
    </span>
  );
}

function LegTable({ legs, photos }: { legs: Leg[]; photos: Photo[] }) {
  const byLeg = new Map<number, Photo[]>();
  for (const p of photos) {
    if (p.leg_index !== undefined) {
      byLeg.set(p.leg_index, [...(byLeg.get(p.leg_index) ?? []), p]);
    }
  }
  return (
    <section className="mt-6">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-neutral-500">
        Legs
      </h2>
      <div className="mt-2 overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-neutral-800 text-left text-xs uppercase tracking-wide text-neutral-500">
              <th className="py-2 pr-4">#</th>
              <th className="py-2 pr-4">Kind</th>
              <th className="py-2 pr-4">From</th>
              <th className="py-2 pr-4">To</th>
              <th className="py-2 pr-4 text-right">Minutes</th>
              <th className="py-2 pr-4 text-right">km</th>
              <th className="py-2 pr-4 text-right">Points</th>
              <th className="py-2">Photos</th>
            </tr>
          </thead>
          <tbody>
            {legs.map((l, i) => (
              <tr key={i} className="border-b border-neutral-900">
                <td className="py-2 pr-4 font-mono text-neutral-500">{i + 1}</td>
                <td className="py-2 pr-4">
                  {l.kind === "observed" ? (
                    <span className="text-emerald-400">observed</span>
                  ) : l.gap_kind === "air" ? (
                    <span className="text-violet-400">gap · air</span>
                  ) : l.gap_kind === "road" ? (
                    <span className="text-sky-400">gap · road</span>
                  ) : (
                    <span className="text-neutral-400">
                      gap · {l.gap_kind}
                    </span>
                  )}
                </td>
                <td className="py-2 pr-4 font-mono">{clock(l.start)}</td>
                <td className="py-2 pr-4 font-mono">{clock(l.end)}</td>
                <td className="py-2 pr-4 text-right">
                  {((Date.parse(l.end) - Date.parse(l.start)) / 60_000).toFixed(0)}
                </td>
                <td className="py-2 pr-4 text-right">{l.distance_km.toFixed(1)}</td>
                <td className="py-2 pr-4 text-right">{l.points.length}</td>
                <td className="py-2">{photoThumbs(byLeg.get(i) ?? [])}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function Stops({ journey, photos }: { journey: Journey; photos: Photo[] }) {
  if (journey.stops.length === 0) return null;
  const byStop = new Map<number, Photo[]>();
  for (const p of photos) {
    if (p.stop_index !== undefined) {
      byStop.set(p.stop_index, [...(byStop.get(p.stop_index) ?? []), p]);
    }
  }
  return (
    <section className="mt-6">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-neutral-500">
        Stops
      </h2>
      <ul className="mt-2 text-sm">
        {journey.stops.map((s, i) => (
          <li key={i} className="py-1">
            <span className="font-mono">
              {clock(s.start)} – {clock(s.end)}
            </span>{" "}
            <span className="text-neutral-400">
              ({((Date.parse(s.end) - Date.parse(s.start)) / 60_000).toFixed(0)}{" "}
              min, moved {s.displacement_km.toFixed(2)} km)
            </span>{" "}
            {photoThumbs(byStop.get(i) ?? [])}
          </li>
        ))}
      </ul>
    </section>
  );
}

// Times keep the journey's own UTC offset (the API preserves it); slicing the
// clock out of the string shows traveller-local time, where `new Date()`
// would silently shift it into the viewer's timezone.
function clock(iso: string): string {
  return iso.slice(11, 16);
}
