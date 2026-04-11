# My Board Game Collection

A small Go web app for browsing your board games, opening rulebook links, and storing player-aid images.

**Stack:** Go, `html/template`, SQLite, HTMX

## What It Does

- Shows your collection in a fast server-rendered UI
- Filters games by category, player count, and play time
- Stores a Google Drive rulebook link per game
- Uploads player-aid images to local disk
- Optionally syncs owned games from BoardGameGeek

## Prerequisites

- Go 1.23 or newer
- No C compiler required

## Quick Start

```sh
make run
```

Then open `http://localhost:8080`.

On first run the app creates `games.db`, creates `data/uploads/`, and seeds a few sample games if the database is empty.

## Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the `boardgames` binary |
| `make run` | Build and run the app |
| `make dev` | Run with `go run .` |
| `make test` | Run all tests |
| `make test-v` | Run tests with verbose output |
| `make cover` | Run tests with coverage report |
| `make cover-html` | Generate HTML coverage report |
| `make clean` | Remove the binary and local database |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `games.db` | SQLite database path |
| `BGG_TOKEN` | unset | Enables BoardGameGeek import |
| `ADMIN_USERNAME` | unset | Required to access admin pages and all write actions |
| `ADMIN_PASSWORD` | unset | Required to access admin pages and all write actions |

Example:

```sh
PORT=3000 DB_PATH=./data/collection.db make run
```

If `BGG_TOKEN` is not set, import remains unavailable.

If `ADMIN_USERNAME` and `ADMIN_PASSWORD` are not set, the app fails closed for admin routes: public read-only pages still work, but import/edit/upload/delete routes are unavailable.

For local development, keep secrets in environment variables or a local `.env` file that is not committed.

## Fly.io Deployment

Use Fly secrets for sensitive values instead of storing them in the repo or in SQLite:

```sh
fly secrets set \
  BGG_TOKEN=your_bgg_token \
  ADMIN_USERNAME=admin \
  ADMIN_PASSWORD='choose-a-long-random-password'
```

Recommended:

- Keep `BGG_TOKEN` server-side only. Never expose it in HTML, JavaScript, or browser storage.
- Use a long random `ADMIN_PASSWORD`.
- Mount a persistent Fly volume for the SQLite database and uploads directory.
- Set `DB_PATH` to your mounted volume, for example `/data/games.db`.
- Run the app behind Fly's HTTPS endpoint; this app sends HSTS when served over HTTPS.

## Project Structure

```text
.
├── main.go
├── handlers.go
├── render.go
├── db.go
├── bgg.go
├── models.go
├── templates/
│   ├── layout.html
│   ├── home.html
│   ├── games.html
│   ├── games_result.html
│   ├── game_detail.html
│   ├── rules.html
│   ├── rules_content.html
│   ├── player_aids_list.html
│   ├── import.html
│   └── import_result.html
├── static/
│   └── style.css
├── data/
│   └── uploads/
└── Makefile
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page |
| GET | `/games` | Collection list with filters |
| GET | `/games/{id}` | Game detail page |
| POST | `/games/{id}/delete` | Remove a game from the collection |
| GET | `/games/{id}/rules` | Rules and player-aids page |
| POST | `/games/{id}/rules/url` | Save or update the Google Drive rulebook link |
| POST | `/games/{id}/rules/upload` | Upload a player-aid image |
| POST | `/games/{id}/rules/aids/{aid_id}/delete` | Delete a player-aid image |
| GET | `/import` | BGG import page |
| POST | `/import` | Sync owned games from BGG |

## Storage

- Templates and static assets are embedded in the Go binary with `go:embed`.
- SQLite data lives in `games.db` by default.
- Uploaded player-aid files are stored in `data/uploads/`.
- Uploaded files are validated as images before being saved.

## Testing

Run the test suite via Make:

```sh
make test           # Run all tests
make test-v         # Run with verbose output
make cover          # Run with coverage report
make cover-html     # Generate HTML coverage report (saved to /tmp/coverage.html)
```

Or run tests directly:

```sh
go test ./...                         # All tests
go test ./... -v                      # Verbose output
go test ./internal/store/ -cover      # Store layer with coverage
go test ./internal/httpx/ -cover      # Middleware with coverage
```

**Coverage:** Phase 1 (security foundations) complete with 57 tests covering password hashing, session management, JWT tokens, CSRF protection, and rate limiting.

## Maintenance

Reset the local database:

```sh
rm -f games.db games.db-shm games.db-wal
```

Remove uploaded files:

```sh
rm -rf data/uploads
```

Update dependencies:

```sh
go get -u ./...
go mod tidy
```

Build for deployment:

```sh
make build
PORT=8080 DB_PATH=/var/data/games.db ./boardgames
```
