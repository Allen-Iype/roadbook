import Link from "next/link";

import { api } from "@/lib/api/client";
import { INSTANCE_LABEL } from "@/lib/instance";
import type { components } from "@/lib/api/schema";
import { DecideCell } from "@/app/decide-cell";
import { RouteThumb } from "@/components/route-thumb";
import { SiteHeader } from "@/components/site-header";
import { TriageBar } from "@/components/triage-bar";
import { TriageSelect } from "@/components/triage-select";

// The triage workbench. Phase 9 CP3 review decided T3: triage is a gallery
// of route-shape cards, not a table — the maintainer's call at the mockup
// STOP (docs/phase-9/DECISIONS.md records the trade: column-scannable
// scores and rows-per-screen went; shape-led recall arrived — you decide
// whether a journey was real by seeing what it looked like). Each card is
// server HTML with the same per-candidate DecideCell island the table rows
// carried since phase 1.
export const dynamic = "force-dynamic";

type Candidate = components["schemas"]["Candidate"];
type CandidateList = components["schemas"]["CandidateList"];
type Leg = components["schemas"]["Leg"];

export default async function CandidatesPage({
  searchParams,
}: {
  searchParams: Promise<{ sort?: string }>;
}) {
  // ?sort=score is the sweep order (phase 11 §6.1): score-descending, built
  // for clearing a long queue in one sitting. Default stays chronological.
  // A search param rather than client state: the order is part of the URL a
  // person can share or reload, and the server sorts before any HTML ships.
  const { sort } = await searchParams;
  const byScore = sort === "score";
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
  const data = res.data;

  // The cards want the journey's shape, so triage fans out like the life
  // map does (BRIEF §2: measured milliseconds per request at charter
  // scale). Undecided and dismissed candidates get art too — the shape is
  // exactly what the decision needs. A failed fetch costs that card its
  // art, never the page.
  const legsById = new Map<number, Leg[]>();
  await Promise.all(
    data.candidates.map(async (c) => {
      const r = await api
        .GET("/candidates/{id}/journey", { params: { path: { id: c.id } } })
        .catch(() => null);
      if (r?.data) legsById.set(c.id, r.data.legs);
    }),
  );

  // Sort a copy — never the fetched array. Unscored candidates (pre-scoring
  // runs) sink to the end rather than interleaving as zeros.
  const ordered = byScore
    ? [...data.candidates].sort((a, b) => (b.score ?? -1) - (a.score ?? -1))
    : data.candidates;

  return (
    <Shell>
      <RunSummary list={data} />
      {data.candidates.length > 0 && <SortToggle byScore={byScore} />}
      {data.candidates.length === 0 ? (
        data.run ? (
          <div className="mt-6 max-w-[58ch]">
            <p className="font-semibold">
              Detection ran and found no candidates.
            </p>
            <p className="mt-2 text-ink-2">
              Your data imported fine — no journeys far from home with a
              real destination were found in it. Detection looks for time
              spent well away from every home base with a genuine stay;
              short trips and pass-throughs don&apos;t count.
            </p>
            <p className="mt-2 text-ink-2">
              More data may change that:{" "}
              <Link
                href="/welcome"
                className="underline decoration-rule underline-offset-2 hover:text-ink"
              >
                add another export
              </Link>{" "}
              — earlier years, another device.
            </p>
          </div>
        ) : (
          <p className="mt-6 text-ink-2">
            No candidates.{" "}
            <Link
              href="/welcome"
              className="underline decoration-rule underline-offset-2 hover:text-ink"
            >
              Add a Timeline export
            </Link>{" "}
            — or import one with the CLI and run{" "}
            <code className="font-mono">roadbook detect -from-db</code>.
          </p>
        )
      ) : (
        <>
          <CandidateGallery candidates={ordered} legsById={legsById} />
          <TriageBar />
        </>
      )}
      {data.orphaned_decisions.length > 0 && (
        <p className="mt-4 text-sm">
          <span className="font-bold text-flag" aria-hidden>
            ⚑
          </span>{" "}
          {data.orphaned_decisions.length} decision
          {data.orphaned_decisions.length === 1 ? "" : "s"} no longer match a
          candidate under the current parameters. They are preserved and will
          re-match if parameters change back.
        </p>
      )}
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <main className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
      <SiteHeader active="candidates" instanceLabel={INSTANCE_LABEL} />
      <h1 className="mt-8 font-display text-2xl font-semibold">Candidates</h1>
      <p className="mt-1 text-sm text-ink-2">
        Adventure candidates detected from your timeline. Confirm the real
        ones; dismiss the rest.
      </p>
      {children}
    </main>
  );
}

