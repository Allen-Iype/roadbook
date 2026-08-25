"use client";

// The photo-batch upload island (phase 11 BRIEF §4C): pick photos from the
// OS picker (`multiple` — on a phone this opens the photo library directly,
// no export step), XHR them to the streaming proxy, then poll for the
// detection that follows. The structure mirrors upload-import.tsx; the
// differences are the batch shape and the response: the import itself
// completes before the 202 returns, carrying a per-file verdict for every
// photo — there is no all-or-nothing rule on a camera roll, so the island's
// job after upload is to say honestly what each file contributed and then
// wait out detection.
//
// The accept list names JPEG, HEIC, and the Takeout JSON sidecars. HEIC is
// deliberate (BRIEF §1.3): position metadata is extractable even though the
// pixels are not decodable server-side — and if the phone converts to JPEG
// at the picker instead, that arrives just as well.

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

type ImportRow = {
  id: number;
  status: "running" | "completed" | "failed";
  error?: string;
  detect_status?: "running" | "completed" | "failed";
  inserted?: number;
  raw_positions?: number;
};

type FileVerdict = {
  file: string;
  status:
    | "fix"
    | "no_position"
    | "no_time"
    | "sidecar_paired"
    | "sidecar_unpaired"
    | "unsupported";
  message?: string;
};

type Phase =
  | { k: "idle" }
  | { k: "uploading"; pct: number | null; count: number }
  | { k: "detecting"; id: number }
  | { k: "done"; id: number; inserted: number | null }
  | { k: "failed"; message?: string }
  | { k: "error"; message: string };

const POLL_MS = 1500;

export function PhotoUploadImport() {
  const [phase, setPhase] = useState<Phase>({ k: "idle" });
  const [verdicts, setVerdicts] = useState<FileVerdict[] | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  const busy = phase.k === "uploading" || phase.k === "detecting";

  function start(files: FileList) {
    setVerdicts(null);
    setPhase({ k: "uploading", pct: null, count: files.length });

    const fd = new FormData();
    for (const f of files) fd.append("file", f, f.name);

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/imports/photos");
    xhr.responseType = "text";
    xhr.upload.onprogress = (e) => {
      setPhase({
        k: "uploading",
        pct: e.lengthComputable ? Math.round((100 * e.loaded) / e.total) : null,
        count: files.length,
      });
    };
    xhr.onerror = () =>
      setPhase({
        k: "error",
        message:
          "The upload didn't complete — nothing was recorded. Check the connection, keep the screen on, and try again.",
      });
    xhr.onabort = xhr.onerror;
    xhr.onload = () => {
      type Answer = {
        import?: ImportRow;
        files?: FileVerdict[];
        error?: string;
      };
      let body: Answer | null = null;
      try {
        body = JSON.parse(xhr.responseText) as Answer;
      } catch {
        // fall through to the generic branch below
      }
      if (xhr.status === 202 && body?.import) {
        // The photo path imports before responding — what remains async is
        // only the detection pass.
        setVerdicts(body.files ?? null);
        setPhase({ k: "detecting", id: body.import.id });
      } else {
        setPhase({
          k: "error",
          message: body?.error ?? `The upload failed (HTTP ${xhr.status}).`,
        });
      }
    };
    xhr.send(fd);
  }

  useEffect(() => {
    if (phase.k !== "detecting") return;
    const id = phase.id;
    const t = setInterval(async () => {
      let row: ImportRow;
      try {
        const res = await fetch(`/api/imports/${id}`, { cache: "no-store" });
        row = (await res.json()) as ImportRow;
      } catch {
        return; // transient — keep polling; the row is durable
      }
      if (row.detect_status === "completed") {
        setPhase({ k: "done", id, inserted: row.inserted ?? null });
        router.refresh();
      } else if (row.detect_status === "failed") {
        setPhase({
          k: "failed",
          message:
            "Your photos imported fine, but the detection step failed — the server log has details.",
        });
        router.refresh();
      }
    }, POLL_MS);
    return () => clearInterval(t);
  }, [phase, router]);

  return (
    <section className="mt-6 border border-rule bg-land p-4">
      <h2 className="font-display text-lg font-semibold">Add photos</h2>
      <p className="mt-1 text-sm text-ink-2">
        Geotagged photos become position evidence: pick the trips (and some
        everyday photos, so home can be worked out), and detection runs on
        what they show. JPEG and HEIC both work; Google Photos Takeout JSON
        sidecars fill in photos whose location was stripped. The photos
        themselves are not kept — only their positions, times, and small
        thumbnails. Keep the screen on until the upload finishes.
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-3">
        <input
          ref={inputRef}
          type="file"
          multiple
          accept="image/jpeg,image/heic,image/heif,.jpg,.jpeg,.heic,.heif,.json"
          disabled={busy}
          className="hidden"
          onChange={(e) => {
            if (e.target.files && e.target.files.length > 0) {
              start(e.target.files);
            }
            e.target.value = "";
          }}
        />
        <button
          type="button"
          disabled={busy}
          onClick={() => inputRef.current?.click()}
          className="border border-ink px-4 py-2 text-sm font-semibold hover:bg-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          {phase.k === "idle" ? "Choose photos…" : "Choose more photos…"}
        </button>
      </div>

      <div aria-live="polite" className="mt-3 text-sm">
        <PhotoPhaseView phase={phase} />
      </div>

      {verdicts && <VerdictSummary verdicts={verdicts} />}
    </section>
  );
}

