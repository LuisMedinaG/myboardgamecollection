# My Board Game Collection — CLAUDE.md

Personal app to manage a board game collection — track owned games, store rulebook links,
upload player aids, and import games from BoardGameGeek (BGG).

## Stack

- **Language:** Go 1.25 (standard library HTTP server)
- **Frontend:** Go HTML templates + HTMX — no JS framework
- **CSS:** Pico CSS (classless framework) + custom overrides
- **Database:** SQLite (`modernc.org/sqlite`, pure Go)
- **Auth:** Session-based (username/password, SHA-256 + salt)
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
static/
  style.css          # @import barrel — loads all CSS modules in order
  styles/
    pico.min.css     # Pico CSS framework — DO NOT read this file
    variables.css    # :root tokens — palette, radii, shadows
    layout.css       # top nav, burger, profile dropdown, breakpoints
    components.css   # shared components (cards, badges, etc.)
    login.css        # login / signup pages
    forms.css        # form inputs, filter controls
    game-list.css    # list rows, grid views, multi-select, view toolbar
    game-detail.css  # game profile — hero, stats panel, sections
    import.css       # import flow, CSV preview, BGG username panel
    rules.css        # rules page, player aids grid, lightbox
    vibes.css        # vibe grid, discover filters, action cards
```

## CSS — Pico CSS

We use **Pico CSS** (classless framework). Pico auto-styles semantic HTML elements — avoid
inventing custom classes for basic layout/typography.

**`pico-reference.html`** (project root) — canonical HTML patterns for all Pico components.
**Read this file before building or modifying any UI component.**

Rules:
- **Never read `pico.min.css`** — minified, too many tokens, no structural value.
- Rely on semantic HTML: `<main>`, `<article>`, `<nav>`, `<dialog>`, `<details>`, etc.
- Use `class="container"` / `class="container-fluid"` for page wrappers.
- Use `class="grid"` on a parent for equal-width responsive columns.
- Use `<article>` for cards (with optional `<header>`/`<footer>`).
- Buttons: default = primary; variants via `class="secondary|contrast"` and/or `class="outline"`.
- Use `aria-busy="true"` for loading states on any element.
- Custom overrides go in the appropriate module file under `static/styles/`.
- New CSS variables go in `variables.css` under the relevant group comment.

## Game Model — Key Fields

| Field | Type | DB column | Source |
|---|---|---|---|
| `Weight` | `float64` | `weight` | BGG `averageweight` stat |
| `Rating` | `float64` | `rating` | BGG `average` rating stat |
| `LanguageDependence` | `int` | `language_dependence` | BGG `language_dependence` poll — winning level (0=unknown, 1–5) |
| `RecommendedPlayers` | `string` | `recommended_players` | BGG `suggested_numplayers` poll — comma-separated counts where Best+Rec > Not Rec (e.g. `"2,3,4"`) |

### BGG Import

`internal/bgg/bgg.go` uses a **custom XML fetch** (`fetchThingsParsed`) instead of gobgg's
`GetThings` — required because gobgg's `ThingResult` doesn't expose raw poll data. Both use
the same authenticated, throttled `http.Client`. gobgg is still used for `GetCollection`.

- **Full Refresh** required to backfill `weight`, `rating`, `language_dependence`,
  `recommended_players` on existing games. Normal sync only fetches newly added games.
- Full Refresh is admin-only — UI checkbox on Import page, or `POST /api/v1/import`
  with `{"full_refresh": true}`.

### Filters

All filters flow through `internal/filter/filter.go`, `store.FilterGames`,
`store.FilterGamesByVibe`, and all HTMX + REST API handlers.

| URL param | Filter function | Values |
|---|---|---|
| `players` | `PlayerCondition` | `1`, `2`, `2only`, `3`, `4`, `5plus` |
| `playtime` | `PlaytimeCondition` | `short`, `medium`, `long` |
| `weight` | `WeightCondition` | `light`, `medium`, `heavy` |
| `rating` | `RatingCondition` | `good` (≥6), `great` (≥7), `excellent` (≥8) |
| `lang` | `LanguageCondition` | `free` (level 1), `low` (2), `moderate` (3), `high` (≥4) |
| `rec_players` | `RecommendedPlayersCondition` | `1`–`5` — sentinel-comma LIKE match |

`RecommendedPlayersCondition` embeds validated digit-only values directly in SQL. Input
validation rejects anything non-numeric.

### DB Migration Pattern

New columns: `ALTER TABLE games ADD COLUMN … DEFAULT …` in `store.createTables()`.
Idempotent — SQLite silently ignores `ADD COLUMN` if it already exists.
Also update `migrateGamesTableForPerUserUniqueness`: both the `CREATE TABLE games_new` DDL
and the `INSERT INTO … SELECT` list.

## Test Suite

**88 tests** across `internal/store` and `internal/httpx`.

| Phase | Status | Scope |
|---|---|---|
| Phase 1 | ✅ done | Password hashing, sessions, JWT, CSRF, rate limiting, multi-user ownership |
| Phase 2 | 🔄 in progress | HTTP handlers (httptest + Playwright e2e) |
| Phase 3 | ⏳ | Store layer (CRUD, filtering, taxonomy) |
| Phase 4 | ⏳ | External integrations (BGG, file uploads) |

## JWT REST API

Parallel JSON REST API under `/api/v1/` alongside the HTMX app.

- **Auth:** `github.com/golang-jwt/jwt/v5` — access tokens (15 min) + refresh tokens (30 day, stored in sessions table)
- **Middleware:** `RequireJWT(secret)` in `internal/httpx/` — reads `Authorization: Bearer`, returns 401 JSON on failure
- **Handlers:** `internal/handler/api_*.go` — auth, games, vibes, import, profile, rules, discovery
- **Helpers:** `api_helpers.go` — `requireAPIUserID`, `requireAPIID`, `writeAPIJSON`, model→snake_case converters
- **Responses:** `{ "data": ... }` for success, `{ "error": "..." }` for failures; paginated lists include `total`, `page`, `per_page`
- **Errors:** Sentinel errors (`store.ErrDuplicate`, `store.ErrWrongPassword`) — never expose raw DB errors

See `agent_docs/` for full route list, env vars, and key patterns.

## Branching Strategy

```
main       ← production, stable
staging    ← pre-production / QA gate
dev        ← integration branch — all feature branches target this
```

**Promotion flow:** `feature/*` → PR → `dev` → PR → `staging` → PR → `main`

| Branch | Direct push | PR required | Source enforced |
|--------|------------|-------------|-----------------|
| `dev` | allowed | no | — |
| `staging` | blocked | yes | must be `dev` |
| `main` | blocked | yes | must be `staging` |

Enforced by GitHub rulesets + `.github/workflows/enforce-merge-direction.yml`.
Admin bypass is always on for emergencies — never use it for normal flow.

## Agent Rules

- **Never `git push`** without the user explicitly asking to push. Commit and stop —
  report the commit is ready. Only push when the user separately says "push" or "make a PR".
- **Never read `pico.min.css`** — use `pico-reference.html` instead.
