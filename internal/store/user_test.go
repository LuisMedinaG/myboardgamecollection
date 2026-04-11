package store

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStore creates an in-memory SQLite store for tests.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	require.NoError(t, err, "newTestStore")
	t.Cleanup(func() { s.Close() })
	return s
}

// --- hashPassword ---

func TestHashPasswordFormat(t *testing.T) {
	cases := []struct {
		name     string
		password string
	}{
		{"typical", "mypassword"},
		{"unicode", "pässwörd🔒"},
		{"empty", ""},
		{"long", strings.Repeat("a", 1000)},
		{"special chars", `p@$$w0rd!#%^&*()`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hash, err := hashPassword(c.password)
			require.NoError(t, err)
			parts := strings.Split(hash, ":")
			require.Len(t, parts, 2, "hash must be <salt>:<hash>")
			assert.Len(t, parts[0], 32, "salt should be 16 bytes hex-encoded (32 chars)")
			assert.Len(t, parts[1], 64, "sha256 hash should be 32 bytes hex-encoded (64 chars)")
		})
	}
}

func TestHashPasswordUniqueness(t *testing.T) {
	// Same password → different hash every time (random salt).
	h1, err1 := hashPassword("password")
	h2, err2 := hashPassword("password")
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, h1, h2, "same password must produce different hashes")
}

// --- checkPasswordHash ---

func TestCheckPasswordHashCorrect(t *testing.T) {
	cases := []struct {
		name     string
		password string
	}{
		{"typical", "correcthorsebatterystaple"},
		{"unicode", "pässwörd"},
		{"empty", ""},
		{"special", `!@#$%^&*()`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hash, err := hashPassword(c.password)
			require.NoError(t, err)
			assert.True(t, checkPasswordHash(c.password, hash), "correct password must verify")
			assert.False(t, checkPasswordHash(c.password+"x", hash), "wrong password must not verify")
		})
	}
}

func TestCheckPasswordHashMalformed(t *testing.T) {
	cases := []struct {
		name string
		hash string
	}{
		{"empty", ""},
		{"no colon", "deadbeef"},
		{"too many colons", "a:b:c"},
		{"invalid hex salt", "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ:deadbeef"},
		{"invalid hex hash", "deadbeefdeadbeefdeadbeefdeadbeef:ZZZZ"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.False(t, checkPasswordHash("anything", c.hash),
				"malformed hash must return false")
		})
	}
}

// BenchmarkHashPassword guards against a DoS via slow password hashing.
// SHA-256 should complete well under 1ms per hash.
func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hashPassword("benchmarkpassword123")
	}
}

// --- RegisterUser / AuthenticateUser ---

func TestRegisterAndAuthenticateUser(t *testing.T) {
	s := newTestStore(t)

	id, err := s.RegisterUser("alice", "secret123", "alice_bgg", "alice@example.com")
	require.NoError(t, err)
	assert.Positive(t, id)

	gotID, err := s.AuthenticateUser("alice", "secret123")
	require.NoError(t, err)
	assert.Equal(t, id, gotID)
}

func TestRegisterUserDuplicateUsername(t *testing.T) {
	s := newTestStore(t)
	_, err := s.RegisterUser("bob", "pass1", "", "")
	require.NoError(t, err)

	_, err = s.RegisterUser("bob", "pass2", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "username already taken")
}

func TestAuthenticateUserWrongPassword(t *testing.T) {
	s := newTestStore(t)
	_, err := s.RegisterUser("carol", "rightpass", "", "")
	require.NoError(t, err)

	_, err = s.AuthenticateUser("carol", "wrongpass")
	assert.Error(t, err)
}

func TestAuthenticateUserNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.AuthenticateUser("nobody", "pass")
	assert.Error(t, err)
}

// --- ChangePassword ---

func TestChangePassword(t *testing.T) {
	s := newTestStore(t)
	id, err := s.RegisterUser("kate", "oldpass", "", "")
	require.NoError(t, err)

	require.NoError(t, s.ChangePassword(id, "oldpass", "newpass"))

	_, err = s.AuthenticateUser("kate", "newpass")
	assert.NoError(t, err, "new password must work after change")

	_, err = s.AuthenticateUser("kate", "oldpass")
	assert.Error(t, err, "old password must not work after change")
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("leo", "pass", "", "")
	err := s.ChangePassword(id, "wrongpass", "newpass")
	assert.Error(t, err)
}

// --- CreateSession / ValidateSession ---

