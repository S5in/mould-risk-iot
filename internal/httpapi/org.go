package httpapi

import (
	"context"
	"net/http"
	"regexp"
)

type ctxKey struct{}

// WithOrg returns a copy of ctx carrying the caller's organisation ID.
// Only the auth layer should call this; handlers never read the org from the
// URL, query string or body.
func WithOrg(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, orgID)
}

func orgFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKey{}).(string)
	return v, ok && v != ""
}

// DevOrg is a TEMPORARY stand-in for login: it puts a fixed organisation on
// every request. Step 5 replaces it with a middleware that reads the org
// from a verified token. It must never be used in a deployment.
func DevOrg(orgID string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(WithOrg(r.Context(), orgID)))
	})
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// requireOrg returns the caller's org, or writes 401 and returns ok=false.
func requireOrg(w http.ResponseWriter, r *http.Request) (string, bool) {
	org, ok := orgFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return "", false
	}
	return org, true
}
