# My Board Game Collection

Personal app — track board games, store rulebook links, upload player aids, import from BoardGameGeek (BGG).

## Rules

- **Never `git push`** without the user explicitly saying "push" or "make a PR".
- **Commit workflow** — always ask before committing so the user can review the diff first.
- **React CSS workflow** — edit `react-app/src/index.css` (Tailwind v4 source). The `@tailwindcss/vite` plugin handles compilation automatically during `bun dev`/`bun build`.
- **Tailwind-first UI** — use Tailwind utility classes. No new CSS files or custom classes unless unavoidable. Shared CSS utility classes in `index.css` @layer components: `.section-label` (uppercase section header), `.field-label` (form field label), `.form-input` (standard full-width input), `.alert-error` (inline error message). Use these before reaching for inline styles.
- **React package manager** — use `bun` (not `npm`) inside `react-app/`. Run `bun dev`, `bun build`, `bun install`.
- **React API calls** — all data fetching goes through `react-app/src/lib/api.ts`. Never call `fetch()` directly in components.
- **DB migrations** — new column: append to `addCols` in `shared/db/db.go`. New table: append to `createTables()` stmts. Both are idempotent (run on every startup). Use `/add-migration` skill.
- **Multi-tenancy** — every SQL query must pass `user_id`. No exceptions.
- **Error handling** — use `apierr.ErrDuplicate`, `apierr.ErrWrongPassword`, `apierr.ErrForeignOwnership`. Never expose raw DB errors.
- **BGG client** — `bgg.New(token)` for token auth, `bgg.NewWithCookies(cookie)` for cookie fallback. Token takes priority.

## Stack

**Go backend:** Go 1.25 · stdlib HTTP · REST/JSON API · SQLite (`modernc.org/sqlite`) · Fly.io

**React frontend:** React 19 · React Router v7 · Vite · Tailwind CSS v4 · TypeScript · Bun

## Project Structure

### Go backend

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

### React frontend (`react-app/`)

```
src/
  lib/api.ts           # Centralized API client (JWT, auto-refresh)
  contexts/AuthContext.tsx
  components/          # Layout, FilterBar, GameListItem, GameCard, TagList, …
  pages/               # LoginPage, CollectionPage, GameDetailPage, VibesPage
  types/game.ts        # Game interface + filter types
  index.css            # Tailwind v4 source + theme tokens + component classes
e2e/
  smoke.spec.ts        # Playwright E2E (TEST_TOKEN env var — ephemeral JWT, no static creds)
```

## Key Patterns

**Auth** — JWT only (no session cookies). `shared/httpx.RequireJWT(secret)` middleware guards all protected routes. User ID extracted with `requireUserID(w, r)` inside each handler.

**Routes** — registered in `main.go` using `protected(method, handler)` or `pub(method, handler)` wrappers.

**Middleware** — `httpx.Chain(handler, A, B, C)` executes A → B → C → handler (reversed internally).

**Error flow** — use `apierr.IsDuplicate(err)` to detect constraint violations; use sentinel errors for all others.

## Commands

### Go backend

```sh
make dev        # go run . (recommended for development)
make build      # build binary → ./boardgames
make run        # build + run
make test       # run all tests
make test-v     # verbose tests
make cover      # per-package coverage %
make cover-html # HTML coverage report
make check      # build + test + vet
make dev-all    # Go + React concurrently (trap INT/TERM)
make bgg-login  # fetch BGG auth headers
```

### React frontend

```sh
bun dev               # Vite dev server at localhost:5173
bun build             # type-check + production build → dist/
bun run lint          # ESLint
bun run preview       # preview production build
bun install           # install dependencies
```

E2E tests (requires Go backend running):

```sh
make dev-go  # in one terminal (auto-creates test user if TEST_USER set)
TEST_TOKEN=<optional-ephemeral-jwt> bun run test:e2e  # in react-app/ (auto-logins if no token)
```

If TEST_TOKEN not provided, tests auto-login with TEST_USER/TEST_PASSWORD (defaults: testuser/testpass123).

## Branching

`feature/*` → `dev` (direct push OK) → PR → `staging` → PR → `main`

Enforced by GitHub rulesets. Never use admin bypass.

## Project Skills

Stored in `.claude/skills/` — auto-loaded by Claude Code. Use these for common workflows:

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
- **playwright** — E2E browser automation (`e2e/smoke.spec.ts`)
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
