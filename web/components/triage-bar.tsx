"use client";

// The bulk-triage action bar (phase 11 §6.1): appears when a selection
// exists, sticks to the viewport bottom where a thumb reaches it. Dismiss is
// the primary bulk gesture — the pilot evidence is a reporter who quit at 10
// of 79, and most of a real archive's candidates are noise. Confirm-selected
// rides name suggestions and reports the unnamed rest honestly; it never
// invents a name.
import { useState, useSyncExternalStore, useTransition } from "react";

import {
  confirmSelectedWithSuggestions,
  decideBulk,
} from "@/app/actions";
import { clear, getSelected, subscribe } from "@/lib/selection-store";

export function TriageBar() {
  const selected = useSyncExternalStore(subscribe, getSelected, getSelected);
  const [isPending, startTransition] = useTransition();
  const [note, setNote] = useState<string | null>(null);

  if (selected.size === 0 && !note) return null;
  const ids = [...selected];

  function dismissAll() {
    setNote(null);
    startTransition(async () => {
      const res = await decideBulk(
        ids.map((id) => ({ id, action: "dismissed" as const })),
      );
      if (res.ok) {
        clear();
        setNote(
          `Dismissed ${res.decided} candidate${res.decided === 1 ? "" : "s"}.`,
        );
      } else {
        setNote(`Nothing was recorded — ${res.error}`);
      }
    });
  }

  function confirmAll() {
    setNote(null);
    startTransition(async () => {
      const res = await confirmSelectedWithSuggestions(ids);
      if (!res.ok) {
        setNote(`Nothing was recorded — ${res.error}`);
        return;
      }
      clear();
      const parts: string[] = [
        `Confirmed ${res.confirmed} candidate${res.confirmed === 1 ? "" : "s"}`,
      ];
      if (res.suggested > 0 && res.dateNamed > 0) {
        parts.push(
          ` — ${res.suggested} with suggested names, ${res.dateNamed} named by date.`,
        );
      } else if (res.dateNamed > 0) {
        parts.push(` — all named by date (no suggestions were available).`);
      } else {
        parts.push(` with suggested names.`);
      }
      parts.push(" Rename any from its card or adventure page.");
      setNote(parts.join(""));
    });
  }

  return (
    <div
      role="region"
      aria-label="Bulk decision"
      aria-live="polite"
      className="sticky bottom-0 z-10 mt-6 border-t border-ink bg-paper py-2.5"
    >
      {selected.size > 0 ? (
        <div
          className={`flex flex-wrap items-center gap-x-5 gap-y-1 ${isPending ? "opacity-60" : ""}`}
        >
          <span className="text-sm font-semibold">
            {selected.size} selected
          </span>
          <button
            onClick={dismissAll}
            disabled={isPending}
            className="-my-2 py-2 text-sm text-ink-2 underline decoration-rule underline-offset-2 hover:text-ink disabled:opacity-50"
          >
            Dismiss selected
          </button>
          <button
            onClick={confirmAll}
            disabled={isPending}
            className="-my-2 py-2 text-sm text-emerald-700 underline decoration-rule underline-offset-2 disabled:opacity-50"
          >
            Confirm selected
          </button>
          <button
            onClick={() => {
              clear();
              setNote(null);
            }}
            disabled={isPending}
            className="-my-2 py-2 text-sm text-ink-2 hover:text-ink disabled:opacity-50"
          >
            Clear
          </button>
        </div>
      ) : null}
      {note && <p className="mt-1 text-sm text-ink-2">{note}</p>}
    </div>
  );
}
