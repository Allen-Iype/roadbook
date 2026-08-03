package store

import (
	"context"

	"roadbook/internal/countries"
	"roadbook/internal/domain"
)

// ReplaceCountries loads country polygons wholesale: delete-then-insert in
// one transaction, so the table always mirrors exactly one source file.
// Countries are reference data, not observations — invariant 2's
// never-mutate rule protects inputs, and this table is derived from a
// committed public-domain file that can always be reloaded.
func (s *Store) ReplaceCountries(ctx context.Context, list []countries.Country) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM countries`); err != nil {
		return err
	}
	// ST_GeomFromGeoJSON parses the feature geometry PostGIS-side; ST_Multi
	// promotes the single-Polygon countries to MultiPolygon so one column
	// type fits every feature; SRID 4326 is set explicitly rather than
	// trusting the file's CRS annotation.
	for _, c := range list {
		if _, err := tx.Exec(ctx,
			`INSERT INTO countries (iso_code, name, geom)
			 VALUES ($1, $2, ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($3::json), 4326)))`,
			c.ISOCode, c.Name, string(c.GeomJSON)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CountryRef is a country a journey crossed.
type CountryRef struct {
	ISOCode string
	Name    string
}

// CountriesForPoints attributes route points to countries in one indexed
// query (BRIEF §1.4): unnest the coordinate arrays, ST_Contains against the
// GiST-indexed polygons, group per country, order by the first point that
// hit it — so the returned list reads in journey order, not alphabetically.
// Points over ocean or outside every polygon (possible near borders at 110m
// resolution) simply attribute to nothing. Empty input, or an unloaded
// countries table, yields an empty result — the caller renders nothing
// rather than failing.
func (s *Store) CountriesForPoints(ctx context.Context, pts []domain.LatLng) ([]CountryRef, error) {
	if len(pts) == 0 {
		return nil, nil
	}
	lons := make([]float64, len(pts))
	lats := make([]float64, len(pts))
	for i, p := range pts {
		lons[i] = p.Lon
		lats[i] = p.Lat
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.iso_code, c.name
		FROM unnest($1::float8[], $2::float8[]) WITH ORDINALITY AS p(lon, lat, ord)
		JOIN countries c
		  ON ST_Contains(c.geom, ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326))
		GROUP BY c.iso_code, c.name
		ORDER BY min(p.ord)`, lons, lats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CountryRef
	for rows.Next() {
		var c CountryRef
		if err := rows.Scan(&c.ISOCode, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
