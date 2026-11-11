package store

import (
	"context"

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