// The sweep-order toggle (phase 11 §6.1). Links, not buttons: the order is
// URL state, and the score order is the "clear the queue" mode — highest
// confidence first, keyboard from there.
function SortToggle({ byScore }: { byScore: boolean }) {
  return (
    <p className="mt-2 text-sm">
      <span className="text-ink-2">Order: </span>
      {byScore ? (
        <>
          <Link
            href="/candidates"
            className="underline decoration-rule underline-offset-2 hover:text-ink"
          >
            by date
          </Link>
          <span className="text-ink-2"> · </span>
          <span className="font-semibold">by score</span>
        </>
      ) : (
        <>
          <span className="font-semibold">by date</span>
          <span className="text-ink-2"> · </span>
          <Link
            href="/candidates?sort=score"
            className="underline decoration-rule underline-offset-2 hover:text-ink"
          >
            by score
          </Link>
        </>
      )}
      <span className="ml-3 text-xs text-ink-2">
        Deciding moves focus to the next undecided card — the queue clears
        from the keyboard.
      </span>
    </p>
  );
}

function RunSummary({ list }: { list: CandidateList }) {
  if (!list.run) return null;
  const far = list.run.params["FAR_KM"];
  return (
    <p className="mt-6 text-xs text-ink-2">
      Detection run {list.run.id} · {list.candidates.length} candidates ·{" "}
      {list.run.outliers_dropped} outliers dropped
      {typeof far === "number" && <> · FAR {far} km</>}
    </p>
  );
}

function CandidateGallery({
  candidates,
  legsById,
}: {
  candidates: Candidate[];
  legsById: Map<number, Leg[]>;
}) {
  return (
    <ul className="mt-4 grid list-none grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-6">
      {/* key must be stable across re-renders so React can reconcile cards;
          the candidate id is exactly that within one run. */}
      {candidates.map((c) => (
        <li key={c.id}>
          <CandidateCard candidate={c} legs={legsById.get(c.id)} />
        </li>
      ))}
    </ul>
  );
}

// One triage card (mockup T3): shape leads, figures compress to a mono
// line, the decision lives in the card. A single rule frames it — the
// double-rule plate frame stays reserved for confirmed covers on
// /adventures, so an undecided queue never wears the atlas's clothes.
function CandidateCard({
  candidate: c,
  legs,
}: {
  candidate: Candidate;
  legs: Leg[] | undefined;
}) {
  return (
    // data-candidate-card + data-undecided drive the sweep's focus advance
    // (DecideCell walks to the next undecided card after a decision).
    <div
      data-candidate-card={c.id}
      data-undecided={c.decision ? undefined : "true"}
      className="border border-rule bg-paper"
    >
      <div className="border-b border-rule">
        {legs && legs.length > 0 ? (
          <RouteThumb legs={legs} className="block h-auto w-full" />
        ) : (
          <p className="flex aspect-[28/15] items-center justify-center text-xs text-ink-2">
            route shape unavailable
          </p>
        )}
      </div>
      <div className="px-3.5 py-3">
        <p className="flex items-baseline justify-between gap-2 font-mono text-sm">
          <span className="flex items-baseline gap-2.5">
            <TriageSelect candidateId={c.id} />
            <Link
              href={`/adventure/${c.id}`}
              className="font-medium underline decoration-rule underline-offset-2 hover:text-ink"
            >
              {c.span_start.slice(0, 10)}
            </Link>
            {/* The span_start substring keeps the civil date the traveller
                experienced; new Date(...) would shift it into the viewer's
                timezone. */}
            {c.start_truncated && (
              <span
                className="ml-1 text-flag"
                title="Began before the imported window; the real journey is longer."
              >
                ◂
              </span>
            )}
            {c.end_truncated && (
              <span
                className="ml-1 text-flag"
                title="Still in progress at the end of the imported window."
              >
                ▸
              </span>
            )}
          </span>
          <span
            className="text-xs text-ink-2"
            title="Confidence 0–100; the breakdown appears when confirming."
          >
            {/* Absent means "not scored" (a pre-scoring run) — different
                information than a low score. */}
            score {c.score !== undefined ? c.score.toFixed(0) : "—"}
          </span>
        </p>
        <p className="mb-2.5 mt-1 font-mono text-xs text-ink-2">
          {c.days} day{c.days === 1 ? "" : "s"} · {c.dest_km} km away ·{" "}
          {c.track_km} km track · {c.stops} stop{c.stops === 1 ? "" : "s"}
          {c.repeat > 0 && <> · {c.repeat + 1}× visited</>}
        </p>
        <DecideCell
          candidateId={c.id}
          decision={c.decision}
          score={c.score}
          scoreBreakdown={c.score_breakdown}
        />
      </div>
    </div>
  );
}
