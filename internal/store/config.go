package store

// SetConfig upserts a key-value pair in the config table.
func (s *Store) SetConfig(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	return err
}

// GetConfig returns the value for a config key, or "" if not found.
func (s *Store) GetConfig(key string) string {
	var v string
	_ = s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&v)
	return v
}
