"use client";

// "Download as image" and "Download as overlay" (phase 14 CP3 + CP4, BRIEF
// §2b, §3D, §9): the plate margin's controls that render the plate as a
// file — on the owner's page and on the shared view alike, because neither
// image discloses anything the page does not. Two formats, both offered
// (the maintainer's decision at the CP3 review): the IMAGE is a picture —
// basemap, route, margin, licence line; the OVERLAY is transparent — route
// and figures only, for the person's own photo or story. Four states, each
// in words: idle, rendering, downloaded, failed — a failure names what went
// wrong (a basemap the browser was not allowed to read, a tile that did not
// load, a timeout) and states that nothing was downloaded. It never
// silently does nothing.
//
// What each file contains is stated beside the buttons before anything is
// rendered. Photos are not drawn in either: they are DOM markers on the
// page, not map layers, and the copy says so.
import { useState } from "react";

import {
  PlateExportError,
  downloadBlob,
  exportOverlayImage,
  exportPlateImage,
  type ExportSpec,
} from "@/lib/plate-export";

type Format = "image" | "overlay";

type State =
  | { step: "idle" }
  | { step: "busy"; format: Format }
  | { step: "done"; filename: string }
  | { step: "error"; message: string };

export function PlateExport(props: ExportSpec) {
  const [state, setState] = useState<State>({ step: "idle" });
  const busy = state.step === "busy";

  async function run(format: Format) {
    setState({ step: "busy", format });
    try {
      const { blob, filename } =
        format === "image"
          ? await exportPlateImage(props)
          : await exportOverlayImage(props);
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

  const buttonClass =
    "-my-3.5 cursor-pointer border border-ink px-3 py-3.5 text-xs font-semibold tracking-[0.06em] text-ink hover:bg-land disabled:cursor-progress disabled:opacity-60";

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-rule pt-2.5">
      {/* Negative-margin padding grows each control to a 44px target
          without moving its text (the site-wide pattern; py-3.5 because
          text-xs is a 16px line). */}
      <button
        type="button"
        onClick={() => run("image")}
        disabled={busy}
        aria-busy={busy && state.format === "image"}
        className={buttonClass}
      >
        {busy && state.format === "image" ? "Rendering…" : "Download as image"}
      </button>
      <button
        type="button"
        onClick={() => run("overlay")}
        disabled={busy}
        aria-busy={busy && state.format === "overlay"}
        className={buttonClass}
      >
        {busy && state.format === "overlay" ? "Rendering…" : "Download as overlay"}
      </button>
      <span className="basis-full text-[11.5px] leading-snug text-ink-2 sm:basis-auto sm:flex-1">
        Image: PNG, 2400 × 1600 — the map with its route, and the margin as
        printed here. Overlay: PNG, 1080 × 1920, transparent — route and
        figures only, for your own photo or story. Photos are not drawn in
        either.
      </span>
      <p
        data-testid="plate-export-status"
        role={state.step === "error" ? "alert" : "status"}
        aria-live="polite"
        className={`basis-full text-[12.5px] ${
          state.step === "error" ? "text-red-700" : "text-ink"
        } ${state.step === "idle" ? "hidden" : ""}`}
      >
        {state.step === "busy" &&
          (state.format === "image"
            ? "Rendering the plate — the basemap loads once more at the export size."
            : "Rendering the overlay.")}
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
