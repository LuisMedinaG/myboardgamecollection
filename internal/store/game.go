package store

import (
	"database/sql"
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

// GetAllGames returns all games for a user, ordered by name.
func (s *Store) GetAllGames(userID int64) ([]model.Game, error) {
	rows, err := s.db.Query("SELECT "+gameColumns+" FROM games WHERE user_id = ? ORDER BY name", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// GetGame returns a single game by ID, verifying it belongs to the given user.
func (s *Store) GetGame(id, userID int64) (model.Game, error) {
	return scanGame(s.db.QueryRow(
		"SELECT "+gameColumns+" FROM games WHERE id = ? AND user_id = ?", id, userID,
	))
}

// GetGameByBGGID returns a game by its BoardGameGeek ID for the given user.
func (s *Store) GetGameByBGGID(bggID, userID int64) (model.Game, error) {
	return scanGame(s.db.QueryRow(
		"SELECT "+gameColumns+" FROM games WHERE bgg_id = ? AND user_id = ?", bggID, userID,
	))
}

// GetThumbnailByBGGID returns the thumbnail URL for any game with the given
// BGG ID, regardless of which user owns it. Used by the image cache handler,
// which is a public route where there is no authenticated user in context.
func (s *Store) GetThumbnailByBGGID(bggID int64) (string, error) {
	var thumbnail string
	err := s.db.QueryRow("SELECT thumbnail FROM games WHERE bgg_id = ? LIMIT 1", bggID).Scan(&thumbnail)
	return thumbnail, err
}

// CreateGame inserts a new game owned by userID, populates taxonomy, and returns its ID.
func (s *Store) CreateGame(g model.Game, userID int64) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO games (bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, weight, rules_url, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.BGGID, g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail,
		g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.Weight, g.RulesURL,
		userID,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.upsertGameTaxonomy(id, g.Categories, g.Mechanics); err != nil {
		return id, err
	}
	return id, nil
}

// UpdateGame refreshes a game's BGG data. Only updates the game if it belongs to userID.
func (s *Store) UpdateGame(g model.Game, userID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"UPDATE games SET name=?, description=?, year_published=?, image=?, thumbnail=?, min_players=?, max_players=?, play_time=?, categories=?, mechanics=?, types=?, weight=? WHERE bgg_id=? AND user_id=?",
		g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail,
		g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.Weight,
		g.BGGID, userID,
	)
	if err != nil {
		return err
	}

	gameID, err := updatedGameID(tx, res, g.BGGID, userID)
	if err != nil {
		return err
	}
	if gameID == 0 {
		return nil
	}

	if _, err := tx.Exec("DELETE FROM game_categories WHERE game_id = ?", gameID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM game_mechanics WHERE game_id = ?", gameID); err != nil {
		return err
	}
	if err := upsertGameTaxonomyTx(tx, gameID, g.Categories, g.Mechanics); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateGameRulesURL sets the rules URL for a game owned by userID.
func (s *Store) UpdateGameRulesURL(id int64, rulesURL string, userID int64) error {
	_, err := s.db.Exec("UPDATE games SET rules_url = ? WHERE id = ? AND user_id = ?", rulesURL, id, userID)
	return err
}

// DeleteGame removes a game owned by userID.
func (s *Store) DeleteGame(id, userID int64) error {
	_, err := s.db.Exec("DELETE FROM games WHERE id = ? AND user_id = ?", id, userID)
	return err
}

// GameCount returns the total number of games owned by userID.
func (s *Store) GameCount(userID int64) int {
	var count int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM games WHERE user_id = ?", userID).Scan(&count)
	return count
}

func updatedGameID(tx *sql.Tx, res sql.Result, bggID, userID int64) (int64, error) {
	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return 0, err
	}

	var gameID int64
	err = tx.QueryRow("SELECT id FROM games WHERE bgg_id = ? AND user_id = ?", bggID, userID).Scan(&gameID)
	return gameID, err
}

// OwnedBGGIDs returns a set of BGG IDs already in the user's collection.
func (s *Store) OwnedBGGIDs(userID int64) (map[int64]bool, error) {
	rows, err := s.db.Query("SELECT bgg_id FROM games WHERE user_id = ?", userID)
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

// DefaultPageSize is the default number of games per page.
const DefaultPageSize = 20

// MaxPageSize caps the client-supplied limit to prevent abuse.
const MaxPageSize = 300

// buildGameConditions constructs the shared WHERE conditions and argument list
// used by both FilterGames and the accompanying count query. userID is always
// included so results are scoped to the requesting user.
func buildGameConditions(q, category, players, playtime, weight string, userID int64) ([]string, []any) {
	conditions := []string{"user_id = ?"}
	args := []any{userID}

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
	if cond := filter.WeightCondition(weight, ""); cond != "" {
		conditions = append(conditions, cond)
	}
	return conditions, args
}

// FilterGames returns one page of games matching the given filters plus the
// total number of matching games (for pagination). Page is 1-based.
// pageSize controls how many results per page; use DefaultPageSize if unsure.
func (s *Store) FilterGames(q, category, players, playtime, weight string, page, pageSize int, userID int64) ([]model.Game, int, error) {
	conditions, args := buildGameConditions(q, category, players, playtime, weight, userID)
	where := " WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM games"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	query := "SELECT " + gameColumns + " FROM games" + where + " ORDER BY name LIMIT ? OFFSET ?"

	rows, err := s.db.Query(query, append(args, pageSize, offset)...)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()
	games, err := scanGames(rows)
	return games, total, err
}

// FilterGamesByVibe returns games tagged with the given vibe, scoped to userID.
func (s *Store) FilterGamesByVibe(vibeID int64, typ, category, mechanic, players, playtime, weight string, userID int64) ([]model.Game, error) {
	conditions := []string{
		"g.user_id = ?",
		"g.id IN (SELECT game_id FROM game_vibes WHERE vibe_id = ?)",
	}
	args := []any{userID, vibeID}

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
	if cond := filter.WeightCondition(weight, "g."); cond != "" {
		conditions = append(conditions, cond)
	}

	query := "SELECT g.id, g.bgg_id, g.name, g.description, g.year_published, g.image, g.thumbnail, g.min_players, g.max_players, g.play_time, g.categories, g.mechanics, g.types, g.weight, g.rules_url FROM games g"
	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY g.name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// DistinctCategories returns all category names for the user's games, sorted.
func (s *Store) DistinctCategories(userID int64) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT c.name
		FROM categories c
		JOIN game_categories gc ON c.id = gc.category_id
		JOIN games g ON g.id = gc.game_id
		WHERE g.user_id = ?
		ORDER BY c.name`, userID)
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
