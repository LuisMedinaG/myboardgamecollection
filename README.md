# My Board Game Collection

A small web app to organize your board games, rulebook links, and player-aid images.

**Stack:** Go · `html/template` · SQLite · HTMX

## Prerequisites

- Go 1.23+
- No C compiler needed

## Quick Start

```sh
make run
```

Open `http://localhost:8080`.

On first run, the app creates `games.db` automatically and seeds a few sample games.

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
└── Makefile
```

## Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page |
| GET | `/games` | Game list with filters |
| GET | `/games/{id}` | Game detail |
| POST | `/games/{id}/delete` | Remove a game |
| GET | `/games/{id}/rules` | Rulebook and player aids |
| POST | `/games/{id}/rules/url` | Save or update Google Drive rulebook link |
| POST | `/games/{id}/rules/upload` | Upload a player-aid image |
| POST | `/games/{id}/rules/aids/{aid_id}/delete` | Delete a player-aid image |
| GET | `/import` | BGG import page |
| POST | `/import` | Sync owned games from BGG |

## Notes

- Templates are embedded into the Go binary with `go:embed`.
- Static assets are embedded too; uploaded player-aid images are stored on disk in `data/uploads`.
- BGG import is optional. If `BGG_TOKEN` is not set, the import UI stays visible but disabled with an explanatory message.
- HTMX is used only for partial updates on filters, import results, and rules/player-aid sections.

## Maintenance

Reset the database:

```sh
rm -f games.db
```

Back up the database:

```sh
cp games.db games-backup-$(date +%Y%m%d).db
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
