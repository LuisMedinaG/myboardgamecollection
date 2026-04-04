package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) error {
	var err error
	db, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA foreign_keys=ON")
	return createTables()
}

func createTables() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS games (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			bgg_id         INTEGER NOT NULL UNIQUE,
			name           TEXT    NOT NULL,
			description    TEXT    NOT NULL DEFAULT '',
			year_published INTEGER NOT NULL DEFAULT 0,
			image          TEXT    NOT NULL DEFAULT '',
			thumbnail      TEXT    NOT NULL DEFAULT '',
			min_players    INTEGER NOT NULL DEFAULT 1,
			max_players    INTEGER NOT NULL DEFAULT 4,
			play_time      INTEGER NOT NULL DEFAULT 30,
			categories     TEXT    NOT NULL DEFAULT '',
			mechanics      TEXT    NOT NULL DEFAULT '',
			rules_url      TEXT    NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS player_aids (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id  INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			filename TEXT    NOT NULL,
			label    TEXT    NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vibes (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS game_vibes (
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			vibe_id INTEGER NOT NULL REFERENCES vibes(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, vibe_id)
		)
	`)
	if err != nil {
		return err
	}
	// Add types column if missing (migration for existing DBs)
	_, _ = db.Exec("ALTER TABLE games ADD COLUMN types TEXT NOT NULL DEFAULT ''")
	return seedDefaultVibes()
}

func setConfig(key, value string) error {
	_, err := db.Exec("INSERT INTO config (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?", key, value, value)
	return err
}

func getConfig(key string) string {
	var v string
	_ = db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&v)
	return v
}

const gameColumns = "id, bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, rules_url"

func scanGame(row interface{ Scan(...any) error }) (Game, error) {
	var g Game
	err := row.Scan(&g.ID, &g.BGGID, &g.Name, &g.Description, &g.YearPublished, &g.Image, &g.Thumbnail, &g.MinPlayers, &g.MaxPlayers, &g.PlayTime, &g.Categories, &g.Mechanics, &g.Types, &g.RulesURL)
	return g, err
}

func scanGames(rows *sql.Rows) ([]Game, error) {
	var games []Game
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

// CRUD operations

func getAllGames() ([]Game, error) {
	rows, err := db.Query("SELECT " + gameColumns + " FROM games ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

func getGame(id int64) (Game, error) {
	return scanGame(db.QueryRow("SELECT "+gameColumns+" FROM games WHERE id = ?", id))
}

func getGameByBGGID(bggID int64) (Game, error) {
	return scanGame(db.QueryRow("SELECT "+gameColumns+" FROM games WHERE bgg_id = ?", bggID))
}

func createGame(g Game) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO games (bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, rules_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.BGGID, g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail, g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.Types, g.RulesURL,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateGameRulesURL(id int64, rulesURL string) error {
	_, err := db.Exec("UPDATE games SET rules_url = ? WHERE id = ?", rulesURL, id)
	return err
}

func deleteGame(id int64) error {
	_, err := db.Exec("DELETE FROM games WHERE id = ?", id)
	return err
}

// ownedBGGIDs returns a set of BGG IDs already in the collection.
func ownedBGGIDs() (map[int64]bool, error) {
	rows, err := db.Query("SELECT bgg_id FROM games")
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

// Player aids

func getPlayerAids(gameID int64) ([]PlayerAid, error) {
	rows, err := db.Query("SELECT id, game_id, filename, label FROM player_aids WHERE game_id = ? ORDER BY id", gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aids []PlayerAid
	for rows.Next() {
		var a PlayerAid
		if err := rows.Scan(&a.ID, &a.GameID, &a.Filename, &a.Label); err != nil {
			return nil, err
		}
		aids = append(aids, a)
	}
	return aids, rows.Err()
}

func createPlayerAid(gameID int64, filename, label string) (int64, error) {
	res, err := db.Exec("INSERT INTO player_aids (game_id, filename, label) VALUES (?, ?, ?)", gameID, filename, label)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deletePlayerAid(id int64) error {
	_, err := db.Exec("DELETE FROM player_aids WHERE id = ?", id)
	return err
}

func getPlayerAid(id int64) (PlayerAid, error) {
	var a PlayerAid
	err := db.QueryRow("SELECT id, game_id, filename, label FROM player_aids WHERE id = ?", id).Scan(&a.ID, &a.GameID, &a.Filename, &a.Label)
	return a, err
}

// Seed data

func seedIfEmpty() error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seeds := []Game{
		{
			BGGID: 13, Name: "Catan", YearPublished: 1995,
			Description: "Collect and trade resources to build up the island of Catan in this modern classic. Players try to be the dominant force on the island by building settlements, cities, and roads. On each turn dice are rolled to determine what resources the island produces.",
			Image:     "https://picsum.photos/seed/bgg13/400/400",
			Thumbnail: "https://picsum.photos/seed/bgg13/200/200",
			MinPlayers: 3, MaxPlayers: 4, PlayTime: 90,
			Categories: "Economic, Negotiation",
			Mechanics:  "Dice Rolling, Hand Management, Network and Route Building, Resource Management, Trading",
			Types:      "Family Games, Strategy Games",
		},
		{
			BGGID: 178900, Name: "Codenames", YearPublished: 2015,
			Description: "Give one-word clues to help your team identify secret agents. Two rival spymasters know the secret identities of 25 agents. Their teammates know the agents only by their codenames.",
			Image:     "https://picsum.photos/seed/bgg178900/400/400",
			Thumbnail: "https://picsum.photos/seed/bgg178900/200/200",
			MinPlayers: 2, MaxPlayers: 8, PlayTime: 15,
			Categories: "Card Game, Deduction, Party Game, Word Game",
			Mechanics:  "Communication Limits, Push Your Luck, Team-Based Game",
			Types:      "Family Games, Party Games",
		},
		{
			BGGID: 70323, Name: "King of Tokyo", YearPublished: 2011,
			Description: "Mutant monsters, gigantic robots, and strange aliens battle to become the King of Tokyo. Roll dice, smash your opponents, and claim the city!",
			Image:     "https://picsum.photos/seed/bgg70323/400/400",
			Thumbnail: "https://picsum.photos/seed/bgg70323/200/200",
			MinPlayers: 2, MaxPlayers: 6, PlayTime: 30,
			Categories: "Dice, Fighting, Science Fiction",
			Mechanics:  "Dice Rolling, King of the Hill, Player Elimination, Press Your Luck",
			Types:      "Family Games",
		},
		{
			BGGID: 174430, Name: "Gloomhaven", YearPublished: 2017,
			Description: "Vanquish monsters with strategic cardplay in a persistent legacy campaign. Players take on the role of wandering adventurers with their own special set of skills in this tactical combat game.",
			Image:     "https://picsum.photos/seed/bgg174430/400/400",
			Thumbnail: "https://picsum.photos/seed/bgg174430/200/200",
			MinPlayers: 1, MaxPlayers: 4, PlayTime: 120,
			Categories: "Adventure, Exploration, Fantasy, Fighting, Miniatures",
			Mechanics:  "Action Queue, Campaign, Cooperative Game, Grid Movement, Hand Management, Modular Board",
			Types:      "Strategy Games, Thematic Games",
			RulesURL:  "https://drive.google.com/file/d/1pPpSCCFWOaNUPe2GqXkzsLjjOF6KC2Bi/view",
		},
		{
			BGGID: 167791, Name: "Terraforming Mars", YearPublished: 2016,
			Description: "Compete with rival CEOs to make Mars habitable and build your corporate empire. Initiate huge projects to raise the temperature, oxygen level, and ocean coverage until the environment is livable.",
			Image:     "https://picsum.photos/seed/bgg167791/400/400",
			Thumbnail: "https://picsum.photos/seed/bgg167791/200/200",
			MinPlayers: 1, MaxPlayers: 5, PlayTime: 120,
			Categories: "Economic, Industry, Science Fiction, Territory Building",
			Mechanics:  "Card Drafting, End Game Bonuses, Hand Management, Hexagonal Grid, Tile Placement, Variable Player Powers",
			Types:      "Strategy Games",
		},
	}

	for _, g := range seeds {
		if _, err := createGame(g); err != nil {
			return fmt.Errorf("seed %q: %w", g.Name, err)
		}
	}
	return nil
}

// Filtering

func filterGames(category, players, playtime string) ([]Game, error) {
	var conditions []string
	var args []any

	if category != "" {
		conditions = append(conditions, "categories LIKE ?")
		args = append(args, "%"+category+"%")
	}
	if players != "" {
		switch players {
		case "1":
			conditions = append(conditions, "min_players <= 1")
		case "2only":
			conditions = append(conditions, "min_players = 2 AND max_players = 2")
		case "3":
			conditions = append(conditions, "min_players <= 3")
		case "4":
			conditions = append(conditions, "min_players <= 4")
		case "5plus":
			conditions = append(conditions, "max_players >= 5")
		}
	}
	if playtime != "" {
		switch playtime {
		case "short":
			conditions = append(conditions, "play_time < 30")
		case "medium":
			conditions = append(conditions, "play_time >= 30 AND play_time <= 60")
		case "long":
			conditions = append(conditions, "play_time > 60")
		}
	}

	query := "SELECT " + gameColumns + " FROM games"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

// Vibes

func seedDefaultVibes() error {
	defaults := []string{"Party", "Family Dinner", "Light Friend Night", "Heavy Euro", "Strangers Meeting"}
	for _, name := range defaults {
		_, _ = db.Exec("INSERT OR IGNORE INTO vibes (name) VALUES (?)", name)
	}
	return nil
}

func allVibes() ([]Vibe, error) {
	rows, err := db.Query("SELECT id, name FROM vibes ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vibes []Vibe
	for rows.Next() {
		var v Vibe
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		vibes = append(vibes, v)
	}
	return vibes, rows.Err()
}

func getVibe(id int64) (Vibe, error) {
	var v Vibe
	err := db.QueryRow("SELECT id, name FROM vibes WHERE id = ?", id).Scan(&v.ID, &v.Name)
	return v, err
}

func createVibe(name string) (int64, error) {
	res, err := db.Exec("INSERT INTO vibes (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateVibe(id int64, name string) error {
	_, err := db.Exec("UPDATE vibes SET name = ? WHERE id = ?", name, id)
	return err
}

func deleteVibe(id int64) error {
	_, err := db.Exec("DELETE FROM vibes WHERE id = ?", id)
	return err
}

func vibesForGame(gameID int64) ([]Vibe, error) {
	rows, err := db.Query(`
		SELECT v.id, v.name FROM vibes v
		JOIN game_vibes gv ON gv.vibe_id = v.id
		WHERE gv.game_id = ?
		ORDER BY v.name`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vibes []Vibe
	for rows.Next() {
		var v Vibe
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		vibes = append(vibes, v)
	}
	return vibes, rows.Err()
}

func setGameVibes(gameID int64, vibeIDs []int64) error {
	tx, err := db.Begin()
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

func filterGamesByVibe(vibeID int64, typ, category, mechanic, players, playtime string) ([]Game, error) {
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
	if players != "" {
		switch players {
		case "1":
			conditions = append(conditions, "g.min_players <= 1")
		case "2":
			conditions = append(conditions, "g.min_players <= 2")
		case "2only":
			conditions = append(conditions, "g.min_players = 2 AND g.max_players = 2")
		case "3":
			conditions = append(conditions, "g.min_players <= 3")
		case "4":
			conditions = append(conditions, "g.min_players <= 4")
		case "5plus":
			conditions = append(conditions, "g.max_players >= 5")
		}
	}
	if playtime != "" {
		switch playtime {
		case "short":
			conditions = append(conditions, "g.play_time < 30")
		case "medium":
			conditions = append(conditions, "g.play_time >= 30 AND g.play_time <= 60")
		case "long":
			conditions = append(conditions, "g.play_time > 60")
		}
	}

	query := "SELECT g." + strings.Replace(gameColumns, ", ", ", g.", -1) + " FROM games g"
	// Fix: gameColumns starts with "id" so we need the prefix on first col too
	query = "SELECT g.id, g.bgg_id, g.name, g.description, g.year_published, g.image, g.thumbnail, g.min_players, g.max_players, g.play_time, g.categories, g.mechanics, g.types, g.rules_url FROM games g"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY g.name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

func extractField(games []Game, field func(Game) string) []string {
	seen := make(map[string]bool)
	for _, g := range games {
		for _, v := range strings.Split(field(g), ", ") {
			v = strings.TrimSpace(v)
			if v != "" {
				seen[v] = true
			}
		}
	}
	var result []string
	for v := range seen {
		result = append(result, v)
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

func typesForGames(games []Game) []string {
	return extractField(games, func(g Game) string { return g.Types })
}

func categoriesForGames(games []Game) []string {
	return extractField(games, func(g Game) string { return g.Categories })
}

func mechanicsForGames(games []Game) []string {
	return extractField(games, func(g Game) string { return g.Mechanics })
}

func distinctCategories() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT categories FROM games WHERE categories != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var cats string
		if err := rows.Scan(&cats); err != nil {
			return nil, err
		}
		for _, c := range strings.Split(cats, ", ") {
			c = strings.TrimSpace(c)
			if c != "" {
				seen[c] = true
			}
		}
	}

	var result []string
	for c := range seen {
		result = append(result, c)
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, rows.Err()
}
