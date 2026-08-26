"use client";

// One selection checkbox per triage card (phase 11 §6.1) — a deliberately
// tiny island: the card around it stays server HTML. State lives in the
// module-level selection store shared with the action bar.
import { useSyncExternalStore } from "react";

import { getSelected, subscribe, toggle } from "@/lib/selection-store";

export function TriageSelect({ candidateId }: { candidateId: number }) {
  const selected = useSyncExternalStore(subscribe, getSelected, getSelected);
  return (
    // The label is the hit area: padding reaches the 44px a thumb needs
    // while the rendered box stays 16px (the phase 7/9 tap-target rule).
    <label className="-m-3 inline-flex cursor-pointer items-center p-3">
      <input
        type="checkbox"
        checked={selected.has(candidateId)}
        onChange={() => toggle(candidateId)}
        aria-label="Select this candidate for a bulk decision"
        className="h-4 w-4 accent-ink"
      />
    </label>
  );
}
