package store

import (
	"strings"

	"myboardgamecollection/internal/filter"
	"myboardgamecollection/internal/model"
)

// GetAllGames returns all games ordered by name.
func (s *Store) GetAllGames() ([]model.Game, error) {
	rows, err := s.db.Query("SELECT " + gameColumns + " FROM games ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// GetGame returns a single game by ID.
func (s *Store) GetGame(id int64) (model.Game, error) {
	return scanGame(s.db.QueryRow("SELECT "+gameColumns+" FROM games WHERE id = ?", id))
}

// GetGameByBGGID returns a game by its BoardGameGeek ID.
func (s *Store) GetGameByBGGID(bggID int64) (model.Game, error) {
	return scanGame(s.db.QueryRow("SELECT "+gameColumns+" FROM games WHERE bgg_id = ?", bggID))
}

// CreateGame inserts a new game and returns its ID.
func (s *Store) CreateGame(g model.Game) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO games (bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, rules_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.BGGID, g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail,
		g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.RulesURL,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateGameRulesURL sets the rules URL for a game.
func (s *Store) UpdateGameRulesURL(id int64, rulesURL string) error {
	_, err := s.db.Exec("UPDATE games SET rules_url = ? WHERE id = ?", rulesURL, id)
	return err
}

// DeleteGame removes a game by ID.
func (s *Store) DeleteGame(id int64) error {
	_, err := s.db.Exec("DELETE FROM games WHERE id = ?", id)
	return err
}

// GameCount returns the total number of games.
func (s *Store) GameCount() int {
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	return count
}

// OwnedBGGIDs returns a set of BGG IDs already in the collection.
func (s *Store) OwnedBGGIDs() (map[int64]bool, error) {
	rows, err := s.db.Query("SELECT bgg_id FROM games")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		m[id] = true
	}
	return m, rows.Err()
}

// FilterGames returns games matching the given category, players, and playtime filters.
func (s *Store) FilterGames(category, players, playtime string) ([]model.Game, error) {
	var conditions []string
	var args []any

	if category != "" {
		conditions = append(conditions, "categories LIKE ?")
		args = append(args, "%"+category+"%")
	}
	if cond := filter.PlayerCondition(players, ""); cond != "" {
		conditions = append(conditions, cond)
	}
	if cond := filter.PlaytimeCondition(playtime, ""); cond != "" {
		conditions = append(conditions, cond)
	}

	query := "SELECT " + gameColumns + " FROM games"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// FilterGamesByVibe returns games tagged with the given vibe, with optional extra filters.
func (s *Store) FilterGamesByVibe(vibeID int64, typ, category, mechanic, players, playtime string) ([]model.Game, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "g.id IN (SELECT game_id FROM game_vibes WHERE vibe_id = ?)")
	args = append(args, vibeID)

	if typ != "" {
		conditions = append(conditions, "g.types LIKE ?")
		args = append(args, "%"+typ+"%")
	}
	if category != "" {
		conditions = append(conditions, "g.categories LIKE ?")
		args = append(args, "%"+category+"%")
	}
	if mechanic != "" {
		conditions = append(conditions, "g.mechanics LIKE ?")
		args = append(args, "%"+mechanic+"%")
	}
	if cond := filter.PlayerCondition(players, "g."); cond != "" {
		conditions = append(conditions, cond)
	}
	if cond := filter.PlaytimeCondition(playtime, "g."); cond != "" {
		conditions = append(conditions, cond)
	}

	query := "SELECT g.id, g.bgg_id, g.name, g.description, g.year_published, g.image, g.thumbnail, g.min_players, g.max_players, g.play_time, g.categories, g.mechanics, g.types, g.rules_url FROM games g"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY g.name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// DistinctCategories returns all unique categories across all games, sorted.
func (s *Store) DistinctCategories() ([]string, error) {
	rows, err := s.db.Query("SELECT categories FROM games WHERE categories != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var raw []string
	for rows.Next() {
		var cats string
		if err := rows.Scan(&cats); err != nil {
			return nil, err
		}
		raw = append(raw, cats)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return filter.SplitDedupSort(raw), nil
}
