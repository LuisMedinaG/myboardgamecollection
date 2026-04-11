package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"myboardgamecollection/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigratesLegacyGlobalUniquenessToPerUserConstraints(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	createLegacyMultiUserDB(t, dbPath)

	s, err := New(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	// Existing rows survive the migration with ids and associations intact.
	game, err := s.GetGame(10, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2001), game.BGGID)
	assert.Equal(t, "Legacy Game", game.Name)

	vibes, err := s.VibesForGame(10)
	require.NoError(t, err)
	require.Len(t, vibes, 1)
	assert.Equal(t, int64(20), vibes[0].ID)
	assert.Equal(t, "Cozy", vibes[0].Name)

	aids, err := s.GetPlayerAids(10)
	require.NoError(t, err)
	require.Len(t, aids, 1)
	assert.Equal(t, int64(30), aids[0].ID)
	assert.Equal(t, "aid.png", aids[0].Filename)

	results, total, err := s.FilterGames("Legacy", "", "", "", 1, DefaultPageSize, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, int64(10), results[0].ID)

	// The migrated schema now allows per-user duplicates.
	secondGameID, err := s.CreateGame(model.Game{
		BGGID:       2001,
		Name:        "Legacy Game",
		Description: "Owned by a different user after migration",
	}, 2)
	require.NoError(t, err)
	assert.Positive(t, secondGameID)

	secondVibeID, err := s.CreateVibe("Cozy", 2)
	require.NoError(t, err)
	assert.Positive(t, secondVibeID)
}

func createLegacyMultiUserDB(t *testing.T, path string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()

	statements := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			bgg_username TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_sync_at DATETIME,
			sync_count_today INTEGER NOT NULL DEFAULT 0,
			sync_date TEXT NOT NULL DEFAULT '',
			is_admin INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE games (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bgg_id INTEGER NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			year_published INTEGER NOT NULL DEFAULT 0,
			image TEXT NOT NULL DEFAULT '',
			thumbnail TEXT NOT NULL DEFAULT '',
			min_players INTEGER NOT NULL DEFAULT 1,
			max_players INTEGER NOT NULL DEFAULT 4,
			play_time INTEGER NOT NULL DEFAULT 30,
			categories TEXT NOT NULL DEFAULT '',
			mechanics TEXT NOT NULL DEFAULT '',
			rules_url TEXT NOT NULL DEFAULT '',
			types TEXT NOT NULL DEFAULT '',
			user_id INTEGER REFERENCES users(id)
		)`,
		`CREATE TABLE vibes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			user_id INTEGER REFERENCES users(id)
		)`,
		`CREATE TABLE game_vibes (
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			vibe_id INTEGER NOT NULL REFERENCES vibes(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, vibe_id)
		)`,
		`CREATE TABLE player_aids (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			filename TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE game_categories (
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, category_id)
		)`,
		`CREATE TABLE mechanics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE game_mechanics (
			game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
			mechanic_id INTEGER NOT NULL REFERENCES mechanics(id) ON DELETE CASCADE,
			PRIMARY KEY (game_id, mechanic_id)
		)`,
		`CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			kind TEXT NOT NULL DEFAULT 'session'
		)`,
		`INSERT INTO users (id, username, bgg_username, password_hash, email) VALUES
			(1, 'alice', 'alice_bgg', 'legacyhash', 'alice@example.com'),
			(2, 'bob', 'bob_bgg', 'legacyhash', 'bob@example.com')`,
		`INSERT INTO games (
			id, bgg_id, name, description, year_published, image, thumbnail,
			min_players, max_players, play_time, categories, mechanics,
			rules_url, types, user_id
		) VALUES (
			10, 2001, 'Legacy Game', 'Imported before per-user uniqueness',
			2020, '', '', 2, 4, 60, 'Strategy', 'Drafting', '', 'Board Game', 1
		)`,
		`INSERT INTO vibes (id, name, user_id) VALUES (20, 'Cozy', 1)`,
		`INSERT INTO game_vibes (game_id, vibe_id) VALUES (10, 20)`,
		`INSERT INTO player_aids (id, game_id, filename, label) VALUES (30, 10, 'aid.png', 'Turn order')`,
		`INSERT INTO categories (id, name) VALUES (40, 'Strategy')`,
		`INSERT INTO game_categories (game_id, category_id) VALUES (10, 40)`,
		`INSERT INTO mechanics (id, name) VALUES (50, 'Drafting')`,
		`INSERT INTO game_mechanics (game_id, mechanic_id) VALUES (10, 50)`,
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt)
		require.NoError(t, err, "exec legacy schema statement")
	}
}
