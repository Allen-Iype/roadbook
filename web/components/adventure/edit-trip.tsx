"use client";

// A kept trip's name and the owner's own notes (phase 17, issue #3). Lives
// in the trip page's header, on the owner's page only: the shared view
// never receives notes from the API, and never renders this component.
//
// Two states. Reading: the notes (if any) set in the display serif — the
// atlas's own face, so the traveller's words read as part of the book, not
// as interface text — and one text control. Editing: the name in the same
// face and size class as the heading it replaces, the notes as plain text
// (line breaks kept, no formatting), Save and Cancel. The limits match the
// API's (counted in characters); the browser stops typing at them, and the
// server says so in words if anything gets past.
//
// The write goes through a server action (app/actions.ts editTrip), which
// calls PATCH /candidates/{id}/decision and revalidates the pages that show
// the name, so the heading, the shelf and the map's list all read the new
// name on the next render.
import { useState, useTransition } from "react";

import { editTrip } from "@/app/actions";

export const NAME_MAX = 120;
export const NOTES_MAX = 4000;

// Characters as the API counts them (code points), not UTF-16 units.
const chars = (s: string) => Array.from(s).length;

export function EditTrip({
  candidateId,
  name,
  notes,
}: {
  candidateId: number;
  name: string;
  notes: string | undefined;
}) {
  const [editing, setEditing] = useState(false);
  const [draftName, setDraftName] = useState(name);
  const [draftNotes, setDraftNotes] = useState(notes ?? "");
  const [error, setError] = useState<string | null>(null);
  const [isPending, startTransition] = useTransition();

  function open() {
    setDraftName(name);
    setDraftNotes(notes ?? "");
    setError(null);
    setEditing(true);
  }

  function save() {
    startTransition(async () => {
      const res = await editTrip(candidateId, draftName, draftNotes);
      if (!res.ok) {
        setError(res.error);
        return;
      }
      setEditing(false);
    });
  }

  const nameBlank = draftName.trim() === "";
  const notesCount = chars(draftNotes);

  if (!editing) {
    return (
      <div className="mt-4">
        {notes && (
          <p
            data-testid="trip-notes"
            className="max-w-[54ch] whitespace-pre-line font-display text-lg leading-relaxed text-ink"
          >
            {notes}
          </p>
        )}
        {/* text-sm needs py-3 for a 44px hit area (phase 7 BRIEF §1.5). */}
        <button
          type="button"
          onClick={open}
          className="-my-3 mt-0 cursor-pointer py-3 text-sm text-ink-2 underline decoration-rule underline-offset-2 hover:text-ink hover:decoration-ink"
        >
          {notes ? "Edit name and notes" : "Edit name or add notes"}
        </button>
      </div>
    );
  }

  return (
    <form
      className="mt-4 max-w-[54ch] border-t border-rule pt-4"
      aria-label="Edit name and notes"
      onSubmit={(e) => {
        e.preventDefault();
        if (!nameBlank) save();
      }}
      onKeyDown={(e) => {
        if (e.key === "Escape") setEditing(false);
      }}
    >
      <label className="block text-sm text-ink-2" htmlFor="trip-name">
        Name
      </label>
      <input
        id="trip-name"
        autoFocus
        value={draftName}
        maxLength={NAME_MAX}
        onChange={(e) => setDraftName(e.target.value)}
        className="mt-1 w-full border border-rule bg-paper px-3 py-2 font-display text-2xl font-semibold leading-tight text-ink outline-none focus:border-ink"
      />

      <label className="mt-4 block text-sm text-ink-2" htmlFor="trip-notes">
        Notes <span className="text-ink-2/80">— only you see these</span>
      </label>
      <textarea
        id="trip-notes"
        value={draftNotes}
        maxLength={NOTES_MAX}
        rows={6}
        onChange={(e) => setDraftNotes(e.target.value)}
        placeholder="What happened, who came, what you'd tell someone going."
        className="mt-1 w-full resize-y border border-rule bg-paper px-3 py-2 font-display text-lg leading-relaxed text-ink outline-none placeholder:text-ink-2/70 focus:border-ink"
      />
      <p className="mt-1 text-right font-mono text-xs text-ink-2" aria-live="polite">
        {notesCount.toLocaleString("en-GB")} of {NOTES_MAX.toLocaleString("en-GB")}
      </p>

      {error && (
        <p role="alert" className="mt-2 text-[13px] text-red-700">
          {error}
        </p>
      )}

      <div className="mt-3 flex items-center gap-4">
        <button
          type="submit"
          disabled={nameBlank || isPending}
          className="cursor-pointer border border-ink bg-ink px-4 py-3 text-sm font-semibold text-paper hover:bg-paper hover:text-ink disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isPending ? "Saving…" : "Save"}
        </button>
        <button
          type="button"
          onClick={() => setEditing(false)}
          className="-my-3 cursor-pointer py-3 text-sm text-ink-2 hover:text-ink"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
