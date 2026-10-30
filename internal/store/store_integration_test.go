package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// Runs only when TEST_DB=1 and the PG* variables point at a database that has the
// migrations applied and the dev seed loaded (org demo-org, device sim-001).
func TestInsertReadingIntegration(t *testing.T) {
	if os.Getenv("TEST_DB") != "1" {
		t.Skip("set TEST_DB=1 (and PG* variables) to run database tests")
	}
	ctx := context.Background()
	s, err := New(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	at := time.Now().UTC().Truncate(time.Microsecond)

	if err := s.InsertReading(ctx, "demo-org", "sim-001", at, 21.5, 55.5); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := s.InsertReading(ctx, "demo-org", "sim-001", at, 21.5, 55.5); !errors.Is(err, ErrDuplicate) {
		t.Errorf("second insert = %v, want ErrDuplicate", err)
	}
	if err := s.InsertReading(ctx, "demo-org", "nope", at, 21.5, 55.5); !errors.Is(err, ErrUnknownDevice) {
		t.Errorf("unknown device = %v, want ErrUnknownDevice", err)
	}
	if err := s.InsertReading(ctx, "other-org", "sim-001", at, 21.5, 55.5); !errors.Is(err, ErrUnknownDevice) {
		t.Errorf("device under wrong org = %v, want ErrUnknownDevice", err)
	}
}
