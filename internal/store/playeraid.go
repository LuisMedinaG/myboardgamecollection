package store

import "myboardgamecollection/internal/model"

// GetPlayerAids returns all player aids for a game, ordered by ID.
func (s *Store) GetPlayerAids(gameID int64) ([]model.PlayerAid, error) {
	rows, err := s.db.Query("SELECT id, game_id, filename, label FROM player_aids WHERE game_id = ? ORDER BY id", gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aids []model.PlayerAid
	for rows.Next() {
		var a model.PlayerAid
		if err := rows.Scan(&a.ID, &a.GameID, &a.Filename, &a.Label); err != nil {
			return nil, err
		}
		aids = append(aids, a)
	}
	return aids, rows.Err()
}

// GetPlayerAid returns a single player aid by ID.
func (s *Store) GetPlayerAid(id int64) (model.PlayerAid, error) {
	var a model.PlayerAid
	err := s.db.QueryRow("SELECT id, game_id, filename, label FROM player_aids WHERE id = ?", id).
		Scan(&a.ID, &a.GameID, &a.Filename, &a.Label)
	return a, err
}

// CreatePlayerAid inserts a new player aid and returns its ID.
func (s *Store) CreatePlayerAid(gameID int64, filename, label string) (int64, error) {
	res, err := s.db.Exec("INSERT INTO player_aids (game_id, filename, label) VALUES (?, ?, ?)", gameID, filename, label)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// DeletePlayerAid removes a player aid by ID.
func (s *Store) DeletePlayerAid(id int64) error {
	_, err := s.db.Exec("DELETE FROM player_aids WHERE id = ?", id)
	return err
}
