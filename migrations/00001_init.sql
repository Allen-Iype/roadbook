-- +goose Up
-- The schema divides into three strata with different lifecycles (BRIEF §3):
-- observations are immutable inputs, candidates are derived and disposable,
-- decisions are user data and are never regenerated.

-- One row per import invocation: provenance for the observation tables.
CREATE TABLE imports (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_label text        NOT NULL,
    window_start timestamptz,
    window_end   timestamptz,
    imported_at  timestamptz NOT NULL DEFAULT now(),
    visits       integer     NOT NULL,
    activities   integer     NOT NULL,
    points       integer     NOT NULL,
    skipped      integer     NOT NULL
);

-- Observations: written once, never updated or deleted (CLAUDE.md invariant 2).
-- content_hash is a SHA-256 over the record's canonical form; the unique index
-- plus ON CONFLICT DO NOTHING makes re-import idempotent, so overlapping
-- exports accumulate instead of duplicating.
-- timestamptz stores an instant and forgets the writer's UTC offset, but
-- detection takes civil dates in each timestamp's own offset (home-base eras;
-- see internal/detect/pyport.go). The *_offset_sec columns preserve the offset
-- so a DB round-trip reproduces file-based detection exactly.
CREATE TABLE visits (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    start_ts         timestamptz NOT NULL,
    start_offset_sec integer     NOT NULL,
    end_ts           timestamptz NOT NULL,
    end_offset_sec   integer     NOT NULL,
    lat              double precision,
    lon              double precision,
    semantic_type    text        NOT NULL DEFAULT '',
    content_hash     text        NOT NULL UNIQUE
);
CREATE INDEX visits_start_idx ON visits (start_ts, id);

CREATE TABLE activities (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    start_ts         timestamptz NOT NULL,
    start_offset_sec integer     NOT NULL,
    end_ts           timestamptz NOT NULL,
    end_offset_sec   integer     NOT NULL,
    start_lat        double precision,
    start_lon        double precision,
    end_lat          double precision,
    end_lon          double precision,
    distance_m       double precision NOT NULL DEFAULT 0,
    mode             text        NOT NULL DEFAULT '',
    content_hash     text        NOT NULL UNIQUE
);
CREATE INDEX activities_start_idx ON activities (start_ts, id);

CREATE TABLE path_points (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ts            timestamptz NOT NULL,
    ts_offset_sec integer     NOT NULL,
    lat           double precision,
    lon           double precision,
    content_hash  text        NOT NULL UNIQUE
);
CREATE INDEX path_points_ts_idx ON path_points (ts, id);

-- Every detection run records the parameters that produced it (invariant 3).
-- params is jsonb, not columns: thresholds will be added over time and a run's
-- record must hold exactly the set that produced it without a migration per
-- new knob.
CREATE TABLE detection_runs (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ran_at           timestamptz NOT NULL DEFAULT now(),
    params           jsonb       NOT NULL,
    outliers_dropped integer     NOT NULL,
    bases            jsonb       NOT NULL
);

-- Candidates are derived and disposable: regenerated wholesale by every run.
-- They carry no identity across runs — decisions do, via their anchor
-- (BRIEF §3.1).
CREATE TABLE candidates (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    run_id                bigint  NOT NULL REFERENCES detection_runs (id) ON DELETE CASCADE,
    seq                   integer NOT NULL,
    span_start            timestamptz NOT NULL,
    span_start_offset_sec integer NOT NULL,
    span_end              timestamptz NOT NULL,
    span_end_offset_sec   integer NOT NULL,
    days            double precision NOT NULL,
    dest_lat        double precision NOT NULL,
    dest_lon        double precision NOT NULL,
    dest_km         integer NOT NULL,
    track_km        integer NOT NULL,
    stop_count      integer NOT NULL,
    repeat_count    integer NOT NULL,
    obs_count       integer NOT NULL,
    start_truncated boolean NOT NULL,
    end_truncated   boolean NOT NULL,
    modes           jsonb   NOT NULL,
    UNIQUE (run_id, seq)
);

-- Decisions are user data: never regenerated, never deleted by any automated
-- process. The anchor is the span and destination of the candidate as the user
-- saw it when deciding; association with the current run's candidates is
-- recomputed by matching, not stored (BRIEF §3.1). user_id always holds the
-- same value — the multi-user accommodation from docs/PRODUCT.md, nothing more.
CREATE TABLE decisions (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id           text        NOT NULL DEFAULT 'self',
    action            text        NOT NULL CHECK (action IN ('confirmed', 'dismissed')),
    name              text,
    anchor_span_start timestamptz NOT NULL,
    anchor_span_end   timestamptz NOT NULL,
    anchor_dest_lat   double precision NOT NULL,
    anchor_dest_lon   double precision NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (action <> 'confirmed' OR name IS NOT NULL)
);

-- +goose Down
DROP TABLE decisions;
DROP TABLE candidates;
DROP TABLE detection_runs;
DROP TABLE path_points;
DROP TABLE activities;
DROP TABLE visits;
DROP TABLE imports;
