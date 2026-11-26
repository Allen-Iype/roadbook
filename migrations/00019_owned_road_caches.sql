-- +goose Up
-- The two road caches belong to an account (phase 23, decided 2026-11-25).
-- Migration 11 left route_cache global and migration 18 followed it for
-- match_cache: "derived road geometry, never exposed per-user". But a row
-- is derived from one person's recorded positions, and deleting an account
-- is promised to remove that person's history. A road they travelled,
-- kept after they have gone, is a remnant of it. So each row now has an
-- owner, is read only by that owner, and DeleteUserData removes it.
--
-- Nothing is lost by not sharing: a match row is keyed by a hash of the
-- exact recorded points, and a route row by one gap's two ends, so two
-- accounts all but never ask the same question.
--
-- Existing rows become 'self' by the DEFAULT, which is right on every
-- single-user instance. An instance with accounts that has already run
-- the batch (it reads 'self' only) is in the same position.
ALTER TABLE route_cache ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);
ALTER TABLE match_cache ADD COLUMN user_id text NOT NULL DEFAULT 'self' REFERENCES users (id);

ALTER TABLE route_cache
    DROP CONSTRAINT route_cache_from_lat_e4_from_lon_e4_to_lat_e4_to_lon_e4_pro_key,
    ADD CONSTRAINT route_cache_owner_key
        UNIQUE (user_id, from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile);
ALTER TABLE match_cache
    DROP CONSTRAINT match_cache_points_hash_profile_radius_m_key,
    ADD CONSTRAINT match_cache_owner_key
        UNIQUE (user_id, points_hash, profile, radius_m);

-- +goose Down
-- Rows of other accounts cannot fold back into a global key.
DELETE FROM route_cache WHERE user_id <> 'self';
DELETE FROM match_cache WHERE user_id <> 'self';
ALTER TABLE match_cache
    DROP CONSTRAINT match_cache_owner_key,
    ADD CONSTRAINT match_cache_points_hash_profile_radius_m_key
        UNIQUE (points_hash, profile, radius_m);
ALTER TABLE route_cache
    DROP CONSTRAINT route_cache_owner_key,
    ADD CONSTRAINT route_cache_from_lat_e4_from_lon_e4_to_lat_e4_to_lon_e4_pro_key
        UNIQUE (from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile);
ALTER TABLE match_cache DROP COLUMN user_id;
ALTER TABLE route_cache DROP COLUMN user_id;
