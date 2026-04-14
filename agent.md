---
name: Board Game Collection Agent
description: Full-stack Go/HTMX application for tracking personal board game libraries with BGG integration
---

# Agent.md — My Board Game Collection

Personal app for tracking board games, storing rulebook links, uploading player aids, and importing from BoardGameGeek (BGG).

---

**📖 Division of docs:**
- **[CLAUDE.md](./CLAUDE.md)** — Project rules, architecture overview, branching strategy (concise reference, ~100 lines)
- **This file** — Agent-specific guidance: code style, conventions, detailed examples, boundaries, execution patterns

This file is optimized for AI agent execution; CLAUDE.md is for human reference.

## Persona & Role

You are a full-stack agent for a **Go backend** + **HTMX frontend** application. You understand:
- Go 1.25 with stdlib HTTP (no frameworks)
- HTMX for dynamic interactions (no JavaScript frameworks)
- SQLite with FTS5 for full-text search
- Multi-tenant architecture (session-based auth)
- Pico CSS (semantic, classless styling)
- BGG API integration (token + cookie auth)

**Your expertise covers** backend services, API design, SQL/migrations, frontend templates, CSS styling, authentication, and testing.

## Project Knowledge

**See [CLAUDE.md](./CLAUDE.md)** for tech stack details and full directory structure.

Key execution context:
- **Backend**: Go 1.25, stdlib HTTP, SQLite with FTS5, multi-tenant (user_id filtering required)
- **Frontend**: HTMX + Pico CSS (semantic HTML only, no JS frameworks)
- **Auth**: Dual system — session cookies (HTMX) + JWT (REST API)
- **Testing**: 88 tests (Phase 1), Phase 2 in progress (target 200–240 tests)

Critical directories:
- `internal/handler/` — HTTP handlers (dual HTMX/REST interface)
- `internal/store/` — SQLite queries & migrations (idempotent, in-code)
- `internal/httpx/` — Middleware (auth, CSRF, rate limiting)
- `templates/` — Embedded HTML (layout cloning, partials)

## Commands

Run these frequently:

```sh
make dev              # go run . (hot reload via entr)
make build            # Build binary → ./boardgames
make run              # build + run binary
make test             # Run all tests
make test-v           # Verbose test output
make cover            # Coverage report (terminal)
make cover-html       # Coverage report (open in browser)
make bgg-login        # Fetch BGG OAuth headers for .env
```

**Environment Variables** (`.env` or Fly secrets):
```sh
BGG_CLIENT_ID=<oauth-client-id>
BGG_CLIENT_SECRET=<oauth-client-secret>
SESSION_SECRET=<32+ char random string>
JWT_SECRET=<32+ char random string>
PORT=8080
DATABASE_URL=file:./data.db?cache=shared&mode=rwc
LOG_LEVEL=info
```

## Code Style Go
- Follow Effective Go guidelines
- Use gofmt for formatting
- Keep functions short and focused
- Return errors, don't panic

## Conventions
- Use short variable names in small scopes
- Use descriptive names for exported identifiers
- Prefix interface names with -er when appropriate
- Use table-driven tests

## Error Handling
- Always check returned errors
- Wrap errors with context using fmt.Errorf
- Use errors.Is and errors.As for error checking
- Return errors, don't log and continue

## Concurrency
- Use channels for communication
- Use sync.WaitGroup for goroutine coordination
- Be careful with shared state
- Prefer passing data over sharing memory

## Code Standards

### Go Backend

**Handler pattern** — dual interface (HTMX + REST):
```go
// HTMX handler returns partial on HX-Request, full page otherwise
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
    gameID := r.PathValue("id")
    game, err := h.store.GetGame(r.Context(), gameID, userID(r))
    if err != nil {
        if errors.Is(err, store.ErrNotFound) {
            http.Error(w, "Not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    data := viewmodel.GameDetail{Game: game}
    if isHXRequest(r) {
        h.render.Partial(w, "game-detail-card", data)
    } else {
        h.render.Page(w, "game-detail", data)
    }
}

// REST API handler returns JSON
func (h *Handler) GetGameJSON(w http.ResponseWriter, r *http.Request) {
    gameID := r.PathValue("id")
    game, err := h.store.GetGame(r.Context(), gameID, userID(r))
    if err != nil {
        http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(game)
}
```

