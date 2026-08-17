import Link from "next/link";

import { api } from "@/lib/api/client";
import { INSTANCE_LABEL } from "@/lib/instance";
import { fmtDateRange, sliceDays } from "@/lib/slice-days";
import { LegKindLegend } from "@/components/legend";
import { ProvenanceBar } from "@/components/provenance-bar";
import { RouteThumb } from "@/components/route-thumb";
import { SiteHeader } from "@/components/site-header";
import type { components } from "@/lib/api/schema";

// The adventures atlas (phase 9 CP3, decided at the mockup STOP): every
// confirmed adventure as a plate cover — route art in the four-way kind
// encoding, name, dates, distance with its provenance bar, countries, and
// the flag mark when divergence is flagged. No score: DESIGN §4 retires it
// from every confirmed cover. The life map stays the map; this page is the
// shelf. Covers are static SVG (lib/route-thumb.ts) — a MapLibre instance
// per tile would exhaust the browser's WebGL context cap.
//
// Order, as decided: newest first. Plate numbers are chronological — the
// number is the atlas register (the order travelled), not the sort key,
// so numbering and display order deliberately disagree.
export const dynamic = "force-dynamic";

type Candidate = components["schemas"]["Candidate"];
type Journey = components["schemas"]["Journey"];

type Cover = {
  candidate: Candidate;
  journey: Journey;
  /** Chronological plate number, 1-based, oldest journey = PLATE 1. */
  plate: number;
};

export default async function AdventuresPage() {
  // Connection-refused rejects rather than returning {error} — the same
  // trap the home page documents (phase 9 CP2); catch to a designed state.
  const res = await api.GET("/candidates").catch(() => null);

  if (!res?.data) {
    return (
      <Shell>
        <p className="mt-6 text-red-700">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </Shell>
    );
  }

  const undecided = res.data.candidates.filter((c) => !c.decision).length;
  const confirmed = res.data.candidates.filter(
    (c) => c.decision?.action === "confirmed",
  );

  if (confirmed.length === 0) {
    return (
      <Shell undecided={undecided}>
        <div className="mt-14 border border-rule bg-paper px-5 py-10 text-center sm:px-8">
          <h2 className="font-display text-2xl font-semibold">
            No adventures yet
          </h2>
          <p className="mx-auto mt-3 max-w-[52ch] text-ink-2">
            Your atlas fills as you confirm candidates — journeys detected
            in your location history that look like real trips.
          </p>
          <p className="mt-5">
            <Link
              href="/candidates"
              className="inline-block border border-ink bg-paper px-5 py-2 font-semibold hover:bg-land"
            >
              Review candidates
              {undecided > 0 && <> — {undecided} undecided</>}
            </Link>
          </p>
        </div>
      </Shell>
    );
  }

  // The same fan-out the life map runs (BRIEF §2, measured): every
  // confirmed journey in parallel. A journey that cannot be fetched drops
  // out rather than failing the shelf.
  const withJourneys = (
    await Promise.all(
      confirmed.map(async (candidate) => {
        const r = await api
          .GET("/candidates/{id}/journey", {
            params: { path: { id: candidate.id } },
          })
          .catch(() => null);
        return r?.data ? { candidate, journey: r.data } : null;
      }),
    )
  ).filter((c): c is Omit<Cover, "plate"> => c !== null);

  // Plate numbers come from travel order (span_start ascending — lexical
  // is chronological for RFC 3339 within these strings' shared shape);
  // display order is newest first.
  const chronological = [...withJourneys].sort((a, b) =>
    a.candidate.span_start.localeCompare(b.candidate.span_start),
  );
  const covers: Cover[] = chronological
    .map((c, i) => ({ ...c, plate: i + 1 }))
    .reverse();

  return (
    <Shell undecided={undecided}>
      <p className="mt-1 text-sm text-ink-2">
        {covers.length} confirmed adventure{covers.length === 1 ? "" : "s"},
        most recent first. Plate numbers follow the order travelled.
      </p>
      <ul className="mt-6 grid list-none grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-7">
        {covers.map((c) => (
          <li key={c.candidate.id} className="m-1">
            <PlateCover cover={c} />
          </li>
        ))}
      </ul>
      <div className="mt-8 border-t border-rule pt-4">
        <LegKindLegend />
      </div>
    </Shell>
  );
}

function Shell({
  undecided,
  children,
}: {
  undecided?: number;
  children: React.ReactNode;
}) {
  return (
    <main className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
      <SiteHeader
        undecided={undecided}
        active="adventures"
        instanceLabel={INSTANCE_LABEL}
      />
      <h1 className="mt-8 font-display text-2xl font-semibold">Adventures</h1>
      {children}
    </main>
  );
}

// One plate cover (cover density B, decided at the CP3 review): the whole
// tile is the link. The double rule is the plate signature at tile size;
// the art carries the same reserved inks and non-color channels as every
// map (invariant 8 — kind is never merged, even at thumbnail scale).
function PlateCover({ cover }: { cover: Cover }) {
  const { candidate, journey, plate } = cover;
  const name = candidate.decision?.name ?? "Unnamed adventure";
  const days = sliceDays(journey).length;
  const countries = journey.countries.map((c) => c.name).join(" · ");

  return (
    <Link
      href={`/adventure/${candidate.id}`}
      className="group block border border-ink bg-paper no-underline [outline:1px_solid_var(--color-ink)] [outline-offset:3px] focus-visible:[outline:2px_solid_var(--color-ink)]"
    >
      <div className="relative border-b border-rule">
        <RouteThumb legs={journey.legs} className="block h-auto w-full" />
        {/* Paper behind the number so route art never strikes through it. */}
        <span
          className="absolute right-2.5 top-2 bg-paper pl-1.5 font-display text-[11px] font-semibold tracking-[0.22em] text-ink-2"
          aria-hidden
        >
          PLATE {plate}
        </span>
      </div>
      <div className="px-3.5 py-3">
        <p className="text-[10px] uppercase tracking-[0.22em] text-ink-2">
          {countries || " "}
        </p>
        <p className="mt-0.5 flex items-baseline justify-between gap-2">
          <span className="font-display text-lg font-semibold leading-tight underline-offset-2 group-hover:underline">
            {name}
          </span>
          {journey.divergence_flagged && (
            <span className="shrink-0 text-flag" title="Divergence flagged — details on the adventure page">
              <span aria-hidden>⚑</span>
              <span className="sr-only">divergence flagged</span>
            </span>
          )}
        </p>
        <p className="mb-2 mt-1 flex items-baseline justify-between gap-2 font-mono text-xs text-ink-2">
          <span>
            {fmtDateRange(candidate.span_start, candidate.span_end)} · {days}{" "}
            day{days === 1 ? "" : "s"}
          </span>
          <span className="font-medium text-ink">
            {journey.total_km.toFixed(0)} km
          </span>
        </p>
        <ProvenanceBar
          observed={journey.observed_km}
          routed={journey.routed_km}
          unknown={journey.unknown_km}
          air={journey.air_km}
        />
      </div>
    </Link>
  );
}
