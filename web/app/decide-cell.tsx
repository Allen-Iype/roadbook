"use client";

// The only client component on the page: one decision cell per candidate row.
// The table stays server-rendered; interactivity is opted into per-island, so
// the JavaScript sent to the browser is exactly this file and nothing else.

import { useOptimistic, useState, useTransition } from "react";
import { decideCandidate } from "@/app/actions";
import type { components } from "@/lib/api/schema";

type Decision = components["schemas"]["Decision"];

// A discriminated union: `status` is the discriminant, and narrowing on it
// makes illegal states unrepresentable — a dismissed candidate cannot carry a
// name, and TypeScript enforces it at compile time. This is the state the UI
// renders; the server's Decision maps into it.
type DecisionState =
  | { status: "undecided" }
  | { status: "confirmed"; name: string }
  | { status: "dismissed" };

function toState(d: Decision | undefined): DecisionState {
  if (!d) return { status: "undecided" };
  if (d.action === "confirmed") {
    return { status: "confirmed", name: d.name ?? "" };
  }
  return { status: "dismissed" };
}

export function DecideCell({
  candidateId,
  decision,
}: {
  candidateId: number;
  decision: Decision | undefined;
}) {
  // useOptimistic: `shown` mirrors the server-derived state, except while a
  // transition is in flight, when it shows whatever we set optimistically.
  // If the action succeeds, revalidatePath refreshes the page and the new
  // server state takes over seamlessly; if it fails, React reverts `shown`
  // to the last server truth automatically — no manual rollback code.
  const [shown, setShown] = useOptimistic<DecisionState, DecisionState>(
    toState(decision),
    (_current, next) => next,
  );
  const [isPending, startTransition] = useTransition();
  // Controlled input: React state is the single source of truth for the name
  // field (value + onChange), so "Save is disabled until non-empty" is a
  // derivation, not DOM inspection.
  const [name, setName] = useState("");
  const [editing, setEditing] = useState<"closed" | "naming" | "redeciding">(
    "closed",
  );
  const [error, setError] = useState<string | null>(null);

  function submit(next: DecisionState) {
    setError(null);
    setEditing("closed");
    // Server actions must run inside a transition for useOptimistic to know
    // when to apply and when to settle.
    startTransition(async () => {
      setShown(next);
      const result = await decideCandidate(
        candidateId,
        next.status === "confirmed" ? "confirmed" : "dismissed",
        next.status === "confirmed" ? next.name : undefined,
      );
      if (!result.ok) setError(result.error);
    });
  }

  const trimmed = name.trim();

  if (editing === "naming") {
    return (
      <span className="flex items-center gap-2">
        <input
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && trimmed !== "") {
              submit({ status: "confirmed", name: trimmed });
            }
            if (e.key === "Escape") setEditing("closed");
          }}
          placeholder="Name this adventure"
          className="w-40 rounded border border-neutral-700 bg-neutral-900 px-2 py-0.5 text-sm outline-none focus:border-neutral-400"
        />
        <button
          disabled={trimmed === ""}
          onClick={() => submit({ status: "confirmed", name: trimmed })}
          className="text-emerald-400 disabled:text-neutral-600"
        >
          save
        </button>
        <button
          onClick={() => setEditing("closed")}
          className="text-neutral-500 hover:text-neutral-300"
        >
          cancel
        </button>
      </span>
    );
  }

  const undecidedControls = (
    <span className="flex items-center gap-3">
      <button
        onClick={() => {
          setName(shown.status === "confirmed" ? shown.name : "");
          setEditing("naming");
        }}
        className="text-emerald-400 hover:underline"
      >
        confirm
      </button>
      <button
        onClick={() => submit({ status: "dismissed" })}
        className="text-neutral-400 hover:underline"
      >
        dismiss
      </button>
    </span>
  );

  return (
    <span
      className={`flex items-center gap-3 ${isPending ? "opacity-60" : ""}`}
    >
      {shown.status === "undecided" || editing === "redeciding" ? (
        undecidedControls
      ) : shown.status === "confirmed" ? (
        <span className="text-emerald-400">{shown.name}</span>
      ) : (
        <span className="text-neutral-600 line-through">dismissed</span>
      )}
      {shown.status !== "undecided" && editing !== "redeciding" && (
        <button
          onClick={() => setEditing("redeciding")}
          className="text-xs text-neutral-600 hover:text-neutral-400"
        >
          change
        </button>
      )}
      {editing === "redeciding" && (
        <button
          onClick={() => setEditing("closed")}
          className="text-xs text-neutral-600 hover:text-neutral-400"
        >
          keep as is
        </button>
      )}
      {error && <span className="text-xs text-red-400">{error}</span>}
    </span>
  );
}
