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
	return err
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

const gameColumns = "id, bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, rules_url"

func scanGame(row interface{ Scan(...any) error }) (Game, error) {
	var g Game
	err := row.Scan(&g.ID, &g.BGGID, &g.Name, &g.Description, &g.YearPublished, &g.Image, &g.Thumbnail, &g.MinPlayers, &g.MaxPlayers, &g.PlayTime, &g.Categories, &g.Mechanics, &g.RulesURL)
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
		"INSERT INTO games (bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, rules_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.BGGID, g.Name, g.Description, g.YearPublished, g.Image, g.Thumbnail, g.MinPlayers, g.MaxPlayers, g.PlayTime, g.Categories, g.Mechanics, g.RulesURL,
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
			Description: "In Catan, players try to be the dominant force on the island of Catan by building settlements, cities, and roads. On each turn dice are rolled to determine what resources the island produces. Players build by spending resources and collect points.",
			Image: "https://cf.geekdo-images.com/W3Bsga_uLP9kO91gZ7H8yw__original/img/o-J27MVjJQVQ5IDYaCfRjYhGy64=/0x0/filters:format(jpeg)/pic2419375.jpg",
			Thumbnail: "https://cf.geekdo-images.com/W3Bsga_uLP9kO91gZ7H8yw__thumb/img/8a9PHqLpCNoGkMiRTgMn6XsVTNc=/fit-in/200x150/filters:strip_icc()/pic2419375.jpg",
			MinPlayers: 3, MaxPlayers: 4, PlayTime: 90,
			Categories: "Economic, Negotiation",
			Mechanics:  "Dice Rolling, Hand Management, Network and Route Building, Resource Management, Trading",
		},
		{
			BGGID: 178900, Name: "Codenames", YearPublished: 2015,
			Description: "Two rival spymasters know the secret identities of 25 agents. Their teammates know the agents only by their codenames. Spymasters give one-word clues that can point to multiple words on the board.",
			Image: "https://cf.geekdo-images.com/F_KDEu0GjdClml8N7c8Imw__original/img/r_LWHM5YBRb5T7bT-zzF4_YN8Sk=/0x0/filters:format(jpeg)/pic259733.jpg",
			Thumbnail: "https://cf.geekdo-images.com/F_KDEu0GjdClml8N7c8Imw__thumb/img/i53gMNTzJdKlgBrEHkOZRdFE0aM=/fit-in/200x150/filters:strip_icc()/pic259733.jpg",
			MinPlayers: 2, MaxPlayers: 8, PlayTime: 15,
			Categories: "Card Game, Deduction, Party Game, Word Game",
			Mechanics:  "Communication Limits, Push Your Luck, Team-Based Game",
		},
		{
			BGGID: 70323, Name: "King of Tokyo", YearPublished: 2011,
			Description: "In King of Tokyo, you play mutant monsters, gigantic robots, and strange aliens, all of whom are destroying Tokyo and whacking each other in order to become the one and only King of Tokyo.",
			Image: "https://cf.geekdo-images.com/4HP1YaPgXLVde5NRAsBJjQ__original/img/XFo4GRPN-dlZ4dqfJHIrBP9Bk9Y=/0x0/filters:format(jpeg)/pic3043734.jpg",
			Thumbnail: "https://cf.geekdo-images.com/4HP1YaPgXLVde5NRAsBJjQ__thumb/img/e7T1VeDW28WFmokS4XjTJVGWrQw=/fit-in/200x150/filters:strip_icc()/pic3043734.jpg",
			MinPlayers: 2, MaxPlayers: 6, PlayTime: 30,
			Categories: "Dice, Fighting, Science Fiction",
			Mechanics:  "Dice Rolling, King of the Hill, Player Elimination, Press Your Luck",
		},
		{
			BGGID: 174430, Name: "Gloomhaven", YearPublished: 2017,
			Description: "Gloomhaven is a game of Euro-inspired tactical combat in a persistent world of shifting motives. Players will take on the role of a wandering adventurer with their own special set of skills and their own reasons for traveling to this dark corner of the world.",
			Image: "https://cf.geekdo-images.com/sZYp_3BTDGjh2unaZfZmuA__original/img/7d-lj5Gd1e8PFnD97LYFah2EPe0=/0x0/filters:format(jpeg)/pic2437871.jpg",
			Thumbnail: "https://cf.geekdo-images.com/sZYp_3BTDGjh2unaZfZmuA__thumb/img/veVoCy-9Rc3T-XV0VH-cFKkprmc=/fit-in/200x150/filters:strip_icc()/pic2437871.jpg",
			MinPlayers: 1, MaxPlayers: 4, PlayTime: 120,
			Categories: "Adventure, Exploration, Fantasy, Fighting, Miniatures",
			Mechanics:  "Action Queue, Campaign, Cooperative Game, Grid Movement, Hand Management, Modular Board",
			RulesURL:   "https://drive.google.com/file/d/1pPpSCCFWOaNUPe2GqXkzsLjjOF6KC2Bi/view",
		},
		{
			BGGID: 167791, Name: "Terraforming Mars", YearPublished: 2016,
			Description: "In the 2400s, mankind begins to terraform the planet Mars. Giant corporations, sponsored by the World Government on Earth, initiate huge projects to raise the temperature, the oxygen level, and the ocean coverage until the environment is habitable.",
			Image: "https://cf.geekdo-images.com/wg9oOLcsKvDesSUdZQ4rxw__original/img/thIqWDnH9utKuYlg6kIBbHuf63I=/0x0/filters:format(jpeg)/pic3536616.jpg",
			Thumbnail: "https://cf.geekdo-images.com/wg9oOLcsKvDesSUdZQ4rxw__thumb/img/xcoMDuy6oooSrP7VhYvfZTaZvLY=/fit-in/200x150/filters:strip_icc()/pic3536616.jpg",
			MinPlayers: 1, MaxPlayers: 5, PlayTime: 120,
			Categories: "Economic, Industry, Science Fiction, Territory Building",
			Mechanics:  "Card Drafting, End Game Bonuses, Hand Management, Hexagonal Grid, Tile Placement, Variable Player Powers",
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
