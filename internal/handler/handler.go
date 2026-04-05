package handler

import (
	"net/http"
	"strconv"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	Store    *store.Store
	Renderer *render.Renderer
	BGG      *bgg.Client // may be nil if BGG is not configured
	DataDir  string      // base directory for user uploads and image cache
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// requireID parses the {id} path value and writes a 400 if it is not a valid
// integer. Returns (id, true) on success; (0, false) when an error response
// has already been written and the caller must return immediately.
func requireID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// requireUserID reads the authenticated user's ID from the request context.
// It writes a 401 and returns (0, false) if the user is not in context (should
// not happen when RequireAuth middleware is in place, but guards against misuse).
func (h *Handler) requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return id, true
}

// currentUsername returns the BGG username of the authenticated user from
// context, or "" if the context has no user (e.g. login page).
func (h *Handler) currentUsername(r *http.Request) string {
	return httpx.UsernameFromContext(r.Context())
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
