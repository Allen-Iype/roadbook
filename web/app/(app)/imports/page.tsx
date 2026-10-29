
import { api } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { SiteHeader } from "@/components/site-header";
import { INSTANCE_LABEL } from "@/lib/instance";
import { requireUser } from "@/lib/session";
import { PhotoUploadImport } from "@/components/photo-upload-import";
import { UploadImport } from "@/components/upload-import";
import { DeleteMyData } from "@/components/delete-my-data";

// Server component, same shape as the home page: runs per request on the
// server, calls the Go API, ships HTML. Nothing here is interactive, so no
// client island is needed — a failed import is a row to read, not a thing to
// click (phase 5 BRIEF §3B).
export const dynamic = "force-dynamic";

type Import = components["schemas"]["Import"];

export default async function ImportsPage() {
  // The auth gate (phase 13 CP3): UX only — Go 401s regardless. Mode
  // off returns pass-through and this page renders exactly as before.
  const session = await requireUser();
  const accountEmail =
    session?.mode === "oidc" ? (session.email ?? "") : undefined;

  const { data, error } = await api.GET("/imports");

  if (error || !data) {
    return (
      <Shell accountEmail={accountEmail}>
        <p className="mt-6 text-red-700">
          The Roadbook API is not reachable. Start it with{" "}
          <code className="font-mono">roadbook serve</code> and reload.
        </p>
      </Shell>
    );
  }

  return (
    <Shell accountEmail={accountEmail}>
      {/* The two doors side by side from lg (CP3 review): at full page
          width each panel's two-line intro sat in the left third of a
          wide frame. Half-width panels put the text at a natural measure
          with no cap; on a phone they stack as before. */}
      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        <UploadImport />
        <PhotoUploadImport />
      </div>
      {data.imports.length === 0 ? (
        <p className="mt-8 text-sm text-ink-2">
          Nothing added yet
          {/* The CLI route exists only where the operator is the user:
              a hosted account cannot run anything on the server. */}
          {session?.mode === "oidc" ? (
            "."
          ) : (
            <>
              {" "}— upload above, or run{" "}
              <code className="font-mono">roadbook import -src Timeline.json</code>.
            </>
          )}
        </p>
      ) : (
        <ImportTable imports={data.imports} />
      )}
      {/* Self-serve deletion (phase 13 CP5) lives where the data story
          lives: the page that shows what came in is the page that lets it
          all go. Present in both modes; empty instances show it too — an
          empty instance has nothing to lose, and the control says so. */}
      <DeleteMyData
        multiUser={session?.mode === "oidc"}
        hasData={data.imports.length > 0}
      />
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
    <main className="mx-auto w-full max-w-5xl px-4 py-8 sm:px-6">
      <SiteHeader active="imports" instanceLabel={INSTANCE_LABEL}
        accountEmail={accountEmail} />
      {/* Phase 16 CP3: the heading alone. The paragraph that sat here
          described the table and pointed at the export walkthrough; the
          table now carries its own heading, and the walkthrough link
          lives in the panel whose file it explains. */}
      <h1 className="mt-8 font-display text-2xl font-semibold">Imports</h1>
      {children}
    </main>
  );
}

// What the sniffer's label means in words (phase 16 CP3). The two formats
// that import are named for the person; a rejected file keeps the label the
// sniffer gave it ("pdf", "zip", "kml"), which already says what it was.
// Presentation only — the label itself travels unchanged from Go.
const FORMAT_LABELS: Record<string, string> = {
  "phone-timeline": "Timeline export",
  photos: "Photos",
};

