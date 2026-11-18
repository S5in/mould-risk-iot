package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/S5in/mould-risk-iot/internal/auth"
)

// One known user: admin@demo.test / "right-password", in orgA.
var knownHash = func() string {
	h, err := auth.HashPassword("right-password")
	if err != nil {
		panic(err)
	}
	return h
}()

func (f fakeStore) UserByEmail(_ context.Context, email string) (User, bool, error) {
	if f.err != nil {
		return User{}, false, f.err
	}
	if strings.EqualFold(email, "admin@demo.test") {
		return User{ID: "user-1", OrgID: orgA, Role: "admin", PasswordHash: knownHash}, true, nil
	}
	return User{}, false, nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(userID, orgID, role string) (string, time.Time, error) {
	return "TOKEN:" + userID + ":" + orgID + ":" + role, time.Unix(0, 0), nil
}

const loginPath = "/v1/auth/login"

func TestLoginOK(t *testing.T) {
	rec := doBody(t, fakeStore{}, "", "POST", loginPath,
		`{"email":"admin@demo.test","password":"right-password"}`)
	want(t, rec, 200)
	if !strings.Contains(rec.Body.String(), "TOKEN:user-1:"+orgA+":admin") {
		t.Fatalf("token not returned: %s", rec.Body.String())
	}
}

func TestLoginEmailCaseInsensitive(t *testing.T) {
	want(t, doBody(t, fakeStore{}, "", "POST", loginPath,
		`{"email":"  Admin@Demo.TEST ","password":"right-password"}`), 200)
}

func TestLoginWrongPasswordAndUnknownEmailLookTheSame(t *testing.T) {
	wrong := doBody(t, fakeStore{}, "", "POST", loginPath,
		`{"email":"admin@demo.test","password":"wrong-password"}`)
	unknown := doBody(t, fakeStore{}, "", "POST", loginPath,
		`{"email":"nobody@demo.test","password":"right-password"}`)
	want(t, wrong, 401)
	want(t, unknown, 401)
	if wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("responses differ, which reveals if an email exists:\n%s\n%s",
			wrong.Body.String(), unknown.Body.String())
	}
}

func TestLoginBadBodies(t *testing.T) {
	for _, body := range []string{
		``,
		`not json`,
		`{"email":"admin@demo.test"}`,
		`{"email":"","password":"x"}`,
		`{"email":"admin@demo.test","password":"right-password","org":"other"}`,      // unknown field
		`{"email":"admin@demo.test","password":"` + strings.Repeat("a", 5000) + `"}`, // too big
	} {
		want(t, doBody(t, fakeStore{}, "", "POST", loginPath, body), 400)
	}
}

func TestLoginGetNotAllowed(t *testing.T) {
	want(t, do(t, fakeStore{}, "", "GET", loginPath), 405)
}

func TestLoginStoreErrorIs500(t *testing.T) {
	rec := doBody(t, fakeStore{err: errFake}, "", "POST", loginPath,
		`{"email":"admin@demo.test","password":"right-password"}`)
	want(t, rec, 500)
}

var errFake = fakeErr("secret db detail")

type fakeErr string

func (e fakeErr) Error() string { return string(e) }
