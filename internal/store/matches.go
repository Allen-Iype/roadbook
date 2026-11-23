package store

import (
	"context"
	"encoding/json"
	"fmt"

	"roadbook/internal/domain"
	"roadbook/internal/route"
)

// LookupMatches reads cached match answers for a set of keys. Unlike
// LookupRoutes this is one query per (profile, radius) — in practice one
// query — because a trip holds a recorded stretch for every gap and then
// some (the archive: 4,852 stretches over 67 trips), and a page load that
// made a round trip per stretch would be felt.
func (s *Store) LookupMatches(ctx context.Context, keys []route.MatchKey) (map[route.MatchKey]route.CachedMatch, error) {
	type group struct {
		profile string
		radiusM int32
	}
	hashes := map[group][]string{}
	for _, k := range keys {
		g := group{k.Profile, k.RadiusM}
		hashes[g] = append(hashes[g], k.PointsHash)
	}
	out := make(map[route.MatchKey]route.CachedMatch, len(keys))
	for g, hs := range hashes {
		rows, err := s.pool.Query(ctx, `
			SELECT points_hash, status, geometry, distance_m, unplaced, ends_placed
			FROM match_cache
			WHERE profile = $1 AND radius_m = $2 AND points_hash = ANY($3)`,
			g.profile, g.radiusM, hs)
		if err != nil {
			return nil, fmt.Errorf("store: lookup matches: %w", err)
		}
		for rows.Next() {
			var (
				hash, status string
				geometry     []byte
				distanceM    *float64
				unplaced     *int
				endsPlaced   *bool
			)
			if err := rows.Scan(&hash, &status, &geometry, &distanceM, &unplaced, &endsPlaced); err != nil {
				rows.Close()
				return nil, fmt.Errorf("store: lookup matches: %w", err)
			}
			c := route.CachedMatch{Status: status}
			if distanceM != nil {
				c.DistanceM = *distanceM
			}
			if unplaced != nil {
				c.Unplaced = *unplaced
			}
			if endsPlaced != nil {
				c.EndsPlaced = *endsPlaced
			}
			if geometry != nil {
				// [[lat, lon], ...] — the migration documents the order.
				var pairs [][]float64
				if err := json.Unmarshal(geometry, &pairs); err != nil {
					rows.Close()
					return nil, fmt.Errorf("store: match geometry: %w", err)
				}
				c.Points = make([]domain.LatLng, len(pairs))
				for i, p := range pairs {
					if len(p) < 2 {
						rows.Close()
						return nil, fmt.Errorf("store: match geometry: malformed pair at %d", i)
					}
					c.Points[i] = domain.LatLng{Lat: p[0], Lon: p[1]}
				}
			}
			out[route.MatchKey{PointsHash: hash, Profile: g.profile, RadiusM: g.radiusM}] = c
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("store: lookup matches: %w", err)
		}
	}
	return out, nil
}

// SaveMatch upserts one answered question. The upsert is the -refresh
// path, as in SaveRoute. A no_match row carries no geometry or figures,
// whatever the caller passed.
func (s *Store) SaveMatch(ctx context.Context, tr route.Trace, c route.CachedMatch, router, dataset string) error {
	var geometry []byte
	var distanceM *float64
	var unplaced *int
	var endsPlaced *bool
	if c.Status == route.StatusMatched {
		pairs := make([][]float64, len(c.Points))
		for i, p := range c.Points {
			pairs[i] = []float64{p.Lat, p.Lon}
		}
		b, err := json.Marshal(pairs)
		if err != nil {
			return err
		}
		geometry = b
		distanceM, unplaced, endsPlaced = &c.DistanceM, &c.Unplaced, &c.EndsPlaced
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO match_cache
			(points_hash, profile, radius_m, points,
			 status, geometry, distance_m, unplaced, ends_placed, router, dataset)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (points_hash, profile, radius_m)
		DO UPDATE SET points = EXCLUDED.points, status = EXCLUDED.status,
			geometry = EXCLUDED.geometry, distance_m = EXCLUDED.distance_m,
			unplaced = EXCLUDED.unplaced, ends_placed = EXCLUDED.ends_placed,
			router = EXCLUDED.router, dataset = EXCLUDED.dataset,
			matched_at = now()`,
		tr.Key.PointsHash, tr.Key.Profile, tr.Key.RadiusM, len(tr.Points),
		c.Status, geometry, distanceM, unplaced, endsPlaced, router, dataset)
	if err != nil {
		return fmt.Errorf("store: save match: %w", err)
	}
	return nil
}