function PhotoPhaseView({ phase }: { phase: Phase }) {
  switch (phase.k) {
    case "idle":
      return null;
    case "uploading":
      return (
        <div>
          <p>
            Uploading {phase.count} file{phase.count === 1 ? "" : "s"}
            {phase.pct !== null ? ` — ${phase.pct}%` : "…"}{" "}
            <span className="text-ink-2">(keep this page open)</span>
          </p>
          <progress
            value={phase.pct ?? undefined}
            max={100}
            className="mt-1 h-2 w-full max-w-md"
          />
        </div>
      );
    case "detecting":
      return <p>Photos imported. Looking for adventures…</p>;
    case "done":
      return (
        <div>
          <p>
            Done.
            {phase.inserted !== null && phase.inserted === 0 ? (
              <span className="text-ink-2">
                {" "}
                These photos held nothing new — every position was already
                imported.
              </span>
            ) : null}
          </p>
          <a
            href="/candidates"
            className="mt-1 inline-block font-semibold underline underline-offset-2"
          >
            See your candidates →
          </a>
        </div>
      );
    case "failed":
      return (
        <div className="max-w-prose">
          <p className="font-semibold text-red-700">The import failed</p>
          {phase.message && <p className="mt-1 text-ink-2">{phase.message}</p>}
        </div>
      );
    case "error":
      return (
        <p className="max-w-prose font-semibold text-red-700">{phase.message}</p>
      );
  }
}

// The per-file accounting (BRIEF §4C): a camera roll has no all-or-nothing
// rule, so the summary states what each class of file contributed — counts
// for the good outcomes, names and reasons for the files that carried no
// usable evidence.
function VerdictSummary({ verdicts }: { verdicts: FileVerdict[] }) {
  const count = (s: FileVerdict["status"]) =>
    verdicts.filter((v) => v.status === s).length;
  const fixes = count("fix");
  const paired = count("sidecar_paired");
  const unusable = verdicts.filter(
    (v) =>
      v.status === "no_position" ||
      v.status === "no_time" ||
      v.status === "sidecar_unpaired" ||
      v.status === "unsupported",
  );

  return (
    <div className="mt-3 border-t border-rule pt-3 text-sm">
      <p>
        {fixes} of {verdicts.length} file{verdicts.length === 1 ? "" : "s"}{" "}
        carried a usable position
        {paired > 0
          ? ` (${paired} sidecar${paired === 1 ? "" : "s"} matched)`
          : ""}
        .
      </p>
      {unusable.length > 0 && (
        <details className="mt-1">
          <summary className="cursor-pointer text-ink-2">
            {unusable.length} file{unusable.length === 1 ? "" : "s"} added
            nothing — why
          </summary>
          <ul className="mt-1 space-y-1">
            {unusable.map((v) => (
              <li key={v.file} className="text-ink-2">
                <span className="font-mono">{v.file}</span> —{" "}
                {v.message ?? reasonFor(v.status)}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}

function reasonFor(status: FileVerdict["status"]): string {
  switch (status) {
    case "no_position":
      return "no location in the photo (messenger apps strip it; use camera originals or Takeout sidecars)";
    case "no_time":
      return "a location but no readable capture time";
    case "sidecar_unpaired":
      return "a metadata file naming no photo in this batch — include the photo it describes";
    default:
      return "not a supported photo format";
  }
}
