// One display shape for both photo provenances (phase 11 CP4): photos
// attached to a decision and photo-import records span-joined to the
// candidate. The API keeps them as distinct schemas deliberately — different
// lifecycles, different capabilities (attached photos are deletable, records
// are not) — and this adapter is where the UI flattens them for rendering:
// same markers, same narrative slots, same tiles. `thumb_url` is prebuilt
// here so no component ever branches on which id-space a photo came from;
// null means the record has no thumbnail (HEIC — metadata readable at
// ingest, pixels not decodable).
import type { components } from "@/lib/api/schema";

type Photo = components["schemas"]["Photo"];
type ImportPhoto = components["schemas"]["ImportPhoto"];

export type DisplayPhoto = {
  /** React list key — the two id-spaces can collide, so the key carries the provenance. */
  key: string;
  /** Proxy URL for the thumbnail; null when no thumbnail exists. */
  thumb_url: string | null;
  imported: boolean;
  original_name: string;
  taken_at?: string;
  time_source: string;
  pos?: components["schemas"]["LatLng"];
  place_kind?: Photo["place_kind"];
  leg_index?: number;
  stop_index?: number;
  distance_from_route_m?: number;
  far_flagged?: boolean;
  thumb_w: number;
  thumb_h: number;
};

export function displayAttached(p: Photo): DisplayPhoto {
  return {
    key: `a-${p.id}`,
    thumb_url: `/api/photos/${p.id}/thumb`,
    imported: false,
    original_name: p.original_name,
    taken_at: p.taken_at,
    time_source: p.time_source,
    pos: p.pos,
    place_kind: p.place_kind,
    leg_index: p.leg_index,
    stop_index: p.stop_index,
    distance_from_route_m: p.distance_from_route_m,
    far_flagged: p.far_flagged,
    thumb_w: p.thumb_w,
    thumb_h: p.thumb_h,
  };
}

export function displayImported(p: ImportPhoto): DisplayPhoto {
  return {
    key: `i-${p.id}`,
    thumb_url: p.thumb_w > 0 ? `/api/import-photos/${p.id}/thumb` : null,
    imported: true,
    original_name: p.original_name,
    taken_at: p.taken_at,
    time_source: p.time_source,
    pos: p.pos,
    place_kind: p.place_kind as Photo["place_kind"],
    leg_index: p.leg_index,
    stop_index: p.stop_index,
    distance_from_route_m: p.distance_from_route_m,
    far_flagged: p.far_flagged,
    thumb_w: p.thumb_w,
    thumb_h: p.thumb_h,
  };
}
