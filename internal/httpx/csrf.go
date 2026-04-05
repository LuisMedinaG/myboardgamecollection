package httpx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

const ctxCSRFToken contextKey = "csrfToken"

// computeCSRF derives a CSRF token from the session token using HMAC-SHA256.
// The token is deterministic for a given session, so it doesn't need storage.
func computeCSRF(sessionToken string) string {
	mac := hmac.New(sha256.New, []byte("csrf"))
	mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// SetCSRFToken stores the CSRF token in the request context.
func SetCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxCSRFToken, token)
}

// CSRFTokenFromContext retrieves the CSRF token from the request context.
func CSRFTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxCSRFToken).(string)
	return v
}

// VerifyCSRF is a middleware that checks POST/PUT/PATCH/DELETE requests for a
// valid CSRF token. The token may be sent as a form field (_csrf) or an HTTP
// header (X-CSRF-Token). GET/HEAD/OPTIONS requests are passed through.
func VerifyCSRF() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			expected := CSRFTokenFromContext(r.Context())
			if expected == "" {
				// No CSRF token in context means RequireAuth didn't run
				// (e.g. public POST routes like /login). Pass through.
				next.ServeHTTP(w, r)
				return
			}

			// Check header first (HTMX), then form field (regular forms).
			provided := r.Header.Get("X-CSRF-Token")
			if provided == "" {
				provided = r.FormValue("_csrf")
			}

			if !hmac.Equal([]byte(provided), []byte(expected)) {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
