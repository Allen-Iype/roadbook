import Link from "next/link";

import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

// Server component, same shape as the home page: runs per request on the
// server, calls the Go API, ships HTML. Nothing here is interactive, so no
// client island is needed — a failed import is a row to read, not a thing to
// click (phase 5 BRIEF §3B).
export const dynamic = "force-dynamic";

type Import = components["schemas"]["Import"];

export default async function ImportsPage() {
  const { data, error } = await api.GET("/imports");

  if (error || !data) {
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
      {data.imports.length === 0 ? (
        <p className="mt-6 text-neutral-400">
          No imports yet. Run{" "}
          <code className="font-mono">roadbook import -src Timeline.json</code>.
        </p>
      ) : (
        <ImportTable imports={data.imports} />
      )}
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <main className="mx-auto w-full max-w-5xl px-6 py-10">
      <h1 className="text-2xl font-semibold">Imports</h1>
      <p className="mt-1 text-sm text-neutral-400">
        Every import attempt, newest first — including the failed ones, with
        what the file turned out to be.{" "}
        <Link
          href="/"
          className="underline decoration-neutral-700 underline-offset-2 hover:text-neutral-100"
        >
          Back to candidates
        </Link>
      </p>
      {children}
    </main>
  );
}

function ImportTable({ imports }: { imports: Import[] }) {
  return (
    <div className="mt-6 overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-neutral-800 text-left text-xs uppercase tracking-wide text-neutral-500">
            <th className="py-2 pr-4">When</th>
            <th className="py-2 pr-4">Source</th>
            <th className="py-2 pr-4">Format</th>
            <th className="py-2 pr-4">Window</th>
            <th className="py-2 pr-4 text-right">Visits</th>
            <th className="py-2 pr-4 text-right">Activities</th>
            <th className="py-2 pr-4 text-right">Points</th>
            <th className="py-2 pr-4 text-right">Raw</th>
            <th className="py-2 pr-4 text-right">Skipped</th>
            <th className="py-2 pr-4">Status</th>
          </tr>
        </thead>
        <tbody>
          {imports.map((imp) => (
            <tr key={imp.id} className="border-b border-neutral-900 align-top">
              <td className="py-2 pr-4 font-mono whitespace-nowrap">
                {imp.imported_at.slice(0, 10)}
              </td>
              <td className="py-2 pr-4">{imp.source_label}</td>
              <td className="py-2 pr-4 font-mono text-neutral-400">
                {/* Absent means the input was never recognised (not JSON,
                    unreadable) — different information than a known format. */}
                {imp.detected_format ?? "—"}
              </td>
              <td className="py-2 pr-4 font-mono whitespace-nowrap text-neutral-400">
                {windowLabel(imp)}
              </td>
              <td className="py-2 pr-4 text-right">{imp.visits}</td>
              <td className="py-2 pr-4 text-right">{imp.activities}</td>
              <td className="py-2 pr-4 text-right">{imp.points}</td>
              <td className="py-2 pr-4 text-right">{imp.raw_positions}</td>
              <td className="py-2 pr-4 text-right">{imp.skipped}</td>
              <td className="py-2 pr-4">
                <StatusCell imp={imp} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function StatusCell({ imp }: { imp: Import }) {
  if (imp.status === "failed") {
    return (
      <div>
        <span className="text-red-400">failed</span>
        {imp.error && (
          <p className="mt-1 max-w-md text-xs text-neutral-400">{imp.error}</p>
        )}
      </div>
    );
  }
  if (imp.status === "running") {
    // Either genuinely in progress, or a crash left the row unfinalised —
    // both are worth seeing as-is rather than dressing up.
    return <span className="text-amber-400">running</span>;
  }
  return <span className="text-neutral-400">completed</span>;
}

// Window bounds are dates the operator typed (-from/-to), so the date part is
// the meaning; slice instead of new Date() for the same offset-preserving
// reason as the candidate list.
function windowLabel(imp: Import): string {
  if (!imp.window_start && !imp.window_end) return "all";
  const d = (s?: string) => (s ? s.slice(0, 10) : "…");
  return `${d(imp.window_start)} → ${d(imp.window_end)}`;
}
