-- +goose Up
-- rawSignals position fixes: the export's only observations carrying a
-- reported accuracy. An export holds only its last ~30 days of these, which is
-- why exports are banked; rows accumulate across imports like every other
-- observation table. Same stratum as path_points: immutable input
-- (CLAUDE.md invariant 2), content-hash idempotent, offset preserved beside
-- timestamptz for exact file/DB parity.
CREATE TABLE raw_positions (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ts            timestamptz NOT NULL,
    ts_offset_sec integer     NOT NULL,
    lat           double precision,
    lon           double precision,
    accuracy_m    double precision NOT NULL DEFAULT 0,
    source        text        NOT NULL DEFAULT '',
    content_hash  text        NOT NULL UNIQUE
);
CREATE INDEX raw_positions_ts_idx ON raw_positions (ts, id);

ALTER TABLE imports ADD COLUMN raw_positions integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE imports DROP COLUMN raw_positions;
DROP TABLE raw_positions;
