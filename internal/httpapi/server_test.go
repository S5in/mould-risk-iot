package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, db Pinger, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	s := New(db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthOK(t *testing.T) {
	rec := get(t, fakeDB{}, "GET", "/health")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestHealthDBDown(t *testing.T) {
	rec := get(t, fakeDB{err: errors.New("secret connection string")}, "GET", "/health")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("error detail leaked to client")
	}
}

func TestUnknownRoute404(t *testing.T) {
	if rec := get(t, fakeDB{}, "GET", "/nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestWrongMethod405(t *testing.T) {
	if rec := get(t, fakeDB{}, "POST", "/health"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
}
