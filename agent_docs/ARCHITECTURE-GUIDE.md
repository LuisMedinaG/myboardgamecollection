# Architecture Guide

A personal board game collection manager. Sync from BoardGameGeek, tag games with custom "vibes," filter by players/playtime/category, store rulebook links and player aids. Single Go binary, SQLite database, HTMX frontend + REST API.

## Macro Architecture

```
┌─────────────────────────────────────────────────────┐
│                     main.go                         │
│   Init → Routes → Middleware chains → Server start  │
└──────────────────────────┬──────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
  internal/httpx/   internal/handler/  internal/render/
  (middleware)       (HTTP handlers)   (templates)
         │                 │                 │
         │         ┌───────┴────────┐        │
         │  HTMX handlers   API handlers     │
         │  game.go         api_games.go     │
         │  vibe.go         api_vibes.go     │
         │  import.go       api_import.go    │
         │  ...             ...              │
         │                 │                 │
         └────────┬────────┘                 │
                  │                          │
          internal/store/           internal/viewmodel/
          (SQLite DAL)              (template data structs)
                  │
          internal/bgg/
          (BGG API client)
```

### Why a single binary?

Static files, templates, and the executable are all bundled via `//go:embed`. Nothing to deploy except the binary. Works perfectly with Docker and Fly.io.

## The Request Pipeline

Every request passes through this chain before reaching a handler:

```
Request
  → SecurityHeaders (all routes)
    → MethodGuard (reject wrong HTTP verb)
      → RequireAuth / RequireJWT (if protected)
        → SameOrigin (if form POST)
          → VerifyCSRF (if HTMX/form POST)
            → Handler
```

### Why `Chain()` reverses middleware order

```go
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}
```

You call `Chain(handler, A, B, C)`. Without reversal, C would run first. With reversal, A wraps first — so execution goes A → B → C → handler. Reading the `mux.Handle` line tells you the security stack in natural order.

## The Dual-Interface Design (HTMX + API)

Nearly every operation has two handlers:

| Route | Handler | Returns |
|---|---|---|
| `GET /games` | `HandleGames` | HTML page or HTMX partial |
| `GET /api/v1/games` | `HandleAPIListGames` | JSON |

**Why both?**

The HTMX frontend was built first. The API was added later to support external clients. Rather than refactor, a parallel API layer was added. They share the same Store calls; only response format differs.

**How the HTMX detection works:**

```go
func isHTMX(r *http.Request) bool {
    return r.Header.Get("HX-Request") == "true"
}

// In HandleGames:
if isHTMX(r) {
    h.Renderer.Partial(w, "games_result", data)  // Return HTML fragment
    return
}
h.renderPage(w, r, "games", "My Games", data)  // Return full page
```

HTMX sets the `HX-Request` header automatically. The same handler returns a partial for swaps or a full page for direct loads.

## Authentication: Two Separate Systems

The app runs two auth systems simultaneously:

### 1. Session cookies (HTMX frontend)

- Login → 32 random bytes → hex token → stored in `sessions` table → set as `sid` cookie
- Every authenticated request validates the `sid` cookie against the DB
- Sessions expire after 30 days
- On login, old sessions are deleted (rotation)

### 2. JWT access tokens (REST API)

- Login → JWT (15-min expiry) + opaque refresh token (30-day, in `sessions` table with `kind='api'`)
- API requests send `Authorization: Bearer <token>` — no DB lookup (self-validating)
- New access tokens via `POST /api/v1/auth/refresh`

**Why two systems?**

Sessions require a DB round-trip on every request — fine for browser HTMX flows. JWTs are stateless — better for API clients. The `kind` column keeps them isolated; a browser session cannot be used as an API token.

## CSRF Protection: Stateless but Secure

```go
func computeCSRF(sessionToken string, secret []byte) string {
    mac := hmac.New(sha256.New, secret)
    mac.Write([]byte(sessionToken))
    return hex.EncodeToString(mac.Sum(nil))
}
```

The CSRF token is `HMAC(sessionToken, serverSecret)`. **Never stored** — recomputed on every request:

- Same session always produces the same token (deterministic HMAC)
- Can't be forged without knowing the random session ID
- No database column — pure computation
- Token-fixation safe: attacker can't pre-compute tokens

Forms embed `_csrf` as a hidden field. HTMX sends it as `X-CSRF-Token`. Middleware checks whichever is present.

## Multi-Tenancy: Ownership Enforced at Two Layers

Every user owns their own games and vibes. Isolation enforced defensively at both handler and store layers.

### Layer 1: SQL WHERE clauses

```go
func (s *Store) GetGame(id, userID int64) (model.Game, error) {
    return scanGame(s.db.QueryRow(
        "SELECT ... FROM games WHERE id = ? AND user_id = ?", id, userID,
    ))
}
```

Every query includes `AND user_id = ?`. Wrong user → zero rows → error.

### Layer 2: Ownership verification for bulk operations

