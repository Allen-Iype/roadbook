"use client";

// Share links, the owner's side (phase 13 CP4). Lives on the adventure page
// under the cover, for confirmed adventures only. Two things happen here:
// a link is minted (through a dialog that says, before anything is created,
// exactly what a person with the link will see — the plate is shared whole,
// photos included, or not at all), and a link is revoked (one tap; the
// token stops resolving immediately).
//
// The raw token exists in the browser only in the moment after creation:
// the server keeps a hash, so the URL is shown once with a copy control and
// a plain statement that it cannot be shown again. The URL is composed on
// the browser's own origin — the app never assumes a public hostname.
//
// Native <dialog> like the summoned list (DECISIONS phase 6): showModal()
// traps focus, Escape closes, focus returns to the opener.
import { useRef, useState, useTransition } from "react";

import { createShareLink, revokeShareLink } from "@/app/actions";
import type { components } from "@/lib/api/schema";

type ShareLink = components["schemas"]["ShareLink"];

type DialogState =
  | { step: "confirm" }
  | { step: "created"; url: string }
  | { step: "error"; error: string };

export function ShareLinks({
  candidateId,
  shares,
  photoCount,
}: {
  candidateId: number;
  shares: ShareLink[];
  /** Photos of both provenances on this adventure — stated in the dialog. */
  photoCount: number;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [state, setState] = useState<DialogState>({ step: "confirm" });
  const [copied, setCopied] = useState(false);
  const [isPending, startTransition] = useTransition();
  const [revokeError, setRevokeError] = useState<string | null>(null);

  function open() {
    setState({ step: "confirm" });
    setCopied(false);
    dialog.current?.showModal();
  }

  function create() {
    startTransition(async () => {
      const res = await createShareLink(candidateId);
      if (!res.ok) {
        setState({ step: "error", error: res.error });
        return;
      }
      setState({
        step: "created",
        url: `${window.location.origin}/shared/${res.link.token}`,
      });
    });
  }

  async function copy(url: string, input: HTMLInputElement | null) {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
    } catch {
      // No clipboard permission (or an http origin on a LAN address):
      // select the text so a keyboard copy works.
      input?.select();
    }
  }

  function revoke(id: number) {
    setRevokeError(null);
    startTransition(async () => {
      const res = await revokeShareLink(id, candidateId);
      if (!res.ok) setRevokeError(res.error);
    });
  }

  return (
    <section className="mb-8 border-t border-rule pt-4" aria-label="Sharing">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-2">
        <h2 className="font-display text-base font-semibold">Sharing</h2>
        {/* Negative-margin padding grows the control to a 44px target
            without moving its text (the site-wide pattern). */}
        <button
          type="button"
          onClick={open}
          className="-my-3 cursor-pointer border border-ink px-3 py-3 text-xs font-semibold tracking-[0.06em] text-ink hover:bg-paper-2"
        >
          Create a share link
        </button>
      </div>
      {shares.length === 0 ? (
        <p className="mt-2 max-w-[58ch] text-[13px] text-ink-2">
          Not shared. A share link lets anyone who has it open this plate —
          map, days, countries, and photos — without signing in.
        </p>
      ) : (
        <ul className="mt-2 max-w-[58ch] text-[13px]">
          {shares.map((s) => (
            <li
              key={s.id}
              className="flex items-baseline justify-between gap-4 border-b border-rule py-1.5 last:border-b-0"
            >
              <span>
                Link created {shortDate(s.created_at)}
                <span className="text-ink-2"> — open to anyone who has it</span>
              </span>
              <button
                type="button"
                onClick={() => revoke(s.id)}
                disabled={isPending}
                className="-my-3 shrink-0 cursor-pointer px-2 py-3 text-xs text-ink underline decoration-rule underline-offset-2 hover:decoration-ink disabled:opacity-60"
              >
                Revoke
              </button>
            </li>
          ))}
        </ul>
      )}
      {revokeError && (
        <p role="alert" className="mt-2 text-[13px] text-red-700">
          {revokeError}
        </p>
      )}

      <dialog
        ref={dialog}
        aria-label="Share this adventure"
        className="m-auto w-[32rem] max-w-[calc(100vw-2rem)] border border-rule bg-paper p-0 text-ink shadow-lg backdrop:bg-ink/45"
        onClick={(e) => {
          if (e.target === dialog.current) dialog.current?.close();
        }}
      >
        <div className="p-6">
          {state.step === "created" ? (
            <CreatedPanel
              url={state.url}
              copied={copied}
              onCopy={copy}
              onDone={() => dialog.current?.close()}
            />
          ) : (
            <>
              <h2 className="font-display text-xl font-semibold">
                Share this adventure
              </h2>
              <p className="mt-3 text-sm leading-relaxed">
                Anyone with the link can open this plate, without signing in.
                They will see:
              </p>
              <ul className="mt-2 list-disc space-y-1 pl-5 text-sm leading-relaxed">
                <li>
                  the route map, each leg marked observed, routed, unknown, or
                  air — exactly as it is drawn for you;
                </li>
                <li>the day-by-day narrative, distances, and countries;</li>
                <li>
                  {photoCount === 0
                    ? "any photos you add to this adventure later, including ones from your photo imports."
                    : photoCount === 1
                      ? "the one photo on this adventure, and any you add later — photo imports included."
                      : `all ${photoCount} photos on this adventure, and any you add later — photo imports included.`}
                </li>
              </ul>
              <p className="mt-3 text-sm leading-relaxed text-ink-2">
                Nothing else of yours is reachable through it. You can revoke
                the link from this page at any time.
              </p>
              {state.step === "error" && (
                <p role="alert" className="mt-3 text-sm text-red-700">
                  {state.error}
                </p>
              )}
              <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-3">
                <button
                  type="button"
                  onClick={create}
                  disabled={isPending}
                  className="cursor-pointer border border-ink px-4 py-3 font-display text-sm font-semibold tracking-[0.06em] text-ink hover:bg-paper-2 disabled:opacity-60"
                >
                  {isPending ? "Creating…" : "Create link"}
                </button>
                <button
                  type="button"
                  onClick={() => dialog.current?.close()}
                  className="-my-3 cursor-pointer px-2 py-3 text-sm text-ink-2 hover:text-ink"
                >
                  Cancel
                </button>
              </div>
            </>
          )}
        </div>
      </dialog>
    </section>
  );
}

