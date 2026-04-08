# My Board Game Collection — CLAUDE.md

Personal app to manage a board game collection — track owned games, store rulebook links,
upload player aids, and import games from BoardGameGeek (BGG).

## Stack

- **Language:** Go 1.25 (standard library HTTP server)
- **Frontend:** Go HTML templates + HTMX — no JS framework
- **Database:** SQLite (`modernc.org/sqlite`, pure Go)
- **Auth:** Session-based (username/password, bcrypt)
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
  handler/           # HTTP handlers (auth, games, import, vibes, rules, discover)
  store/             # SQLite data access layer
  httpx/             # Middleware (auth, CSRF, rate-limit, security headers)
  bgg/               # BGG API client wrapper
  render/            # Template renderer (pre-parses all templates on startup)
  model/             # Domain structs
  viewmodel/         # View-layer data passed to templates
  filter/            # Game filtering logic
templates/           # Embedded HTML templates
static/              # Embedded CSS
```

## More Detail

See `agent_docs/` for routes, env vars, and key patterns.
