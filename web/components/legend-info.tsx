"use client";

// The legend behind one button (copy pass, docs/product/words.md decision
// 2, 2026-11-02): the four line kinds and, on the trip page, the two
// markers, opened from a small "What the lines mean" control
// instead of printed under every map. The lines keep their four styles —
// the distinction is in the geometry (CLAUDE.md invariants 5 and 8) — but
// a reader who wants the names taps for them. Same mechanism as Term: the
// native popover with an inline fallback where the attribute is missing.
import { useId, useState, useSyncExternalStore } from "react";

import { LEGEND_ENTRIES } from "@/lib/legend";
import { FixDot, LegSample, StopDot } from "@/components/legend";

/** The control's label — pinned by the e2e legend spec. */
export const LEGEND_BUTTON = "What the lines mean";
const subscribeNever = () => () => {};

export function LegendInfo({
  fix = false,
  stop = false,
  className = "",
}: {
  /** The trip page's single-point marker is on the map. */
  fix?: boolean;
  /** The trip page's stop marker is on the map. */
  stop?: boolean;
  className?: string;
}) {
  const noteId = useId();
  const [open, setOpen] = useState(false);
  const fallback = !useSyncExternalStore(
    subscribeNever,
    () => "popover" in HTMLElement.prototype,
    () => true,
  );

  const list = (
    <ul aria-label="Legend" className="list-none space-y-2.5 text-[13px] leading-snug text-ink">
      {LEGEND_ENTRIES.map((e) => (
        <li key={e.key} className="flex items-baseline gap-x-2.5">
          <LegSample kind={e.key} className="h-2 w-11 shrink-0 self-center" />
          <span>
            <span className="font-semibold tracking-wide">{e.label}</span>
            <span data-legend-desc className="text-ink-2">
              {" — "}
              {e.desc}
            </span>
          </span>
        </li>
      ))}
      {fix && (
        <li className="flex items-baseline gap-x-2.5">
          <span className="inline-flex w-11 shrink-0 justify-center self-center">
            <FixDot />
          </span>
          <span>
            <span className="font-semibold tracking-wide">Single point</span>
            <span className="text-ink-2"> — one recorded spot</span>
          </span>
        </li>
      )}
      {stop && (
        <li className="flex items-baseline gap-x-2.5">
          <span className="inline-flex w-11 shrink-0 justify-center self-center">
            <StopDot />
          </span>
          <span className="font-semibold tracking-wide">Stop</span>
        </li>
      )}
    </ul>
  );

  // Same hit-area arithmetic as Term: text-xs is a 16px line, so py-3.5 on
  // both sides reaches 44px without moving the surrounding text.
  const button = `legend-info inline-flex cursor-pointer items-center gap-1.5 -my-3.5 py-3.5 text-xs text-ink underline decoration-dotted decoration-1 underline-offset-[3px] ${className}`;
  const glyph = (
    <span
      aria-hidden
      className="inline-flex h-4 w-4 items-center justify-center rounded-full border border-ink font-display text-[10px] font-semibold leading-none"
    >
      i
    </span>
  );

  if (fallback) {
    return (
      <>
        <button
          type="button"
          className={button}
          aria-expanded={open}
          aria-controls={noteId}
          onClick={() => setOpen((o) => !o)}
        >
          {glyph}
          {LEGEND_BUTTON}
        </button>
        {open && (
          <div id={noteId} role="note" className="mt-3 border-l-2 border-ink pl-3">
            {list}
          </div>
        )}
      </>
    );
  }

  return (
    <>
      <button type="button" className={button} popoverTarget={noteId}>
        {glyph}
        {LEGEND_BUTTON}
      </button>
      <div id={noteId} popover="auto" role="note" className="term-note">
        <p className="font-display text-base font-semibold leading-snug text-ink">
          {LEGEND_BUTTON}
        </p>
        <div className="mt-3">{list}</div>
        <button
          type="button"
          popoverTarget={noteId}
          popoverTargetAction="hide"
          className="mt-3 -mb-3.5 -ml-2 block px-2 py-3.5 text-[13px] font-semibold text-ink"
        >
          Got it
        </button>
      </div>
    </>
  );
}
