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

// Pinger is anything that can say whether the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server wires dependencies to routes.
type Server struct {
	db  Pinger
	log *slog.Logger
}

func New(db Pinger, log *slog.Logger) *Server {
	return &Server{db: db, log: log}
}

// Handler returns the router. Later steps add /v1/... routes here.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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
