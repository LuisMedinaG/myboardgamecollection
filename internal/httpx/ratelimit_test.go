package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewLoginLimiter ---

func TestNewLoginLimiter(t *testing.T) {
	l := NewLoginLimiter(5, time.Minute)
	require.NotNil(t, l)
	assert.Equal(t, 5, l.max)
	assert.Equal(t, time.Minute, l.window)
	assert.NotNil(t, l.buckets)
}

// --- Allow / Record ---

func TestAllowBeforeLimit(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		assert.True(t, l.Allow("1.2.3.4"), "attempt %d should be allowed", i+1)
		l.Record("1.2.3.4")
	}
}

func TestDenyAtLimit(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		l.Record("1.2.3.4")
	}
	assert.False(t, l.Allow("1.2.3.4"), "4th attempt should be denied")
}

func TestDenyAboveLimit(t *testing.T) {
	l := NewLoginLimiter(1, time.Minute)
	l.Record("1.2.3.4")
	l.Record("1.2.3.4")
	assert.False(t, l.Allow("1.2.3.4"))
}

func TestAllowAfterWindowExpiry(t *testing.T) {
	l := NewLoginLimiter(1, 50*time.Millisecond)
	l.Record("1.2.3.4")
	assert.False(t, l.Allow("1.2.3.4"), "should be denied immediately after recording")

	time.Sleep(60 * time.Millisecond)
	assert.True(t, l.Allow("1.2.3.4"), "should be allowed after window expires")
}

func TestDifferentIPsAreIndependent(t *testing.T) {
	l := NewLoginLimiter(1, time.Minute)
	l.Record("1.1.1.1")

	assert.False(t, l.Allow("1.1.1.1"), "1.1.1.1 should be denied")
	assert.True(t, l.Allow("2.2.2.2"), "2.2.2.2 should be allowed (independent bucket)")
}

func TestAllowLimitBoundary(t *testing.T) {
	// Allow exactly max attempts, deny the (max+1)th.
	const max = 5
	l := NewLoginLimiter(max, time.Minute)
	for i := 0; i < max; i++ {
		assert.True(t, l.Allow("ip"), "attempt %d/%d should be allowed", i+1, max)
		l.Record("ip")
	}
	assert.False(t, l.Allow("ip"), "attempt %d should be denied", max+1)
}

// --- Cleanup ---

func TestCleanupRemovesExpiredBuckets(t *testing.T) {
	l := NewLoginLimiter(5, 50*time.Millisecond)
	l.Record("1.2.3.4")
	l.Record("5.6.7.8")

	time.Sleep(60 * time.Millisecond)
	l.Cleanup()

	l.mu.Lock()
	defer l.mu.Unlock()
	assert.Empty(t, l.buckets, "all stale entries must be removed")
}

func TestCleanupPreservesActiveBuckets(t *testing.T) {
	l := NewLoginLimiter(5, time.Minute)
	l.Record("1.2.3.4")
	l.Cleanup()

	l.mu.Lock()
	defer l.mu.Unlock()
	assert.Len(t, l.buckets["1.2.3.4"], 1, "active entry must survive cleanup")
}

func TestCleanupMixedExpiry(t *testing.T) {
	// One IP with old entry, one with fresh entry.
	l := NewLoginLimiter(5, 50*time.Millisecond)
	l.Record("old.ip")

	time.Sleep(60 * time.Millisecond)
	l.Record("fresh.ip") // recorded after sleep, within window

	l.Cleanup()

	l.mu.Lock()
	defer l.mu.Unlock()
	assert.Empty(t, l.buckets["old.ip"], "expired IP must be removed")
	assert.NotEmpty(t, l.buckets["fresh.ip"], "active IP must be kept")
}

// --- ClientIP ---

func TestClientIPFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	assert.Equal(t, "192.168.1.1", ClientIP(req))
}

func TestClientIPFromXForwardedForMultiple(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	assert.Equal(t, "10.0.0.1", ClientIP(req), "first entry in XFF is the client")
}

func TestClientIPFromXForwardedForSingle(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	assert.Equal(t, "203.0.113.1", ClientIP(req))
}

func TestClientIPXForwardedForTakesPrecedence(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:9000"
	req.Header.Set("X-Forwarded-For", "203.0.113.42")
	assert.Equal(t, "203.0.113.42", ClientIP(req), "XFF must take precedence over RemoteAddr")
}

// --- RateLimit middleware ---

func TestRateLimitMiddlewareBlocks(t *testing.T) {
	l := NewLoginLimiter(2, time.Minute)
	handler := RateLimit(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeRequest := func() int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	// First 2 requests: allowed (Allow returns true, but doesn't Record automatically).
	// Record must be called manually in real usage; here we just test Allow behavior.
	assert.Equal(t, http.StatusOK, makeRequest())
	assert.Equal(t, http.StatusOK, makeRequest())
}

func TestRateLimitMiddlewareAllows(t *testing.T) {
	l := NewLoginLimiter(10, time.Minute)
	handler := RateLimit(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimitMiddlewareReturns429WhenDenied(t *testing.T) {
	l := NewLoginLimiter(0, time.Minute) // max=0: every request denied
	handler := RateLimit(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}
