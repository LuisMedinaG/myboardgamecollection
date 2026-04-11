package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	Store        *store.Store
	Renderer     *render.Renderer
	BGG          *bgg.Client         // may be nil if BGG is not configured
	DataDir      string              // base directory for user uploads and image cache
	LoginLimiter *httpx.LoginLimiter // may be nil; records failed login attempts
	JWTSecret    string              // signing key for API access tokens
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

// csrfToken returns the CSRF token for the current request.
func (h *Handler) csrfToken(r *http.Request) string {
	return httpx.CSRFTokenFromContext(r.Context())
}

// renderPage renders a full page with username and CSRF token from context.
func (h *Handler) renderPage(w http.ResponseWriter, r *http.Request, name, title string, data any) error {
	return h.Renderer.Page(w, name, title, data, h.currentUsername(r), h.csrfToken(r))
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func syncLimit(r *http.Request) int {
	if httpx.IsAdminFromContext(r.Context()) {
		return syncLimitAdmin
	}
	return syncLimitRegular
}

// parsePagination reads page and limit from query parameters, applying defaults
// and clamping to [1, MaxPageSize].
func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = store.DefaultPageSize
	} else if limit > store.MaxPageSize {
		limit = store.MaxPageSize
	}
	return page, limit
}

// populateGameVibes fetches and assigns vibes for each game in the slice.
func (h *Handler) populateGameVibes(games []model.Game) {
	if len(games) == 0 {
		return
	}
	gameIDs := make([]int64, len(games))
	for i, g := range games {
		gameIDs[i] = g.ID
	}
	gameVibes, err := h.Store.VibesForGames(gameIDs)
	if err != nil {
		slog.Error("populateGameVibes", "error", err)
		return
	}
	for i := range games {
		games[i].Vibes = gameVibes[games[i].ID]
	}
}
