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

	"golang.org/x/crypto/argon2"
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
		if isDuplicateError(err) {
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
	// Argon2id parameters (OWASP recommended minimums).
	argon2idMemory      = 64 * 1024 // 64 MB
	argon2idTime        = 1
	argon2idParallelism = 4
	argon2idKeyLen      = 32
	argon2idSaltLen     = 16

	// Legacy SHA-256 constants kept for verifying existing hashes.
	sha256IterationsV2    = 120000
	sha256HashVersion     = 2
	sha256HashBytesLength = 32
)

// hashPassword returns an argon2id-encoded hash string.
func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2idSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2idTime, argon2idMemory, argon2idParallelism, argon2idKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2idMemory, argon2idTime, argon2idParallelism,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash),
	), nil
}

// checkPasswordHash verifies password against encoded. Returns (match, legacy)
// where legacy=true means the hash uses SHA-256 and should be upgraded.
func checkPasswordHash(password, hash string) (match bool, legacy bool) {
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2idHash(password, hash), false
	}
	if strings.HasPrefix(hash, "$sha256$") {
		return verifySHA256V2Hash(password, hash), true
	}
	return verifyLegacySHA256Hash(password, hash), true
}

func verifyArgon2idHash(password, encoded string) bool {
	// Format: $argon2id$v=<version>$m=<mem>,t=<time>,p=<par>$<hex-salt>$<hex-hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false
	}
	var memory uint32
	var timeVal uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeVal, &parallelism); err != nil {
		return false
	}
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	originalHash, err := hex.DecodeString(parts[5])
	if err != nil || len(originalHash) == 0 {
		return false
	}
	newHash := argon2.IDKey([]byte(password), salt, timeVal, memory, parallelism, uint32(len(originalHash)))
	return subtle.ConstantTimeCompare(originalHash, newHash) == 1
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

// createToken generates a cryptographically random token, stores it in the
// sessions table with the given kind, and returns the token string.
// kind="" for browser sessions, kind="api" for API refresh tokens.
func (s *Store) createToken(userID int64, kind string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at, kind) VALUES (?, ?, ?, ?)",
		token, userID, expires, kind,
	)
	return token, err
}

// CreateSession generates a cryptographically random session token, stores it,
// and returns the token string.
func (s *Store) CreateSession(userID int64) (string, error) {
	return s.createToken(userID, "")
}

// validateToken checks that a token exists (optionally filtered by kind) and
// has not expired. Returns the user's ID, login username, and admin flag.
func (s *Store) validateToken(token, kind, notFoundMsg string) (int64, string, bool, error) {
	var userID int64
	var isAdminInt int
	var username, expiresAt string

	query := `SELECT s.user_id, u.username, s.expires_at, u.is_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?`
	args := []any{token}
	if kind != "" {
		query += " AND s.kind = ?"
		args = append(args, kind)
	}

	err := s.db.QueryRow(query, args...).Scan(&userID, &username, &expiresAt, &isAdminInt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, errors.New(notFoundMsg)
	}
	if err != nil {
		return 0, "", false, err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, "", false, err
	}
	if time.Now().After(exp) {
		return 0, "", false, errors.New(notFoundMsg)
	}
	return userID, username, isAdminInt == 1, nil
}

// ValidateSession checks that the token exists and has not expired.
// On success it returns the user's ID, login username, and admin flag.
func (s *Store) ValidateSession(token string) (int64, string, bool, error) {
	return s.validateToken(token, "", "session not found")
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
	return s.createToken(userID, "api")
}

// ValidateAPIRefreshToken checks that the token exists, has kind='api', and has
// not expired. Returns the user's ID, login username, and admin flag on success.
func (s *Store) ValidateAPIRefreshToken(token string) (int64, string, bool, error) {
	return s.validateToken(token, "api", "invalid or expired refresh token")
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
