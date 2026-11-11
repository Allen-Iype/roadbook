import Link from "next/link";
import { notFound } from "next/navigation";

import { api } from "@/lib/api/client";
import { MAP_STYLE_URL } from "@/lib/basemap";
import { SiteHeader } from "@/components/site-header";
import { Unreachable } from "@/components/unreachable";
import { INSTANCE_LABEL } from "@/lib/instance";
import { requireUser } from "@/lib/session";
import { AdventureView } from "@/components/adventure/adventure-view";
import { EditTrip } from "@/components/adventure/edit-trip";
import { ShareLinks } from "@/components/share-links";
import { PhotosSection } from "./photos-section";

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
  // The auth gate (phase 13 CP3): UX only — Go 401s regardless. Mode
  // off returns pass-through and this page renders exactly as before.
  const session = await requireUser();
  const accountEmail =
    session?.mode === "oidc" ? (session.email ?? "") : undefined;

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
      <main className="mx-auto w-full max-w-[91rem] px-4 py-8 sm:px-6">
        <Unreachable operator={session?.mode === "off"} className="text-red-700" />
      </main>
    );
  }
  const journey = journeyRes.data;
  const candidates = listRes.data?.candidates ?? [];
  const candidate = candidates.find((c) => c.id === id);
  const confirmed = candidate?.decision?.action === "confirmed";

  // Photos exist only for confirmed adventures (phase 4 BRIEF §3A) — the
  // list call is skipped otherwise rather than made to 409. Import-photo
  // records are different (CP4): span-joined at read time for any candidate,
  // because during triage they are evidence of what the trip was.
  // Share links (phase 13 CP4) exist for confirmed adventures only, like
  // photos; the list is fetched alongside them.
  const [photos, importPhotos, shares] = await Promise.all([
    confirmed
      ? api
          .GET("/candidates/{id}/photos", { params: { path: { id } } })
          .then((r) => r.data?.photos ?? [])
      : Promise.resolve(null),
    api
      .GET("/candidates/{id}/import-photos", { params: { path: { id } } })
      .then((r) => r.data?.photos ?? []),
    confirmed
      ? api
          .GET("/candidates/{id}/shares", { params: { path: { id } } })
          .then((r) => r.data?.shares ?? [])
      : Promise.resolve([]),
  ]);

  const undecided = candidates.filter((c) => !c.decision).length;

  return (
    <main className="mx-auto w-full max-w-[91rem] px-4 py-8 sm:px-6">
      <SiteHeader undecided={undecided} instanceLabel={INSTANCE_LABEL}
        accountEmail={accountEmail} />
      <p className="mt-6 text-sm">
        <Link href="/" className="text-ink-2 hover:text-ink">
          ← Map
        </Link>
      </p>
      <AdventureView
        journey={journey}
        candidate={candidate}
        photos={photos}
        importPhotos={importPhotos}
        styleUrl={MAP_STYLE_URL}
        photosSection={
          photos !== null && candidate ? (
            <PhotosSection candidateId={candidate.id} photos={photos} />
          ) : undefined
        }
        ownerWords={
          confirmed && candidate ? (
            <EditTrip
              candidateId={candidate.id}
              name={candidate.decision?.name ?? ""}
              notes={candidate.decision?.notes}
            />
          ) : undefined
        }
        shareControls={
          confirmed && candidate ? (
            <ShareLinks
              candidateId={candidate.id}
              shares={shares}
              photoCount={(photos?.length ?? 0) + importPhotos.length}
            />
          ) : undefined
        }
      />
    </main>
  );
}
