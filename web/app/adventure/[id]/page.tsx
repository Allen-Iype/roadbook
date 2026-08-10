import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api/client";
import { MAP_STYLE_URL } from "@/lib/basemap";
import { SiteHeader } from "@/components/site-header";
import { AdventureView } from "./adventure-view";

// Server component for one adventure. Next 16 dynamic route: the [id] folder
// segment arrives via `params`, which is a Promise here (async request APIs,
// Next 15+) — it must be awaited before use. This component only fetches;
// everything below the masthead renders in the adventure view, a client
// component, because the selected day is state shared between the narrative
// and the map plate.
export const dynamic = "force-dynamic";

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
      <main className="mx-auto w-full max-w-[91rem] px-6 py-8">
        <p className="text-red-700">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </main>
    );
  }
  const journey = journeyRes.data;
  const candidates = listRes.data?.candidates ?? [];
  const candidate = candidates.find((c) => c.id === id);
  const confirmed = candidate?.decision?.action === "confirmed";

  // Photos exist only for confirmed adventures (phase 4 BRIEF §3A) — the
  // list call is skipped otherwise rather than made to 409.
  const photos = confirmed
    ? (
        await api.GET("/candidates/{id}/photos", { params: { path: { id } } })
      ).data?.photos ?? []
    : null;

  // The plate number: this adventure's position among confirmed adventures
  // in date order — derived from the data on every render, never stored.
  const confirmedIds = candidates
    .filter((c) => c.decision?.action === "confirmed")
    .sort((a, b) => a.span_start.localeCompare(b.span_start))
    .map((c) => c.id);
  const plate = confirmed ? confirmedIds.indexOf(id) + 1 : null;
  const undecided = candidates.filter((c) => !c.decision).length;

  return (
    <main className="mx-auto w-full max-w-[91rem] px-6 py-8">
      <SiteHeader undecided={undecided} />
      <p className="mt-6 text-sm">
        <Link href="/" className="text-ink-2 hover:text-ink">
          ← Life map
        </Link>
      </p>
      <AdventureView
        journey={journey}
        candidate={candidate}
        photos={photos}
        styleUrl={MAP_STYLE_URL}
        plate={plate}
      />
    </main>
  );
}
