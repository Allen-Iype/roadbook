package store

import (
	"context"

	"roadbook/internal/domain"
	"roadbook/internal/journey"
	"roadbook/internal/places"
)

// ReplacePlaces loads the list of named settlements wholesale —
// delete-then-insert in one transaction, so the table always mirrors exactly
// one source list — as ReplaceStates does. Reference data, not observations.
//
// One INSERT for the whole list, not one per row: the list is about 114,000
// rows (twenty-five times the states table), and this runs at every fresh
// instance's first start. The columns travel as six parallel arrays and
// unnest() turns them back into rows inside Postgres, where the position is
// built — a round trip per statement instead of per place.
func (s *Store) ReplacePlaces(ctx context.Context, list []places.Place, source string) error {
	ids := make([]string, len(list))
	names := make([]string, len(list))
	classes := make([]string, len(list))
	pops := make([]int64, len(list))
	lons := make([]float64, len(list))
	lats := make([]float64, len(list))
	for i, p := range list {
		ids[i], names[i], classes[i], pops[i], lons[i], lats[i] = p.ID, p.Name, p.Class, p.Population, p.Lon, p.Lat
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM places`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO places (id, name, class, population, source, geog)
		SELECT p.id, p.name, p.class, p.population, $7,
		       ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326)::geography
		FROM unnest($1::text[], $2::text[], $3::text[], $4::bigint[], $5::float8[], $6::float8[])
		     AS p(id, name, class, population, lon, lat)`,
		ids, names, classes, pops, lons, lats, source); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CountPlaces reports how many places are loaded — the `places -if-empty`
// startup check, the countries and states precedent.
func (s *Store) CountPlaces(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM places`).Scan(&n)
	return n, err
}

// PlacesForPoints finds, for each point, the nearest row of the places
// table and the distance to it — one query for the whole list, answers in
// the points' order. An empty input or an unloaded table yields answers
// with Found false, never an error: no names is a state the page shows.
//
// How the query reads. unnest turns the two arrays back into one row per
// point, numbered by WITH ORDINALITY. CROSS JOIN LATERAL then runs the
// inner SELECT once per point, and may refer to that point — a plain join
// could not. The inner SELECT is the nearest-neighbour search: ORDER BY
// geog <-> point LIMIT 1 walks the GiST index outward from the point and
// stops at the first row, instead of measuring every place.
//
// The distance is on a sphere (ST_Distance's last argument false), the same
// model as geo.HaversineM, which every other distance in Roadbook uses; the
// spheroid's extra accuracy is a fraction of a percent, far below what
// "near" means.
func (s *Store) PlacesForPoints(ctx context.Context, pts []domain.LatLng) ([]journey.NearestPlace, error) {
	out := make([]journey.NearestPlace, len(pts))
	if len(pts) == 0 {
		return out, nil
	}
	lons := make([]float64, len(pts))
	lats := make([]float64, len(pts))
	for i, p := range pts {
		lons[i] = p.Lon
		lats[i] = p.Lat
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.ord, n.name, n.source, n.dist_m
		FROM unnest($1::float8[], $2::float8[]) WITH ORDINALITY AS p(lon, lat, ord)
		CROSS JOIN LATERAL (
			SELECT pl.name, pl.source,
			       ST_Distance(pl.geog, ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326)::geography, false) AS dist_m
			FROM places pl
			ORDER BY pl.geog <-> ST_SetSRID(ST_MakePoint(p.lon, p.lat), 4326)::geography
			LIMIT 1
		) n`, lons, lats)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ord int
		var n journey.NearestPlace
		var distM float64
		if err := rows.Scan(&ord, &n.Name, &n.Source, &distM); err != nil {
			return nil, err
		}
		n.Found = true
		n.DistanceKm = distM / 1000
		out[ord-1] = n
	}
	return out, rows.Err()
}
