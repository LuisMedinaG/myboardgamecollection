package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

// FindOrCreateUser returns the ID for bggUsername, inserting a new row if the
// username has not been seen before.
func (s *Store) FindOrCreateUser(bggUsername string) (int64, error) {
	res, err := s.db.Exec("INSERT OR IGNORE INTO users (bgg_username) VALUES (?)", bggUsername)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return res.LastInsertId()
	}
	var id int64
	err = s.db.QueryRow("SELECT id FROM users WHERE bgg_username = ?", bggUsername).Scan(&id)
	return id, err
}

// GetUsername returns the BGG username for a user ID.
func (s *Store) GetUsername(userID int64) (string, error) {
	var username string
	err := s.db.QueryRow("SELECT bgg_username FROM users WHERE id = ?", userID).Scan(&username)
	return username, err
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
// On success it returns the user's ID and BGG username.
func (s *Store) ValidateSession(token string) (int64, string, error) {
	var userID int64
	var username, expiresAt string
	err := s.db.QueryRow(`
		SELECT s.user_id, u.bgg_username, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?`, token,
	).Scan(&userID, &username, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", errors.New("session not found")
	}
	if err != nil {
		return 0, "", err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, "", err
	}
	if time.Now().After(exp) {
		return 0, "", errors.New("session expired")
	}
	return userID, username, nil
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
func (s *Store) CanSync(userID int64) (bool, error) {
	today := time.Now().Format("2006-01-02")
	var syncDate string
	var count int
	err := s.db.QueryRow(
		"SELECT sync_date, sync_count_today FROM users WHERE id = ?", userID,
	).Scan(&syncDate, &count)
	if err != nil {
		return false, err
	}
	return syncDate != today || count < 1, nil
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
