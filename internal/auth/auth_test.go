package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = strings.Repeat("k", 40) // test-only key, deliberately not random

func newT(t *testing.T) *Tokens {
	t.Helper()
	tk, err := NewTokens(secret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func TestShortSecretRefused(t *testing.T) {
	if _, err := NewTokens("short", time.Hour); err == nil {
		t.Fatal("want error for short secret")
	}
}

func TestIssueVerifyRoundTrip(t *testing.T) {
	tk := newT(t)
	s, exp, err := tk.Issue("user-1", "org-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) < 59*time.Minute {
		t.Fatalf("expiry too soon: %v", exp)
	}
	c, err := tk.Verify(s)
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "user-1" || c.OrgID != "org-1" || c.Role != "admin" {
		t.Fatalf("wrong claims: %+v", c)
	}
}

func TestExpiredRefused(t *testing.T) {
	tk := newT(t)
	s, _, _ := tk.Issue("u", "o", "viewer")
	tk.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err := tk.Verify(s); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestWrongSecretRefused(t *testing.T) {
	s, _, _ := newT(t).Issue("u", "o", "viewer")
	other, _ := NewTokens(strings.Repeat("x", 40), time.Hour)
	if _, err := other.Verify(s); err == nil {
		t.Fatal("token from another secret accepted")
	}
}

func TestTamperedRefused(t *testing.T) {
	s, _, _ := newT(t).Issue("u", "org-1", "viewer")
	parts := strings.Split(s, ".")
	// Swap in a payload claiming another org, keep the old signature.
	other, _, _ := newT(t).Issue("u", "org-2", "viewer")
	forged := parts[0] + "." + strings.Split(other, ".")[1] + "." + parts[2]
	if _, err := newT(t).Verify(forged); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestAlgNoneRefused(t *testing.T) {
	c := Claims{OrgID: "o", RegisteredClaims: jwt.RegisteredClaims{
		Subject: "u", Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodNone, c).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newT(t).Verify(s); err == nil {
		t.Fatal(`"alg":"none" token accepted`)
	}
}

func TestGarbageRefused(t *testing.T) {
	if _, err := newT(t).Verify("not.a.token"); err == nil {
		t.Fatal("garbage accepted")
	}
}

func TestPasswordHashAndCheck(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h, "correct horse") || !strings.HasPrefix(h, "$2") {
		t.Fatal("hash looks wrong")
	}
	if !CheckPassword(h, "correct horse") {
		t.Fatal("right password refused")
	}
	if CheckPassword(h, "wrong horse") {
		t.Fatal("wrong password accepted")
	}
	if CheckPassword("", "anything") {
		t.Fatal("unknown user accepted")
	}
}

func TestPasswordRules(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
	if _, err := HashPassword(strings.Repeat("a", 73)); err == nil {
		t.Fatal("73-byte password accepted")
	}
}
