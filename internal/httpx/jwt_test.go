package httpx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	token, err := GenerateAccessToken(42, "alice", false, "testsecret")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ParseAccessToken(token, "testsecret")
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.False(t, claims.IsAdmin)
}

func TestGenerateAccessTokenAdminFlag(t *testing.T) {
	token, err := GenerateAccessToken(99, "admin", true, "secret")
	require.NoError(t, err)

	claims, err := ParseAccessToken(token, "secret")
	require.NoError(t, err)
	assert.True(t, claims.IsAdmin)
	assert.Equal(t, int64(99), claims.UserID)
}

func TestParseAccessTokenWrongSecret(t *testing.T) {
	token, err := GenerateAccessToken(1, "user", false, "secret1")
	require.NoError(t, err)

	_, err = ParseAccessToken(token, "secret2")
	assert.Error(t, err, "wrong secret must be rejected")
}

func TestParseAccessTokenMalformed(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"random string", "not.a.jwt"},
		{"single segment", "invalid"},
		{"two segments", "header.payload"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseAccessToken(c.token, "secret")
			assert.Error(t, err)
		})
	}
}

func TestParseAccessTokenAlgNone(t *testing.T) {
	// alg=none tokens must be rejected.
	// A manually crafted alg:none token.
	noneToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOjEsInVzZXJuYW1lIjoiYWRtaW4iLCJpc19hZG1pbiI6dHJ1ZX0."
	_, err := ParseAccessToken(noneToken, "secret")
	assert.Error(t, err, "alg=none token must be rejected")
}

func TestTokenExpiryWindow(t *testing.T) {
	token, err := GenerateAccessToken(1, "user", false, "secret")
	require.NoError(t, err)

	claims, err := ParseAccessToken(token, "secret")
	require.NoError(t, err)

	exp := claims.ExpiresAt.Time
	now := time.Now()
	// Must expire between 14 and 16 minutes from now.
	assert.True(t, exp.After(now.Add(14*time.Minute)),
		"token should expire no sooner than 14 minutes from now")
	assert.True(t, exp.Before(now.Add(16*time.Minute)),
		"token should expire no later than 16 minutes from now")
}

func TestTokenIssuedAt(t *testing.T) {
	before := time.Now().Add(-time.Second)
	token, _ := GenerateAccessToken(1, "user", false, "secret")
	after := time.Now().Add(time.Second)

	claims, err := ParseAccessToken(token, "secret")
	require.NoError(t, err)

	iat := claims.IssuedAt.Time
	assert.True(t, iat.After(before), "iat should be after test start")
	assert.True(t, iat.Before(after), "iat should be before test end")
}

func TestDifferentUsersGetDifferentTokens(t *testing.T) {
	t1, _ := GenerateAccessToken(1, "alice", false, "secret")
	t2, _ := GenerateAccessToken(2, "bob", false, "secret")
	assert.NotEqual(t, t1, t2)
}
