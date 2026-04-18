# My Board Game Collection — Monolith (Go backend only)

Personal app — track board games, store rulebook links, upload player aids, import from BoardGameGeek (BGG).

> **Migration in progress.** Frontend lives in `mbgc-web` (deployed at `lumedina.dev`).
> This repo is the Go monolith backend — being incrementally decomposed into microservices.
> See GitHub issues #113–#120 for the migration roadmap.

## Rules

- **Never `git push`** without the user explicitly saying "push" or "make a PR".
- **Commit workflow** — always ask before committing so the user can review the diff first.
- **DB migrations** — new column: append to `addCols` in `shared/db/db.go`. New table: append to `createTables()` stmts. Both are idempotent (run on every startup). Use `/add-migration` skill.
- **Multi-tenancy** — every SQL query must pass `user_id`. No exceptions.
- **Error handling** — use `apierr.ErrDuplicate`, `apierr.ErrWrongPassword`, `apierr.ErrForeignOwnership`. Never expose raw DB errors.
- **BGG client** — `bgg.New(token)` for token auth, `bgg.NewWithCookies(cookie)` for cookie fallback. Token takes priority.
- **CORS** — `REACT_ORIGIN` env var (comma-separated). Prod value: `https://lumedina.dev`.

## Stack

**Go backend:** Go 1.25 · stdlib HTTP · REST/JSON API · SQLite (`modernc.org/sqlite`) · Fly.io (`myboardgamecollection`)

**Frontend:** React 19 · Cloudflare Pages · repo: `LuisMedinaG/mbgc-web`

## Project Structure

```
main.go              # Server setup, routes, middleware wiring
services/
  auth/              # Login, logout, JWT refresh, sessions
  games/             # Game CRUD, filtering, player aids
  collections/       # Collections (vibes), discovery
  files/             # Player aid uploads, rules URL
  importer/          # BGG sync, CSV import
  profile/           # User profile, BGG username, password change
shared/
  db/db.go           # Schema + all migrations (createTables, addCols, FTS5)
  httpx/httpx.go     # Middleware: RequireJWT, MethodGuard, Chain, CORS, SecurityHeaders
  apierr/errors.go   # Sentinel errors + IsDuplicate() helper
internal/
  bgg/               # BGG API client (token + cookie auth transports)
  model/             # Domain structs shared across services
```

Each `services/<domain>/` has `handler.go` (HTTP) + `store.go` (SQL). Routes are registered in `main.go`.

## Key Patterns

**Auth** — JWT only (no session cookies). `shared/httpx.RequireJWT(secret)` middleware guards all protected routes. User ID extracted with `requireUserID(w, r)` inside each handler.

**Routes** — registered in `main.go` using `protected(method, handler)` or `pub(method, handler)` wrappers.

**Middleware** — `httpx.Chain(handler, A, B, C)` executes A → B → C → handler (reversed internally).

**Error flow** — use `apierr.IsDuplicate(err)` to detect constraint violations; use sentinel errors for all others.

## Commands

```sh
make dev        # go run . (recommended for development)
make build      # build binary → ./boardgames
make run        # build + run
make test       # run all tests
make test-v     # verbose tests
make cover      # per-package coverage %
make cover-html # HTML coverage report
make check      # build + test + vet
make bgg-login  # fetch BGG auth headers
```

## Branching

`feature/*` → `dev` (direct push OK) → PR → `staging` → PR → `main`

Enforced by GitHub rulesets. Never use admin bypass.

## Project Skills

| Skill | Trigger |
|-------|---------|
| `/add-feature` | Adding a new API endpoint to a service |
| `/add-migration` | Adding a new DB table or column |
| `/ship` | Full test → commit → push → PR workflow |
| `/run-tests` | Running Go unit tests and/or E2E tests |

## Plugins & Skills

### Installed Plugins

- **gopls-lsp** — Go LSP (code nav, hover, definitions)
- **claude-mem** — Persistent cross-session memory
- **code-review** — PR code review
- **code-simplifier** — Refactor review for changed code
- **ralph-loop** — `/loop` recurring prompts
- **security-guidance** — Security analysis
- **commit-commands** — `/commit`, `/commit-push-pr`

### MCP Servers

- **sqlite** — Direct DB queries on `games.db`
- **github-official** — PRs, issues, Actions, code search
- **brave-search** — Web search in-context
- **fetch** — Web content to markdown
- **fly** — Fly.io deploy/monitor

### Global Skills

`/commit` · `/review-pr` · `/loop` · `/simplify` · `/update-config` · `/schedule` · `/claude-api`
`/claude-mem:make-plan` · `/claude-mem:do` · `/claude-mem:smart-explore` · `/claude-mem:knowledge-agent`

## Deep Reference

- `agent_docs/ARCHITECTURE-GUIDE.md` — design decisions, request pipeline
- `agent_docs/ARCHITECTURE-REF.md` — env vars, route table
