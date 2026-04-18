---
name: adding-game-filter
description: Adds a new game-list filter (query param, SQL condition, UI chip, API wiring) to this Go+SQLite board game app. Use when the user asks to filter games by some attribute, add a filter chip or dropdown, or expose a new filter URL param on /games or /api/v1/games or /discover.
---

# Adding a game filter

Filters are parameterized SQL fragments plus a thin fan-out through the store, HTMX handler, discover handler, and REST API handler. The pattern is consistent across every existing filter (`players`, `playtime`, `weight`, `rating`, `lang`, `rec_players`).

## Required touch points

```
- [ ] 1. internal/filter/filter.go — add XxxCondition(value, prefix) returning a SQL fragment
- [ ] 2. internal/filter/filter.go — add ValidXxxOptions(games) for UI option culling
- [ ] 3. internal/viewmodel/viewmodel.go — add XxxOption struct + field on the games view model
- [ ] 4. internal/store/game.go — wire into buildGameConditions (and FilterGamesByVibe if discover-relevant)
- [ ] 5. internal/store/game.go — thread the new param through FilterGames and FilterGamesByVibe signatures
- [ ] 6. internal/handler/game.go — read r.URL.Query().Get("xxx") and pass to FilterGames
- [ ] 7. internal/handler/discover.go — same, for FilterGamesByVibe
- [ ] 8. internal/handler/api_games.go — same, for the REST API
- [ ] 9. internal/handler/api_discover.go — same, for the REST discover endpoint
- [ ] 10. templates/ — render the filter chip/dropdown (HTMX)
```

## Step 1 — SQL condition

Conditions live in `internal/filter/filter.go`. Each returns a parenthesis-free SQL fragment, or `""` when the param is empty or invalid. The `prefix` argument supports unaliased queries (`""`) and aliased queries (`"g."` from `FilterGamesByVibe`).

**For enum-like filters** (e.g. `weight`, `rating`, `players`) use a `switch`:

```go
func XxxCondition(xxx, prefix string) string {
    switch xxx {
    case "low":
        return prefix + "xxx < 2.0"
    case "high":
        return prefix + "xxx >= 3.0"
    default:
        return ""
    }
}
```

**For filters that embed a validated literal in SQL** (needed when `?` placeholders can't be used, e.g. LIKE-wrapped constants), follow `RecommendedPlayersCondition` exactly — validate every character first and reject anything unexpected. Example for a digit-only value:

```go
for _, ch := range count {
    if ch < '0' || ch > '9' {
        return ""
    }
}
return "',' || " + prefix + "recommended_players || ',' LIKE '%," + count + ",%'"
```

Never concatenate user input into SQL without a char-by-char allowlist. If the values are enum-like, a `switch` is safer.

## Step 2 — Valid-options helper

`ValidXxxOptions(games []model.Game) []viewmodel.XxxOption` returns only the options that match at least one game in the supplied slice. This drives UI culling so users don't see filter chips that return empty results.

Copy the shape of `ValidWeightOptions` or `ValidLanguageOptions` — inline `def` struct with `value`, `label`, and a `match func(model.Game) bool`, iterate, break on first hit.

## Step 3 — Viewmodel

In `internal/viewmodel/viewmodel.go`, add `XxxOption{Value, Label string}` and surface a `XxxOptions []XxxOption` field on the games-page view model. Follow the existing `PlayerOption` / `WeightOption` / `RatingOption` pattern.

## Step 4 — Store condition wiring

`buildGameConditions` in `internal/store/game.go` is the single place that aggregates filter SQL for `FilterGames`. Append:

```go
if cond := filter.XxxCondition(xxx, ""); cond != "" {
    conditions = append(conditions, cond)
}
```

`FilterGamesByVibe` is a separate function with its own conditions block (uses the `"g."` prefix because it joins `game_vibes`). Add the same wiring there.

## Step 5 — Thread the param

`FilterGames` and `FilterGamesByVibe` take filter values as positional string params. Add your new one **in alphabetical-ish order matching the existing signature**. Example current signature:

```go
func (s *Store) FilterGames(q, category, players, playtime, weight, rating, lang, recPlayers string, page, pageSize int, userID int64) ([]model.Game, int, error)
```

Update every call site — `buildGameConditions` takes the same list. The compiler will find them all, but the order must be consistent across caller and callee.

## Step 6–9 — Handlers

Four handlers read query params and call the store. They're near-identical; keep the variable naming consistent (`xxx := r.URL.Query().Get("xxx")`).

- `internal/handler/game.go` — HTMX games page (`/games`)
- `internal/handler/discover.go` — HTMX vibe discover page
- `internal/handler/api_games.go` — `GET /api/v1/games`
- `internal/handler/api_discover.go` — `GET /api/v1/discover`

The HTMX handlers also populate the view model with `filter.ValidXxxOptions(games)` so the template can render chip buttons.

## Step 10 — Template

Add the filter control to the relevant template(s) under `templates/`. Use the existing chip/dropdown markup for `weight` or `rating` as the pattern. Templates are pre-parsed on startup by `internal/render/render.go`.

## URL param naming

Keep param names short and lowercase (`players`, `playtime`, `weight`, `rating`, `lang`, `rec_players`). Use underscores when disambiguation helps. Don't rename existing params — they're in user bookmarks and the REST API contract.

## Verification

```sh
make test        # covers store.FilterGames with the new param
make dev         # exercise the chip in the browser
curl 'localhost:8080/api/v1/games?xxx=low' -H "Authorization: Bearer $TOKEN"
```
