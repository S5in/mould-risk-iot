package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/S5in/mould-risk-iot/internal/auth"
)

// User is what login needs to know about an account.
type User struct {
	ID           string
	OrgID        string
	Role         string
	PasswordHash string
}

// TokenIssuer creates a signed login token.
type TokenIssuer interface {
	Issue(userID, orgID, role string) (token string, expires time.Time, err error)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// login checks email + password and returns a token. Every failure gives the
// same 401 message, so the response never says whether the email exists.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10) // 4 KB is plenty for a login
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req loginRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body must be JSON with email and password")
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	u, found, err := s.db.UserByEmail(r.Context(), email)
	if err != nil {
		s.serverError(w, "login lookup", err)
		return
	}
	hash := ""
	if found {
		hash = u.PasswordHash
	}
	if !auth.CheckPassword(hash, req.Password) { // same work whether found or not
		s.log.Info("login failed") // never log the email or password
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, exp, err := s.tokens.Issue(u.ID, u.OrgID, u.Role)
	if err != nil {
		s.serverError(w, "issue token", err)
		return
	}
	s.log.Info("login ok", "user", u.ID)
	writeJSON(w, http.StatusOK, loginResponse{Token: token, ExpiresAt: exp.UTC()})
}
