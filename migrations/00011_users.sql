-- +goose Up
-- Phase 13 CP1: identity schema. Row-scoped tenancy in one instance
-- (BRIEF §3A): every user-data table gains an owner, and every store query
-- filters by it. 'self' is the single-user default — an authless instance
-- (ROADBOOK_AUTH=off, the self-host reference) is user 'self' throughout
-- and behaves exactly as before this migration; decisions has carried the
-- column since 00001 as the PRODUCT.md accommodation, now made real.
--
-- Deliberately global (docs/phase-13/DECISIONS.md): countries (public
-- reference data) and route_cache/route_runs (derived road geometry keyed
-- by rounded coordinates, never exposed per-user; shared so one user's
-- routing warms another's).
--
-- id is text, not a sequence: 'self' must be a valid id, and OIDC-created
-- users (CP2) get an opaque generated id — never the provider subject,
-- which is provider-scoped and belongs in its own column.
CREATE TABLE users (
    id           text PRIMARY KEY,
    oidc_subject text UNIQUE,
    email        text,
    created_at   timestamptz NOT NULL DEFAULT now()
);
INSERT INTO users (id) VALUES ('self');

ALTER TABLE decisions ADD CONSTRAINT decisions_user_fk
    FOREIGN KEY (user_id) REFERENCES users (id);

-- Owner column on every user-data table. Existing rows are 'self' by the
-- DEFAULT; the default STAYS so an authless instance needs no special
-- handling anywhere — inserts name the owner explicitly regardless.
ALTER TABLE imports        ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE visits         ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE activities     ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE path_points    ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE raw_positions  ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE detection_runs ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE candidates     ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE photos         ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE photo_records  ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);

-- Content-hash dedupe becomes per-user: "this instance already holds these
-- bytes" was really "this PERSON already holds these bytes" — two users may
-- import the same export or photo and each owns their copy. (Thumbnail
-- files stay content-addressed and shared on disk; identical bytes are one
-- file either way.)
ALTER TABLE visits         DROP CONSTRAINT visits_content_hash_key;
ALTER TABLE activities     DROP CONSTRAINT activities_content_hash_key;
ALTER TABLE path_points    DROP CONSTRAINT path_points_content_hash_key;
ALTER TABLE raw_positions  DROP CONSTRAINT raw_positions_content_hash_key;
ALTER TABLE photos         DROP CONSTRAINT photos_content_hash_key;
ALTER TABLE photo_records  DROP CONSTRAINT photo_records_content_hash_key;
ALTER TABLE visits         ADD CONSTRAINT visits_user_hash_key        UNIQUE (user_id, content_hash);
ALTER TABLE activities     ADD CONSTRAINT activities_user_hash_key    UNIQUE (user_id, content_hash);
ALTER TABLE path_points    ADD CONSTRAINT path_points_user_hash_key   UNIQUE (user_id, content_hash);
ALTER TABLE raw_positions  ADD CONSTRAINT raw_positions_user_hash_key UNIQUE (user_id, content_hash);
ALTER TABLE photos         ADD CONSTRAINT photos_user_hash_key        UNIQUE (user_id, content_hash);
ALTER TABLE photo_records  ADD CONSTRAINT photo_records_user_hash_key UNIQUE (user_id, content_hash);

-- The chronological read path is now per-user; the time indexes follow it.
DROP INDEX visits_start_idx;
DROP INDEX activities_start_idx;
DROP INDEX path_points_ts_idx;
DROP INDEX raw_positions_ts_idx;
CREATE INDEX visits_start_idx        ON visits (user_id, start_ts, id);
CREATE INDEX activities_start_idx    ON activities (user_id, start_ts, id);
CREATE INDEX path_points_ts_idx      ON path_points (user_id, ts, id);
CREATE INDEX raw_positions_ts_idx    ON raw_positions (user_id, ts, id);

-- +goose Down
DROP INDEX visits_start_idx;
DROP INDEX activities_start_idx;
DROP INDEX path_points_ts_idx;
DROP INDEX raw_positions_ts_idx;
CREATE INDEX visits_start_idx     ON visits (start_ts, id);
CREATE INDEX activities_start_idx ON activities (start_ts, id);
CREATE INDEX path_points_ts_idx   ON path_points (ts, id);
CREATE INDEX raw_positions_ts_idx ON raw_positions (ts, id);
ALTER TABLE visits         DROP CONSTRAINT visits_user_hash_key;
ALTER TABLE activities     DROP CONSTRAINT activities_user_hash_key;
ALTER TABLE path_points    DROP CONSTRAINT path_points_user_hash_key;
ALTER TABLE raw_positions  DROP CONSTRAINT raw_positions_user_hash_key;
ALTER TABLE photos         DROP CONSTRAINT photos_user_hash_key;
ALTER TABLE photo_records  DROP CONSTRAINT photo_records_user_hash_key;
ALTER TABLE visits         ADD CONSTRAINT visits_content_hash_key        UNIQUE (content_hash);
ALTER TABLE activities     ADD CONSTRAINT activities_content_hash_key    UNIQUE (content_hash);
ALTER TABLE path_points    ADD CONSTRAINT path_points_content_hash_key   UNIQUE (content_hash);
ALTER TABLE raw_positions  ADD CONSTRAINT raw_positions_content_hash_key UNIQUE (content_hash);
ALTER TABLE photos         ADD CONSTRAINT photos_content_hash_key        UNIQUE (content_hash);
ALTER TABLE photo_records  ADD CONSTRAINT photo_records_content_hash_key UNIQUE (content_hash);
ALTER TABLE imports        DROP COLUMN user_id;
ALTER TABLE visits         DROP COLUMN user_id;
ALTER TABLE activities     DROP COLUMN user_id;
ALTER TABLE path_points    DROP COLUMN user_id;
ALTER TABLE raw_positions  DROP COLUMN user_id;
ALTER TABLE detection_runs DROP COLUMN user_id;
ALTER TABLE candidates     DROP COLUMN user_id;
ALTER TABLE photos         DROP COLUMN user_id;
ALTER TABLE photo_records  DROP COLUMN user_id;
ALTER TABLE decisions      DROP CONSTRAINT decisions_user_fk;
DROP TABLE users;
