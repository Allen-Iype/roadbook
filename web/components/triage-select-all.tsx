"use client";

// Select-all for bulk triage (phase 11 §6.1, maintainer's CP4-review ask).
// Selects every UNDECIDED candidate — never the decided ones, whose curation
// a bulk action must not silently overwrite. The ids arrive as a server
// prop (numbers only; the cards' geometry stays server HTML). Confirm-all
// itself stays what Gate 1 decided: bulk confirm rides suggested names and
// reports the unnamed rest — nothing is auto-confirmed.
import { useSyncExternalStore } from "react";

import { getSelected, selectAll, subscribe } from "@/lib/selection-store";

export function TriageSelectAll({ undecidedIds }: { undecidedIds: number[] }) {
  const selected = useSyncExternalStore(subscribe, getSelected, getSelected);
  if (undecidedIds.length === 0) return null;
  const allSelected =
    selected.size > 0 && undecidedIds.every((id) => selected.has(id));
  return (
    <button
      onClick={() => selectAll(undecidedIds)}
      disabled={allSelected}
      className="-my-2 py-2 text-sm underline decoration-rule underline-offset-2 hover:text-ink disabled:no-underline disabled:opacity-60"
    >
      {allSelected
        ? `All ${undecidedIds.length} undecided selected`
        : `Select all ${undecidedIds.length} undecided`}
    </button>
  );
}
