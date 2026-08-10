"use client";

// The photos island: upload, pending previews, per-file results, the strip,
// and per-photo delete. One client component so the page stays
// server-rendered around it (the decide-cell pattern, scaled up).
//
// Optimistic UI at seconds-scale (phase 4 BRIEF §1.5): the browser can show
// a preview of a not-yet-uploaded file via URL.createObjectURL — a temporary
// in-memory URL for the File object, no server involved. Those previews are
// overlaid on the server-confirmed list with useOptimistic while the upload
// transition is in flight; when the server action completes and
// revalidatePath refreshes the page, React swaps the overlay for the real
// rows automatically — no manual reconciliation. Failures are per-file: the
// API returns one result per uploaded file, and the panel renders each
// (BRIEF §3F), so one HEIC in a batch of five fails inline while four land.

import { useEffect, useOptimistic, useState, useTransition } from "react";
import { deletePhoto, uploadPhotos } from "@/app/actions";
import { fmtDistanceM, placeStatement } from "@/lib/format";
import type { components } from "@/lib/api/schema";

type Photo = components["schemas"]["Photo"];
type UploadResult = components["schemas"]["PhotoUploadResult"];

// A pending tile: what we can show about a file before the server has seen
// it. Sidecar JSONs get no preview image — there is nothing to preview.
type Pending = { name: string; previewUrl?: string };

export function PhotosSection({
  candidateId,
  photos,
}: {
  candidateId: number;
  photos: Photo[];
}) {
  const [isPending, startTransition] = useTransition();

  // Overlay 1: pending upload tiles. Set inside the transition; reverts to
  // [] by itself when the transition settles (success or failure).
  const [pending, setPending] = useOptimistic<Pending[], Pending[]>(
    [],
    (_current, next) => next,
  );
  // Overlay 2: optimistically hidden photo ids while a delete is in flight.
  const [hiddenIds, setHiddenIds] = useOptimistic<number[], number>(
    [],
    (current, id) => [...current, id],
  );

  // Object URLs must be released or the blobs live until tab close; the
  // effect cleanup revokes the previous set whenever it changes.
  const [urls, setUrls] = useState<string[]>([]);
  useEffect(() => {
    return () => urls.forEach((u) => URL.revokeObjectURL(u));
  }, [urls]);

  const [results, setResults] = useState<UploadResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  function onSelect(fileList: FileList | null) {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList);
    setError(null);
    setResults(null);

    const previews: Pending[] = files.map((f) => ({
      name: f.name,
      previewUrl: f.type === "image/jpeg" ? URL.createObjectURL(f) : undefined,
    }));
    setUrls(previews.flatMap((p) => (p.previewUrl ? [p.previewUrl] : [])));

    const form = new FormData();
    for (const f of files) form.append("files", f, f.name);

    startTransition(async () => {
      setPending(previews);
      try {
        const res = await uploadPhotos(candidateId, form);
        if (res.ok) {
          setResults(res.results);
        } else {
          setError(res.error);
        }
      } catch {
        // The one whole-batch failure: the request never reached the API —
        // most likely the batch exceeded the transport's body limit.
        setError(
          "The batch could not be sent — most likely too large in one request. Try fewer photos at a time.",
        );
      }
    });
  }

  function onDelete(photoId: number) {
    setError(null);
    startTransition(async () => {
      setHiddenIds(photoId);
      const res = await deletePhoto(photoId, candidateId);
      if (!res.ok) setError(res.error);
    });
  }

  const shown = photos.filter((p) => !hiddenIds.includes(p.id));

  return (
    <section className="mt-6">
      <div className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-2">
          Photos
        </h2>
        <label
          className={`cursor-pointer rounded border border-rule px-3 py-1 text-xs ${
            isPending
              ? "cursor-wait text-ink-2/70"
              : "text-ink hover:border-ink-2"
          }`}
        >
          {isPending ? "Uploading…" : "Add photos"}
          <input
            type="file"
            multiple
            // JPEGs and Takeout sidecar JSONs; everything else is rejected
            // server-side with an actionable reason shown per file.
            accept=".jpg,.jpeg,image/jpeg,.json,application/json"
            className="hidden"
            disabled={isPending}
            onChange={(e) => {
              onSelect(e.target.files);
              e.target.value = ""; // allow re-selecting the same files
            }}
          />
        </label>
      </div>

      {error && <p className="mt-2 text-sm text-red-700">{error}</p>}

      {shown.length === 0 && pending.length === 0 && (
        <p className="mt-2 text-sm text-ink-2">
          None yet. Photos carry the most accurate positions in the project —
          EXIF stays on camera originals; WhatsApp and most messengers strip
          it.
        </p>
      )}

      <div className="mt-3 flex flex-wrap gap-3">
        {shown.map((p) => (
          <PhotoTile key={p.id} photo={p} onDelete={onDelete} busy={isPending} />
        ))}
        {pending.map((p) => (
          <PendingTile key={p.name} pending={p} />
        ))}
      </div>

      {results && <ResultsPanel results={results} />}
    </section>
  );
}

