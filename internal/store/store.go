package store

import (
	"database/sql"
	"errors"
	"os"
	"strings"

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
	if err := s.migrateUserData(); err != nil {
		return nil, err
	}
	if err := s.migrateAdminUser(); err != nil {
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
			bgg_id         INTEGER NOT NULL,
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
			rules_url      TEXT    NOT NULL DEFAULT '',
			user_id        INTEGER,
			UNIQUE (user_id, bgg_id)
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
			name TEXT NOT NULL,
			user_id INTEGER,
			UNIQUE (user_id, name)
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

	// Normalized category and mechanic tables (used for filtering).
	// The comma-string columns on games remain as a denormalized display cache.
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS game_categories (
			game_id     INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, category_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS mechanics (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS game_mechanics (
			game_id     INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			mechanic_id INTEGER NOT NULL REFERENCES mechanics(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, mechanic_id)
		)
	`)
	if err != nil {
		return err
	}

	// Users and sessions for multi-user support.
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			username         TEXT    NOT NULL UNIQUE,
			bgg_username     TEXT    NOT NULL DEFAULT '',
			password_hash    TEXT    NOT NULL DEFAULT '',
			email            TEXT    NOT NULL DEFAULT '',
			created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_sync_at     DATETIME,
			sync_count_today INTEGER NOT NULL DEFAULT 0,
			sync_date        TEXT    NOT NULL DEFAULT '',
			is_admin         INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token      TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Migration: add user_id columns to games and vibes (no-op if already present).
	_, _ = s.db.Exec("ALTER TABLE games ADD COLUMN user_id INTEGER REFERENCES users(id)")
	_, _ = s.db.Exec("ALTER TABLE vibes ADD COLUMN user_id INTEGER REFERENCES users(id)")
	// Migration: add kind column to sessions to distinguish browser sessions from
	// API refresh tokens. Existing rows get kind='session' via the DEFAULT.
	_, _ = s.db.Exec("ALTER TABLE sessions ADD COLUMN kind TEXT NOT NULL DEFAULT 'session'")
	// Migration: add is_admin and password_hash column (no-op if already present).
	_, _ = s.db.Exec("ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0")
	_, _ = s.db.Exec("ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''")

	// Migration: add username and email columns for username-based auth (#49).
	// username becomes the login identity; bgg_username is kept for BGG syncing.
	_, _ = s.db.Exec("ALTER TABLE users ADD COLUMN username TEXT NOT NULL DEFAULT ''")
	_, _ = s.db.Exec("ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''")
	// Populate username from bgg_username for existing users that lack one.
	_, _ = s.db.Exec("UPDATE users SET username = bgg_username WHERE username = ''")
	// Create unique index on username (idempotent via IF NOT EXISTS).
	_, _ = s.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE username != ''")
	// bgg_username is no longer unique — multiple users can share a BGG username
	// (e.g. household members syncing the same collection). Drop any pre-existing
	// unique index from older schemas. Auto-named SQLite indexes from inline
	// UNIQUE constraints can't be dropped, but a fresh DB created with the new
	// schema won't have one.
	_, _ = s.db.Exec("DROP INDEX IF EXISTS idx_users_bgg_username")

	if err := s.migratePerUserConstraints(); err != nil {
		return err
	}

	// FTS5 virtual table for full-text search over game name + description.
	// content=games makes the FTS index reference the games table rows.
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS games_fts
		USING fts5(name, description, content=games, content_rowid=id)
	`)
	if err != nil {
		return err
	}
	// Triggers keep the FTS index in sync when games are inserted or deleted.
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS games_fts_insert AFTER INSERT ON games BEGIN
			INSERT INTO games_fts(rowid, name, description)
			VALUES (new.id, new.name, new.description);
		END
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS games_fts_delete AFTER DELETE ON games BEGIN
			INSERT INTO games_fts(games_fts, rowid, name, description)
			VALUES ('delete', old.id, old.name, old.description);
		END
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS games_fts_update AFTER UPDATE ON games BEGIN
			INSERT INTO games_fts(games_fts, rowid, name, description)
			VALUES ('delete', old.id, old.name, old.description);
			INSERT INTO games_fts(rowid, name, description)
			VALUES (new.id, new.name, new.description);
		END
	`)
	if err != nil {
		return err
	}
	// Rebuild the FTS index from the content table. This is idempotent and
	// ensures rows that existed before the FTS table was created are indexed.
	_, err = s.db.Exec("INSERT INTO games_fts(games_fts) VALUES ('rebuild')")
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) migratePerUserConstraints() error {
	gamesNeedsMigration, err := hasSingleColumnUniqueIndex(s.db, "games", "bgg_id")
	if err != nil {
		return err
	}
	if gamesNeedsMigration {
		if err := s.migrateGamesTableForPerUserUniqueness(); err != nil {
			return err
		}
	}

	vibesNeedsMigration, err := hasSingleColumnUniqueIndex(s.db, "vibes", "name")
	if err != nil {
		return err
	}
	if vibesNeedsMigration {
		if err := s.migrateVibesTableForPerUserUniqueness(); err != nil {
			return err
		}
	}

	return nil
}

func hasSingleColumnUniqueIndex(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query("PRAGMA index_list(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return false, err
		}
		if unique == 0 {
			continue
		}

		infoRows, err := db.Query("PRAGMA index_info(" + quoteSQLiteIdentifier(name) + ")")
		if err != nil {
			return false, err
		}

		var indexedColumns []string
		for infoRows.Next() {
			var seqno, cid int
			var indexedColumn string
			if err := infoRows.Scan(&seqno, &cid, &indexedColumn); err != nil {
				infoRows.Close()
				return false, err
			}
			indexedColumns = append(indexedColumns, indexedColumn)
		}
		if err := infoRows.Err(); err != nil {
			infoRows.Close()
			return false, err
		}
		infoRows.Close()

		if len(indexedColumns) == 1 && indexedColumns[0] == columnName {
			return true, nil
		}
	}

	return false, rows.Err()
}

func quoteSQLiteIdentifier(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func (s *Store) migrateGamesTableForPerUserUniqueness() error {
	_, err := s.db.Exec("DROP TRIGGER IF EXISTS games_fts_insert")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DROP TRIGGER IF EXISTS games_fts_delete")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DROP TRIGGER IF EXISTS games_fts_update")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DROP TABLE IF EXISTS games_fts")
	if err != nil {
		return err
	}

	_, err = s.db.Exec("PRAGMA foreign_keys = OFF")
	if err != nil {
		return err
	}
	defer s.db.Exec("PRAGMA foreign_keys = ON")

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		CREATE TABLE games_new (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			bgg_id         INTEGER NOT NULL,
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
			rules_url      TEXT    NOT NULL DEFAULT '',
			types          TEXT    NOT NULL DEFAULT '',
			user_id        INTEGER,
			UNIQUE (user_id, bgg_id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO games_new (
			id, bgg_id, name, description, year_published, image, thumbnail,
			min_players, max_players, play_time, categories, mechanics,
			rules_url, types, user_id
		)
		SELECT
			id, bgg_id, name, description, year_published, image, thumbnail,
			min_players, max_players, play_time, categories, mechanics,
			rules_url, COALESCE(types, ''), user_id
		FROM games
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DROP TABLE games")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE games_new RENAME TO games")
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) migrateVibesTableForPerUserUniqueness() error {
	_, err := s.db.Exec("PRAGMA foreign_keys = OFF")
	if err != nil {
		return err
	}
	defer s.db.Exec("PRAGMA foreign_keys = ON")

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		CREATE TABLE vibes_new (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			name    TEXT NOT NULL,
			user_id INTEGER,
			UNIQUE (user_id, name)
		)
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO vibes_new (id, name, user_id)
		SELECT id, name, user_id
		FROM vibes
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DROP TABLE vibes")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE vibes_new RENAME TO vibes")
	if err != nil {
		return err
	}

	return tx.Commit()
}

// migrateAdminUser promotes the ADMIN_USERNAME user (if they already exist in
// the DB) to is_admin=1. This is idempotent and handles the case where the admin
// user registered before this column was added.
func (s *Store) migrateAdminUser() error {
	admin := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	if admin == "" {
		return nil
	}
	_, err := s.db.Exec("UPDATE users SET is_admin = 1 WHERE username = ? OR bgg_username = ?", admin, admin)
	return err
}

// migrateUserData assigns games and vibes that were created before multi-user
// support (user_id IS NULL) to the user stored in config["bgg_username"]. This
// is a one-time, idempotent migration for upgrading single-user installations.
func (s *Store) migrateUserData() error {
	var orphaned int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM games WHERE user_id IS NULL").Scan(&orphaned); err != nil || orphaned == 0 {
		return err
	}
	username := s.GetConfig("bgg_username")
	if username == "" {
		return nil // no known owner; leave orphaned rows for now
	}

	// For legacy migrations, we check if the user exists.
	var userID int64
	err := s.db.QueryRow("SELECT id FROM users WHERE bgg_username = ?", username).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		// If the user doesn't exist, we create them with a placeholder password.
		// The user will need to use "forgot password" (if implemented) or
		// an admin will need to reset it.
		userID, err = s.RegisterUser(username, "MIGRATED_USER_CHANGE_ME", username, "")
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := s.db.Exec("UPDATE games SET user_id = ? WHERE user_id IS NULL", userID); err != nil {
		return err
	}
	_, err = s.db.Exec("UPDATE vibes SET user_id = ? WHERE user_id IS NULL", userID)
	return err
}

// PopulateTaxonomy fills the normalized category and mechanic tables from the
// denormalized comma-string columns on every existing game row. It is safe to
// call on every startup because all inserts use INSERT OR IGNORE.
func (s *Store) PopulateTaxonomy() error {
	rows, err := s.db.Query("SELECT id, categories, mechanics FROM games")
	if err != nil {
		return err
	}
	defer rows.Close()

	type entry struct {
		id         int64
		categories string
		mechanics  string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.categories, &e.mechanics); err != nil {
			return err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, e := range entries {
		if err := s.upsertGameTaxonomy(e.id, e.categories, e.mechanics); err != nil {
			return err
		}
	}
	return nil
}

// upsertGameTaxonomy inserts category and mechanic rows (and their join rows)
// for a single game. All inserts are INSERT OR IGNORE so it is idempotent.
func (s *Store) upsertGameTaxonomy(gameID int64, categories, mechanics string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertGameTaxonomyTx(tx, gameID, categories, mechanics); err != nil {
		return err
	}

	return tx.Commit()
}

func upsertGameTaxonomyTx(tx *sql.Tx, gameID int64, categories, mechanics string) error {
	for _, name := range splitTaxonomy(categories) {
		if _, err := tx.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", name); err != nil {
			return err
		}
		if _, err := tx.Exec(
			"INSERT OR IGNORE INTO game_categories (game_id, category_id) SELECT ?, id FROM categories WHERE name = ?",
			gameID, name,
		); err != nil {
			return err
		}
	}

	for _, name := range splitTaxonomy(mechanics) {
		if _, err := tx.Exec("INSERT OR IGNORE INTO mechanics (name) VALUES (?)", name); err != nil {
			return err
		}
		if _, err := tx.Exec(
			"INSERT OR IGNORE INTO game_mechanics (game_id, mechanic_id) SELECT ?, id FROM mechanics WHERE name = ?",
			gameID, name,
		); err != nil {
			return err
		}
	}

	return nil
}

// splitTaxonomy splits a comma-separated tag string into trimmed, non-empty terms.
func splitTaxonomy(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ", ") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
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
