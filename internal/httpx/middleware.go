package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Middleware wraps an HTTP handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares from left to right.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// --- Context helpers ---

type contextKey string

const (
	ctxUserID   contextKey = "userID"
	ctxUsername contextKey = "username"
	ctxIsAdmin  contextKey = "isAdmin"
)

// SetUser stores userID, username, and admin flag in the context.
func SetUser(ctx context.Context, id int64, username string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, id)
	ctx = context.WithValue(ctx, ctxUsername, username)
	ctx = context.WithValue(ctx, ctxIsAdmin, isAdmin)
	return ctx
}

// UserIDFromContext retrieves the authenticated user's ID from the context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(ctxUserID).(int64)
	return v, ok
}

// UsernameFromContext retrieves the authenticated user's BGG username.
func UsernameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUsername).(string)
	return v
}

// IsAdminFromContext reports whether the authenticated user has admin privileges.
func IsAdminFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxIsAdmin).(bool)
	return v
}

// --- Session auth middleware ---

// SessionValidator is satisfied by any store that can validate a session token.
// Using an interface avoids an import cycle between httpx and store.
type SessionValidator interface {
	ValidateSession(token string) (int64, string, bool, error)
}

// RequireAuth reads the session cookie and populates the request context with
// the user's ID, username, and admin flag. Unauthenticated requests are
// redirected to /login. HTMX requests receive an HX-Redirect header instead of
// a 302 so the client can do a full-page navigation rather than swapping partial
// content.
// secret is used to sign CSRF tokens and must match the application's SESSION_SECRET.
func RequireAuth(sv SessionValidator, secret []byte) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("sid")
			if err != nil {
				redirectToLogin(w, r)
				return
			}
			userID, username, isAdmin, err := sv.ValidateSession(cookie.Value)
			if err != nil {
				// Clear the stale/invalid cookie.
				http.SetCookie(w, &http.Cookie{Name: "sid", Path: "/", MaxAge: -1})
				redirectToLogin(w, r)
				return
			}
			ctx := SetUser(r.Context(), userID, username, isAdmin)
			ctx = SetCSRFToken(ctx, computeCSRF(cookie.Value, secret))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// redirectToLogin performs a full-page redirect or an HTMX-friendly redirect.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- Security middleware ---

// SecurityHeaders adds basic browser hardening headers.
func SecurityHeaders() Middleware {
	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' https://unpkg.com",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https:",
		"frame-src https://drive.google.com",
		"object-src 'none'",
		"base-uri 'self'",
		"frame-ancestors 'none'",
		"form-action 'self'",
	}, "; ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", csp)
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
			if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SameOrigin rejects cross-site unsafe requests to reduce CSRF risk.
func SameOrigin() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			if !sameOrigin(r.Header.Get("Origin"), r.Host) && !sameOrigin(r.Header.Get("Referer"), r.Host) {
				http.Error(w, "cross-site request blocked", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func sameOrigin(raw, host string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, host)
}

// MethodGuard returns 405 when a route is reached with an unexpected method.
func MethodGuard(method string) Middleware {
	allow := strings.ToUpper(method)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != allow {
				w.Header().Set("Allow", allow)
				http.Error(w, fmt.Sprintf("method %s not allowed", r.Method), http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- JWT auth middleware ---

// RequireJWT validates an Authorization: Bearer <token> header and populates the
// request context with the user's identity. API requests that fail auth receive a
// 401 JSON response — never a redirect.
func RequireJWT(secret string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, prefix) {
				WriteJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			claims, err := ParseAccessToken(strings.TrimPrefix(auth, prefix), secret)
			if err != nil {
				WriteJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := SetUser(r.Context(), claims.UserID, claims.Username, claims.IsAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WriteJSONError writes a JSON error response with the given status code.
// Used by API middlewares and handlers so they never return HTML.
func WriteJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
