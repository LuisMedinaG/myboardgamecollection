# My Board Game Collection — CLAUDE.md

Personal app to manage a board game collection — track owned games, store rulebook links,
upload player aids, and import games from BoardGameGeek (BGG).

## Stack

- **Language:** Go 1.25 (standard library HTTP server)
- **Frontend:** Go HTML templates + HTMX — no JS framework
- **Database:** SQLite (`modernc.org/sqlite`, pure Go)
- **Auth:** Session-based (username/password, SHA-256 + salt — upgrade to argon2 is planned)
- **Deployment:** Docker + Fly.io (persistent volume at `/data`)

## Commands

```sh
make dev        # go run .
make run        # build + run
make build      # outputs ./boardgames binary
make bgg-login  # utility to grab BGG auth headers
```

No test suite — verification is manual.

## Project Structure

```
main.go              # Server setup, routes, middleware wiring
internal/
  handler/           # HTTP handlers — HTMX (auth, games, import, vibes, rules, discover)
                     #                  API  (api_auth, api_games, api_vibes, api_rules,
                     #                        api_discover, api_import, api_profile)
  store/             # SQLite data access layer
  httpx/             # Middleware (auth, CSRF, rate-limit, security headers, JWT)
  bgg/               # BGG API client wrapper
  render/            # Template renderer (pre-parses all templates on startup)
  model/             # Domain structs
  viewmodel/         # View-layer data passed to templates
  filter/            # Game filtering logic
templates/           # Embedded HTML templates
static/              # Embedded CSS
```

## In-Progress: JWT REST API

A parallel JSON REST API is being added under `/api/v1/` alongside the existing HTMX app.
The HTMX frontend and all existing routes remain untouched throughout this work.

- **Auth:** `github.com/golang-jwt/jwt/v5` — access tokens (15 min JWT) + refresh tokens (30 day, stored in sessions table)
- **Middleware:** `RequireJWT(secret)` in `internal/httpx/` — reads `Authorization: Bearer`, returns 401 JSON on failure
- **Handlers:** All API handlers live in `internal/handler/api_*.go`
- **Responses:** `{ "data": ... }` for success, `{ "error": "..." }` for failures

Phases:
1. JWT foundation + `/api/v1/auth/*` — login, refresh, logout
2. Core data — games, vibes, import, profile
3. Rules, player aids, discovery

## More Detail

See `agent_docs/` for routes, env vars, and key patterns.

Never run `git push` without the user explicitly asking to push.

**Why:** User was surprised when a push happened as part of a "commit" request — they only wanted a commit, not a push.

**How to apply:** After committing, stop and tell the user the commit is ready. Only push if they separately say "push" or "make a PR" (which requires a push). When in doubt, ask.
