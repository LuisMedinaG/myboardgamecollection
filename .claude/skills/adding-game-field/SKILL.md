---
name: adding-game-field
description: Adds a new column to the games table in this Go+SQLite board game app, wiring it through the model, store DDL, legacy migration, BGG import, and API converter. Use when the user asks to add a game attribute or BGG field (e.g. "add min-age", "store the publisher"), when adding any column to the games table, or when backfilling BGG metadata onto existing games.
---

# Adding a field to the games table

Every new column on `games` touches five places. Miss any one and either the schema, fresh installs, legacy installs, or the API will disagree.

## Required touch points

Copy this checklist and tick off items as you go:

```
- [ ] 1. internal/model/model.go — add field to Game struct
- [ ] 2. internal/store/store.go — ALTER TABLE in createTables()
- [ ] 3. internal/store/store.go — games_new DDL + INSERT list in migrateGamesTableForPerUserUniqueness()
- [ ] 4. internal/store/store.go — gameColumns const + scanGame()
- [ ] 5. internal/store/game.go — CreateGame INSERT and UpdateGame SET lists
- [ ] 6. internal/bgg/bgg.go — populate in bggItemToGame() (if BGG-sourced)
- [ ] 7. internal/handler/api_helpers.go — gameToAPI snake_case mapping
- [ ] 8. Call out Full Refresh requirement if backfilling existing games
```

## Step 1 — Model

Add the field to the `Game` struct in `internal/model/model.go`. Keep the Go name PascalCase and match the existing alignment of the BGG-sourced block:

```go
Weight             float64
Rating             float64
LanguageDependence int
RecommendedPlayers string
MyNewField         T      // one-line comment if the WHY is non-obvious
```

## Step 2 — Schema additions for fresh DBs

In `internal/store/store.go`, inside `createTables()`, add an idempotent `ALTER TABLE` near the existing block (around the `weight` / `rating` / `language_dependence` / `recommended_players` additions):

```go
_, _ = s.db.Exec("ALTER TABLE games ADD COLUMN my_new_field TYPE NOT NULL DEFAULT default_value")
```

Rules:
- Always `NOT NULL DEFAULT …` (no nullable columns).
- Errors are intentionally ignored — SQLite returns an error when the column already exists; that's fine.
- Use snake_case for the DB column; PascalCase for the Go field.

## Step 3 — Legacy table-recreation migration

`migrateGamesTableForPerUserUniqueness` rebuilds the `games` table for users whose old schema had `UNIQUE(bgg_id)` instead of `UNIQUE(user_id, bgg_id)`. It must know about every column that exists today or it silently drops data on upgrade.

Update **both**:

1. The `CREATE TABLE games_new (...)` DDL — add your column with the same type and default.
2. The `INSERT INTO games_new (...) SELECT ... FROM games` statement — add your column to **both** the column list and the SELECT list. Wrap the SELECT expression in `COALESCE(col, default)` to tolerate rows that predate the column.

Example of the existing pattern:

```go
SELECT
    id, bgg_id, name, …,
    COALESCE(types, ''), COALESCE(weight, 0.0),
    COALESCE(rating, 0.0), COALESCE(language_dependence, 0),
    COALESCE(recommended_players, ''),
    user_id
FROM games
```

## Step 4 — Read path

In `internal/store/store.go`:

- Append the new column to the `gameColumns` const (SELECT list order).
- Append a matching `&g.MyNewField` to `scanGame()` in the exact same order.

The order in `gameColumns` and `scanGame` must match each other and the column list used by `FilterGamesByVibe` (the explicit `SELECT g.id, g.bgg_id, …` string in `internal/store/game.go`). Update that string too.

## Step 5 — Write path

In `internal/store/game.go`:

- `CreateGame` — add the column to the `INSERT INTO games (...)` list, a `?` to the `VALUES (...)`, and pass `g.MyNewField` to `db.Exec`.
- `UpdateGame` — add `my_new_field=?` to the `SET` list and pass `g.MyNewField`.

## Step 6 — BGG mapping (if applicable)

If the field comes from BGG, populate it in `bggItemToGame` in `internal/bgg/bgg.go`. For poll-derived fields, add a `parseXxx(item.Poll)` helper next to `parseLanguageDependence` / `parseRecommendedPlayers`. For stats, extend the `bggRatingsXML` / `bggStatisticsXML` structs.

The XML structs are custom because `gobgg.ThingResult` doesn't expose polls. See [importing-from-bgg](../importing-from-bgg/SKILL.md) if you need to add a new BGG endpoint rather than a new field.

## Step 7 — JSON API

In `internal/handler/api_helpers.go`, add the field to `gameToAPI` with a snake_case JSON key. This is the only converter; all API handlers use it.

```go
"my_new_field": g.MyNewField,
```

## Step 8 — Backfill warning

Normal BGG sync only fetches games that aren't already owned, so existing collections will keep the default value forever. Tell the user: **run a Full Refresh** (admin-only checkbox on the Import page, or `POST /api/v1/import` with `{"full_refresh": true}`) after deploying the change.

## Verification

```sh
make build   # catches type mismatches in model/store/BGG
make test    # migration_test.go exercises the legacy-table rebuild
```

If you want the field to appear in filters or the UI, that's a separate workflow — see [adding-game-filter](../adding-game-filter/SKILL.md) and the view templates in `templates/`.
