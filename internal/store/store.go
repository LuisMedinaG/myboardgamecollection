package store

import (
	"database/sql"

	"myboardgamecollection/internal/model"

	_ "modernc.org/sqlite"
)

// Store wraps the database connection and provides all data access methods.
type Store struct {
	db *sql.DB
}

// New opens the SQLite database, runs migrations, and returns a ready Store.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA foreign_keys=ON")

	s := &Store{db: db}
	if err := s.createTables(); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) createTables() error {
	_, err := s.db.Exec(`
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
	_, err = s.db.Exec(`
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
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS vibes (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS game_vibes (
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			vibe_id INTEGER NOT NULL REFERENCES vibes(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, vibe_id)
		)
	`)
	if err != nil {
		return err
	}
	// Migration: add types column if missing (for existing DBs).
	_, _ = s.db.Exec("ALTER TABLE games ADD COLUMN types TEXT NOT NULL DEFAULT ''")
	return nil
}

const gameColumns = "id, bgg_id, name, description, year_published, image, thumbnail, min_players, max_players, play_time, categories, mechanics, types, rules_url"

func scanGame(row interface{ Scan(...any) error }) (model.Game, error) {
	var g model.Game
	err := row.Scan(&g.ID, &g.BGGID, &g.Name, &g.Description, &g.YearPublished,
		&g.Image, &g.Thumbnail, &g.MinPlayers, &g.MaxPlayers, &g.PlayTime,
		&g.Categories, &g.Mechanics, &g.Types, &g.RulesURL)
	return g, err
}

func scanGames(rows *sql.Rows) ([]model.Game, error) {
	var games []model.Game
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	return games, rows.Err()
}
