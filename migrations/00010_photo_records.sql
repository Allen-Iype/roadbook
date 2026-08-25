-- +goose Up
-- Photo records (phase 11 BRIEF §4D): the durable identity of each ingested
-- photograph. Ingestion extracts a position fix into raw_positions and then
-- discards the original bytes — this row (plus a thumbnail file where the
-- format is decodable) is what keeps every fix on a map traceable back to
-- its photo evidence, without hosting originals. The three recorded photo
-- futures (proof-of-location, points view, attachment follow-up) all read
-- this table; none re-reads originals.
--
-- Chosen over relaxing photos.decision_id (the CP2 schema decision, argued
-- in docs/phase-11/DECISIONS.md): the two tables share a shape but not a
-- lifecycle — photos ride decision re-matching, photo records ride imports —
-- and a nullable FK would make every photos query carry both meanings.
-- The thumbnail directory IS shared: both tables name thumbnails by
-- content_hash, so an identical photo attached and ingested stores one file.
--
-- Records exist only for usable photos (position AND instant extracted):
-- per-file verdicts at upload time report the unusable ones, and retaining
-- rows for photos that place nothing would be storage with no product use.
--
-- fix_content_hash is the record→fix join: raw_positions rows are
-- content-hashed (unique), so the record names its fix without adding a
-- column to the observation stratum. thumb_w/thumb_h = 0 means no thumbnail
-- (HEIC: metadata readable, pixels not decodable here).
CREATE TABLE photo_records (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    import_id        bigint NOT NULL REFERENCES imports (id) ON DELETE RESTRICT,
    content_hash     text NOT NULL UNIQUE,
    original_name    text NOT NULL,
    taken_at         timestamptz NOT NULL,
    taken_offset_sec integer NOT NULL,
    time_source      text NOT NULL CHECK (time_source IN ('gps', 'exif_offset', 'sidecar', 'exif_local')),
    lat              double precision NOT NULL,
    lon              double precision NOT NULL,
    pos_source       text NOT NULL CHECK (pos_source IN ('exif', 'sidecar')),
    thumb_w          integer NOT NULL DEFAULT 0,
    thumb_h          integer NOT NULL DEFAULT 0,
    fix_content_hash text NOT NULL,
    uploaded_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX photo_records_import_idx ON photo_records (import_id, id);
CREATE INDEX photo_records_fix_idx ON photo_records (fix_content_hash);

-- +goose Down
DROP TABLE photo_records;
