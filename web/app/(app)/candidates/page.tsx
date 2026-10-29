import Link from "next/link";

import { api } from "@/lib/api/client";
import { INSTANCE_LABEL } from "@/lib/instance";
import { requireUser } from "@/lib/session";
import type { components } from "@/lib/api/schema";
import { DecideCell } from "@/app/decide-cell";
import { RouteThumb } from "@/components/route-thumb";
import { SiteHeader } from "@/components/site-header";
import { Term } from "@/components/term";
import { TriageBar } from "@/components/triage-bar";
import { TriageSelect } from "@/components/triage-select";
import { TriageSelectAll } from "@/components/triage-select-all";

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
  // The auth gate (phase 13 CP3): UX only — Go 401s regardless. Mode
  // off returns pass-through and this page renders exactly as before.
  const session = await requireUser();
  const accountEmail =
    session?.mode === "oidc" ? (session.email ?? "") : undefined;

  // ?sort=score is the sweep order (phase 11 §6.1): score-descending, built
  // for clearing a long queue in one sitting. Default stays chronological.
  // A search param rather than client state: the order is part of the URL a
  // person can share or reload, and the server sorts before any HTML ships.
  const { sort } = await searchParams;
  const byScore = sort === "score";
  const res = await api.GET("/candidates").catch(() => null);

  if (!res?.data) {
    return (
      <Shell accountEmail={accountEmail}>
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
    <Shell accountEmail={accountEmail}>
      {data.candidates.length > 0 && (
        <>
          <RunSummary list={data} />
          <SortToggle byScore={byScore} />
          <p className="mt-1.5">
            <TriageSelectAll
              undecidedIds={data.candidates
                .filter((c) => !c.decision)
                .map((c) => c.id)}
            />
          </p>
        </>
      )}
      {data.candidates.length === 0 ? (
        data.run ? (
          <div className="mt-6 max-w-[58ch]">
            <p className="font-semibold">
              Roadbook looked and found no trips.
            </p>
            <p className="mt-2 text-ink-2">
              Your data imported fine. A trip is time spent well away from
              home with a real stay somewhere; short outings and
              pass-throughs don&apos;t count.
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

function Shell({
  children,
  accountEmail,
}: {
  children: React.ReactNode;
  accountEmail?: string;
}) {
  return (
    <main className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
      <SiteHeader active="candidates" instanceLabel={INSTANCE_LABEL}
        accountEmail={accountEmail} />
      <h1 className="mt-8 font-display text-2xl font-semibold">Candidates</h1>
      {/* Phase 16 CP3: the page says what it asks (RESEARCH §4), and the
          one term it needs is taught where it is met. */}
      {/* text-balance, not text-pretty: pretty only rescues a one-word
          last line, and this intro's natural break leaves two ("the
          rest."); balance evens the lines, which lands the break on the
          sentence boundary at laptop width. */}
      <p className="mt-1 max-w-[72ch] text-balance text-sm text-ink-2">
        Each <Term id="candidate">candidate</Term> is a trip Roadbook found
        in your record. Confirm the adventures; dismiss the rest.
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
      {/* A keyboard fact, so it is said only where a keyboard is likely. */}
      <span className="ml-3 hidden text-xs text-ink-2 sm:inline">
        After each decision, focus moves to the next undecided card.
      </span>
    </p>
  );
}

// The run's own figures, folded (phase 16 BRIEF §4 CP3). They are what
// `roadbook detect` prints — the run id, the outliers it dropped, the FAR
// threshold — and none of them helps a person decide whether a trip was
// real (RESEARCH §4), so they sit behind a closed summary that carries
// the one figure a person does want: how many there are. The words inside
// are plain; the figures are the CLI's, unchanged (invariant 13).
function RunSummary({ list }: { list: CandidateList }) {
  if (!list.run) return null;
  const n = list.candidates.length;
  const far = list.run.params["FAR_KM"];
  const dropped = list.run.outliers_dropped;
  return (
    <details className="group mt-5 max-w-[72ch] text-xs text-ink-2">
      <summary className="-my-3.5 inline-flex cursor-pointer list-none items-baseline gap-2 py-3.5 marker:hidden">
        <span aria-hidden className="font-mono transition-transform group-open:rotate-90">
          ▸
        </span>
        <span className="underline decoration-rule underline-offset-2 group-hover:text-ink">
          How these {n} were found
        </span>
      </summary>
      <p className="mt-2 pl-5 leading-relaxed">
        {typeof far === "number" ? (
          <>
            Time spent more than{" "}
            <span className="font-mono">{far} km</span> from home, with
            somewhere you stayed, counts as a trip.{" "}
          </>
        ) : (
          <>Time spent far from home, with somewhere you stayed, counts as a trip.{" "}</>
        )}
        {dropped > 0 ? (
          <>
            <span className="font-mono">{dropped}</span> recorded position
            {dropped === 1 ? "" : "s"} too fast to be real{" "}
            {dropped === 1 ? "was" : "were"} left out first.{" "}
          </>
        ) : null}
        <span className="font-mono">Run {list.run.id}</span>.
      </p>
    </details>
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
          {/* data-score is the e2e sort check's hook (it used to read a
              hover title, which a phone never showed — the Term now
              teaches the figure, phase 16 CP3). */}
          <span data-score className="text-xs text-ink-2">
            <Term id="score">score</Term>{" "}
            {/* Absent means "not scored" (a pre-scoring run) — different
                information than a low score. */}
            {c.score !== undefined ? c.score.toFixed(0) : "—"}
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
