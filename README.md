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
| `make clean` | Remove the binary and local database |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `games.db` | SQLite database path |
| `BGG_TOKEN` | unset | Enables BoardGameGeek import |

Example:

```sh
PORT=3000 DB_PATH=./data/collection.db make run
```

If `BGG_TOKEN` is not set, the import page stays visible but the sync form is disabled.

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
