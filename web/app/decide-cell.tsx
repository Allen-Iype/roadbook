"use client";

// The only client component on the page: one decision cell per candidate row.
// The table stays server-rendered; interactivity is opted into per-island, so
// the JavaScript sent to the browser is exactly this file and nothing else.

import { useOptimistic, useRef, useState, useTransition } from "react";
import { decideCandidate, suggestName } from "@/app/actions";
import type { components } from "@/lib/api/schema";

type Decision = components["schemas"]["Decision"];
type ScoreComponent = components["schemas"]["ScoreComponent"];

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
  score,
  scoreBreakdown,
}: {
  candidateId: number;
  decision: Decision | undefined;
  score?: number;
  scoreBreakdown?: ScoreComponent[];
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
  const rootRef = useRef<HTMLSpanElement>(null);

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
      else advanceFocus(rootRef.current);
    });
  }

  const trimmed = name.trim();

  if (editing === "naming") {
    return (
      <span ref={rootRef} className="flex flex-col gap-1.5">
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
            className="w-40 rounded border border-rule bg-paper px-2 py-1 text-sm outline-none focus:border-ink"
          />
          {/* Negative-margin padding on every control: the hit area reaches
              the 44px a thumb needs (phase 7 BRIEF §1.5) while the rendered
              text moves not at all. The padding depends on the text size —
              text-sm (20px line) needs py-3, text-xs (16px line) py-3.5 —
              which is how the decided-state buttons shipped at 40px until
              the phase 9 harness measured them. */}
          <button
            disabled={trimmed === ""}
            onClick={() => submit({ status: "confirmed", name: trimmed })}
            className="-mx-2 -my-3 px-2 py-3 text-emerald-700 disabled:text-rule"
          >
            save
          </button>
          <button
            onClick={() => setEditing("closed")}
            className="-mx-2 -my-3 px-2 py-3 text-ink-2 hover:text-ink"
          >
            cancel
          </button>
        </span>
        <ScoreBreakdownBox score={score} components={scoreBreakdown} />
      </span>
    );
  }

  const undecidedControls = (
    <span className="flex items-center gap-3">
      <button
        onClick={() => {
          setName(shown.status === "confirmed" ? shown.name : "");
          setEditing("naming");
          // Prefill, never auto-apply (BRIEF §1.7): the suggestion lands in
          // the input only if it is still empty when the lookup returns —
          // anything the user typed in the meantime wins.
          suggestName(candidateId).then((s) => {
            const suggested = s.name;
            if (suggested) setName((cur) => (cur === "" ? suggested : cur));
          });
        }}
        className="-mx-2 -my-3 px-2 py-3 text-emerald-700 hover:underline"
      >
        confirm
      </button>
      <button
        onClick={() => submit({ status: "dismissed" })}
        className="-mx-2 -my-3 px-2 py-3 text-ink-2 hover:underline"
      >
        dismiss
      </button>
    </span>
  );

  return (
    <span
      ref={rootRef}
      className={`flex items-center gap-3 ${isPending ? "opacity-60" : ""}`}
    >
      {shown.status === "undecided" || editing === "redeciding" ? (
        undecidedControls
      ) : shown.status === "confirmed" ? (
        <span className="text-emerald-700">{shown.name}</span>
      ) : (
        <span className="text-ink-2 line-through">dismissed</span>
      )}
      {shown.status !== "undecided" && editing !== "redeciding" && (
        <button
          onClick={() => setEditing("redeciding")}
          className="-mx-2 -my-3.5 px-2 py-3.5 text-xs text-ink-2 hover:text-ink"
        >
          change
        </button>
      )}
      {editing === "redeciding" && (
        <button
          onClick={() => setEditing("closed")}
          className="-mx-2 -my-3.5 px-2 py-3.5 text-xs text-ink-2 hover:text-ink"
        >
          keep as is
        </button>
      )}
      {error && <span className="text-xs text-red-700">{error}</span>}
    </span>
  );
}

// The sweep's keyboard pace (phase 11 §6.1): after a decision lands, focus
// jumps to the next undecided card's first control, so a queue clears
// without reaching for the mouse. DOM-order walk over the gallery's
// data-candidate-card attributes — the cards are server HTML with no shared
// React ancestor, so the document is the honest source of "next". A no-op
// off the gallery (the attribute doesn't exist elsewhere) and on the last
// card.
function advanceFocus(from: HTMLElement | null) {
  if (!from) return;
  const own = from.closest("[data-candidate-card]");
  if (!own) return;
  const cards = [...document.querySelectorAll("[data-candidate-card]")];
  const start = cards.indexOf(own as Element);
  for (let i = start + 1; i < cards.length; i++) {
    if (cards[i].getAttribute("data-undecided") === "true") {
      const btn = cards[i].querySelector<HTMLButtonElement>(
        "button:not([disabled])",
      );
      if (btn) {
        btn.focus();
        return;
      }
    }
  }
}

// Why the machine proposed this candidate: the stored per-component
// arithmetic, shown at the moment of confirming (BRIEF §1.6). Decision
// support only — nothing here confirms anything. Candidates from runs before
// scoring existed have no breakdown and the box simply doesn't render.
function ScoreBreakdownBox({
  score,
  components,
}: {
  score?: number;
  components?: ScoreComponent[];
}) {
  if (score === undefined || !components || components.length === 0) {
    return null;
  }
  return (
    <span className="w-max rounded border border-rule bg-land px-2.5 py-1.5 text-xs text-ink-2">
      <span className="block pb-1 text-ink">
        confidence {score.toFixed(1)} / 100
      </span>
      {components.map((c) => (
        <span key={c.name} className="flex items-baseline gap-2 py-px">
          <span className="w-36">{c.name.replaceAll("_", " ")}</span>
          {c.present ? (
            <>
              <span className="w-24 text-right font-mono">
                {c.raw} {c.unit}
              </span>
              <span className="w-14 text-right font-mono text-ink">
                +{c.contribution.toFixed(1)}
              </span>
            </>
          ) : (
            <span className="text-ink-2/80">
              not measurable — weight redistributed
            </span>
          )}
        </span>
      ))}
    </span>
  );
}
