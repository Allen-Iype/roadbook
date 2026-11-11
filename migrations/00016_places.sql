-- +goose Up
-- Named settlements (cities and towns) beside the countries and states
-- tables: the same kind of row — reference data loaded wholesale by
-- `roadbook places` from the embedded OpenStreetMap list, replaced in one
-- transaction, never accumulated, the same for every account (phase 18
-- BRIEF §3, §4D). A stop is named from the nearest row.
--
-- The position is geography, not geometry like the two polygon tables.
-- Those answer "is this point inside?", where flat coordinates are exact.
-- This table answers "which row is nearest, and how far?": on flat
-- coordinates nearest means nearest in degrees, and a degree of longitude
-- is 111 km at the equator and 45 km at 66° north — the wrong town wins.
-- geography measures on the globe, in metres, and its GiST index serves
-- nearest-neighbour search (ORDER BY geog <-> point).
--
-- The key is the OpenStreetMap element with its type letter ("n60741085"),
-- so a later list that also carries ways or relations cannot collide.
-- Nothing references this table by foreign key, and nothing should: it is
-- replaced wholesale when a new list is loaded, so a row that wants to
-- remember a place keeps the id and the name as written at the time (the
-- same caution states.country_code carries).
CREATE TABLE places (
    id         text PRIMARY KEY,
    name       text   NOT NULL,
    class      text   NOT NULL,
    population bigint NOT NULL DEFAULT 0,
    -- Which list the row came from ('osm'); served with every name.
    source     text   NOT NULL,
    geog       geography(Point, 4326) NOT NULL
);
CREATE INDEX places_geog_idx ON places USING GIST (geog);

-- +goose Down
DROP TABLE places;
