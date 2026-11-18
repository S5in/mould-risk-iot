// Package auth issues and checks login tokens (JWTs) and hashes passwords.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "mould-risk-api"

// MinSecretLen is the shortest signing secret we accept (32 bytes = 256 bits,
// the key size HS256 is designed for).
const MinSecretLen = 32

// Claims is what a token says about its holder. The organisation comes from
// here and nowhere else.
type Claims struct {
	OrgID string `json:"org"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// Tokens signs and verifies HS256 tokens with one server-side secret.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time // replaceable in tests
}

func NewTokens(secret string, ttl time.Duration) (*Tokens, error) {
	if len(secret) < MinSecretLen {
		return nil, fmt.Errorf("JWT secret must be at least %d characters", MinSecretLen)
	}
	return &Tokens{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

// Issue creates a signed token for a user. It returns the token and when it expires.
func (t *Tokens) Issue(userID, orgID, role string) (string, time.Time, error) {
	now := t.now()
	exp := now.Add(t.ttl)
	c := Claims{
		OrgID: orgID,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
	return s, exp, err
}

// ErrInvalidToken covers every reason a token is refused. Callers should not
// tell the client which reason it was.
var ErrInvalidToken = errors.New("invalid token")

// Verify checks signature, algorithm, issuer and expiry, and returns the claims.
func (t *Tokens) Verify(token string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(token, c,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), // refuses "none" and algorithm swaps
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || c.OrgID == "" || c.Subject == "" {
		return nil, ErrInvalidToken
	}
	return c, nil
}
