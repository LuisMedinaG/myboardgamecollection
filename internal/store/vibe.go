package store

import "myboardgamecollection/internal/model"

// AllVibes returns all vibes ordered by name.
func (s *Store) AllVibes() ([]model.Vibe, error) {
	rows, err := s.db.Query("SELECT id, name FROM vibes ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vibes []model.Vibe
	for rows.Next() {
		var v model.Vibe
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		vibes = append(vibes, v)
	}
	return vibes, rows.Err()
}

// GetVibe returns a single vibe by ID.
func (s *Store) GetVibe(id int64) (model.Vibe, error) {
	var v model.Vibe
	err := s.db.QueryRow("SELECT id, name FROM vibes WHERE id = ?", id).Scan(&v.ID, &v.Name)
	return v, err
}

// CreateVibe inserts a new vibe and returns its ID.
func (s *Store) CreateVibe(name string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO vibes (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateVibe renames a vibe.
func (s *Store) UpdateVibe(id int64, name string) error {
	_, err := s.db.Exec("UPDATE vibes SET name = ? WHERE id = ?", name, id)
	return err
}

// DeleteVibe removes a vibe by ID.
func (s *Store) DeleteVibe(id int64) error {
	_, err := s.db.Exec("DELETE FROM vibes WHERE id = ?", id)
	return err
}

// VibesForGame returns all vibes associated with a game.
func (s *Store) VibesForGame(gameID int64) ([]model.Vibe, error) {
	rows, err := s.db.Query(`
		SELECT v.id, v.name FROM vibes v
		JOIN game_vibes gv ON gv.vibe_id = v.id
		WHERE gv.game_id = ?
		ORDER BY v.name`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vibes []model.Vibe
	for rows.Next() {
		var v model.Vibe
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		vibes = append(vibes, v)
	}
	return vibes, rows.Err()
}

// SetGameVibes replaces all vibe associations for a game.
func (s *Store) SetGameVibes(gameID int64, vibeIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM game_vibes WHERE game_id = ?", gameID); err != nil {
		return err
	}
	for _, vid := range vibeIDs {
		if _, err := tx.Exec("INSERT INTO game_vibes (game_id, vibe_id) VALUES (?, ?)", gameID, vid); err != nil {
			return err
		}
	}
	return tx.Commit()
}
