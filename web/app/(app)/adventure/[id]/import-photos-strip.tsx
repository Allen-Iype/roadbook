// The photo-import strip (phase 11 CP4): records whose capture falls inside
// this candidate's span, joined at read time — never stored, never deletable
// here (they belong to their import; DECISIONS 2026-08-26). Rendered for any
// candidate, not only confirmed ones: during triage the photos are evidence
// of what the trip was. Tiles mirror the attached-photo strip; a record with
// no thumbnail (HEIC) renders its facts as a tile rather than vanishing.
import { fmtDistanceM, placeStatement } from "@/lib/format";
import type { DisplayPhoto } from "@/lib/photo-display";

export function ImportPhotosStrip({ photos }: { photos: DisplayPhoto[] }) {
  if (photos.length === 0) return null;
  return (
    <section className="mt-10">
      <h2 className="font-display text-xl font-semibold">
        From your photo imports
      </h2>
      <p className="mt-1 max-w-[58ch] text-sm text-ink-2">
        {photos.length === 1 ? "One photo" : `${photos.length} photos`} from
        your photo imports {photos.length === 1 ? "was" : "were"} taken inside
        this journey&apos;s window — placed by capture time against the drawn
        route. They stay with their import; deleting is not done from here.
      </p>
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
                No preview — this format&apos;s pixels aren&apos;t decodable
                here
              </div>
            )}
            <figcaption className="mt-1 text-[11px] leading-tight text-ink-2">
              <span className="block truncate" title={p.original_name}>
                {p.original_name}
              </span>
              {p.taken_at && (
                <span title={`capture time from ${p.time_source}`}>
                  {p.taken_at.slice(0, 10)} {p.taken_at.slice(11, 16)}
                  <span className="text-ink-2/80"> · {p.time_source}</span>
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
                  "outside the drawn route's elements"
                )}
              </span>
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  );
}
