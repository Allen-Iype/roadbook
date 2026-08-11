"use client";

// The upload island (phase 7 BRIEF §§1.1–1.3): file input → XHR to the
// streaming proxy → poll the imports row → land on candidates. A client
// component because everything here is interaction; the browser still never
// talks to the Go API — both the upload and the polling go through Next
// route handlers.
//
// XMLHttpRequest, not fetch, for the upload itself: fetch cannot report
// upload progress, and a multi-hundred-MB file on hotel Wi-Fi with no
// progress bar reads as a hang. This is the one legitimate XHR left.
//
// The state machine mirrors the two async facts the brief names (§3D): the
// import completes, then detection completes a beat later. "Done" is only
// declared when detect_status resolves — the front door must not say "see
// your candidates" while the candidate list is still being written.

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { redirectFor } from "@/lib/rejection-anchors";

type ImportRow = {
  id: number;
  status: "running" | "completed" | "failed";
  error?: string;
  detected_format?: string;
  detect_status?: "running" | "completed" | "failed";
  inserted?: number;
  visits?: number;
  activities?: number;
  points?: number;
  raw_positions?: number;
};

type Phase =
  | { k: "idle" }
  | { k: "uploading"; pct: number | null }
  | { k: "importing"; id: number }
  | { k: "detecting"; id: number }
  | { k: "done"; id: number; inserted: number | null }
  | { k: "rejected"; message: string; format?: string }
  | { k: "failed"; message?: string; format?: string }
  | { k: "error"; message: string };

const POLL_MS = 1500;

export function UploadImport() {
  const [phase, setPhase] = useState<Phase>({ k: "idle" });
  const [fileName, setFileName] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  const busy =
    phase.k === "uploading" || phase.k === "importing" || phase.k === "detecting";

  function start(file: File) {
    setFileName(file.name);
    setPhase({ k: "uploading", pct: null });

    const fd = new FormData();
    fd.append("file", file, file.name);

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/imports");
    xhr.responseType = "text";
    xhr.upload.onprogress = (e) => {
      setPhase({
        k: "uploading",
        pct: e.lengthComputable ? Math.round((100 * e.loaded) / e.total) : null,
      });
    };
    // A dead connection, a suspended tab, a killed in-app browser: the
    // all-or-nothing rule (BRIEF §1.2) means nothing was recorded, and
    // saying so is what makes "try again" safe advice.
    xhr.onerror = () =>
      setPhase({
        k: "error",
        message:
          "The upload didn't complete — nothing was recorded. Check the connection and try again.",
      });
    xhr.onabort = xhr.onerror;
    xhr.onload = () => {
      type UploadAnswer = Partial<ImportRow> & {
        error?: string;
        detected_format?: string;
      };
      let body: UploadAnswer | null = null;
      try {
        body = JSON.parse(xhr.responseText) as UploadAnswer;
      } catch {
        // fall through to the generic branch below
      }
      if (xhr.status === 202 && body && typeof body.id === "number") {
        setPhase({ k: "importing", id: body.id });
      } else if (xhr.status === 400) {
        setPhase({
          k: "rejected",
          message: body?.error ?? "This file was not recognised.",
          format: body?.detected_format,
        });
      } else {
        // 409 (an import is already running), 413 (too large), 5xx — the
        // API's message is the useful part; show it as given.
        setPhase({
          k: "error",
          message: body?.error ?? `The upload failed (HTTP ${xhr.status}).`,
        });
      }
    };
    xhr.send(fd);
  }

  // The polling loop (BRIEF §1.2): ask the row every couple of seconds
  // while an import or its auto-detect runs. No sockets — the imports row
  // is the only status channel, and /imports remains the durable fallback
  // if this tab closes.
  useEffect(() => {
    if (phase.k !== "importing" && phase.k !== "detecting") return;
    const id = phase.id;
    const t = setInterval(async () => {
      let row: ImportRow;
      try {
        const res = await fetch(`/api/imports/${id}`, { cache: "no-store" });
        row = (await res.json()) as ImportRow;
      } catch {
        return; // transient — keep polling; the row is durable
      }
      if (row.status === "failed") {
        setPhase({ k: "failed", message: row.error, format: row.detected_format });
        router.refresh();
      } else if (row.status === "completed") {
        if (row.detect_status === "completed") {
          setPhase({ k: "done", id, inserted: row.inserted ?? null });
          router.refresh();
        } else if (row.detect_status === "failed") {
          setPhase({
            k: "failed",
            message:
              "Your data imported fine, but the detection step failed — the server log has details.",
          });
          router.refresh();
        } else {
          // The beat between "imported" and "detected" (BRIEF §3D).
          setPhase({ k: "detecting", id });
        }
      }
    }, POLL_MS);
    return () => clearInterval(t);
  }, [phase, router]);

  return (
    <section className="mt-6 border border-rule bg-land p-4">
      <h2 className="font-display text-lg font-semibold">
        Add a Timeline export
      </h2>
      <p className="mt-1 text-sm text-ink-2">
        Upload the file exported from Google Maps Timeline on your phone. It
        imports here and detection runs automatically; your export stays
        stored on this instance. Large files take a while — use Wi-Fi and
        keep this page open until the upload finishes.
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-3">
        <input
          ref={inputRef}
          type="file"
          accept=".json,application/json"
          disabled={busy}
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) start(f);
            // Allow re-selecting the same file after a failure.
            e.target.value = "";
          }}
        />
        <button
          type="button"
          disabled={busy}
          onClick={() => inputRef.current?.click()}
          className="border border-ink px-4 py-2 text-sm font-semibold hover:bg-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          {phase.k === "idle" ? "Choose file…" : "Choose another file…"}
        </button>
        {fileName && (
          <span className="font-mono text-sm text-ink-2">{fileName}</span>
        )}
      </div>

      {/* aria-live: the phases below are exactly what a screen reader
          should hear as they happen. */}
      <div aria-live="polite" className="mt-3 text-sm">
        <PhaseView phase={phase} />
      </div>
    </section>
  );
}

