-- +goose Up
-- Admin-1 polygons (states, provinces, regions) beside the countries table:
-- the same kind of row — reference data loaded wholesale by `roadbook
-- states` from the embedded Natural Earth 1:10m file, replaced in one
-- transaction, never accumulated (phase 14 BRIEF §1, §3B). Attribution is
-- the same one indexed ST_Contains query the countries table answers.
--
-- The key is Natural Earth's adm1_code: unique across the file, where the
-- ISO 3166-2 column is not (eight codes repeat upstream). country_code
-- aligns with countries.iso_code by construction (same two-letter-else-
-- ADM0_A3 fallback) but is not a foreign key: the two tables are loaded
-- independently, and an operator's higher-resolution country file may name
-- a different set.
CREATE TABLE states (
    adm1_code    text PRIMARY KEY,
    iso_3166_2   text NOT NULL,
    name         text NOT NULL,
    country_code text NOT NULL,
    geom         geometry(MultiPolygon, 4326) NOT NULL
);
CREATE INDEX states_geom_idx ON states USING GIST (geom);

-- +goose Down
DROP TABLE states;
