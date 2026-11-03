// A read-only strip of photo tiles: thumbnail (or the honest no-preview
// tile for HEIC records), capture time with its source, and the placement
// statement against the drawn route. Two consumers with different words
// around the same tiles: the owner's photo-import strip (phase 11 CP4 —
// records ride their import, never deletable here) and the shared view
// (phase 13 CP4), where both provenances render read-only for a stranger
// and the copy says whose photos they are. Attached photos on the owner's
// own page keep their upload/delete island (photos-section.tsx).
import { fmtDistanceM, placeStatement, sourceWords } from "@/lib/format";
import { fmtDateRange } from "@/lib/slice-days";
import type { DisplayPhoto } from "@/lib/photo-display";

export function PhotoStrip({
  heading,
  intro,
  photos,
}: {
  heading: string;
  intro: React.ReactNode;
  photos: DisplayPhoto[];
}) {
  if (photos.length === 0) return null;
  return (
    <section className="mt-10">
      <h2 className="font-display text-xl font-semibold">{heading}</h2>
      <p className="mt-1 max-w-[58ch] text-sm text-ink-2">{intro}</p>
      <div className="mt-4 flex flex-wrap gap-4">
        {photos.map((p) => (
          <figure key={p.key} className="w-36">
            {p.thumb_url !== null ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={p.thumb_url}
                alt={p.original_name}
                width={p.thumb_w}
                height={p.thumb_h}
                className={`h-28 w-36 rounded object-cover ${
                  p.far_flagged ? "ring-2 ring-flag" : ""
                }`}
              />
            ) : (
              <div className="flex h-28 w-36 items-center justify-center rounded border border-rule bg-land px-2 text-center text-[11px] leading-snug text-ink-2">
                No preview for this format
              </div>
            )}
            <figcaption className="mt-1 text-[11px] leading-tight text-ink-2">
              <span className="block truncate" title={p.original_name}>
                {p.original_name}
              </span>
              {p.taken_at && (
                <span>
                  {fmtDateRange(p.taken_at, p.taken_at)} {p.taken_at.slice(11, 16)}
                  <span className="text-ink-2/80"> · {sourceWords(p.time_source)}</span>
                </span>
              )}
              <span className="block">
                {p.place_kind ? (
                  <>
                    {p.far_flagged && (
                      <span className="font-bold text-flag" aria-hidden>
                        ⚑{" "}
                      </span>
                    )}
                    {p.distance_from_route_m !== undefined
                      ? `${fmtDistanceM(p.distance_from_route_m)} ${placeStatement(p.place_kind)}`
                      : placeStatement(p.place_kind)}
                  </>
                ) : (
                  "away from the route"
                )}
              </span>
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  );
}

// The owner's photo-import strip: records whose capture falls inside this
// candidate's span, joined at read time (DECISIONS 2026-08-26). Rendered
// for any candidate, not only confirmed ones: during triage the photos are
// evidence of what the trip was.
export function ImportPhotosStrip({ photos }: { photos: DisplayPhoto[] }) {
  return (
    <PhotoStrip
      heading="From your photo uploads"
      intro={
        <>
          {photos.length === 1 ? "One photo" : `${photos.length} photos`} from
          your photo uploads {photos.length === 1 ? "was" : "were"} taken
          during this trip. To remove {photos.length === 1 ? "it" : "them"},
          delete that upload under Uploads.
        </>
      }
      photos={photos}
    />
  );
}
