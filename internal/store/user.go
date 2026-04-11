package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// RegisterUser creates a new user with a hashed password.
// bggUsername and email are optional profile fields.
func (s *Store) RegisterUser(username, password, bggUsername, email string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, err
	}

	isAdmin := 0
	if admin := strings.TrimSpace(os.Getenv("ADMIN_USERNAME")); admin != "" && strings.EqualFold(username, admin) {
		isAdmin = 1
	}

	res, err := s.db.Exec(
		"INSERT INTO users (username, bgg_username, password_hash, email, is_admin) VALUES (?, ?, ?, ?, ?)",
		username, bggUsername, hash, email, isAdmin,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

// ChangePassword verifies currentPassword then replaces the hash with a new one.
func (s *Store) ChangePassword(userID int64, currentPassword, newPassword string) error {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&hash)
	if err != nil {
		return errors.New("user not found")
	}
	if ok, _ := checkPasswordHash(currentPassword, hash); !ok {
		return ErrWrongPassword
	}
	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, userID)
	return err
}

// AuthenticateUser verifies the username and password, returning the user's ID
// on success. Authentication is against the username column only.
func (s *Store) AuthenticateUser(username, password string) (int64, error) {
	var id int64
	var hash string
	err := s.db.QueryRow(
		"SELECT id, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("invalid username or password")
	}
	if err != nil {
		return 0, err
	}

	ok, legacy := checkPasswordHash(password, hash)
	if !ok {
		return 0, errors.New("invalid username or password")
	}
	if legacy {
		upgradedHash, hashErr := hashPassword(password)
		if hashErr == nil {
			_, _ = s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", upgradedHash, id)
		}
	}
	return id, nil
}

const (
	passwordSaltLen       = 16
	sha256IterationsV2    = 120000
	sha256HashVersion     = 2
	sha256HashBytesLength = 32
)

// hashPassword returns an encoded salted+iterated SHA-256 hash string.
func hashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := iterativeSHA256([]byte(password), salt, sha256IterationsV2)
	return fmt.Sprintf("$sha256$v=%d$i=%d$%s$%s", sha256HashVersion, sha256IterationsV2, hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

func checkPasswordHash(password, hash string) (match bool, legacy bool) {
	if strings.HasPrefix(hash, "$sha256$") {
		return verifySHA256V2Hash(password, hash), false
	}
	return verifyLegacySHA256Hash(password, hash), true
}

func verifyLegacySHA256Hash(password, hash string) bool {
	parts := strings.Split(hash, ":")
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	originalHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	newHash := h.Sum(nil)

	return subtle.ConstantTimeCompare(originalHash, newHash) == 1
}

func verifySHA256V2Hash(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "sha256" {
		return false
	}
	var version, iterations int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != sha256HashVersion {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "i=%d", &iterations); err != nil || iterations <= 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	originalHash, err := hex.DecodeString(parts[5])
	if err != nil || len(originalHash) != sha256HashBytesLength {
		return false
	}
	newHash := iterativeSHA256([]byte(password), salt, iterations)
	return subtle.ConstantTimeCompare(originalHash, newHash) == 1
}

func iterativeSHA256(password, salt []byte, iterations int) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write(password)
	sum := h.Sum(nil)
	for i := 1; i < iterations; i++ {
		h.Reset()
		h.Write(sum)
		sum = h.Sum(nil)
	}
	return sum
}

// GetUsername returns the login username for a user ID.
func (s *Store) GetUsername(userID int64) (string, error) {
	var username string
	err := s.db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	return username, err
}

// SetBGGUsername updates the BGG username for a user.
func (s *Store) SetBGGUsername(userID int64, bggUsername string) error {
	_, err := s.db.Exec("UPDATE users SET bgg_username = ? WHERE id = ?", bggUsername, userID)
	return err
}

// GetBGGUsername returns the BGG username for a user ID. Returns empty string
// if the user has no BGG username set.
func (s *Store) GetBGGUsername(userID int64) (string, error) {
	var bgg string
	err := s.db.QueryRow("SELECT bgg_username FROM users WHERE id = ?", userID).Scan(&bgg)
	return bgg, err
}