func TestCreateAndValidateSession(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("dave", "pass", "", "")

	token, err := s.CreateSession(id)
	require.NoError(t, err)
	assert.Len(t, token, 64, "session token should be 32 bytes hex-encoded (64 chars)")

	gotID, username, isAdmin, err := s.ValidateSession(token)
	require.NoError(t, err)
	assert.Equal(t, id, gotID)
	assert.Equal(t, "dave", username)
	assert.False(t, isAdmin)
}

func TestValidateSessionNotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, _, err := s.ValidateSession("nonexistenttoken")
	assert.Error(t, err)
}

func TestValidateSessionExpired(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("expired", "pass", "", "")
	token, _ := s.CreateSession(id)

	// Manually expire the session.
	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?", past, token)
	require.NoError(t, err)

	_, _, _, err = s.ValidateSession(token)
	assert.Error(t, err)
}

func TestDeleteSession(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("eve", "pass", "", "")
	token, _ := s.CreateSession(id)

	require.NoError(t, s.DeleteSession(token))

	_, _, _, err := s.ValidateSession(token)
	assert.Error(t, err)
}

func TestDeleteExpiredSessions(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("frank", "pass", "", "")
	token, _ := s.CreateSession(id)

	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?", past, token)
	require.NoError(t, err)

	require.NoError(t, s.DeleteExpiredSessions())

	// Session row is gone now.
	_, _, _, err = s.ValidateSession(token)
	assert.Error(t, err)
}

func TestDeleteUserSessions(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("multi", "pass", "", "")
	t1, _ := s.CreateSession(id)
	t2, _ := s.CreateSession(id)

	require.NoError(t, s.DeleteUserSessions(id))

	_, _, _, err1 := s.ValidateSession(t1)
	_, _, _, err2 := s.ValidateSession(t2)
	assert.Error(t, err1, "first session should be gone")
	assert.Error(t, err2, "second session should be gone")
}

func TestSessionTokenUniqueness(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("grace", "pass", "", "")

	t1, _ := s.CreateSession(id)
	t2, _ := s.CreateSession(id)
	assert.NotEqual(t, t1, t2, "session tokens must be unique")
}

// --- CreateAPIRefreshToken / ValidateAPIRefreshToken ---

func TestCreateAndValidateAPIRefreshToken(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("henry", "pass", "", "")

	token, err := s.CreateAPIRefreshToken(id)
	require.NoError(t, err)
	assert.Len(t, token, 64)

	gotID, username, _, err := s.ValidateAPIRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, id, gotID)
	assert.Equal(t, "henry", username)
}

func TestValidateAPIRefreshTokenInvalid(t *testing.T) {
	s := newTestStore(t)
	_, _, _, err := s.ValidateAPIRefreshToken("bogustoken")
	assert.Error(t, err)
}

func TestAPIRefreshTokenKindIsolation(t *testing.T) {
	// A browser session token must not validate as an API refresh token.
	s := newTestStore(t)
	id, _ := s.RegisterUser("ivy", "pass", "", "")

	sessionToken, _ := s.CreateSession(id)
	_, _, _, err := s.ValidateAPIRefreshToken(sessionToken)
	assert.Error(t, err, "browser session token must not be valid as an API refresh token")
}

func TestDeleteAPIRefreshToken(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("jack", "pass", "", "")
	token, _ := s.CreateAPIRefreshToken(id)

	require.NoError(t, s.DeleteAPIRefreshToken(token))

	_, _, _, err := s.ValidateAPIRefreshToken(token)
	assert.Error(t, err)
}

func TestAPIRefreshTokenExpired(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("expired_api", "pass", "", "")
	token, _ := s.CreateAPIRefreshToken(id)

	past := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?", past, token)
	require.NoError(t, err)

	_, _, _, err = s.ValidateAPIRefreshToken(token)
	assert.Error(t, err)
}

// --- GetUserInfo / GetUsername ---

func TestGetUserInfo(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("mia", "pass", "", "")

	username, isAdmin, err := s.GetUserInfo(id)
	require.NoError(t, err)
	assert.Equal(t, "mia", username)
	assert.False(t, isAdmin)
}

func TestGetUsername(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("nora", "pass", "", "")

	username, err := s.GetUsername(id)
	require.NoError(t, err)
	assert.Equal(t, "nora", username)
}

// --- SetBGGUsername / GetBGGUsername ---

func TestSetAndGetBGGUsername(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.RegisterUser("oscar", "pass", "oscar_bgg", "")

	bgg, err := s.GetBGGUsername(id)
	require.NoError(t, err)
	assert.Equal(t, "oscar_bgg", bgg)

	require.NoError(t, s.SetBGGUsername(id, "oscar_bgg_updated"))

	bgg, err = s.GetBGGUsername(id)
	require.NoError(t, err)
	assert.Equal(t, "oscar_bgg_updated", bgg)
}
