// Package httpapi holds the REST API's HTTP handlers. It depends on small
// interfaces, not on the database driver, so handlers can be tested with fakes.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Property, Room and Reading are the shapes the API returns as JSON.
type Property struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Reading struct {
	Time         time.Time `json:"time"`
	TemperatureC float64   `json:"temperature_c"`
	HumidityRH   float64   `json:"humidity_rh"`
}

// Store is everything the handlers need from the database. Every method that
// returns tenant data takes orgID and must filter by it: that is the
// organisation-isolation rule. found=false means "no such thing in this
// organisation", which the handlers turn into 404.
type Store interface {
	Ping(ctx context.Context) error
	ListProperties(ctx context.Context, orgID string) ([]Property, error)
	ListRooms(ctx context.Context, orgID, propertyID string) (rooms []Room, found bool, err error)
	LatestReading(ctx context.Context, orgID, roomID string) (r Reading, found bool, err error)
	Readings(ctx context.Context, orgID, roomID string, since time.Time, limit int) (rs []Reading, found bool, err error)
}

// Server wires dependencies to routes.
type Server struct {
	db  Store
	log *slog.Logger
}

func New(db Store, log *slog.Logger) *Server {
	return &Server{db: db, log: log}
}

// Handler returns the router. /health is public; everything under /v1/ needs
// an organisation in the request context (set by the auth middleware).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /v1/properties", s.listProperties)
	mux.HandleFunc("GET /v1/properties/{id}/rooms", s.listRooms)
	mux.HandleFunc("GET /v1/rooms/{id}/readings/latest", s.latestReading)
	mux.HandleFunc("GET /v1/rooms/{id}/readings", s.readings)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// health reports 200 if the database answers within 2 seconds, otherwise 503.
// It never returns error details to the caller (they could leak internals);
// the detail goes to the server log instead.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		s.log.Error("health: database ping failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
