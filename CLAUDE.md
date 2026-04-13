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

## Game Model — Key Fields

| Field | Type | DB column | Source |
|---|---|---|---|
| `Weight` | `float64` | `weight` | BGG `averageweight` stat |
| `Rating` | `float64` | `rating` | BGG `average` rating stat |
| `LanguageDependence` | `int` | `language_dependence` | BGG `language_dependence` poll — winning level (0=unknown, 1–5) |
| `RecommendedPlayers` | `string` | `recommended_players` | BGG `suggested_numplayers` poll — comma-separated counts where Best+Rec > Not Rec (e.g. `"2,3,4"`) |

### BGG Import

`internal/bgg/bgg.go` uses a **custom XML fetch** (`fetchThingsParsed`) instead of gobgg's `GetThings`. This is required because gobgg's `ThingResult` doesn't expose raw poll data. Both use the same authenticated, throttled `http.Client` so rate limiting is shared. gobgg is still used for `GetCollection`.

- **Full Refresh required** to backfill `weight`, `rating`, `language_dependence`, `recommended_players` on existing games. Normal sync only fetches newly added games.
- Full Refresh is admin-only — UI checkbox on the Import page, or `POST /api/v1/import` with `{"full_refresh": true}`.

### Filters

All filters are wired through `internal/filter/filter.go`, `store.FilterGames`, `store.FilterGamesByVibe`, and all HTMX + REST API handlers.

| URL param | Filter function | Values |
|---|---|---|
| `players` | `PlayerCondition` | `1`, `2`, `2only`, `3`, `4`, `5plus` |
| `playtime` | `PlaytimeCondition` | `short`, `medium`, `long` |
| `weight` | `WeightCondition` | `light`, `medium`, `heavy` |
| `rating` | `RatingCondition` | `good` (≥6), `great` (≥7), `excellent` (≥8) |
| `lang` | `LanguageCondition` | `free` (level 1), `low` (2), `moderate` (3), `high` (≥4) |
| `rec_players` | `RecommendedPlayersCondition` | `1`–`5` — sentinel-comma LIKE match |

`RecommendedPlayersCondition` embeds validated digit-only values directly in SQL (same pattern as other filter conditions that embed literals). Input validation rejects anything non-numeric.

### DB Migration Pattern

New columns are added via `ALTER TABLE games ADD COLUMN … DEFAULT …` in `store.createTables()`. These are idempotent (SQLite silently ignores `ADD COLUMN` if it already exists). The legacy table-recreation function `migrateGamesTableForPerUserUniqueness` must also be updated whenever a new column is added (both the `CREATE TABLE games_new` DDL and the `INSERT INTO … SELECT` list).

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
static/              # Embedded CSS + JS
  style.css          # Entry point — @imports all modules (see below)
  styles/            # CSS modules (native nesting, one concern per file)
    variables.css    # :root tokens — palette, radii, shadows, profile vars
    reset.css        # box-sizing reset, html/body base
    layout.css       # top nav, burger, profile dropdown, .main-content, breakpoints
    login.css        # login page
    forms.css        # form inputs, filter controls, toggle switch
    buttons.css      # .btn variants, .badge, .page-btn, .view-btn
    utilities.css    # pagination, spinner, modal, bulk-bar, quickref, empty states
    tags.css         # .tag, .pill-btn, .vibe-pill, vibe-color-* palette
    game-list.css    # list rows, grid-sm/md/lg (nested), multi-select, view toolbar
    game-detail.css  # game profile page — hero, stats panel, sections, lang card, aids
    import.css       # import flow, CSV preview, BGG username panel
    rules.css        # rules page, player aids grid, lightbox
    vibes.css        # vibe grid, discover filters, action cards, vibe management
```

## CSS Architecture

`static/style.css` is a thin `@import` barrel — it only lists the module files in cascade order and contains no rules of its own.

**Module conventions:**
- Each file owns one feature area. Do not mix concerns across files.
- Use **native CSS nesting** (`&`) for pseudo-classes, modifier classes, and tightly-coupled descendants. No preprocessor.
- New CSS variables go in `styles/variables.css` under the appropriate group comment.
- All colour literals used more than once must be a variable.

**Production build** — the Go server serves each `@import` as a separate HTTP request (fine for development). To bundle for production:
```sh
cat static/styles/variables.css \
    static/styles/reset.css \
    static/styles/layout.css \
    static/styles/login.css \
    static/styles/forms.css \
    static/styles/buttons.css \
    static/styles/utilities.css \
    static/styles/tags.css \
    static/styles/game-list.css \
    static/styles/game-detail.css \
    static/styles/import.css \
    static/styles/rules.css \
    static/styles/vibes.css \
    > static/style.bundle.css
```
Then swap the `<link>` href in `templates/layout.html` to `style.bundle.css`.

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
