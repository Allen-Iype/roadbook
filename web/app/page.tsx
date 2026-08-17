import Link from "next/link";
import { redirect } from "next/navigation";

import { api } from "@/lib/api/client";
import { MAP_STYLE_URL } from "@/lib/basemap";
import { LifeMap, type Adventure } from "@/components/life-map";
import { SummonedList } from "@/components/summoned-list";
import { LegKindLegend } from "@/components/legend";
import { SiteHeader } from "@/components/site-header";

// The home page is the life map (phase 6, the anchor vision): one map, the
// union of every confirmed adventure's route, and the map is the
// navigation. Everything else — triage, imports — is an entry point
// floating in a corner.
//
// Data path (BRIEF §2, measured before it was decided): this server
// component fans out — the candidate list, then every confirmed journey in
// parallel. No aggregate endpoint exists for this page; at charter scale
// (tens of confirmed adventures) the fan-out costs milliseconds. The
// revisit trigger is recorded in the BRIEF, not here.
export const dynamic = "force-dynamic";

export default async function HomePage() {
  // A rejected fetch (API container down, connection refused) THROWS — the
  // generated client's {error} shape only covers HTTP responses — and an
  // uncaught throw here lands on the generic error boundary instead of the
  // designed API-down state below. Found by phase 9 CP2's empty-state
  // render check: the designed copy was dead code for the very failure it
  // described. null = "no answer", distinct from an HTTP error.
  const safe = <T,>(p: Promise<T>): Promise<T | null> =>
    p.catch(() => null);
  const [candidatesRes, importsRes] = await Promise.all([
    safe(api.GET("/candidates")),
    safe(api.GET("/imports")),
  ]);
  const data = candidatesRes?.data;
  const error = candidatesRes ? candidatesRes.error : undefined;

  // A cold instance — nothing has ever been imported — routes to the front
  // door (phase 7 BRIEF §3E): a shared link lands on the pitch, and the same
  // link lands here once adventures exist. Only an answered-and-empty
  // imports list redirects; an unreachable API renders as exactly that.
  if (importsRes?.data && importsRes.data.imports.length === 0) {
    redirect("/welcome");
  }

  if (error || !data) {
    return (
      <Invitation>
        <p className="text-red-700">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </Invitation>
    );
  }

  const candidates = data.candidates;
  const undecided = candidates.filter((c) => !c.decision).length;
  const confirmed = candidates.filter(
    (c) => c.decision?.action === "confirmed",
  );

  // Empty state one: imports exist but no candidates. Two different facts
  // deserve two different sentences (BRIEF §3E, the zero-candidates ending):
  // detection ran and found nothing — designed copy, never a blank screen —
  // versus detection never ran (a CLI import without a detect).
  if (candidates.length === 0) {
    return (
      <Invitation>
        {data.run ? (
          <>
            <h1 className="font-display text-2xl font-semibold">
              No adventures found — yet
            </h1>
            <p className="mx-auto mt-3 max-w-[52ch] text-ink-2">
              Your data imported fine. Detection ran and found no journeys
              far from home with a real destination — it looks for time
              spent well away from every home base, staying somewhere, and
              pass-throughs don&apos;t count.
            </p>
            <p className="mx-auto mt-3 max-w-[52ch] text-ink-2">
              More data may change that:{" "}
              <Link
                href="/welcome"
                className="underline decoration-rule underline-offset-2 hover:text-ink"
              >
                add another export
              </Link>{" "}
              — earlier years, another device — or see{" "}
              <Link
                href="/imports"
                className="underline decoration-rule underline-offset-2 hover:text-ink"
              >
                Imports
              </Link>{" "}
              for what was ingested.
            </p>
          </>
        ) : (
          <>
            <h1 className="font-display text-2xl font-semibold">
              No adventures yet
            </h1>
            <p className="mx-auto mt-3 max-w-[52ch] text-ink-2">
              Roadbook draws your life map from the journeys in a Google
              Timeline export. Data has been imported but detection has not
              run yet.
            </p>
            <pre className="mx-auto mt-4 w-fit bg-land px-4 py-3 text-left font-mono text-sm">
              {"roadbook detect -from-db"}
            </pre>
            <p className="mt-3 text-sm text-ink-2">
              Or{" "}
              <Link
                href="/welcome"
                className="underline decoration-rule underline-offset-2 hover:text-ink"
              >
                add your data through the browser
              </Link>
              , which runs detection automatically.
            </p>
          </>
        )}
      </Invitation>
    );
  }

  // Empty state two: candidates exist, none confirmed. The life map draws
  // only confirmed adventures (the curation boundary — never coverage), so
  // the invitation points at triage.
  if (confirmed.length === 0) {
    return (
      <Invitation undecided={undecided}>
        <h1 className="font-display text-2xl font-semibold">
          {candidates.length} candidate{candidates.length === 1 ? "" : "s"}{" "}
          await review
        </h1>
        <p className="mt-3 text-ink-2">
          The life map draws confirmed adventures only — none are confirmed
          yet. Confirm the real journeys and they appear here as routes.
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
      </Invitation>
    );
  }

  // The fan-out: every confirmed journey, in parallel. A journey that
  // cannot be fetched (stale id mid-re-detection) drops out of the map
  // rather than failing the page; the summoned list and the map always
  // agree because both render from this same array.
  const adventures = (
    await Promise.all(
      confirmed.map(async (c): Promise<Adventure | null> => {
        const r = await api.GET("/candidates/{id}/journey", {
          params: { path: { id: c.id } },
        });
        if (!r.data) return null;
        return {
          id: c.id,
          name: c.decision?.name ?? "Unnamed adventure",
          start: c.span_start,
          end: c.span_end,
          journey: r.data,
        };
      }),
    )
  ).filter((a): a is Adventure => a !== null);

  if (adventures.length === 0) {
    return (
      <Invitation undecided={undecided}>
        <p className="text-red-700">
          {confirmed.length} confirmed adventure
          {confirmed.length === 1 ? "" : "s"} exist, but no journey could be
          loaded from the API. Reload; if it persists, check that the API and
          database are up.
        </p>
      </Invitation>
    );
  }

  return (
    <main className="fixed inset-0 overflow-hidden bg-land">
      <LifeMap adventures={adventures} styleUrl={MAP_STYLE_URL} />

      {/* The floating chrome. pointer-events-none on the frame lets map
          drags pass through everywhere except the panels themselves. */}
      <div className="pointer-events-none absolute inset-0 flex flex-col justify-between p-4 sm:p-5">
        <header className="flex flex-wrap items-start justify-between gap-3">
          <p className="pointer-events-auto border border-rule bg-paper px-4 py-2.5 font-display text-lg font-semibold tracking-[0.28em] shadow-sm">
            ROADBOOK
          </p>
          <nav className="pointer-events-auto flex flex-wrap items-center gap-x-5 gap-y-1 border border-rule bg-paper px-4 py-2.5 text-sm shadow-sm">
            {/* The summoned list first: it is the accessible enumeration of
                the map (DESIGN §5), not a secondary link. */}
            <SummonedList adventures={adventures} />
            <Link
              href="/candidates"
              className="-mx-2 -my-3 px-2 py-3 underline decoration-rule underline-offset-2 hover:text-ink"
            >
              Candidates
              {undecided > 0 && <> — {undecided} undecided</>}
            </Link>
            <Link
              href="/imports"
              className="-mx-2 -my-3 px-2 py-3 underline decoration-rule underline-offset-2 hover:text-ink"
            >
              Imports
            </Link>
          </nav>
        </header>

        <footer className="flex items-end justify-between">
          {/* The plate-margin legend, permanent (DESIGN §6 treatment T3):
              the map never appears without its language. */}
          <div className="pointer-events-auto max-w-2xl border border-rule bg-paper px-4 py-2.5 shadow-sm">
            <LegKindLegend />
          </div>
        </footer>
      </div>
    </main>
  );
}

// The non-map states share the workbench shell: same header as candidates
// and imports, content centered as a plate. Used for both invitations and
// the API-down report.
function Invitation({
  undecided,
  children,
}: {
  undecided?: number;
  children: React.ReactNode;
}) {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <SiteHeader undecided={undecided} />
      <div className="mt-16 border border-rule bg-paper px-5 py-10 text-center sm:px-8">
        {children}
      </div>
    </main>
  );
}