// CreateSession generates a cryptographically random session token, stores it,
// and returns the token string.
func (s *Store) CreateSession(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires,
	)
	return token, err
}

// ValidateSession checks that the token exists and has not expired.
// On success it returns the user's ID, login username, and admin flag.
func (s *Store) ValidateSession(token string) (int64, string, bool, error) {
	var userID int64
	var isAdminInt int
	var username, expiresAt string
	err := s.db.QueryRow(`
		SELECT s.user_id, u.username, s.expires_at, u.is_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?`, token,
	).Scan(&userID, &username, &expiresAt, &isAdminInt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, errors.New("session not found")
	}
	if err != nil {
		return 0, "", false, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, "", false, err
	}
	if time.Now().After(exp) {
		return 0, "", false, errors.New("session expired")
	}
	return userID, username, isAdminInt == 1, nil
}

// DeleteUserSessions removes all sessions for a user (session rotation on login).
func (s *Store) DeleteUserSessions(userID int64) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// DeleteSession removes a session token (logout).
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// DeleteExpiredSessions removes all sessions that have passed their expiry time.
func (s *Store) DeleteExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC().Format(time.RFC3339))
	return err
}

// GetUserInfo returns the username and admin flag for a user ID.
// Used by the JWT login handler to encode claims at token issuance time.
func (s *Store) GetUserInfo(userID int64) (username string, isAdmin bool, err error) {
	var isAdminInt int
	err = s.db.QueryRow(
		"SELECT username, is_admin FROM users WHERE id = ?", userID,
	).Scan(&username, &isAdminInt)
	return username, isAdminInt == 1, err
}

// CreateAPIRefreshToken generates a random token stored in the sessions table
// with kind='api', used as a long-lived refresh token for the JSON API.
func (s *Store) CreateAPIRefreshToken(userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, kind) VALUES (?, ?, ?, 'api')",
		token, userID, expires,
	)
	return token, err
}

// ValidateAPIRefreshToken checks that the token exists, has kind='api', and has
// not expired. Returns the user's ID, login username, and admin flag on success.
func (s *Store) ValidateAPIRefreshToken(token string) (int64, string, bool, error) {
	var userID int64
	var isAdminInt int
	var username, expiresAt string
	err := s.db.QueryRow(`
		SELECT s.user_id, u.username, s.expires_at, u.is_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.kind = 'api'`, token,
	).Scan(&userID, &username, &expiresAt, &isAdminInt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, errors.New("invalid or expired refresh token")
	}
	if err != nil {
		return 0, "", false, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, "", false, err
	}
	if time.Now().After(exp) {
		return 0, "", false, errors.New("invalid or expired refresh token")
	}
	return userID, username, isAdminInt == 1, nil
}

// DeleteAPIRefreshToken removes a single API refresh token (API logout).
func (s *Store) DeleteAPIRefreshToken(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ? AND kind = 'api'", token)
	return err
}

// CanSync returns true if the user has not yet consumed their daily sync quota.
// limit is the maximum number of syncs allowed per day for this user.
func (s *Store) CanSync(userID int64, limit int) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var syncDate string
	var count int
	err := s.db.QueryRow(
		"SELECT sync_date, sync_count_today FROM users WHERE id = ?", userID,
	).Scan(&syncDate, &count)
	if err != nil {
		return false, err
	}
	return syncDate != today || count < limit, nil
}

// RecordSync increments the user's daily sync counter.
func (s *Store) RecordSync(userID int64) error {
	today := time.Now().Format("2006-01-02")
	_, err := s.db.Exec(`
		UPDATE users SET
			last_sync_at     = CURRENT_TIMESTAMP,
			sync_count_today = CASE WHEN sync_date = ? THEN sync_count_today + 1 ELSE 1 END,
			sync_date        = ?
		WHERE id = ?`, today, today, userID)
	return err
}
