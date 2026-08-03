import Link from "next/link";

import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { DecideCell } from "@/app/decide-cell";

// Server component (the default in the App Router): this async function runs
// on the server per request, calls the Go API, and ships rendered HTML — no
// fetch code, no API URL, and no data logic reach the browser. Interactivity
// (confirm/dismiss) arrives in the next checkpoint as a small client-component
// island; everything that can stay on the server stays there.

// Without this, Next's default ('auto') would prerender this page once at
// build time and freeze the candidate list into the build output. The list
// must reflect the database on every request.
export const dynamic = "force-dynamic";

// Types come from the generated schema — inferred from the contract, never
// hand-written. `components["schemas"][...]` is the generated mirror of
// api/openapi.yaml's components/schemas.
type Candidate = components["schemas"]["Candidate"];
type CandidateList = components["schemas"]["CandidateList"];

export default async function CandidatesPage() {
  // openapi-fetch returns { data, error } — exactly one is set, and both are
  // typed from the contract's response schemas.
  const { data, error } = await api.GET("/candidates");

  if (error || !data) {
    // Expected failure (Go API down, DB unreachable): render a factual state,
    // don't throw. error.tsx is reserved for unexpected exceptions.
    return (
      <Shell>
        <p className="mt-6 text-red-400">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </Shell>
    );
  }

  return (
    <Shell>
      <RunSummary list={data} />
      {data.candidates.length === 0 ? (
        <p className="mt-6 text-neutral-400">
          No candidates. Import an export and run{" "}
          <code className="font-mono">roadbook detect -from-db</code>.
        </p>
      ) : (
        <CandidateTable candidates={data.candidates} />
      )}
      {data.orphaned_decisions.length > 0 && (
        <p className="mt-4 text-sm text-amber-400">
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
    <main className="mx-auto w-full max-w-5xl px-6 py-10">
      <h1 className="text-2xl font-semibold">Roadbook</h1>
      <p className="mt-1 text-sm text-neutral-400">
        Adventure candidates detected from your timeline. Confirm the real
        ones; dismiss the rest.
      </p>
      {children}
    </main>
  );
}

function RunSummary({ list }: { list: CandidateList }) {
  if (!list.run) return null;
  const far = list.run.params["FAR_KM"];
  return (
    <p className="mt-6 text-xs text-neutral-500">
      Detection run {list.run.id} · {list.candidates.length} candidates ·{" "}
      {list.run.outliers_dropped} outliers dropped
      {typeof far === "number" && <> · FAR {far} km</>}
    </p>
  );
}

function CandidateTable({ candidates }: { candidates: Candidate[] }) {
  return (
    <div className="mt-2 overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-neutral-800 text-left text-xs uppercase tracking-wide text-neutral-500">
            <th className="py-2 pr-4">Start</th>
            <th className="py-2 pr-4 text-right">Days</th>
            <th className="py-2 pr-4 text-right">Distance</th>
            <th className="py-2 pr-4 text-right">Track</th>
            <th className="py-2 pr-4 text-right">Stops</th>
            <th className="py-2 pr-4 text-right">Visits</th>
            <th className="py-2 pr-4 text-right">Score</th>
            <th className="py-2 pr-4">Status</th>
          </tr>
        </thead>
        <tbody>
          {/* key must be stable across re-renders so React can reconcile rows;
              the candidate row id is exactly that within one run. */}
          {candidates.map((c) => (
            <tr key={c.id} className="border-b border-neutral-900">
              <td className="py-2 pr-4 font-mono">
                {/* Client-side navigation to the adventure detail page. */}
                <Link
                  href={`/adventure/${c.id}`}
                  className="underline decoration-neutral-700 underline-offset-2 hover:text-neutral-100"
                >
                  {startDate(c)}
                </Link>
                {c.start_truncated && (
                  <span
                    className="ml-1 text-amber-400"
                    title="Began before the imported window; the real journey is longer."
                  >
                    ◂
                  </span>
                )}
                {c.end_truncated && (
                  <span
                    className="ml-1 text-amber-400"
                    title="Still in progress at the end of the imported window."
                  >
                    ▸
                  </span>
                )}
              </td>
              <td className="py-2 pr-4 text-right">{c.days}</td>
              <td className="py-2 pr-4 text-right">{c.dest_km} km</td>
              <td className="py-2 pr-4 text-right">{c.track_km} km</td>
              <td className="py-2 pr-4 text-right">{c.stops}</td>
              <td className="py-2 pr-4 text-right">
                {c.repeat > 0 ? `${c.repeat + 1}×` : "first"}
              </td>
              <td
                className="py-2 pr-4 text-right font-mono"
                title="Confidence 0–100; the breakdown appears when confirming."
              >
                {/* Absent means "not scored" (a pre-scoring run), which is
                    different information than a low score — show a dash. */}
                {c.score !== undefined ? c.score.toFixed(0) : "—"}
              </td>
              <td className="py-2 pr-4">
                {/* The row is server HTML; only this cell hydrates. */}
                <DecideCell
                  candidateId={c.id}
                  decision={c.decision}
                  score={c.score}
                  scoreBreakdown={c.score_breakdown}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// The span_start string carries the journey's own UTC offset (the Go side
// preserves it). Taking the date as a substring keeps the civil date the
// traveller experienced; `new Date(...)` would shift it into the viewer's
// timezone — off by one for evening departures viewed from the west.
function startDate(c: Candidate): string {
  return c.span_start.slice(0, 10);
}

