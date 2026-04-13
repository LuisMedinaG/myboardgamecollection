# Quick Reference

## Environment Variables

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `games.db` | SQLite file path |
| `DATA_DIR` | `data` | Uploads + image cache |
| `SESSION_SECRET` | — | 32+ byte hex; required in prod |
| `BGG_TOKEN` | — | Primary BGG auth (takes priority over cookie) |
| `BGG_COOKIE` | — | Fallback when no token set |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | — | Initial admin credentials |

## Routes

| Method | Path | Feature |
|--------|------|---------|
| GET | `/` | Dashboard |
| GET/POST | `/login`, `/signup`, `/logout` | Auth |
| GET | `/profile/change-password` | Password change |
| GET | `/games` | Collection (filterable) |
| GET | `/games/{id}` | Game detail |
| GET/POST | `/games/{id}/edit` | Edit metadata |
| GET/POST | `/games/{id}/rules` | Rulebook + player aids |
| GET/POST | `/vibes` | Vibe tags |
| GET/POST | `/import` | BGG sync |
| GET | `/discover` | Recommendations |
| GET | `/images/{bgg_id}` | BGG image proxy |

All routes above have REST API equivalents under `/api/v1/…` (JSON in/out, JWT auth).