function ImportTable({ imports }: { imports: Import[] }) {
  return (
    <section className="mt-10">
      <h2 className="font-display text-lg font-semibold">Added so far</h2>
      <div className="mt-3 overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          {/* The detail columns hide at phone width (small-screen pass,
              BRIEF §1.5): Added/Source/New/Status is the story a thumb
              needs — the status cell already carries any failure prose.
              Format joins from sm:, the full accounting from md:, and
              overflow-x-auto remains the belt.
              Phase 16 CP3: the headers are words, not column names —
              "Window" was the import's date bounds, "Points" the path
              points and "Raw" the recorded positions (RESEARCH §4). The
              figures are unchanged and still what `roadbook import`
              prints. */}
          <tr className="border-b border-ink/40 text-left text-xs uppercase tracking-wide text-ink-2">
            <th className="py-2 pr-4">Added</th>
            <th className="py-2 pr-4">Source</th>
            <th className="hidden py-2 pr-4 sm:table-cell">Format</th>
            <th className="hidden py-2 pr-4 md:table-cell">Dates</th>
            <th className="hidden py-2 pr-4 text-right md:table-cell">Visits</th>
            <th className="hidden py-2 pr-4 text-right md:table-cell">Activities</th>
            <th className="hidden py-2 pr-4 text-right md:table-cell">Path points</th>
            <th className="hidden py-2 pr-4 text-right md:table-cell">Positions</th>
            <th className="hidden py-2 pr-4 text-right md:table-cell">Skipped</th>
            <th className="py-2 pr-4 text-right">New</th>
            <th className="py-2 pr-4">Status</th>
          </tr>
        </thead>
        <tbody>
          {imports.map((imp) => (
            <tr key={imp.id} className="border-b border-rule align-top">
              <td className="py-2 pr-4 font-mono whitespace-nowrap">
                {imp.imported_at.slice(0, 10)}
              </td>
              <td className="py-2 pr-4">{imp.source_label}</td>
              <td className="hidden py-2 pr-4 text-ink-2 sm:table-cell">
                {/* Absent means the input was never recognised (not JSON,
                    unreadable) — different information than a known format. */}
                {imp.detected_format
                  ? (FORMAT_LABELS[imp.detected_format] ?? imp.detected_format)
                  : "—"}
              </td>
              <td className="hidden py-2 pr-4 font-mono whitespace-nowrap text-ink-2 md:table-cell">
                {windowLabel(imp)}
              </td>
              <td className="hidden py-2 pr-4 text-right md:table-cell">{imp.visits}</td>
              <td className="hidden py-2 pr-4 text-right md:table-cell">{imp.activities}</td>
              <td className="hidden py-2 pr-4 text-right md:table-cell">{imp.points}</td>
              <td className="hidden py-2 pr-4 text-right md:table-cell">{imp.raw_positions}</td>
              <td className="hidden py-2 pr-4 text-right md:table-cell">{imp.skipped}</td>
              <td className="py-2 pr-4 text-right">
                {/* How many observations were genuinely new. Absent on rows
                    from before the column existed — unknown, not zero. */}
                {imp.inserted ?? "—"}
              </td>
              <td className="py-2 pr-4">
                <StatusCell imp={imp} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      </div>
    </section>
  );
}

function StatusCell({ imp }: { imp: Import }) {
  if (imp.status === "failed") {
    return (
      <div>
        <span className="text-red-700">failed</span>
        {imp.error && (
          <p className="mt-1 max-w-md text-xs text-ink-2">{imp.error}</p>
        )}
      </div>
    );
  }
  if (imp.status === "running") {
    // Either genuinely in progress, or a crash left the row unfinalised —
    // both are worth seeing as-is rather than dressing up.
    return (
      <span>
        <span className="text-flag" aria-hidden>
          ⚑
        </span>{" "}
        running
      </span>
    );
  }
  // Completed — but the automatic detection that follows an upload import
  // has its own outcome, and a failure there must not hide (BRIEF §3D).
  if (imp.detect_status === "failed") {
    return (
      <div>
        <span className="text-ink-2">completed</span>
        <p className="mt-1 max-w-md text-xs text-red-700">
          detection failed — the server log has details
        </p>
      </div>
    );
  }
  return <span className="text-ink-2">completed</span>;
}

// Window bounds are dates the operator typed (-from/-to), so the date part is
// the meaning; slice instead of new Date() for the same offset-preserving
// reason as the candidate list.
function windowLabel(imp: Import): string {
  if (!imp.window_start && !imp.window_end) return "everything";
  const d = (s?: string) => (s ? s.slice(0, 10) : "…");
  return `${d(imp.window_start)} → ${d(imp.window_end)}`;
}
