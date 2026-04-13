# My Board Game Collection

Personal app — track board games, store rulebook links, upload player aids, import from BoardGameGeek (BGG).

## Rules

- **Never `git push`** without the user explicitly saying "push" or "make a PR".
- **Never read `pico.min.css`** — use `pico-reference.html` (project root) for all Pico patterns.
- **Commit workflow** — always ask before committing so the user can review the diff first.
- **Pico CSS is classless** — use semantic HTML (`<article>`, `<dialog>`, `<nav>`, `<hgroup>`, etc.). Don't invent classes for things Pico handles. Read `pico-reference.html` before any UI work.
- **CSS overrides** go in the matching module under `static/styles/`. New tokens go in `variables.css`.
- **DB migrations** — `ALTER TABLE … ADD COLUMN … DEFAULT …` in `store.createTables()`. Idempotent. Also update `migrateGamesTableForPerUserUniqueness` DDL + SELECT list.
- **Multi-tenancy** — every SQL query must include `AND user_id = ?`. Bulk ops use the `ownedIDs()` pattern.
- **Error handling** — use sentinel errors (`store.ErrDuplicate`, `store.ErrWrongPassword`). Never expose raw DB errors.
- **BGG client** — `fetchThingsParsed` (custom XML) for game details, gobgg for `GetCollection`. Don't switch these.

## Stack

Go 1.25 · stdlib HTTP server · HTMX (no JS framework) · Pico CSS · SQLite (`modernc.org/sqlite`) · Docker + Fly.io

## Commands

```sh
make dev          # go run .
make run          # build + run
make build        # outputs ./boardgames binary
make test         # run all tests
make test-v       # verbose tests
make cover        # coverage report
make cover-html   # HTML coverage report
make bgg-login    # grab BGG auth headers
```

## Project Structure

```
main.go                # Server setup, routes, middleware
internal/
  handler/             # HTTP handlers (HTMX: game.go, vibe.go, … | API: api_games.go, …)
  store/               # SQLite DAL — all queries, migrations, FTS5
  httpx/               # Middleware (auth, CSRF, rate-limit, security headers, JWT)
  bgg/                 # BGG API client (auth + throttle transports)
  render/              # Template renderer (embedded, layout cloning, partials)
  model/               # Domain structs
  viewmodel/           # Template data structs
  filter/              # Game filtering (players, playtime, weight, rating, language, rec_players)
templates/             # Embedded HTML templates
static/
  style.css            # @import barrel for all CSS modules
  styles/              # pico.min.css, variables.css, layout.css, components.css, …
```

## Key Patterns

**Dual interface** — every feature has an HTMX handler (returns HTML/partials) and a REST API handler (`/api/v1/…`, returns JSON). They share the same Store calls.

**HTMX detection** — `HX-Request: true` header → return partial; otherwise → full page with layout.

**Auth** — two systems: session cookies (HTMX frontend, 30-day, DB-backed) and JWT (REST API, 15-min access + 30-day refresh). `kind` column in sessions table keeps them isolated.

**CSRF** — stateless HMAC of session token. Never stored. Forms use `_csrf` hidden field; HTMX sends `X-CSRF-Token`.

**Middleware** — `httpx.Chain(handler, A, B, C)` executes A → B → C → handler (reversed internally).

**Templates** — embedded via `go:embed`, layout cloned per page, buffered rendering (no partial output on error). Partials registered both standalone and inside full pages for HTMX.

## Branching

`feature/*` → `dev` (direct push OK) → PR → `staging` → PR → `main`

Enforced by GitHub rulesets. Never use admin bypass for normal flow.

## Deep Reference

`agent_docs/ARCHITECTURE-GUIDE.md` — macro architecture, request pipeline, design decisions.
`agent_docs/ARCHITECTURE-REF.md` — env vars, route table.