function PhotoTile({
  photo,
  onDelete,
  busy,
}: {
  photo: Photo;
  onDelete: (id: number) => void;
  busy: boolean;
}) {
  return (
    <figure className="group relative w-36">
      {/* The src points at the Next proxy route handler — the browser never
          talks to the Go API (BRIEF §1.3). The amber ring is the far flag:
          the photo (a measurement) disagreeing with the drawn route. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={`/api/photos/${photo.id}/thumb`}
        alt={photo.original_name}
        width={photo.thumb_w}
        height={photo.thumb_h}
        className={`h-28 w-36 rounded object-cover ${
          photo.far_flagged ? "ring-2 ring-flag" : ""
        }`}
      />
      <button
        onClick={() => onDelete(photo.id)}
        disabled={busy}
        title="Delete this photo"
        className="absolute right-1 top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-land/80 text-xs text-ink hover:text-red-700 group-hover:flex"
      >
        ×
      </button>
      <figcaption className="mt-1 text-[11px] leading-tight text-ink-2">
        <span className="block truncate" title={photo.original_name}>
          {photo.original_name}
        </span>
        {photo.taken_at ? (
          // taken_at carries the photo's own civil offset; slicing shows
          // traveller-local time (the page-wide convention).
          <span title={`capture time from ${photo.time_source}`}>
            {photo.taken_at.slice(0, 10)} {photo.taken_at.slice(11, 16)}
            <span className="text-ink-2/80"> · {photo.time_source}</span>
          </span>
        ) : (
          <span className="text-ink-2">no capture time</span>
        )}
        <span className="block">
          {photo.pos ? (
            <span title={`position from ${photo.pos_source}`}>
              positioned · {photo.pos_source}
            </span>
          ) : (
            <span className="text-ink-2">no position — strip only</span>
          )}
        </span>
        {/* Placement (BRIEF §3G): where the journey held this instant, and
            how far the photo sits from that claim. Derived server-side
            against the same geometry the map draws. */}
        {photo.place_kind &&
          (photo.distance_from_route_m !== undefined ? (
            <span className="block">
              {/* Amber marks, ink words: the flag token fails contrast as
                  running text (CP4 a11y pass), so the glyph carries it. */}
              {photo.far_flagged && (
                <span className="font-bold text-flag">⚑ </span>
              )}
              {fmtDistanceM(photo.distance_from_route_m)}{" "}
              {placeStatement(photo.place_kind)}
            </span>
          ) : (
            <span className="block">{placeStatement(photo.place_kind)}</span>
          ))}
        {!photo.place_kind && photo.pos && photo.taken_at && (
          <span className="block text-ink-2">
            outside this journey&apos;s timeline
          </span>
        )}
      </figcaption>
    </figure>
  );
}

function PendingTile({ pending }: { pending: Pending }) {
  return (
    <figure className="w-36 opacity-60">
      {pending.previewUrl ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={pending.previewUrl}
          alt={pending.name}
          className="h-28 w-36 animate-pulse rounded object-cover"
        />
      ) : (
        <div className="flex h-28 w-36 animate-pulse items-center justify-center rounded bg-land text-xs text-ink-2">
          sidecar
        </div>
      )}
      <figcaption className="mt-1 truncate text-[11px] text-ink-2">
        {pending.name}
      </figcaption>
    </figure>
  );
}

// One line per uploaded file — the API's per-file verdicts rendered
// truthfully (BRIEF §1.5): a rejected file names its reason beside the files
// that landed.
function ResultsPanel({ results }: { results: UploadResult[] }) {
  const label: Record<
    UploadResult["status"],
    { text: string; cls: string; flagged?: boolean }
  > = {
    accepted: { text: "added", cls: "text-emerald-700" },
    duplicate: { text: "already uploaded", cls: "text-ink-2" },
    rejected: { text: "rejected", cls: "text-red-700" },
    sidecar_paired: { text: "metadata applied", cls: "text-emerald-700" },
    sidecar_unpaired: { text: "unpaired sidecar", cls: "text-ink", flagged: true },
  };
  return (
    <ul className="mt-3 space-y-1 text-xs">
      {results.map((r, i) => (
        <li key={i}>
          <span className="font-mono text-ink">{r.file}</span>{" "}
          {label[r.status].flagged && (
            <span className="font-bold text-flag" aria-hidden>
              ⚑{" "}
            </span>
          )}
          <span className={label[r.status].cls}>{label[r.status].text}</span>
          {r.paired_with && (
            <span className="text-ink-2"> → {r.paired_with}</span>
          )}
          {r.reason && <span className="text-ink-2"> — {r.reason}</span>}
        </li>
      ))}
    </ul>
  );
}
