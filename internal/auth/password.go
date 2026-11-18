package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost 12 takes roughly 0.2-0.3 s per check on a laptop: slow enough to
// make guessing expensive, fast enough for a login.
const BcryptCost = 12

// MinPasswordLen follows NIST SP 800-63B's minimum of 8 characters.
const MinPasswordLen = 8

// bcrypt only uses the first 72 bytes of a password.
const maxPasswordBytes = 72

func HashPassword(pw string) (string, error) {
	if len(pw) < MinPasswordLen {
		return "", errors.New("password must be at least 8 characters")
	}
	if len(pw) > maxPasswordBytes {
		return "", errors.New("password must be at most 72 bytes")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), BcryptCost)
	return string(h), err
}

// dummyHash is compared against when the email does not exist, so a login for
// an unknown email takes as long as one for a known email. Otherwise response
// time would reveal which emails have accounts.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), BcryptCost)

// CheckPassword reports whether pw matches hash. Pass hash = "" for an
// unknown user; it still does the same amount of work and returns false.
func CheckPassword(hash, pw string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
