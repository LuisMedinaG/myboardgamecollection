package store

import (
	"regexp"
	"strings"

	"myboardgamecollection/internal/filter"
	"myboardgamecollection/internal/model"
)

// ftsSpecialChars matches characters that have special meaning in FTS5 queries.
var ftsSpecialChars = regexp.MustCompile(`[^a-zA-Z0-9 ]`)

// sanitizeFTSQuery strips FTS5 operator characters and builds a prefix-match
// query ("word1* word2*") safe for use in a MATCH expression.
func sanitizeFTSQuery(q string) string {
	safe := strings.TrimSpace(ftsSpecialChars.ReplaceAllString(q, " "))
	words := strings.Fields(safe)
	if len(words) == 0 {
		return ""
	}
	for i, w := range words {
		words[i] = w + "*"
	}
	return strings.Join(words, " ")
}

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

// CreateGame inserts a new game, populates the taxonomy tables, and returns its ID.
func (s *Store) CreateGame(g model.Game) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO games (bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, rules_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.BGGID, g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail,
		g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.RulesURL,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	// Keep normalized taxonomy tables in sync.
	if err := s.upsertGameTaxonomy(id, g.Categories, g.Mechanics); err != nil {
		return id, err
	}
	return id, nil
}

// UpdateGame refreshes a game's BGG data by its BGG ID.
func (s *Store) UpdateGame(g model.Game) error {
	_, err := s.db.Exec(
		"UPDATE games SET name=?, description=?, year_published=?, image=?, thumbnail=?, min_players=?, max_players=?, play_time=?, categories=?, mechanics=?, types=? WHERE bgg_id=?",
		g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail,
		g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.BGGID,
	)
	return err
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

// GamesPageSize is the number of games returned per page by FilterGames.
const GamesPageSize = 20

// buildGameConditions constructs the shared WHERE conditions and argument list
// used by both FilterGames and the accompanying count query.
func buildGameConditions(q, category, players, playtime string) ([]string, []any) {
	var conditions []string
	var args []any

	if q != "" {
		// Guard: sanitizeFTSQuery may return "" if the input is all special
		// chars. Passing an empty string to MATCH would be an FTS5 error.
		if safe := sanitizeFTSQuery(q); safe != "" {
			conditions = append(conditions, "id IN (SELECT rowid FROM games_fts WHERE games_fts MATCH ?)")
			args = append(args, safe)
		}
	}
	if category != "" {
		// Exact match via normalized table — no LIKE substring ambiguity.
		conditions = append(conditions, "id IN (SELECT game_id FROM game_categories gc JOIN categories c ON c.id = gc.category_id WHERE c.name = ?)")
		args = append(args, category)
	}
	if cond := filter.PlayerCondition(players, ""); cond != "" {
		conditions = append(conditions, cond)
	}
	if cond := filter.PlaytimeCondition(playtime, ""); cond != "" {
		conditions = append(conditions, cond)
	}
	return conditions, args
}

// FilterGames returns one page of games matching the given filters plus the
// total number of matching games (for pagination). Page is 1-based.
func (s *Store) FilterGames(q, category, players, playtime string, page int) ([]model.Game, int, error) {
	conditions, args := buildGameConditions(q, category, players, playtime)

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count all matching rows first (same WHERE, no LIMIT).
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM games"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Paginated result set.
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * GamesPageSize
	query := "SELECT " + gameColumns + " FROM games" + where + " ORDER BY name LIMIT ? OFFSET ?"

	rows, err := s.db.Query(query, append(args, GamesPageSize, offset)...)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()
	games, err := scanGames(rows)
	return games, total, err
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
		conditions = append(conditions, "g.id IN (SELECT game_id FROM game_categories gc JOIN categories c ON c.id = gc.category_id WHERE c.name = ?)")
		args = append(args, category)
	}
	if mechanic != "" {
		conditions = append(conditions, "g.id IN (SELECT game_id FROM game_mechanics gm JOIN mechanics m ON m.id = gm.mechanic_id WHERE m.name = ?)")
		args = append(args, mechanic)
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

// DistinctCategories returns all category names from the normalized table, sorted.
func (s *Store) DistinctCategories() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cats = append(cats, name)
	}
	return cats, rows.Err()
}
