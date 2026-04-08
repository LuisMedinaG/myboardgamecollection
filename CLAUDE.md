# My Board Game Collection — CLAUDE.md

## Stack

- **Language:** Go 1.25 (standard library HTTP server)
- **Frontend:** Go HTML templates (`html/template`) + HTMX — no JS framework
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no C dependency)
- **Auth:** Session-based (username/password, bcrypt hashed)
- **Deployment:** Docker + Fly.io (persistent volume at `/data`)
- **External API:** BoardGameGeek (BGG) for importing game collections

## Commands

```sh
make dev        # go run . (hot-ish reload)
make run        # build + run
make build      # outputs ./boardgames binary
make clean      # remove binary + local DB
make bgg-login  # utility to grab BGG auth headers
```

No test suite — verification is manual.

## Environment Variables

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `games.db` | SQLite file path |
| `DATA_DIR` | `data` | Base dir for uploads + image cache |
| `SESSION_SECRET` | — | 32+ byte hex; required in prod (`openssl rand -hex 32`) |
| `BGG_TOKEN` | — | BGG API token; takes priority over cookie |
| `BGG_COOKIE` | — | BGG cookie fallback when no token set |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | — | Initial admin credentials |

Use `.env` locally, `fly secrets set` in production.

## Project Structure

```
main.go              # Server setup, routes, middleware wiring
internal/
  handler/           # HTTP handlers (auth, games, import, vibes, rules, discover)
  store/             # SQLite data access layer (CRUD for all entities)
  httpx/             # Middleware (auth, CSRF, rate-limit, security headers)
  bgg/               # BGG API client wrapper
  render/            # Template renderer (pre-parses all templates on startup)
  model/             # Domain structs (Game, PlayerAid, Vibe, CollectionEntry)
  viewmodel/         # View-layer data passed to templates
  filter/            # Game filtering logic (category, players, playtime)
templates/           # Embedded HTML templates
static/              # Embedded CSS
data/                # Runtime: games.db, uploads/, images/ (image cache)
cmd/bgg-login/       # CLI utility for BGG auth
```

## Key Patterns

**Middleware:** Composable via `httpx.Chain()` — `MethodGuard`, `RequireAuth`, `SameOrigin`, `VerifyCSRF`, `RateLimit`, `SecurityHeaders`.

**Templates:** All embedded in binary via `go:embed`. Renderer supports full-page (with layout) and partial (HTMX swap) renders. Custom funcs: `split`, `add`, `playerAidsData`.

**BGG Auth:** Token is primary; cookies attach only as fallback when no token is set.

**SQLite:** WAL mode + foreign keys enabled. Schema migrations run automatically on startup.

**CSRF:** Tokens derived from `SESSION_SECRET`, validated on all mutating POST routes.

## Routes (summary)

| Method | Path | Feature |
|--------|------|---------|
| GET | `/` | Home/dashboard |
| GET/POST | `/login`, `/signup`, `/logout` | Auth |
| GET | `/profile/change-password` | Password change |
| GET | `/games` | Collection list (filterable) |
| GET | `/games/{id}` | Game detail |
| GET/POST | `/games/{id}/edit` | Edit game metadata |
| GET/POST | `/games/{id}/rules` | Rulebook link + player aids |
| GET/POST | `/vibes` | Vibe tag management |
| GET/POST | `/import` | BGG collection sync (delta only) |
| GET | `/discover` | Recommendations by vibe |
| GET | `/images/{bgg_id}` | BGG image proxy/cache |
