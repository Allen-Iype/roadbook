"use client";

// The teaching control (phase 16 BRIEF §5D): a load-bearing term rendered
// as a button with a dotted underline that opens its note from lib/terms.ts
// — one or two sentences, no figures — right where the term is met. The
// alternatives were rejected in the brief: hover titles are invisible on
// touch (the problem being fixed), a once-per-visit card needs stored
// state and teaches before the term appears, a glossary page is never
// reached.
//
// Mechanism: the native popover (`popover="auto"` + `popoverTarget`), which
// React 19 supports as plain attributes. It works without any script of
// ours, sits in the top layer, and dismisses on a tap outside or Escape.
// Where the attribute is unsupported (older in-app browsers), an effect
// swaps in a plain toggle that shows the note inline — the note is always
// reachable, only its presentation differs. Both forms render as spans so
// a Term can sit inside any text element (an eyebrow <p>, a <dt>, a list).
import { useId, useState, useSyncExternalStore, type ReactNode } from "react";

import { TERMS, type TermKey } from "@/lib/terms";

const subscribeNever = () => () => {};

export function Term({
  id,
  children,
  className = "",
}: {
  id: TermKey;
  /** The visible text; defaults to the term itself. */
  children?: ReactNode;
  className?: string;
}) {
  const entry = TERMS[id];
  const noteId = useId();
  const [open, setOpen] = useState(false);
  // A browser capability read the React 19 way: the server snapshot says
  // "supported" (the popover form is the one streamed), the client
  // snapshot is the real check, and React reconciles the two after
  // hydration without an effect-driven setState.
  const fallback = !useSyncExternalStore(
    subscribeNever,
    () => "popover" in HTMLElement.prototype,
    () => true,
  );

  // The hit area: inline-block with padding and an equal negative margin
  // grows the touch target to 44px without moving the surrounding text
  // (the site-wide tap-target pattern). leading-4 pins the line box so the
  // arithmetic (16 + 2×14) holds at every font size the term appears in.
  // The dotted mark is a text-decoration, not a border: a border would sit
  // at the padded box's edge, 14px under the words (the first build did).
  const button = `term inline-block cursor-pointer align-baseline leading-4 -my-3.5 py-3.5 text-inherit underline decoration-dotted decoration-1 underline-offset-[3px] ${className}`;

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
          {children ?? entry.term}
        </button>
        {open && (
          <span
            id={noteId}
            role="note"
            className="mt-2 block max-w-[38ch] border-l-2 border-ink pl-3 font-sans text-[13px] normal-case leading-relaxed tracking-normal text-ink"
          >
            {entry.note}
          </span>
        )}
      </>
    );
  }

  return (
    <>
      <button type="button" className={button} popoverTarget={noteId}>
        {children ?? entry.term}
      </button>
      <span id={noteId} popover="auto" role="note" className="term-note">
        <span className="block font-display text-base font-semibold leading-snug text-ink">
          {entry.term}
        </span>
        <span className="mt-2 block text-[13.5px] leading-relaxed text-ink">
          {entry.note}
        </span>
        {/* One-tap close for anyone who does not know a tap outside works;
            Escape and the light dismiss are native. */}
        <button
          type="button"
          popoverTarget={noteId}
          popoverTargetAction="hide"
          className="mt-3 -mb-3.5 -ml-2 block px-2 py-3.5 text-[13px] font-semibold text-ink"
        >
          Got it
        </button>
      </span>
    </>
  );
}
