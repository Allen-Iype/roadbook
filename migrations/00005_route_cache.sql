-- +goose Up
-- The routing cache (phase 3 BRIEF §3C): the one derived table in the
-- schema, because what it holds cannot be re-derived from what the database
-- holds — route answers come from an external engine that is slow,
-- rate-limited, possibly absent, and non-deterministic across OSM
-- snapshots. Persisting the answers is what removes the runtime dependency:
-- `roadbook route` (batch) writes, `roadbook serve` only reads. Only route
-- ANSWERS live here, never OSM road data — the network stays OSRM's own
-- files on disk. TRUNCATE is always safe; decisions never reference this.
--
-- The key is the rounded endpoint pair as e4 fixed-point integers
-- (coordinate × 10^4, ~11 m — below source accuracy). Integers because the
-- key must match by equality and 4-decimal values are not exactly
-- representable in binary floating point. The rounded coordinate is itself
-- what was routed, so a hit is exact. Directional: one-way roads exist.
CREATE TABLE route_cache (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    from_lat_e4  integer NOT NULL,
    from_lon_e4  integer NOT NULL,
    to_lat_e4    integer NOT NULL,
    to_lon_e4    integer NOT NULL,
    profile      text    NOT NULL,
    -- 'no_route' is a remembered negative: the road network cannot connect
    -- these points (patchy OSM coverage — designed behaviour, the gap stays
    -- visibly unknown). Operational failures are never cached.
    status       text    NOT NULL CHECK (status IN ('routed', 'no_route')),
    -- [[lat, lon], ...] — lat-first, the repository's domain convention;
    -- the lon-lat flip lives only in the frontend's GeoJSON conversion.
    -- NULL when status is 'no_route'.
    geometry     jsonb,
    distance_m   double precision,
    -- Unused by the renderer, kept deliberately: routed travel time
    -- exceeding the gap's duration proves a wrong route regardless of
    -- distance, and replay animation needs per-leg timing (BRIEF §3C).
    duration_s   double precision,
    -- Provenance in the spirit of invariant 3: which implementation and URL
    -- answered, against which OSM snapshot (OSRM's data_version, or the
    -- extract name stated at batch time via -dataset).
    router       text    NOT NULL,
    dataset      text    NOT NULL DEFAULT '',
    routed_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile)
);

-- Every batch invocation records what it did (invariant 3 applied to
-- routing): the accumulating rows turn "is OSM coverage improving on these
-- roads?" from anecdote into a query.
CREATE TABLE route_runs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ran_at      timestamptz NOT NULL DEFAULT now(),
    router      text  NOT NULL,
    dataset     text  NOT NULL DEFAULT '',
    params      jsonb NOT NULL,
    gaps_found  integer NOT NULL,
    cache_hits  integer NOT NULL,
    routed      integer NOT NULL,
    no_route    integer NOT NULL,
    failures    integer NOT NULL
);

-- +goose Down
DROP TABLE route_runs;
DROP TABLE route_cache;
