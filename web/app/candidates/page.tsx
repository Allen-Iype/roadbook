import Link from "next/link";

import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { DecideCell } from "@/app/decide-cell";
import { SiteHeader } from "@/components/site-header";

// The triage workbench, moved here from / (phase 6: the home page becomes the
// life map; triage demotes to an entry point). Server component: runs on the
// server per request, calls the Go API, ships HTML. Interactivity is the
// per-row DecideCell island, unchanged in behaviour since phase 1.

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
        <p className="mt-6 text-red-700">
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
        // The zero-candidates ending is designed copy (BRIEF §3E): an
        // import that completes but detects nothing must say so in words —
        // this is where the upload island's "see your candidates" link
        // lands when detection found none.
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
        <CandidateTable candidates={data.candidates} />
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
    <main className="mx-auto w-full max-w-5xl px-4 py-8 sm:px-6">
      <SiteHeader />
      <h1 className="mt-8 font-display text-2xl font-semibold">Candidates</h1>
      <p className="mt-1 text-sm text-ink-2">
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
    <p className="mt-6 text-xs text-ink-2">
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
          {/* At phone width the row keeps Start/Days/Distance/Status — the
              decide cell sits under the thumb without a horizontal scroll
              (small-screen pass, BRIEF §1.5). Track, stops, repeat visits,
              and score are triage detail for wider screens; the score's
              breakdown still appears in the confirm cell everywhere. */}
          <tr className="border-b border-ink/40 text-left text-xs uppercase tracking-wide text-ink-2">
            <th className="py-2 pr-4">Start</th>
            <th className="py-2 pr-4 text-right">Days</th>
            <th className="py-2 pr-4 text-right">Distance</th>
            <th className="hidden py-2 pr-4 text-right sm:table-cell">Track</th>
            <th className="hidden py-2 pr-4 text-right sm:table-cell">Stops</th>
            <th className="hidden py-2 pr-4 text-right sm:table-cell">Visits</th>
            <th className="hidden py-2 pr-4 text-right sm:table-cell">Score</th>
            <th className="py-2 pr-4">Status</th>
          </tr>
        </thead>
        <tbody>
          {/* key must be stable across re-renders so React can reconcile rows;
              the candidate row id is exactly that within one run. */}
          {candidates.map((c) => (
            <tr key={c.id} className="border-b border-rule">
              <td className="whitespace-nowrap py-2 pr-4 font-mono">
                {/* Client-side navigation to the adventure detail page. */}
                <Link
                  href={`/adventure/${c.id}`}
                  className="underline decoration-rule underline-offset-2 hover:text-ink"
                >
                  {startDate(c)}
                </Link>
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
              </td>
              <td className="py-2 pr-4 text-right">{c.days}</td>
              <td className="py-2 pr-4 text-right">{c.dest_km} km</td>
              <td className="hidden py-2 pr-4 text-right sm:table-cell">{c.track_km} km</td>
              <td className="hidden py-2 pr-4 text-right sm:table-cell">{c.stops}</td>
              <td className="hidden py-2 pr-4 text-right sm:table-cell">
                {c.repeat > 0 ? `${c.repeat + 1}×` : "first"}
              </td>
              <td
                className="hidden py-2 pr-4 text-right font-mono sm:table-cell"
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
