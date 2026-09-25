"use client";

// "Download as image" (phase 14 CP3, BRIEF §2b, §3D): the one control in
// the plate margin that renders the plate as a PNG — on the owner's page
// and on the shared view alike, because the image discloses nothing the
// page does not. Four states, each in words: idle, rendering, downloaded,
// and failed — a failure names what went wrong (a basemap the browser was
// not allowed to read, a tile that did not load, a timeout) and states
// that nothing was downloaded. It never silently does nothing.
//
// What the image contains is stated beside the button before anything is
// rendered: the map with its route, stops and fixes, and the margin —
// name, dates, distance with its split, countries and regions, the legend,
// the basemap credit. Photos are not drawn: they are DOM markers on the
// page, not map layers, and the copy says so.
import { useState } from "react";

import {
  PlateExportError,
  downloadBlob,
  exportPlateImage,
  type ExportSpec,
} from "@/lib/plate-export";

type State =
  | { step: "idle" }
  | { step: "busy" }
  | { step: "done"; filename: string }
  | { step: "error"; message: string };

export function PlateExport(props: ExportSpec) {
  const [state, setState] = useState<State>({ step: "idle" });
  const busy = state.step === "busy";

  async function run() {
    setState({ step: "busy" });
    try {
      const { blob, filename } = await exportPlateImage(props);
      downloadBlob(blob, filename);
      setState({ step: "done", filename });
    } catch (e) {
      setState({
        step: "error",
        message:
          e instanceof PlateExportError
            ? e.message
            : `The export failed (${e instanceof Error ? e.message : String(e)}). Nothing was downloaded.`,
      });
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-rule pt-2.5">
      {/* Negative-margin padding grows the control to a 44px target
          without moving its text (the site-wide pattern). */}
      <button
        type="button"
        onClick={run}
        disabled={busy}
        aria-busy={busy}
        className="-my-3.5 cursor-pointer border border-ink px-3 py-3.5 text-xs font-semibold tracking-[0.06em] text-ink hover:bg-land disabled:cursor-progress disabled:opacity-60"
      >
        {busy ? "Rendering…" : "Download as image"}
      </button>
      <span className="text-[11.5px] text-ink-2">
        PNG, 2400 × 1600 — the map with its route, and the margin as printed
        here. Photos are not drawn.
      </span>
      <p
        data-testid="plate-export-status"
        role={state.step === "error" ? "alert" : "status"}
        aria-live="polite"
        className={`basis-full text-[12.5px] ${
          state.step === "error" ? "text-red-700" : "text-ink"
        } ${state.step === "idle" ? "hidden" : ""}`}
      >
        {state.step === "busy" && "Rendering the plate — the basemap loads once more at the export size."}
        {state.step === "done" && (
          <>
            Downloaded <span className="font-mono">{state.filename}</span>.
          </>
        )}
        {state.step === "error" && state.message}
      </p>
    </div>
  );
}
