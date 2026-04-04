package httpx

import (
	"crypto/subtle"
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

// AdminAuth protects admin routes with HTTP Basic Auth.
func AdminAuth(username, password string) Middleware {
	configured := username != "" && password != ""

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !configured {
				http.Error(w, "admin auth is not configured", http.StatusServiceUnavailable)
				return
			}

			user, pass, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
				subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="admin", charset="UTF-8"`)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
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
