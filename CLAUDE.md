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
# Development
make dev        # go run .
make run        # build + run
make build      # outputs ./boardgames binary
make clean      # remove binary and database

# Testing
make test       # run all tests
make test-v     # run tests with verbose output
make cover      # run tests with coverage report
make cover-html # generate HTML coverage report

# Utilities
make bgg-login  # grab BGG auth headers
```

## Test Suite

**66 tests** across `internal/store` and `internal/httpx`.

### Coverage by Phase
1. ✅ **Phase 1:** Password hashing, sessions, JWT, CSRF, rate limiting, multi-user ownership (100% critical functions)
2. 🔄 **Phase 2:** HTTP handlers (httptest + Playwright e2e)
3. ⏳ **Phase 3:** Store layer (CRUD, filtering, taxonomy)
4. ⏳ **Phase 4:** External integrations (BGG, file uploads)

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

- **Auth:** `github.com/golang-jwt/jwt/v5` — access tokens (15 min) + refresh tokens (30 day, stored in sessions table)
- **Middleware:** `RequireJWT(secret)` in `internal/httpx/` — reads `Authorization: Bearer`, returns 401 JSON on failure
- **Handlers:** `internal/handler/api_*.go` — auth, games, vibes, import, profile, rules, discovery
- **Helpers:** `api_helpers.go` — `requireAPIUserID`, `requireAPIID`, `writeAPIJSON`, model→snake_case converters
- **Responses:** `{ "data": ... }` for success, `{ "error": "..." }` for failures; paginated lists include `total`, `page`, `per_page` at top level
- **Error handling:** Sentinel errors (`store.ErrDuplicate`, `store.ErrWrongPassword` in `store/errors.go`) — never expose raw DB errors to clients

See `agent_docs/` for routes, env vars, and key patterns.

Never run `git push` without the user explicitly asking to push.

**Why:** User was surprised when a push happened as part of a "commit" request — they only wanted a commit, not a push.

**How to apply:** After committing, stop and tell the user the commit is ready. Only push if they separately say "push" or "make a PR" (which requires a push). When in doubt, ask.
