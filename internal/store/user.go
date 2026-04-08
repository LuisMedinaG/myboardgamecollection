package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
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
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, errors.New("username already taken")
		}
		return 0, err
	}
	return res.LastInsertId()
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

	if !checkPasswordHash(password, hash) {
		return 0, errors.New("invalid username or password")
	}
	return id, nil
}

// hashPassword returns a salt+hash string.
// NOTE: For a real product, use golang.org/x/crypto/argon2.
// This is a simplified SHA-256 implementation for demonstration.
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	hash := h.Sum(nil)
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash), nil
}

func checkPasswordHash(password, hash string) bool {
	parts := strings.Split(hash, ":")
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	originalHash, _ := hex.DecodeString(parts[1])

	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	newHash := h.Sum(nil)

	return subtle.ConstantTimeCompare(originalHash, newHash) == 1
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
