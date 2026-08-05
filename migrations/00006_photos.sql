-- +goose Up
-- Photos join the user-data stratum beside decisions (phase 4 BRIEF §1.4):
-- never regenerated, never deleted by any automated process — and wider than
-- a decision, because each row pairs with a thumbnail file on disk (named by
-- content_hash, under the photos directory) and the original is discarded
-- after extraction. Losing a row or its file means re-upload is the only
-- recovery.
--
-- Durable identity (BRIEF §3A): a photo belongs to a decision — the one
-- durable representation of an adventure — so re-detection that renumbers
-- candidates carries photos along via the decision's anchored re-matching.
-- ON DELETE RESTRICT documents the invariant that decisions are never
-- deleted rather than choosing a cascade behaviour for an event that must
-- not happen.
--
-- Placement (which leg, distance from route, the far flag) is deliberately
-- NOT here: derived at read time from taken_at against the assembled
-- journey, so re-detection re-places photos automatically (BRIEF §3B).
CREATE TABLE photos (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    decision_id   bigint NOT NULL REFERENCES decisions (id) ON DELETE RESTRICT,
    -- SHA-256 of the original upload's bytes. Uniqueness makes re-upload
    -- idempotent (the imports precedent) and doubles as the thumbnail's
    -- filename, so row↔file association needs no extra bookkeeping.
    content_hash  text NOT NULL UNIQUE,
    original_name text NOT NULL,
    -- The resolved capture instant and the civil offset for display —
    -- nullable pair, the schema's convention for instants that round-trip
    -- with their offset. NULL taken_at = no usable capture time: the photo
    -- is stored and shown, but unplaced (absence rendered as absence).
    taken_at         timestamptz,
    taken_offset_sec integer,
    -- Provenance, invariant 3's spirit: which rung of the resolution ladder
    -- produced the instant, and which reading produced the position — so a
    -- wrongly-placed photo is explainable a year later.
    time_source   text NOT NULL CHECK (time_source IN ('gps', 'exif_offset', 'sidecar', 'exif_local', 'none')),
    lat           double precision,
    lon           double precision,
    pos_source    text NOT NULL CHECK (pos_source IN ('exif', 'sidecar', 'none')),
    -- Thumbnail pixel dimensions, post-orientation: the page renders
    -- without decoding the file.
    thumb_w       integer NOT NULL,
    thumb_h       integer NOT NULL,
    uploaded_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX photos_decision_idx ON photos (decision_id, id);

-- +goose Down
DROP TABLE photos;
