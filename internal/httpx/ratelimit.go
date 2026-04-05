package httpx

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// LoginLimiter tracks login attempts per IP address and rejects requests that
// exceed the configured threshold within a sliding window.
type LoginLimiter struct {
	mu      sync.Mutex
	buckets map[string][]time.Time
	max     int
	window  time.Duration
}

// NewLoginLimiter creates a limiter allowing max attempts per window per IP.
func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		buckets: make(map[string][]time.Time),
		max:     max,
		window:  window,
	}
}

// Allow returns true if the IP has not exceeded its rate limit.
func (l *LoginLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	attempts := l.buckets[ip]
	// Prune old entries.
	n := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			attempts[n] = t
			n++
		}
	}
	attempts = attempts[:n]
	l.buckets[ip] = attempts

	return len(attempts) < l.max
}

// Record adds a failed attempt for the given IP.
func (l *LoginLimiter) Record(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets[ip] = append(l.buckets[ip], time.Now())
}

// Cleanup removes stale entries older than the window. Call periodically.
func (l *LoginLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.window)
	for ip, attempts := range l.buckets {
		n := 0
		for _, t := range attempts {
			if t.After(cutoff) {
				attempts[n] = t
				n++
			}
		}
		if n == 0 {
			delete(l.buckets, ip)
		} else {
			l.buckets[ip] = attempts[:n]
		}
	}
}

// ClientIP extracts the client IP from the request, preferring X-Forwarded-For
// (first entry) for reverse-proxy setups, falling back to RemoteAddr.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First comma-separated value is the original client.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// RateLimit wraps a handler and rejects requests when the IP exceeds the limit.
func RateLimit(limiter *LoginLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			if !limiter.Allow(ip) {
				http.Error(w, "too many login attempts, please try again later", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
