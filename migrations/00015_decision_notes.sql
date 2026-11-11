-- +goose Up
-- The owner's own words on a kept trip (phase 17, issue #3). Nullable: no
-- note is the common case and reads as absent, never as an empty string.
-- User data like the rest of the row — never regenerated, never touched by
-- re-detection; a re-decide (decideCandidate) leaves it alone, so dismissing
-- and re-keeping a trip does not blank what was written. Written only by
-- PATCH /candidates/{id}/decision, read only by the owner: the shared view
-- builds its own response and never carries it.
ALTER TABLE decisions ADD COLUMN notes text;

-- +goose Down
ALTER TABLE decisions DROP COLUMN notes;
