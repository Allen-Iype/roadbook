package store

import (
	"context"

	"roadbook/internal/domain"
	"roadbook/internal/states"
)

// ReplaceStates loads admin-1 polygons wholesale — delete-then-insert in
// one transaction, so the table always mirrors exactly one source file —
// exactly as ReplaceCountries does. Reference data, not observations.
func (s *Store) ReplaceStates(ctx context.Context, list []states.State) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM states`); err != nil {
		return err
	}
	for _, st := range list {
		if _, err := tx.Exec(ctx,
			`INSERT INTO states (adm1_code, iso_3166_2, name, country_code, geom)
			 VALUES ($1, $2, $3, $4, ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($5::json), 4326)))`,
			st.Code, st.ISO3166_2, st.Name, st.CountryCode, string(st.GeomJSON)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CountStates reports how many admin-1 polygons are loaded — the
// `states -if-empty` startup check, the countries precedent.
func (s *Store) CountStates(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM states`).Scan(&n)
	return n, err
}

// StateRef is an admin-1 region a journey crossed.
type StateRef struct {
	Code        string // adm1_code
	Name        string
	CountryCode string
}

// StatesForPoints attributes route points to admin-1 polygons in one indexed
// query, ordered by the first point that hit each — journey order, like
// CountriesForPoints. Points outside every polygon attribute to nothing;
// an empty input or an unloaded table yields an empty result.
func (s *Store) StatesForPoints(ctx context.Context, pts []domain.LatLng) ([]StateRef, error) {
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
		SELECT st.adm1_code, st.name, st.country_code
		FROM unnest($1::float8[], $2::float8[]) WITH ORDINALITY AS p(lon, lat, ord)
		JOIN states st
		  ON ST_Contains(st.geom, ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326))
		GROUP BY st.adm1_code, st.name, st.country_code
		ORDER BY min(p.ord)`, lons, lats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StateRef
	for rows.Next() {
		var r StateRef
		if err := rows.Scan(&r.Code, &r.Name, &r.CountryCode); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
