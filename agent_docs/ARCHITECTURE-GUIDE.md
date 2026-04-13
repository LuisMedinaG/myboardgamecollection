# Architecture Guide

Read this when you need to understand *why* things are built this way — not *how* (read the code for that).

## Macro Architecture

```
main.go  →  Routes + middleware chains  →  Server start
             │
     ┌───────┼───────────┐
  httpx/   handler/    render/
  (middleware) (HTMX + API)  (templates)
             │
     ┌───────┴───────┐
   store/         bgg/
   (SQLite DAL)   (BGG client)
```

Single binary — static files, templates, and executable bundled via `//go:embed`. Nothing to deploy except the binary.

## Request Pipeline

```
Request → SecurityHeaders → MethodGuard → RequireAuth/RequireJWT → SameOrigin → VerifyCSRF → Handler
```

`Chain(handler, A, B, C)` reverses internally so A runs first. Read the `mux.Handle` line to see the security stack in natural order.

## Design Decisions That Affect How You Code

| Decision | What it means for you |
|---|---|
| Dual interface (HTMX + API) | Every feature needs two handlers sharing the same Store call. Don't refactor one without the other. |
| Session cookies + JWT | Sessions for browser (DB-backed), JWT for API (stateless). `kind` column isolates them — don't mix. |
| HMAC-derived CSRF | Stateless, derived from session token. Never add a DB column for CSRF. |
| `user_id` in every query | Multi-tenancy at SQL layer. Never write a query without `AND user_id = ?`. |
| `ownedIDs()` for bulk ops | Validates ownership before mutating. Don't skip this for batch operations. |
| Migrations in `store.New()` | Idempotent, run on every startup. Use `ALTER TABLE … ADD COLUMN … DEFAULT`. Also update `migrateGamesTableForPerUserUniqueness`. |
| Denormalized + normalized columns | `categories TEXT` for display, `game_categories` table for filtering. Keep both in sync within transactions. |
| FTS5 for search | Triggers keep `games_fts` in sync. Don't bypass them with direct inserts. |
| Password hash upgrade | Old SHA256 hashes silently upgrade to v2 on login. Don't break the `checkPasswordHash` → upgrade flow. |
| Buffered template rendering | Templates render to buffer first. Errors don't produce partial output. Don't write directly to `http.ResponseWriter` in template paths. |
| BGG three-layer transport | auth → throttle → network. Rate limiting is automatic. Don't add manual sleeps or retry logic around BGG calls. |
| Partials in both registries | HTMX partials are registered standalone AND embedded in full pages. When adding a new partial, register it in both places in `render/`. |
