-- +goose Up
-- The match cache (phase 23, #9): answers to "which road do these recorded
-- points lie on?", the second derived table beside route_cache and for the
-- same reason — the answer comes from an external engine that is slow,
-- possibly absent, and changes with the OSM snapshot. `roadbook route`
-- (batch) writes; `roadbook serve` only reads. TRUNCATE is always safe:
-- every recorded stretch is then drawn point to point, as before this
-- table existed. Nothing references it.
--
-- The key is the request itself: a SHA-256 over the stretch's points
-- (coordinates rounded to five decimals, times in whole seconds — exactly
-- what the matcher was sent), the profile, and the search radius. A hit is
-- therefore exact. Global like route_cache (migration 11): a row can only
-- be found by someone who already holds the same points.
--
-- The length guard (MATCH_MAX_RATIO) is NOT stored and not part of the
-- key: it is applied when a row is read, so changing it needs no new
-- requests. That is why a match far longer than its points is still kept
-- here as 'matched'.
CREATE TABLE match_cache (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    points_hash  text    NOT NULL,
    profile      text    NOT NULL,
    radius_m     integer NOT NULL,
    -- How many points the request held; kept for reporting.
    points       integer NOT NULL,
    -- 'no_match' is a remembered negative: no single road line fits these
    -- points within the radius (or the matcher could only fit them as
    -- several disconnected pieces). Operational failures are never cached.
    status       text    NOT NULL CHECK (status IN ('matched', 'no_match')),
    -- [[lat, lon], ...] — lat-first, as in route_cache. NULL on 'no_match'.
    geometry     jsonb,
    distance_m   double precision,
    -- Request points the matcher left off the road as outliers, and
    -- whether the first and last were both placed. A line whose end point
    -- was not placed stops short of the stretch and is not drawn.
    unplaced     integer,
    ends_placed  boolean,
    router       text    NOT NULL,
    dataset      text    NOT NULL DEFAULT '',
    matched_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (points_hash, profile, radius_m)
);

-- A batch run records what it did for matching beside what it did for
-- routing (invariant 3). Existing rows predate matching: zeros.
ALTER TABLE route_runs
    ADD COLUMN stretches_found  integer NOT NULL DEFAULT 0,
    ADD COLUMN match_cache_hits integer NOT NULL DEFAULT 0,
    ADD COLUMN matched          integer NOT NULL DEFAULT 0,
    ADD COLUMN no_match         integer NOT NULL DEFAULT 0,
    ADD COLUMN match_failures   integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE route_runs
    DROP COLUMN stretches_found,
    DROP COLUMN match_cache_hits,
    DROP COLUMN matched,
    DROP COLUMN no_match,
    DROP COLUMN match_failures;
DROP TABLE match_cache;
