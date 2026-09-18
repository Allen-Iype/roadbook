"use client";

// Self-serve deletion (phase 13 CP5, BRIEF §1 "Data lifecycle"): everything
// of the person's goes, structurally, without the operator. A native
// <dialog> (the site's pattern) states what goes and what does not, and
// the confirming button unlocks only after the word "delete" is typed —
// the one place in the product where a typed confirmation is warranted,
// because the action has no undo and the rows are a person's history.
import { useRef, useState, useTransition } from "react";

import { deleteMyData } from "@/app/actions";

export function DeleteMyData({
  multiUser,
  hasData,
}: {
  multiUser: boolean;
  hasData: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [typed, setTyped] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();
  const armed = typed.trim().toLowerCase() === "delete";

  function run() {
    setError(null);
    startTransition(async () => {
      const res = await deleteMyData();
      if (!res.ok) {
        setError(res.error);
        return;
      }
      // A full navigation, not a router push: the session cookie is gone
      // and every page's state with it. "/" lands on /welcome (an empty
      // authless instance) or /signin (a signed-out multi-user one).
      window.location.assign("/");
    });
  }

  return (
    <section className="mt-14 border-t border-rule pt-5" aria-label="Delete everything">
      <h2 className="font-display text-base font-semibold">
        Delete everything
      </h2>
      <p className="mt-1 max-w-[58ch] text-[13px] text-ink-2">
        {hasData
          ? multiUser
            ? "Removes every import, adventure, decision, photo, and share link of yours, and forgets your account. Nothing of anyone else's is touched."
            : "Removes every import, adventure, decision, photo, and share link. The instance is empty again, as if freshly installed."
          : "Nothing is stored yet — there is nothing to delete."}
      </p>
      <p className="mt-3">
        <button
          type="button"
          onClick={() => {
            setTyped("");
            setError(null);
            dialog.current?.showModal();
          }}
          disabled={!hasData}
          className="-my-3 cursor-pointer border border-ink px-3 py-3 text-xs font-semibold tracking-[0.06em] text-ink hover:bg-paper-2 disabled:cursor-default disabled:opacity-50"
        >
          Delete everything…
        </button>
      </p>

      <dialog
        ref={dialog}
        aria-label="Delete everything"
        className="m-auto w-[32rem] max-w-[calc(100vw-2rem)] border border-rule bg-paper p-0 text-ink shadow-lg backdrop:bg-ink/45"
        onClick={(e) => {
          if (e.target === dialog.current) dialog.current?.close();
        }}
      >
        <div className="p-6">
          <h2 className="font-display text-xl font-semibold">
            Delete everything?
          </h2>
          <p className="mt-3 text-sm leading-relaxed">This removes, for good:</p>
          <ul className="mt-2 list-disc space-y-1 pl-5 text-sm leading-relaxed">
            <li>every import and the observations it brought in;</li>
            <li>every detection run, candidate, and decision — names included;</li>
            <li>every photo thumbnail and photo record;</li>
            <li>every share link — the links stop opening immediately;</li>
            {multiUser ? (
              <li>your account here — signing in again starts from nothing.</li>
            ) : (
              <li>the retained copies of your uploaded exports.</li>
            )}
          </ul>
          <p className="mt-3 text-sm leading-relaxed text-ink-2">
            Your own export files, wherever you keep them, are untouched —
            they are the canonical copy of your history. There is no undo.
          </p>
          <label className="mt-5 block text-sm">
            Type <span className="font-mono">delete</span> to confirm
            <input
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              autoComplete="off"
              className="mt-1.5 block w-full border border-rule bg-paper-2 px-3 py-2.5 font-mono text-sm text-ink"
            />
          </label>
          {error && (
            <p role="alert" className="mt-3 text-sm text-red-700">
              {error}
            </p>
          )}
          <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-3">
            <button
              type="button"
              onClick={run}
              disabled={!armed || isPending}
              className="cursor-pointer border border-ink px-4 py-3 font-display text-sm font-semibold tracking-[0.06em] text-ink hover:bg-paper-2 disabled:cursor-default disabled:opacity-50"
            >
              {isPending ? "Deleting…" : "Delete everything"}
            </button>
            <button
              type="button"
              onClick={() => dialog.current?.close()}
              className="-my-3 cursor-pointer px-2 py-3 text-sm text-ink-2 hover:text-ink"
            >
              Keep my data
            </button>
          </div>
        </div>
      </dialog>
    </section>
  );
}