function PhaseView({ phase }: { phase: Phase }) {
  switch (phase.k) {
    case "idle":
      return null;
    case "uploading":
      return (
        <div>
          <p>
            Uploading{phase.pct !== null ? ` — ${phase.pct}%` : "…"}{" "}
            <span className="text-ink-2">(keep this page open)</span>
          </p>
          <progress
            value={phase.pct ?? undefined}
            max={100}
            className="mt-1 h-2 w-full max-w-md"
          />
        </div>
      );
    case "importing":
      return <p>Uploaded. Importing your observations…</p>;
    case "detecting":
      return <p>Imported. Looking for adventures…</p>;
    case "done":
      return (
        <div>
          <p>
            Done.
            {phase.inserted !== null && phase.inserted === 0 ? (
              <span className="text-ink-2">
                {" "}
                This file held nothing new — every observation was already
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
    case "rejected":
      return (
        <div className="max-w-prose">
          <p className="font-semibold text-red-700">
            That file isn&apos;t a Timeline export
            {phase.format ? (
              <span className="font-mono text-ink-2"> ({phase.format})</span>
            ) : null}
          </p>
          <p className="mt-1 text-ink-2">{phase.message}</p>
          <p className="mt-1 text-ink-2">Nothing was stored — try the right file.</p>
          <RedirectionLink format={phase.format} />
        </div>
      );
    case "failed":
      return (
        <div className="max-w-prose">
          <p className="font-semibold text-red-700">The import failed</p>
          {phase.message && <p className="mt-1 text-ink-2">{phase.message}</p>}
          <p className="mt-1 text-ink-2">
            The attempt is recorded on the{" "}
            <a href="/imports" className="underline underline-offset-2">
              imports list
            </a>
            .
          </p>
          {/* Only when the failure named a format: a detect failure or an
              unexplained one has no file-shaped fix to point at. */}
          {phase.format && <RedirectionLink format={phase.format} />}
        </div>
      );
    case "error":
      return (
        <p className="max-w-prose font-semibold text-red-700">{phase.message}</p>
      );
  }
}

// Rejection as redirection (BRIEF §3E): the API's message explains what the
// file was; this link says where on the front door the fix lives. A plain
// anchor, not <Link> — on /welcome it scrolls in place, elsewhere it
// navigates there.
function RedirectionLink({ format }: { format?: string }) {
  const r = redirectFor(format);
  return (
    <p className="mt-2">
      <a
        href={`/welcome#${r.anchor}`}
        className="font-semibold underline underline-offset-2"
      >
        {r.link} →
      </a>
    </p>
  );
}
