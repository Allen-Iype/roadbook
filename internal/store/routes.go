package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"roadbook/internal/domain"
	"roadbook/internal/route"
)

// LookupRoutes reads cached answers for a set of keys. One query per key:
// a journey holds a handful of gaps and the batch a few dozen — building
// for thousands is theatre (CLAUDE.md).
func (s *Store) LookupRoutes(ctx context.Context, keys []route.Key) (map[route.Key]route.Cached, error) {
	out := make(map[route.Key]route.Cached, len(keys))
	for _, k := range keys {
		var (
			status              string
			geometry            []byte
			distanceM, duration *float64
		)
		err := s.pool.QueryRow(ctx, `
			SELECT status, geometry, distance_m, duration_s
			FROM route_cache
			WHERE from_lat_e4 = $1 AND from_lon_e4 = $2
			  AND to_lat_e4 = $3 AND to_lon_e4 = $4 AND profile = $5`,
			k.FromLatE4, k.FromLonE4, k.ToLatE4, k.ToLonE4, k.Profile,
		).Scan(&status, &geometry, &distanceM, &duration)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("store: lookup route: %w", err)
		}
		c := route.Cached{Status: status}
		if distanceM != nil {
			c.DistanceM = *distanceM
		}
		if duration != nil {
			c.DurationS = *duration
		}
		if geometry != nil {
			// [[lat, lon], ...] — the migration documents the order.
			var pairs [][]float64
			if err := json.Unmarshal(geometry, &pairs); err != nil {
				return nil, fmt.Errorf("store: route geometry: %w", err)
			}
			c.Points = make([]domain.LatLng, len(pairs))
			for i, p := range pairs {
				if len(p) < 2 {
					return nil, fmt.Errorf("store: route geometry: malformed pair at %d", i)
				}
				c.Points[i] = domain.LatLng{Lat: p[0], Lon: p[1]}
			}
		}
		out[k] = c
	}
	return out, nil
}

// SaveRoute upserts one answered question. The upsert (not insert-ignore)
// is the -refresh path: re-asking with a better extract replaces the row.
func (s *Store) SaveRoute(ctx context.Context, k route.Key, c route.Cached, router, dataset string) error {
	var geometry []byte
	var distanceM, durationS *float64
	if c.Status == route.StatusRouted {
		pairs := make([][]float64, len(c.Points))
		for i, p := range c.Points {
			pairs[i] = []float64{p.Lat, p.Lon}
		}
		b, err := json.Marshal(pairs)
		if err != nil {
			return err
		}
		geometry = b
		distanceM, durationS = &c.DistanceM, &c.DurationS
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO route_cache
			(from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile,
			 status, geometry, distance_m, duration_s, router, dataset)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (from_lat_e4, from_lon_e4, to_lat_e4, to_lon_e4, profile)
		DO UPDATE SET status = EXCLUDED.status, geometry = EXCLUDED.geometry,
			distance_m = EXCLUDED.distance_m, duration_s = EXCLUDED.duration_s,
			router = EXCLUDED.router, dataset = EXCLUDED.dataset,
			routed_at = now()`,
		k.FromLatE4, k.FromLonE4, k.ToLatE4, k.ToLonE4, k.Profile,
		c.Status, geometry, distanceM, durationS, router, dataset)
	if err != nil {
		return fmt.Errorf("store: save route: %w", err)
	}
	return nil
}

// RouteRunCounts is a batch invocation's report, persisted (invariant 3
// applied to routing: a batch that writes derived data records what
// produced it).
type RouteRunCounts struct {
	GapsFound int
	CacheHits int
	Routed    int
	NoRoute   int
	Failures  int
}

func (s *Store) InsertRouteRun(ctx context.Context, router, dataset string, params any, c RouteRunCounts) (int64, error) {
	p, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
		INSERT INTO route_runs (router, dataset, params, gaps_found, cache_hits, routed, no_route, failures)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		router, dataset, p, c.GapsFound, c.CacheHits, c.Routed, c.NoRoute, c.Failures,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: insert route run: %w", err)
	}
	return id, nil
}
