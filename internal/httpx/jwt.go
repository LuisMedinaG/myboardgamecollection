package httpx

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims holds the application-specific fields embedded in every access token.
type JWTClaims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// GenerateAccessToken mints a signed HS256 JWT that expires in 15 minutes.
func GenerateAccessToken(userID int64, username string, isAdmin bool, secret string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken validates a signed JWT and returns its claims.
// Returns an error if the token is invalid, expired, or uses an unexpected algorithm.
func ParseAccessToken(tokenStr, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return token.Claims.(*JWTClaims), nil
}
