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

A test suite is planned. Verification is currently manual (`make dev` + curl).

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

## Branching Strategy

```
main       <- production, stable
staging    <- pre-production / QA sign-off gate
dev        <- integration branch — all feature branches target this
```

**Promotion flow:** `feature/*` → PR → `dev` → PR → `staging` → PR → `main`

### Branch protection

| Branch | Direct push | Force push | Deletion | PR required | Source enforced |
|--------|------------|------------|----------|-------------|-----------------|
| `dev` | allowed | blocked | blocked | no | — |
| `staging` | blocked | blocked | blocked | yes (0 approvals) | must be `dev` |
| `main` | blocked | blocked | blocked | yes (0 approvals) | must be `staging` |

- Enforced by two GitHub rulesets ("Protect dev", "Protect main and staging")
- Source branch restriction enforced by `.github/workflows/enforce-merge-direction.yml` — PRs to `staging`/`main` fail if source is wrong
- Admin bypass is **always on** — you are never locked out for emergency fixes
- Never push directly to `main` or `staging` (admin bypass exists for emergencies only)

## JWT REST API

A parallel JSON REST API lives under `/api/v1/` alongside the existing HTMX app.
The HTMX frontend and all existing routes remain untouched.

- **Auth:** `github.com/golang-jwt/jwt/v5` — access tokens (15 min JWT) + refresh tokens (30 day, stored in sessions table)
- **Middleware:** `RequireJWT(secret)` in `internal/httpx/` — reads `Authorization: Bearer`, returns 401 JSON on failure
- **Handlers:** All API handlers live in `internal/handler/api_*.go`
- **Helpers:** `api_helpers.go` — `requireAPIUserID`, `requireAPIID`, `writeAPIJSON`, model→snake_case converters
- **Responses:** `{ "data": ... }` for success, `{ "error": "..." }` for failures; paginated lists include `total`, `page`, `per_page` at the top level

### Completed phases
1. ✅ JWT foundation — `POST /api/v1/auth/login|refresh|logout`
2. ✅ Core data — games, vibes, import, profile (16 endpoints)
3. ✅ Rules, player aids, discovery (4 endpoints)

### Next
4. Test suite

## More Detail

See `agent_docs/` for routes, env vars, and key patterns.

Never run `git push` without the user explicitly asking to push.

**Why:** User was surprised when a push happened as part of a "commit" request — they only wanted a commit, not a push.

**How to apply:** After committing, stop and tell the user the commit is ready. Only push if they separately say "push" or "make a PR" (which requires a push). When in doubt, ask.
