package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	orgA  = "11111111-1111-1111-1111-111111111111"
	orgB  = "22222222-2222-2222-2222-222222222222"
	propA = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	roomA = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

// fakeStore holds one property and one room, both owned by orgA, and
// behaves like the real store: asking as any other org finds nothing.
type fakeStore struct {
	pingErr error
	err     error
}

func (f fakeStore) Ping(context.Context) error { return f.pingErr }

func (f fakeStore) ListProperties(_ context.Context, org string) ([]Property, error) {
	if f.err != nil {
		return nil, f.err
	}
	if org != orgA {
		return nil, nil
	}
	return []Property{{ID: propA, Name: "House"}}, nil
}

func (f fakeStore) ListRooms(_ context.Context, org, prop string) ([]Room, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	if org != orgA || prop != propA {
		return nil, false, nil
	}
	return []Room{{ID: roomA, Name: "Bedroom"}}, true, nil
}

func (f fakeStore) LatestReading(_ context.Context, org, room string) (Reading, bool, error) {
	if org != orgA || room != roomA {
		return Reading{}, false, f.err
	}
	return Reading{Time: time.Unix(0, 0).UTC(), TemperatureC: 20.5, HumidityRH: 65}, true, nil
}

func (f fakeStore) Readings(_ context.Context, org, room string, _ time.Time, _ int) ([]Reading, bool, error) {
	if org != orgA || room != roomA {
		return nil, false, nil
	}
	return nil, true, nil // room exists, no readings in window
}

func do(t *testing.T, db Store, org, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doBody(t, db, org, method, path, "")
}

func doBody(t *testing.T, db Store, org, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	s := New(db, fakeTokens{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if org != "" {
		req = req.WithContext(WithOrg(req.Context(), org))
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func want(t *testing.T, rec *httptest.ResponseRecorder, code int) {
	t.Helper()
	if rec.Code != code {
		t.Fatalf("want %d, got %d: %s", code, rec.Code, rec.Body.String())
	}
}

func TestHealthOK(t *testing.T) {
	rec := do(t, fakeStore{}, "", "GET", "/health")
	want(t, rec, 200)
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatal(rec.Body.String())
	}
}

func TestHealthDBDown(t *testing.T) {
	rec := do(t, fakeStore{pingErr: errors.New("secret connection string")}, "", "GET", "/health")
	want(t, rec, 503)
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("error detail leaked to client")
	}
}

func TestUnknownRoute404(t *testing.T) { want(t, do(t, fakeStore{}, "", "GET", "/nope"), 404) }
func TestWrongMethod405(t *testing.T)  { want(t, do(t, fakeStore{}, "", "POST", "/health"), 405) }
func TestNoOrgIs401(t *testing.T)      { want(t, do(t, fakeStore{}, "", "GET", "/v1/properties"), 401) }
func TestPropertiesOwnOrg(t *testing.T) {
	rec := do(t, fakeStore{}, orgA, "GET", "/v1/properties")
	want(t, rec, 200)
	if !strings.Contains(rec.Body.String(), "House") {
		t.Fatal(rec.Body.String())
	}
}

func TestPropertiesOtherOrgSeesEmptyList(t *testing.T) {
	rec := do(t, fakeStore{}, orgB, "GET", "/v1/properties")
	want(t, rec, 200)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("want [], got %s", rec.Body.String())
	}
}

func TestOtherOrgGets404NotForbidden(t *testing.T) {
	for _, p := range []string{
		"/v1/properties/" + propA + "/rooms",
		"/v1/rooms/" + roomA + "/readings/latest",
		"/v1/rooms/" + roomA + "/readings",
	} {
		want(t, do(t, fakeStore{}, orgB, "GET", p), 404)
		want(t, do(t, fakeStore{}, orgA, "GET", p), 200)
	}
}

func TestBadIDIs404(t *testing.T) {
	want(t, do(t, fakeStore{}, orgA, "GET", "/v1/rooms/not-a-uuid/readings"), 404)
}

func TestHoursValidation(t *testing.T) {
	base := "/v1/rooms/" + roomA + "/readings?hours="
	for _, h := range []string{"0", "-3", "abc", "99999"} {
		want(t, do(t, fakeStore{}, orgA, "GET", base+h), 400)
	}
	want(t, do(t, fakeStore{}, orgA, "GET", base+"6"), 200)
}

func TestEmptyReadingsIsEmptyArray(t *testing.T) {
	rec := do(t, fakeStore{}, orgA, "GET", "/v1/rooms/"+roomA+"/readings")
	want(t, rec, 200)
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("want [], got %s", rec.Body.String())
	}
}

func TestStoreErrorIs500WithoutDetail(t *testing.T) {
	rec := do(t, fakeStore{err: errors.New("secret sql")}, orgA, "GET", "/v1/properties")
	want(t, rec, 500)
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("error detail leaked")
	}
}
