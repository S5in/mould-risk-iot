// Package store writes validated readings to PostgreSQL/TimescaleDB.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUnknownDevice means no device with that org slug and device key exists.
var ErrUnknownDevice = errors.New("unknown device")

// ErrDuplicate means a reading for that device and time is already stored.
var ErrDuplicate = errors.New("duplicate reading")

// Store inserts readings.
type Store struct {
	pool *pgxpool.Pool
}

// New connects using a connection string. An empty string uses the PG* environment variables.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the connections.
func (s *Store) Close() { s.pool.Close() }

// The device is looked up by org slug and device key, and org_id, room_id and device_id
// come from the devices table, never from the message.
const insertSQL = `
INSERT INTO readings (device_id, org_id, room_id, device_time, temperature_c, humidity_rh)
SELECT d.id, d.org_id, d.room_id, $3, $4, $5
FROM devices d
JOIN organisations o ON o.id = d.org_id
WHERE o.slug = $1 AND d.device_key = $2
ON CONFLICT DO NOTHING`

const deviceExistsSQL = `
SELECT EXISTS (
	SELECT 1 FROM devices d JOIN organisations o ON o.id = d.org_id
	WHERE o.slug = $1 AND d.device_key = $2)`

// InsertReading stores one reading. It returns ErrUnknownDevice or ErrDuplicate when nothing was stored.
func (s *Store) InsertReading(ctx context.Context, orgSlug, deviceKey string, at time.Time, tempC, humRH float64) error {
	tag, err := s.pool.Exec(ctx, insertSQL, orgSlug, deviceKey, at, tempC, humRH)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, deviceExistsSQL, orgSlug, deviceKey).Scan(&exists); err != nil {
		return fmt.Errorf("check device: %w", err)
	}
	if !exists {
		return ErrUnknownDevice
	}
	return ErrDuplicate
}