**Multi-tenancy** — all queries must filter by user:
```go
// ✅ Correct: includes user_id filter
err := h.store.GetGame(ctx, gameID, userID)

// ❌ Wrong: missing user_id filter
err := h.store.GetGame(ctx, gameID) // FORBIDDEN

// Bulk operations use ownedIDs() pattern
ids := h.store.OwnedGameIDs(ctx, userID, filter)
```

**Error handling** — use sentinel errors:
```go
var (
    ErrDuplicate     = errors.New("duplicate entry")
    ErrWrongPassword = errors.New("wrong password")
    ErrNotFound      = errors.New("not found")
)

// Never expose raw DB errors to client
if err != nil {
    if errors.Is(err, store.ErrDuplicate) {
        http.Error(w, "Game already in collection", http.StatusConflict)
        return
    }
    // Log, don't expose
    h.log.Error("db error", "err", err)
    http.Error(w, "Internal error", http.StatusInternalServerError)
}
```

**Middleware chain** — reversed order:
```go
// httpx.Chain(handler, A, B, C) → A → B → C → handler
h := httpx.Chain(
    h.GetGame,
    h.requireAuth,      // 1st (outer)
    h.withRateLimit,    // 2nd
    h.withCSRF,         // 3rd (inner)
)
```

### HTMX Frontend

**Semantic HTML** — no custom classes, use Pico:
```html
<!-- ✅ Correct: semantic HTML + Pico classes -->
<article>
  <h1>Game Library</h1>
  <div class="grid">
    <button class="secondary">Filter</button>
    <button class="outline">Export</button>
  </div>
</article>

<!-- ❌ Wrong: custom classes -->
<div class="game-list-wrapper">
  <div class="game-item">...</div>
</div>
```

**HTMX interactions** — use `hx-*` attributes:
```html
<!-- Load game detail on click -->
<tr hx-get="/api/games/{{ .ID }}" 
    hx-target="#detail-panel" 
    hx-swap="innerHTML">
  <td>{{ .Title }}</td>
</tr>

<!-- Form submission with CSRF -->
<form hx-post="/games" hx-target="closest article">
  <input type="hidden" name="_csrf" value="{{ .CSRF }}">
  <input type="text" name="title" required>
  <button type="submit">Add Game</button>
</form>
```

**Template layout** — partials + full pages:
```go
// render.Page() wraps partial in <html><head><body> layout
h.render.Page(w, "game-detail", data)

// render.Partial() returns just the template (for HTMX)
h.render.Partial(w, "game-card", game)

// Template registration (partial registered inside full page)
<div id="game-card">
  <h3>{{ .Title }}</h3>
  <p>Players: {{ .MinPlayers }}-{{ .MaxPlayers }}</p>
</div>
```

### CSS (Pico)

**Read pico-reference.html, never pico.min.css** — it's minified.

**Custom overrides** live in matching module under `static/styles/`:
```css
/* ✅ Correct: module-specific override */
/* static/styles/game-detail.css */
article[data-section="game"] {
  --form-element-spacing: 1.5rem;
  background: var(--muted-background);
}

/* ❌ Wrong: creating custom classes Pico already handles */
.game-card { ... }  /* Use <article> instead */
.button-primary { ... }  /* Use <button> instead */
```

**Design tokens** in `variables.css`:
```css
:root {
  --primary-color: #2e7d32;
  --spacing-unit: 1rem;
  --game-card-radius: 0.5rem;
}
```

### SQL & Migrations

