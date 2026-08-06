-- +goose Up
-- The anchor is a decision's durable identity (phase 1: anchored matching;
-- phase 4: photos ride it). Restore (phase 5 BRIEF §3D) merges archives by
-- that identity with ON CONFLICT, which needs the uniqueness to be
-- structural. It already holds in practice: the decide flow updates an
-- existing decision in place rather than inserting a twin, so two rows with
-- one anchor would be a bug, and this index makes it a loud one.
CREATE UNIQUE INDEX decisions_anchor_unique ON decisions
    (user_id, anchor_span_start, anchor_span_end, anchor_dest_lat, anchor_dest_lon);

-- +goose Down
DROP INDEX decisions_anchor_unique;
