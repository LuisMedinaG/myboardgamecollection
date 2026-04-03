package main

import (
	"database/sql"
	"fmt"
	"strconv"
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
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT    NOT NULL,
			genre      TEXT    NOT NULL DEFAULT '',
			subgenre   TEXT    NOT NULL DEFAULT '',
			min_players INTEGER NOT NULL DEFAULT 1,
			max_players INTEGER NOT NULL DEFAULT 4,
			playtime   INTEGER NOT NULL DEFAULT 30,
			quickref   TEXT    NOT NULL DEFAULT '',
			rules_url  TEXT    NOT NULL DEFAULT ''
		)
	`)
	return err
}

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
			Name: "Catan", Genre: "strategy", Subgenre: "resource-management",
			MinPlayers: 3, MaxPlayers: 4, Playtime: 90,
			QuickRef: "Each turn: roll dice, collect resources, trade, build.\n\nEasy to forget:\n- Robber activates on a 7 (move robber, steal 1 card from adjacent player)\n- Players with 8+ cards on a 7 must discard half\n- Longest road needs 5+ segments\n- Largest army needs 3+ knight cards\n- You can trade 4:1 with the bank anytime\n- Ports let you trade 3:1 (general) or 2:1 (specific resource)",
			RulesURL: "https://boardgamegeek.com/boardgame/13/catan",
		},
		{
			Name: "Codenames", Genre: "party", Subgenre: "word-game",
			MinPlayers: 4, MaxPlayers: 8, Playtime: 20,
			QuickRef: "Two teams, each with a spymaster. Spymasters give one-word clues + a number.\n\nEasy to forget:\n- Clue must be ONE word and a number\n- If you guess the assassin word, your team loses instantly\n- You always get one extra guess beyond the number given\n- Spymasters must keep a straight face",
			RulesURL: "https://boardgamegeek.com/boardgame/178900/codenames",
		},
		{
			Name: "King of Tokyo", Genre: "family", Subgenre: "dice-game",
			MinPlayers: 2, MaxPlayers: 6, Playtime: 30,
			QuickRef: "Roll 6 dice up to 3 times (Yahtzee-style). Score points, attack, heal, or gain energy.\n\nEasy to forget:\n- You cannot heal while in Tokyo\n- You MUST enter Tokyo if it's empty\n- When you take damage in Tokyo, you can yield (leave)\n- 3 of a kind scores that number in points; each extra die = +1 point\n- First to 20 points OR last monster standing wins",
			RulesURL: "https://boardgamegeek.com/boardgame/70323/king-of-tokyo",
		},
	}

	for _, g := range seeds {
		if _, err := createGame(g); err != nil {
			return fmt.Errorf("seed %q: %w", g.Name, err)
		}
	}
	return nil
}

// CRUD operations

func getAllGames() ([]Game, error) {
	rows, err := db.Query("SELECT id, name, genre, subgenre, min_players, max_players, playtime, quickref, rules_url FROM games ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGames(rows)
}

func getGame(id int64) (Game, error) {
	var g Game
	err := db.QueryRow(
		"SELECT id, name, genre, subgenre, min_players, max_players, playtime, quickref, rules_url FROM games WHERE id = ?", id,
	).Scan(&g.ID, &g.Name, &g.Genre, &g.Subgenre, &g.MinPlayers, &g.MaxPlayers, &g.Playtime, &g.QuickRef, &g.RulesURL)
	return g, err
}

func createGame(g Game) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO games (name, genre, subgenre, min_players, max_players, playtime, quickref, rules_url) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		g.Name, g.Genre, g.Subgenre, g.MinPlayers, g.MaxPlayers, g.Playtime, g.QuickRef, g.RulesURL,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateGame(g Game) error {
	_, err := db.Exec(
		"UPDATE games SET name=?, genre=?, subgenre=?, min_players=?, max_players=?, playtime=?, quickref=?, rules_url=? WHERE id=?",
		g.Name, g.Genre, g.Subgenre, g.MinPlayers, g.MaxPlayers, g.Playtime, g.QuickRef, g.RulesURL, g.ID,
	)
	return err
}

func deleteGame(id int64) error {
	_, err := db.Exec("DELETE FROM games WHERE id = ?", id)
	return err
}

// Filtering

func filterGames(genre, players, playtime string) ([]Game, error) {
	var conditions []string
	var args []any

	if genre != "" {
		conditions = append(conditions, "genre = ?")
		args = append(args, genre)
	}
	if players != "" {
		if n, err := strconv.Atoi(players); err == nil {
			conditions = append(conditions, "min_players <= ? AND max_players >= ?")
			args = append(args, n, n)
		}
	}
	if playtime != "" {
		switch playtime {
		case "short":
			conditions = append(conditions, "playtime < 30")
		case "medium":
			conditions = append(conditions, "playtime >= 30 AND playtime <= 60")
		case "long":
			conditions = append(conditions, "playtime > 60")
		}
	}

	query := "SELECT id, name, genre, subgenre, min_players, max_players, playtime, quickref, rules_url FROM games"
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

func distinctSubgenres() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT subgenre FROM games WHERE subgenre != '' ORDER BY subgenre")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func distinctGenres() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT genre FROM games WHERE genre != '' ORDER BY genre")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		genres = append(genres, g)
	}
	return genres, rows.Err()
}

func scanGames(rows *sql.Rows) ([]Game, error) {
	var games []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.Name, &g.Genre, &g.Subgenre, &g.MinPlayers, &g.MaxPlayers, &g.Playtime, &g.QuickRef, &g.RulesURL); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}
