---
name: adding-api-endpoint
description: Adds a new JSON endpoint under /api/v1/ to this Go board game app, following the repo's JWT + snake_case + sentinel-error conventions. Use when the user asks to add a REST API route, a JSON endpoint, a /api/v1/... handler, or anything that extends the JWT-authenticated REST API (parallel to the HTMX site).
---

# Adding a /api/v1/ endpoint

The REST API lives alongside the HTMX app. It shares the store but has its own handlers, helpers, auth middleware, error shape, and JSON converters. Don't mix helpers across the two — HTMX uses `http.Error` / `requireID`; the API uses `writeAPIError` / `requireAPIID`.

## Required touch points

```
- [ ] 1. internal/handler/api_<area>.go — add HandleAPIXxx method on Handler
- [ ] 2. main.go — register the route under /api/v1/ with the right verb wrapper
- [ ] 3. internal/handler/api_helpers.go — add model→snake_case converter if needed
- [ ] 4. internal/store/errors.go — add a sentinel error if introducing a new failure mode
```

## Handler conventions

All API handlers are methods on `*Handler` and live in `internal/handler/api_<area>.go`. Pick the file matching the resource: `api_games.go`, `api_vibes.go`, `api_rules.go`, `api_discover.go`, `api_import.go`, `api_profile.go`, `api_auth.go`. Create a new `api_<area>.go` only for a genuinely new resource.

### Skeleton

```go
// HandleAPIDoThing does thing.
//
// POST /api/v1/things/{id}/do
// Body: {"name": "..."}
func (h *Handler) HandleAPIDoThing(w http.ResponseWriter, r *http.Request) {
    id, ok := requireAPIID(w, r)         // 400 on bad {id}
    if !ok { return }
    userID, ok := h.requireAPIUserID(w, r) // 401 if JWT missing
    if !ok { return }

    var body struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeAPIError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    result, err := h.Store.DoThing(userID, id, body.Name)
    if err != nil {
        if errors.Is(err, store.ErrDuplicate) {
            writeAPIError(w, http.StatusConflict, "already exists")
            return
        }
        slog.Error("HandleAPIDoThing: DoThing", "error", err)
        writeAPIError(w, http.StatusInternalServerError, "internal error")
        return
    }

    writeAPIData(w, http.StatusOK, thingToAPI(result))
}
```

### Helpers (all in `internal/handler/`)

| Helper | Purpose |
|---|---|
| `h.requireAPIUserID(w, r)` | Read user ID from JWT-populated context; writes 401 JSON on miss. |
| `requireAPIID(w, r)` | Parse `{id}` path value; writes 400 JSON on bad parse. |
| `writeAPIData(w, status, v)` | Success — wraps in `{"data": v}`. |
| `writeAPIJSON(w, status, v)` | Success — **no** `data` wrapper. Use only for paginated lists with top-level `total`/`page`/`per_page`. |
| `writeAPIError(w, status, msg)` | Failure — `{"error": msg}`. |

## Response shapes

**Single item:**
```json
{ "data": { "id": 1, "name": "..." } }
```

**Paginated list** — use `writeAPIJSON` and include pagination at top level:
```json
{
  "data": [ {...}, {...} ],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

**Error:**
```json
{ "error": "game not found" }
```

## Status codes

- `200 OK` — GET, PUT, POST success with body
- `201 Created` — POST that creates a resource (return the created object in `data`)
- `204 No Content` — DELETE success (no body, use `w.WriteHeader` directly)
- `400 Bad Request` — malformed JSON, bad path param
- `401 Unauthorized` — missing/invalid JWT (handled by middleware, not usually by handler)
- `403 Forbidden` — authenticated but not allowed (e.g. wrong password on password-change)
- `404 Not Found` — resource doesn't exist or belongs to another user
- `409 Conflict` — duplicate on unique constraint (`store.ErrDuplicate`)
- `413 Request Entity Too Large` — upload over limit
- `422 Unprocessable Entity` — validation error on well-formed input
- `500 Internal Server Error` — anything unexpected; log with `slog.Error` first

## Error handling

**Never expose raw DB errors.** Translate known errors via sentinels defined in `internal/store/errors.go`:

```go
if errors.Is(err, store.ErrDuplicate)   { writeAPIError(w, 409, "already exists"); return }
if errors.Is(err, store.ErrWrongPassword) { writeAPIError(w, 403, "current password incorrect"); return }
if store.IsOwnershipError(err)          { writeAPIError(w, 404, "not found"); return }
```

Anything else: `slog.Error(...)` with a descriptive key, then `writeAPIError(w, 500, "internal error")`. The log line should include the handler name and the wrapped operation (see existing handlers for the `"HandleAPIX: StoreMethodY"` pattern).

## Route registration

Routes live in `main.go` near the other `/api/v1/` lines. Use the right verb wrapper:

| Wrapper | Auth | Use for |
|---|---|---|
| `apiPOSTPublic(...)` | none | Login / refresh / logout only |
| `apiGET(...)` | JWT | GET endpoints |
| `apiPOST(...)` | JWT | POST endpoints |
| `apiPUT(...)` | JWT | PUT endpoints |
| `apiDELETE(...)` | JWT | DELETE endpoints |

Each wrapper chains `MethodGuard` + `RequireJWT`. Stick to the wrappers — don't hand-roll chains.

```go
mux.Handle("POST /api/v1/things/{id}/do", apiPOST(h.HandleAPIDoThing))
```

Path params use Go 1.22+ `mux.Handle` syntax (`{id}`, `{aid_id}`). They're read via `r.PathValue("id")`, wrapped by `requireAPIID`.

## JSON converters

All model→JSON conversion goes through one converter per type in `internal/handler/api_helpers.go`. Keys are **snake_case**. When adding a new type:

```go
func thingToAPI(t model.Thing) map[string]any {
    return map[string]any{
        "id":         t.ID,
        "name":       t.Name,
        "created_at": t.CreatedAt,
    }
}

func thingsToAPI(ts []model.Thing) []map[string]any {
    out := make([]map[string]any, len(ts))
    for i, t := range ts { out[i] = thingToAPI(t) }
    return out
}
```

Don't expose `*model.Thing` directly — a map with explicit keys keeps the wire format decoupled from struct tags.

## Input validation

Validate at the handler boundary. For body fields, check required/empty/length before calling the store. Return `400` for structural problems, `422` for semantic ones.

```go
if body.Name == "" {
    writeAPIError(w, http.StatusBadRequest, "name required")
    return
}
if len(body.Name) > 100 {
    writeAPIError(w, http.StatusUnprocessableEntity, "name too long (max 100)")
    return
}
```

## Verification

```sh
make build
make test
# Manual:
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login -d '{"username":"…","password":"…"}' | jq -r .data.access_token)
curl -X POST localhost:8080/api/v1/things/1/do -H "Authorization: Bearer $TOKEN" -d '{"name":"x"}'
```

See `agent_docs/ARCHITECTURE-REF.md` for the full route list and env vars.