The `vibe_scope.go` pattern handles operations crossing multiple entities:

```go
func ownedIDs(tx *sql.Tx, table string, userID int64, ids []int64) (map[int64]bool, error) {
    // SELECT id FROM {table} WHERE user_id = ? AND id IN (...)
    // Returns which IDs the user actually owns
}

// Used in AddVibesToGames:
ownedGames, _ := ownedIDs(tx, "games", userID, gameIDs)
if len(ownedGames) != len(gameIDs) {
    return ErrForeignOwnership  // User tried to tag someone else's game
}
```

If a user submits 5 game IDs but only owns 4, the entire operation is rejected.

## The Store Layer: Design Decisions

### Schema evolution without a migration tool

All migrations are embedded in `store.New()` and run on every startup. They use `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE IF NOT EXISTS`, and check for existing indexes. This means:

- Old databases upgrade automatically on first run
- Migrations are idempotent (safe to re-run)
- No separate migration CLI tool needed

The biggest migration (`migratePerUserConstraints`) detects if the database has the old single-user schema (global `UNIQUE(bgg_id)`) and atomically rebuilds tables with per-user uniqueness (`UNIQUE(user_id, bgg_id)`).

### Denormalized + normalized columns for games

```
games table:
  categories TEXT  → "Strategy, Economic, Negotiation"  (for display)
  
game_categories table:
  game_id, category_id                                  (for filtering)
```

**Why both?** Displaying is trivial — print the string. Filtering requires exact matching. Parsing CSV on every filter query is slow. The normalized table is kept in sync inside transactions.

### FTS5 for search

```sql
CREATE VIRTUAL TABLE games_fts USING fts5(name, description, content=games)
```

Three triggers keep it in sync. Queries sanitize input (strip special chars, add `*` for prefix matching). Avoids `LIKE '%cage%'` row scans.

### Password hashing: transparent upgrade

Users with old single-SHA256 hashes are silently upgraded to v2 (120,000 iterations + salt) on successful login:

```go
ok, legacy := checkPasswordHash(password, hash)
if ok && legacy {
    upgradedHash, _ := hashPassword(password)
    db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", upgradedHash, id)
}
```

No re-registration needed.

## Template Rendering: Layout Cloning

```go
layout := template.Must(...)  // Parse layout.html once

fullPage := func(files ...string) *template.Template {
    return template.Must(template.Must(layout.Clone()).ParseFS(templateFS, files...))
}
```

**Why clone?** Each page gets its own isolated layout copy so template state doesn't bleed. Isolation is the benefit.

**Partials appear in both lists:**

```go
"games":        fullPage("templates/games.html", "templates/games_result.html"),
"games_result": partial("templates/games_result.html"),
```

`games_result.html` is both embedded in the full page *and* available as a standalone partial for HTMX. Same component, two use modes.

**Buffered rendering:**

```go
var buf bytes.Buffer
if err := t.ExecuteTemplate(&buf, "layout.html", data); err != nil {
    return err  // No partial output sent
}
buf.WriteTo(w)
```

Execution is buffered. If it fails, nothing is written and the handler can still set a 500 status.

## The BGG Client: Three-Layer Transport

```
HTTP request
  → authTransport        (inject Bearer token or cookies)
    → throttledTransport (wait for rate-limit tick, retry 429s)
      → http.DefaultTransport (actual network call)
```

Each layer wraps the one below — same `http.RoundTripper` composition as middleware.

**Throttling:** `time.NewTicker(time.Second / 2)` fires twice per second. Every request waits for a tick. If BGG returns 429, exponential backoff kicks in. Caller never thinks about rate limits.

**Resilient to partial failure:** If 1 batch of 20 games fails, the next 20 are tried. Only if every batch fails does import error. You might import 100 of 120 games.

**CSV import never touches disk:** Preview parses the file, extracts BGG IDs, returns them as a hidden form field. Confirm re-submits the ID list. No temp files.

## The Most Important Design Decisions

| Decision | Rationale |
|---|---|
| Single binary with `embed.FS` | Zero-dependency deployment |
| SQLite | Sufficient for single-user, no infra, persistent volume on Fly.io |
| HTMX + parallel JSON API | Reuse business logic, support both browser and API clients |
| Sessions for browser, JWT for API | DB lookup once per page load is fine; JWT avoids DB per API call |
| HMAC-derived CSRF tokens | No storage, cryptographically sound, immune to fixation |
| `user_id` in every query | Multi-tenancy enforced at DB layer, not just handler layer |
| `ownedIDs()` pattern | Bulk operations can't exploit DB to touch other users' data |
| Migrations embedded in `New()` | Old databases upgrade automatically, no migration CLI |
| Denormalized + normalized columns | Display without parsing; filter without string splitting |
| Buffered template rendering | No partial responses on template errors |
| Three-layer BGG transport | Separation of auth, rate-limiting, and networking concerns |
| Error sentinels | Type-safe business logic checks, no string matching |
