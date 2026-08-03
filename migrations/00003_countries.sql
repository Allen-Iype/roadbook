-- +goose Up
-- PostGIS enters the stack here, for point-in-polygon country attribution
-- only (BRIEF §1.4). Leg segmentation stays in the pure Go core — that
-- decision is closed; PostGIS provides the geometry type, the GiST index,
-- and ST_Contains, nothing more. Creating the extension needs database
-- superuser or CREATE privilege; on the brew install the local user has it.
CREATE EXTENSION IF NOT EXISTS postgis;

-- Reference data, not observations: country polygons loaded wholesale by
-- `roadbook countries` from the bundled Natural Earth admin-0 file (public
-- domain), or from a higher-resolution file on disk. The loader replaces the
-- table's contents in one transaction, so the table always mirrors exactly
-- one source file — it is never accumulated into like the observation tables.
CREATE TABLE countries (
    iso_code text PRIMARY KEY,
    name     text NOT NULL,
    geom     geometry(MultiPolygon, 4326) NOT NULL
);
CREATE INDEX countries_geom_idx ON countries USING GIST (geom);

-- +goose Down
DROP TABLE countries;
DROP EXTENSION IF EXISTS postgis;
