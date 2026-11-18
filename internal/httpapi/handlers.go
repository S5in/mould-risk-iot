package httpapi

import (
	"net/http"
	"strconv"
	"time"
)

const (
	defaultHours = 24
	maxHours     = 24 * 31
	maxReadings  = 2000
)

func (s *Server) listProperties(w http.ResponseWriter, r *http.Request) {
	org, ok := requireOrg(w, r)
	if !ok {
		return
	}
	ps, err := s.db.ListProperties(r.Context(), org)
	if err != nil {
		s.serverError(w, "list properties", err)
		return
	}
	if ps == nil {
		ps = []Property{} // JSON [] not null
	}
	writeJSON(w, http.StatusOK, ps)
}

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	org, ok := requireOrg(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rooms, found, err := s.db.ListRooms(r.Context(), org, id)
	if err != nil {
		s.serverError(w, "list rooms", err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "not found") // also for other orgs' IDs
		return
	}
	if rooms == nil {
		rooms = []Room{}
	}
	writeJSON(w, http.StatusOK, rooms)
}

func (s *Server) latestReading(w http.ResponseWriter, r *http.Request) {
	org, ok := requireOrg(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rd, found, err := s.db.LatestReading(r.Context(), org, id)
	if err != nil {
		s.serverError(w, "latest reading", err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, rd)
}

func (s *Server) readings(w http.ResponseWriter, r *http.Request) {
	org, ok := requireOrg(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !uuidRE.MatchString(id) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	hours := defaultHours
	if h := r.URL.Query().Get("hours"); h != "" {
		n, err := strconv.Atoi(h)
		if err != nil || n < 1 || n > maxHours {
			writeError(w, http.StatusBadRequest, "hours must be a whole number from 1 to "+strconv.Itoa(maxHours))
			return
		}
		hours = n
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	rs, found, err := s.db.Readings(r.Context(), org, id, since, maxReadings)
	if err != nil {
		s.serverError(w, "readings", err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if rs == nil {
		rs = []Reading{}
	}
	writeJSON(w, http.StatusOK, rs)
}

// serverError logs the real error and gives the client a generic 500.
func (s *Server) serverError(w http.ResponseWriter, what string, err error) {
	s.log.Error("request failed", "op", what, "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