**All migrations in code** — no separate SQL files:
```go
// store.go createTables()
func (s *Store) createTables(ctx context.Context) error {
    return s.execAll(ctx, []string{
        // Users table
        `CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
        // Games table (add columns, never remove)
        `CREATE TABLE IF NOT EXISTS games (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            title TEXT NOT NULL,
            bgg_id INTEGER,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id),
            UNIQUE(user_id, title)
        )`,
    })
}
```

**FTS5 for search**:
```go
// Full-text search on game titles + descriptions
`CREATE VIRTUAL TABLE games_fts USING fts5(
    title,
    description,
    content=games,
    content_rowid=id
)`
```

## Testing & CI

**Phase 1 ✅ Complete** (88 tests, ~52% coverage)
- `internal/httpx` — 44 tests, 47.4% coverage (auth, CSRF, rate limiting)
- `internal/store` — 44 tests, 56.4% coverage (DB queries, migrations)

**Phase 2 🔄 In Progress** (target 200–240 tests, ~70% coverage)
- Handler tests (auth, game CRUD, vibe CRUD)
- Filter logic + import flows
- BGG integration (VCR cassettes for mocking)
- See Issue #101 for priorities

**Run tests**:
```sh
make test           # All tests, coverage summary
make test-v         # Verbose (show each test)
make cover          # % coverage per package
make cover-html     # HTML report
```

**BGG API mocking** — use VCR cassettes (recorded responses):
```go
// testdata/cassettes/bgg-get-collection.json
// Records real API responses for replay
cassette := vcr.New("bgg-get-collection")
resp, err := cassette.GetCollection(userID)
```

## Git Workflow

Branch strategy: `feature/*` → `dev` (direct push) → `staging` (PR) → `main` (PR)

**Before committing**:
1. Run `make test` locally
2. Use `/commit` skill to stage and create commit
3. Push to `feature/*` branch
4. Open PR to `dev`

**Commit message format**:
```
type: brief description

Longer explanation if needed.

type: fix|feat|refactor|docs|chore
```

**Never force-push** to shared branches. GitHub rulesets enforce clean history.

## Boundaries

### ✅ Always Do

- **Filter by `user_id`** on every SQL query (multi-tenancy is core)
- **Ask before committing** — user reviews diff first
- **Use semantic HTML** — let Pico handle styling
- **Test new handlers** — add to Phase 2 test plan
- **Document migration changes** — update schema docs
- **Handle errors with sentinel values** — never expose raw DB errors
- **Validate at boundaries** — user input, external APIs (BGG)

### ⚠️ Ask First

- **New database tables** — impacts schema, migrations, multi-tenancy
- **Significant CSS changes** — check pico-reference.html first
- **Auth system changes** — affects both session + JWT flows
- **BGG API changes** — confirm auth strategy (token vs. cookies)
- **Large refactors** — get scope approval before starting
- **Force-pushing** — destructive, ask user explicitly

### ❌ Never Do

- **Read or modify `pico.min.css`** — it's minified; use pico-reference.html
- **Create custom CSS classes** for things Pico handles (buttons, forms, grids)
- **Query without user_id filter** — breaks multi-tenancy security
- **Push to `main` directly** — only via PR from staging
- **Expose raw database errors** to clients — use sentinel errors
- **Commit sensitive data** (API keys, session secrets, passwords)
- **Skip pre-commit hooks** — they catch real issues
- **Mock the database** in integration tests — use real SQLite
- **Create classes for things Pico handles** — semantic HTML only
- **Change BGG auth strategy** mid-flight — token is primary, cookies fallback only

## Resources

**Project Rules & Overview:**
- **[CLAUDE.md](./CLAUDE.md)** — Rules, branching, architecture overview, commands (start here for project context)

**Deep Dives:**
- `agent_docs/ARCHITECTURE-GUIDE.md` — Request pipeline, design decisions, patterns
- `agent_docs/ARCHITECTURE-REF.md` — Routes, env vars, DB schema, testing roadmap
- `pico-reference.html` — Pico CSS component patterns (always consult before CSS changes)
- `static/styles/variables.css` — Design tokens (color, spacing, typography)
- Issue #101 — Phase 2 testing roadmap and priorities

---

**Last Updated**: April 13, 2026
