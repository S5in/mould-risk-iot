// Package apistore is the PostgreSQL implementation of httpapi.Store.
// Every query filters by org_id: that is where tenant isolation is enforced.
package apistore

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/S5in/mould-risk-iot/internal/httpapi"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// OrgIDBySlug is used only by the temporary dev login stand-in.
func (s *Store) OrgIDBySlug(ctx context.Context, slug string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM organisations WHERE slug = $1`, slug).Scan(&id)
	return id, err
}

func (s *Store) ListProperties(ctx context.Context, orgID string) ([]httpapi.Property, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, name FROM properties WHERE org_id = $1 ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []httpapi.Property
	for rows.Next() {
		var p httpapi.Property
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListRooms(ctx context.Context, orgID, propertyID string) ([]httpapi.Room, bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM properties WHERE id = $1 AND org_id = $2)`,
		propertyID, orgID).Scan(&exists)
	if err != nil || !exists {
		return nil, false, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, name FROM rooms WHERE property_id = $1 AND org_id = $2 ORDER BY name`,
		propertyID, orgID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var out []httpapi.Room
	for rows.Next() {
		var r httpapi.Room
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, false, err
		}
		out = append(out, r)
	}
	return out, true, rows.Err()
}

func (s *Store) roomInOrg(ctx context.Context, orgID, roomID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM rooms WHERE id = $1 AND org_id = $2)`,
		roomID, orgID).Scan(&exists)
	return exists, err
}

func (s *Store) LatestReading(ctx context.Context, orgID, roomID string) (httpapi.Reading, bool, error) {
	var r httpapi.Reading
	err := s.pool.QueryRow(ctx,
		`SELECT device_time, temperature_c, humidity_rh
   FROM readings
  WHERE room_id = $1 AND org_id = $2
  ORDER BY device_time DESC LIMIT 1`, roomID, orgID).
		Scan(&r.Time, &r.TemperatureC, &r.HumidityRH)
	if err == pgx.ErrNoRows {
		return r, false, nil // no readings, or not this org's room: same answer
	}
	return r, err == nil, err
}

func (s *Store) Readings(ctx context.Context, orgID, roomID string, since time.Time, limit int) ([]httpapi.Reading, bool, error) {
	ok, err := s.roomInOrg(ctx, orgID, roomID)
	if err != nil || !ok {
		return nil, false, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT device_time, temperature_c, humidity_rh
   FROM readings
  WHERE room_id = $1 AND org_id = $2 AND device_time >= $3
  ORDER BY device_time DESC LIMIT $4`, roomID, orgID, since, limit)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var out []httpapi.Reading
	for rows.Next() {
		var r httpapi.Reading
		if err := rows.Scan(&r.Time, &r.TemperatureC, &r.HumidityRH); err != nil {
			return nil, false, err
		}
		out = append(out, r)
	}
	return out, true, rows.Err()
}