function CreatedPanel({
  url,
  copied,
  onCopy,
  onDone,
}: {
  url: string;
  copied: boolean;
  onCopy: (url: string, input: HTMLInputElement | null) => void;
  onDone: () => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  return (
    <>
      <h2 className="font-display text-xl font-semibold">Link created</h2>
      <p className="mt-3 text-sm leading-relaxed">
        Copy it now. Roadbook keeps only a fingerprint of this link, so it
        cannot be shown again — but it can be revoked from this page at any
        time.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <input
          ref={input}
          readOnly
          value={url}
          aria-label="Share link"
          onFocus={(e) => e.currentTarget.select()}
          className="min-w-0 flex-1 border border-rule bg-paper-2 px-3 py-2.5 font-mono text-xs text-ink"
        />
        <button
          type="button"
          onClick={() => onCopy(url, input.current)}
          className="cursor-pointer border border-ink px-4 py-2.5 text-sm font-semibold text-ink hover:bg-paper-2"
        >
          {copied ? "Copied" : "Copy link"}
        </button>
      </div>
      <p aria-live="polite" className="mt-2 min-h-[1.25rem] text-xs text-ink-2">
        {copied ? "The link is on your clipboard." : ""}
      </p>
      <div className="mt-4">
        <button
          type="button"
          onClick={onDone}
          className="-my-3 cursor-pointer px-2 py-3 text-sm text-ink-2 hover:text-ink"
        >
          Done
        </button>
      </div>
    </>
  );
}

const SHORT_MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
function shortDate(iso: string): string {
  const [y, m, d] = iso.slice(0, 10).split("-").map(Number);
  return `${d} ${SHORT_MONTHS[m - 1]} ${y}`;
}
