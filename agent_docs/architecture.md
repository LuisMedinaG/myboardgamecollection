# Architecture & Reference

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

## Key Patterns

**Middleware:** Composable via `httpx.Chain()` — `MethodGuard`, `RequireAuth`, `SameOrigin`, `VerifyCSRF`, `RateLimit`, `SecurityHeaders`.

**Templates:** All embedded in binary via `go:embed`. Renderer supports full-page (with layout) and partial (HTMX swap) renders. Custom funcs: `split`, `add`, `playerAidsData`.

**BGG Auth:** Token is primary; cookies attach only as fallback when no token is set.

**SQLite:** WAL mode + foreign keys enabled. Schema migrations run automatically on startup.

**CSRF:** Tokens derived from `SESSION_SECRET`, validated on all mutating POST routes.

## Routes

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
