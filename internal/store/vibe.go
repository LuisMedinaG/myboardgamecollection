package store

import (
	"database/sql"
	"fmt"
	"strings"

	"myboardgamecollection/internal/model"
)

func scanVibes(rows *sql.Rows) ([]model.Vibe, error) {
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

// AllVibes returns all vibes owned by userID, ordered by name.
func (s *Store) AllVibes(userID int64) ([]model.Vibe, error) {
	rows, err := s.db.Query("SELECT id, name FROM vibes WHERE user_id = ? ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanVibes(rows)
}

// GetVibe returns a single vibe by ID, verifying it belongs to userID.
func (s *Store) GetVibe(id, userID int64) (model.Vibe, error) {
	var v model.Vibe
	err := s.db.QueryRow("SELECT id, name FROM vibes WHERE id = ? AND user_id = ?", id, userID).
		Scan(&v.ID, &v.Name)
	return v, err
}

// CreateVibe inserts a new vibe owned by userID and returns its ID.
// Returns ErrDuplicate if the name already exists for this user.
func (s *Store) CreateVibe(name string, userID int64) (int64, error) {
	res, err := s.db.Exec("INSERT INTO vibes (name, user_id) VALUES (?, ?)", name, userID)
	if err != nil {
		if isDuplicateError(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateVibe renames a vibe, verifying ownership.
// Returns ErrDuplicate if the new name already exists for this user.
func (s *Store) UpdateVibe(id int64, name string, userID int64) error {
	_, err := s.db.Exec("UPDATE vibes SET name = ? WHERE id = ? AND user_id = ?", name, id, userID)
	if isDuplicateError(err) {
		return ErrDuplicate
	}
	return err
}

// BatchUpdateVibes renames multiple vibes in a single transaction, verifying
// ownership for each. Returns ErrDuplicate if any name conflicts.
func (s *Store) BatchUpdateVibes(userID int64, updates map[int64]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for id, name := range updates {
		if _, err := tx.Exec("UPDATE vibes SET name = ? WHERE id = ? AND user_id = ?", name, id, userID); err != nil {
			if isDuplicateError(err) {
				return ErrDuplicate
			}
			return err
		}
	}
	return tx.Commit()
}

// DeleteVibe removes a vibe, verifying ownership.
func (s *Store) DeleteVibe(id, userID int64) error {
	_, err := s.db.Exec("DELETE FROM vibes WHERE id = ? AND user_id = ?", id, userID)
	return err
}

// VibesForGame returns all vibes associated with a game.
// Ownership is verified at the handler level (GetGame checks user_id).
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
	return scanVibes(rows)
}

// AddVibesToGames adds vibes to multiple games owned by userID.
func (s *Store) AddVibesToGames(userID int64, gameIDs, vibeIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	gameIDs = uniqueInt64s(gameIDs)
	vibeIDs = uniqueInt64s(vibeIDs)

	ownedGames, err := ownedIDs(tx, "games", userID, gameIDs)
	if err != nil {
		return err
	}
	if len(ownedGames) != len(gameIDs) {
		return ErrForeignOwnership
	}

	ownedVibes, err := ownedIDs(tx, "vibes", userID, vibeIDs)
	if err != nil {
		return err
	}
	if len(ownedVibes) != len(vibeIDs) {
		return ErrForeignOwnership
	}

	insertAssoc, err := tx.Prepare("INSERT OR IGNORE INTO game_vibes (game_id, vibe_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer insertAssoc.Close()

	for _, gid := range gameIDs {
		for _, vid := range vibeIDs {
			if _, err := insertAssoc.Exec(gid, vid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// VibesForGames returns a map of game ID → vibes for the given game IDs.
// Games with no vibes are omitted from the map.
func (s *Store) VibesForGames(gameIDs []int64) (map[int64][]model.Vibe, error) {
	gameIDs = uniqueInt64s(gameIDs)
	if len(gameIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(gameIDs))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(`
		SELECT gv.game_id, v.id, v.name
		FROM game_vibes gv
		JOIN vibes v ON v.id = gv.vibe_id
		WHERE gv.game_id IN (%s)
		ORDER BY v.name`, placeholders)
	args := make([]any, len(gameIDs))
	for i, id := range gameIDs {
		args[i] = id
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64][]model.Vibe)
	for rows.Next() {
		var gameID int64
		var v model.Vibe
		if err := rows.Scan(&gameID, &v.ID, &v.Name); err != nil {
			return nil, err
		}
		result[gameID] = append(result[gameID], v)
	}
	return result, rows.Err()
}

// SetGameVibes replaces all vibe associations for a game owned by userID.
func (s *Store) SetGameVibes(userID, gameID int64, vibeIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	vibeIDs = uniqueInt64s(vibeIDs)

	ownedGames, err := ownedIDs(tx, "games", userID, []int64{gameID})
	if err != nil {
		return err
	}
	if !ownedGames[gameID] {
		return ErrForeignOwnership
	}

	ownedVibes, err := ownedIDs(tx, "vibes", userID, vibeIDs)
	if err != nil {
		return err
	}
	if len(ownedVibes) != len(vibeIDs) {
		return ErrForeignOwnership
	}

	if _, err := tx.Exec("DELETE FROM game_vibes WHERE game_id = ?", gameID); err != nil {
		return err
	}
	insertAssoc, err := tx.Prepare("INSERT INTO game_vibes (game_id, vibe_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer insertAssoc.Close()

	for _, vid := range vibeIDs {
		if _, err := insertAssoc.Exec(gameID, vid); err != nil {
			return err
		}
	}
	return tx.Commit()
}
