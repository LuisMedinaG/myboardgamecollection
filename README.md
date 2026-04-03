# My Board Game Collection

A minimal web app to organise your board game collection. Browse, search, add, edit, and delete games — with quick-reference rules for each.

**Stack:** Go · Templ · SQLite · HTMX

## Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- No C compiler needed (uses pure-Go SQLite)

> The Makefile resolves the `templ` CLI via `$(go env GOPATH)/bin/templ` automatically, so it works even if `GOPATH/bin` is not in your `PATH`.

## Quick Start

```sh
# One-time setup: install templ CLI, generate code, fetch dependencies
make setup

# Run the server
make run
```

Open [http://localhost:8080](http://localhost:8080).

The database (`games.db`) is created automatically on first run with 3 sample games.

## Commands

| Command        | Description                                          |
|----------------|------------------------------------------------------|
| `make setup`   | Install templ CLI, generate code, resolve Go modules |
| `make build`   | Generate templ + compile binary (`./boardgames`)     |
| `make run`     | Build and run the server                             |
| `make dev`     | Generate and run with `go run` (for development)     |
| `make clean`   | Remove binary, database, and generated `_templ.go`   |

## Configuration

Environment variables:

| Variable  | Default    | Description          |
|-----------|------------|----------------------|
| `PORT`    | `8080`     | HTTP server port     |
| `DB_PATH` | `games.db` | SQLite database path |

Example:

```sh
PORT=3000 DB_PATH=./data/collection.db make run
```

## Project Structure

```
.
├── main.go            # Entry point, routing, embedded static files
├── handlers.go        # HTTP handlers for all routes
├── db.go              # SQLite initialisation, CRUD, filtering, seed data
├── models.go          # Game type and page data structs
├── layout.templ       # Base HTML layout (head, body, HTMX)
├── home.templ         # Home page
├── games.templ        # Game list with filters
├── game_detail.templ  # Single game view
├── game_form.templ    # Add / edit form
├── static/
│   └── style.css      # All styles
├── Makefile           # Build commands
├── go.mod             # Go module definition
└── README.md
```

## Routes

| Method | Path                 | Description        |
|--------|----------------------|--------------------|
| GET    | `/`                  | Home page          |
| GET    | `/games`             | List all games     |
| GET    | `/games/new`         | Add game form      |
| POST   | `/games`             | Create a game      |
| GET    | `/games/{id}`        | Game detail        |
| GET    | `/games/{id}/edit`   | Edit game form     |
| POST   | `/games/{id}`        | Update a game      |
| POST   | `/games/{id}/delete` | Delete a game      |

## How It Works

- **Templ** compiles `.templ` files into type-safe Go code (`*_templ.go`). These generated files are gitignored — run `make generate` after cloning.
- **SQLite** stores everything in a single `games.db` file. WAL mode is enabled for concurrent reads.
- **HTMX** powers the game list filters (genre, player count, duration) without full page reloads.
- Static assets are embedded into the binary via `go:embed`, so the compiled `boardgames` binary is fully self-contained.

## Maintenance

### Reset the database

```sh
rm games.db && make run
```

The app re-seeds 3 sample games on startup when the database is empty.

### Back up the database

```sh
cp games.db games-backup-$(date +%Y%m%d).db
```

### Update dependencies

```sh
go get -u ./...
go mod tidy
```

### Update templ

```sh
go install github.com/a-h/templ/cmd/templ@latest
make generate
```

> After cloning, run `make generate` before `make build` — the `*_templ.go` files are gitignored.

### Deploy

Build a self-contained binary:

```sh
make build
```

Copy `./boardgames` to your server and run it. The binary includes all static assets. Only the `games.db` file is external.

```sh
PORT=8080 DB_PATH=/var/data/games.db ./boardgames
```

## License

MIT
