"use client";

// The summoned list (DESIGN §5): the life map is map-maximal, and this
// "List (n)" control summons the adventure list as a modal overlay. It is
// the accessible path — the map canvas is aria-hidden decoration, so for
// keyboard and screen-reader users (and small screens) THIS is how
// adventures are reached. That is why it is a first-class dialog, not a
// hover affordance.
//
// The native <dialog> element does the accessibility work by specification
// (see DECISIONS.md "Summoned list"): showModal() traps focus inside,
// Escape closes, and on close focus returns to the button that opened it.
// The two things the platform leaves to us: closing on backdrop click
// (below — a click that hits the dialog element itself can only be on the
// backdrop, because the inner div covers the whole content box), and
// closing after a navigation (Next's client-side <Link> keeps this page's
// DOM alive under the new route otherwise).
import { useRef } from "react";
import Link from "next/link";

import { ProvenanceBar } from "@/components/provenance-bar";
import type { Adventure } from "@/components/life-map";

export function SummonedList({ adventures }: { adventures: Adventure[] }) {
  const ref = useRef<HTMLDialogElement>(null);

  return (
    <>
      <button
        type="button"
        onClick={() => ref.current?.showModal()}
        className="cursor-pointer font-semibold text-ink underline decoration-rule underline-offset-2 hover:decoration-ink"
      >
        List ({adventures.length})
      </button>
      <dialog
        ref={ref}
        aria-label="Confirmed adventures"
        className="m-auto max-h-[85vh] w-[30rem] max-w-[calc(100vw-2rem)] overflow-y-auto border border-rule bg-paper p-0 text-ink shadow-lg backdrop:bg-ink/45"
        onClick={(e) => {
          if (e.target === ref.current) ref.current?.close();
        }}
      >
        <div className="p-6">
          <div className="flex items-baseline justify-between">
            <h2 className="font-display text-lg font-semibold">
              Adventures ({adventures.length})
            </h2>
            <button
              type="button"
              onClick={() => ref.current?.close()}
              className="cursor-pointer text-sm text-ink-2 hover:text-ink"
            >
              Close
            </button>
          </div>
          <ul className="mt-4">
            {adventures.map((a) => {
              const j = a.journey;
              return (
                <li key={a.id} className="border-t border-rule">
                  <Link
                    href={`/adventure/${a.id}`}
                    onClick={() => ref.current?.close()}
                    className="block px-1 py-3 hover:bg-land"
                  >
                    <span className="flex items-baseline justify-between gap-3">
                      <span className="font-display font-semibold">{a.name}</span>
                      <span className="shrink-0 font-mono text-sm">
                        {j.total_km.toFixed(0)} km
                      </span>
                    </span>
                    <span className="mt-0.5 block font-mono text-xs text-ink-2">
                      {a.start.slice(0, 10)} → {a.end.slice(0, 10)}
                    </span>
                    <ProvenanceBar
                      observed={j.observed_km}
                      routed={j.routed_km}
                      unknown={j.unknown_km}
                      air={j.air_km}
                      className="mt-2"
                    />
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      </dialog>
    </>
  );
}
