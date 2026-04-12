package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- computeCSRF ---

func TestComputeCSRFDeterministic(t *testing.T) {
	secret := []byte("test-secret-key")
	t1 := computeCSRF("session-abc", secret)
	t2 := computeCSRF("session-abc", secret)
	assert.Equal(t, t1, t2, "computeCSRF must be deterministic for the same inputs")
}

func TestComputeCSRFDifferentSession(t *testing.T) {
	secret := []byte("test-secret-key")
	t1 := computeCSRF("session-abc", secret)
	t2 := computeCSRF("session-xyz", secret)
	assert.NotEqual(t, t1, t2, "different sessions must produce different CSRF tokens")
}

func TestComputeCSRFDifferentSecret(t *testing.T) {
	t1 := computeCSRF("session-abc", []byte("secret1"))
	t2 := computeCSRF("session-abc", []byte("secret2"))
	assert.NotEqual(t, t1, t2, "different secrets must produce different CSRF tokens")
}

func TestComputeCSRFOutputFormat(t *testing.T) {
	token := computeCSRF("session", []byte("secret"))
	// HMAC-SHA256 = 32 bytes → 64 hex chars.
	assert.Len(t, token, 64, "CSRF token should be 64 hex chars")
}

func TestComputeCSRFEmptyInputs(t *testing.T) {
	// Should not panic on empty inputs.
	t1 := computeCSRF("", []byte("secret"))
	t2 := computeCSRF("session", []byte{})
	assert.Len(t, t1, 64)
	assert.Len(t, t2, 64)
	assert.NotEqual(t, t1, t2)
}

// --- context helpers ---

func TestCSRFTokenContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, CSRFTokenFromContext(ctx), "empty context returns empty string")

	ctx = SetCSRFToken(ctx, "my-csrf-token")
	assert.Equal(t, "my-csrf-token", CSRFTokenFromContext(ctx))
}

func TestCSRFTokenContextOverwrite(t *testing.T) {
	ctx := SetCSRFToken(context.Background(), "first")
	ctx = SetCSRFToken(ctx, "second")
	assert.Equal(t, "second", CSRFTokenFromContext(ctx))
}

// --- VerifyCSRF middleware ---

func newCSRFHandler(expectedToken string) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := VerifyCSRF()(inner)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedToken != "" {
			r = r.WithContext(SetCSRFToken(r.Context(), expectedToken))
		}
		h.ServeHTTP(w, r)
	})
}

func TestVerifyCSRFSafeMethodsPassThrough(t *testing.T) {
	handler := newCSRFHandler("expected")
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		req := httptest.NewRequest(method, "/", nil)
		// No CSRF token provided — safe methods must not be blocked.
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "%s must pass without CSRF token", method)
	}
}

func TestVerifyCSRFNoContextToken(t *testing.T) {
	// No CSRF token in context → public route (e.g. /login). Must pass through.
	handler := newCSRFHandler("") // no context token
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestVerifyCSRFValidHeader(t *testing.T) {
	handler := newCSRFHandler("valid-token")
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.Header.Set("X-CSRF-Token", "valid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestVerifyCSRFValidFormField(t *testing.T) {
	handler := newCSRFHandler("valid-token")
	body := strings.NewReader("_csrf=valid-token&other=data")
	req := httptest.NewRequest(http.MethodPost, "/action", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestVerifyCSRFHeaderTakesPrecedence(t *testing.T) {
	// When both header and form field are present, header is used.
	handler := newCSRFHandler("header-token")
	body := strings.NewReader("_csrf=form-token")
	req := httptest.NewRequest(http.MethodPost, "/action", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", "header-token") // correct
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestVerifyCSRFInvalidToken(t *testing.T) {
	handler := newCSRFHandler("correct-token")
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	req.Header.Set("X-CSRF-Token", "wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestVerifyCSRFMissingToken(t *testing.T) {
	handler := newCSRFHandler("correct-token")
	req := httptest.NewRequest(http.MethodPost, "/action", nil)
	// No X-CSRF-Token header, no _csrf form field.
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestVerifyCSRFMutatingMethods(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		handler := newCSRFHandler("correct-token")
		req := httptest.NewRequest(method, "/action", nil)
		req.Header.Set("X-CSRF-Token", "wrong-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code, "%s with bad CSRF must be 403", method)
	}
}
