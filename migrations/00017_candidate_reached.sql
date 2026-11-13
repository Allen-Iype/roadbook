-- +goose Up
-- A candidate that qualified without a dwelt destination (phase 20, issue #8):
-- the span left one home base, was continuously beyond FAR_KM for MIN_FAR_MIN
-- minutes, and came back to the same base. Its dest_lat/dest_lon/dest_km then
-- describe a place reached, not a place stayed, and the column is what lets
-- every reader tell the two apart. Derived like the rest of the row: written
-- by a detection run, never edited. Rows from runs before this migration are
-- false, which is true of them — the rule that sets it did not exist.
ALTER TABLE candidates ADD COLUMN reached boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE candidates DROP COLUMN reached;
