package handler

import (
	"context"
	"net/http"
	"strconv"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/service"
	"myboardgamecollection/internal/store"
	"myboardgamecollection/internal/viewmodel"
)

type GameHandler struct {
	gameService *service.GameService
	store       *store.Store
	renderer    *render.Renderer
}

func NewGameHandler(gameService *service.GameService, store *store.Store, renderer *render.Renderer) *GameHandler {
	return &GameHandler{
		gameService: gameService,
		store:       store,
		renderer:    renderer,
	}
}

func (h *GameHandler) currentUsername(r *http.Request) string {
	return httpx.UsernameFromContext(r.Context())
}

func (h *GameHandler) csrfToken(r *http.Request) string {
	return httpx.CSRFTokenFromContext(r.Context())
}

// renderPage renders a full page with username and CSRF token from context.
func (h *GameHandler) renderPage(w http.ResponseWriter, r *http.Request, name, title string, data any) error {
	return h.renderer.Page(w, name, title, data, h.currentUsername(r), h.csrfToken(r))
}

func (h *GameHandler) HandleGames(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrUnauthorized)
		return
	}

	req := parseGamesRequest(r)
	data, err := h.loadGamesPageData(r.Context(), userID, req)
	if err != nil {
		httpx.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		if err := h.renderer.Partial(w, "games_result", data); err != nil {
			httpx.HandleError(w, err, http.StatusInternalServerError)
		}
		return
	}

	if err := h.renderPage(w, r, "games", "My Games", data); err != nil {
		httpx.HandleError(w, err, http.StatusInternalServerError)
	}
}

func (h *GameHandler) HandleGameDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrInvalidID)
		return
	}

	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrUnauthorized)
		return
	}

	game, err := h.gameService.GetGame(r.Context(), id, userID)
	if err != nil {
		httpx.HandleError(w, err, http.StatusNotFound)
		return
	}

	aids, err := h.store.GetPlayerAids(id)
	if err != nil {
		httpx.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	if err := h.renderPage(w, r, "game_detail", game.Name, viewmodel.GameDetailData{Game: *game, Aids: aids}); err != nil {
		httpx.HandleError(w, err, http.StatusInternalServerError)
	}
}

func (h *GameHandler) HandleBulkVibeAssign(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		httpx.HandleError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.gameService.AssignVibesToGames(r.Context(), userID, parseInt64Values(r.Form["game_ids"]), parseInt64Values(r.Form["vibes"])); err != nil {
		httpx.HandleError(w, err, http.StatusBadRequest)
		return
	}

	redirectToGames(w, r)
}

func (h *GameHandler) HandleGameDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrInvalidID)
		return
	}

	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, httpx.ErrUnauthorized)
		return
	}

	if err := h.gameService.DeleteGame(r.Context(), id, userID); err != nil {
		httpx.HandleError(w, err, http.StatusInternalServerError)
		return
	}

	redirectToGames(w, r)
}

type gamesRequest struct {
	filters service.GameFilters
	page    int
	limit   int
}

func parseGamesRequest(r *http.Request) gamesRequest {
	page, limit := parsePagination(r)
	return gamesRequest{
		filters: service.GameFilters{
			Query:      r.URL.Query().Get("q"),
			Category:   r.URL.Query().Get("category"),
			Players:    r.URL.Query().Get("players"),
			Playtime:   r.URL.Query().Get("playtime"),
			Weight:     r.URL.Query().Get("weight"),
			Rating:     r.URL.Query().Get("rating"),
			Language:   r.URL.Query().Get("lang"),
			RecPlayers: r.URL.Query().Get("rec_players"),
		},
		page:  page,
		limit: limit,
	}
}

func (h *GameHandler) loadGamesPageData(ctx context.Context, userID int64, req gamesRequest) (viewmodel.GamesPageData, error) {
	result, err := h.gameService.ListGames(ctx, userID, req.filters, req.page, req.limit)
	if err != nil {
		return viewmodel.GamesPageData{}, err
	}

	categories, err := h.gameService.GetCategories(ctx, userID)
	if err != nil {
		return viewmodel.GamesPageData{}, err
	}

	vibes, err := h.gameService.GetAllVibes(ctx, userID)
	if err != nil {
		return viewmodel.GamesPageData{}, err
	}

	return viewmodel.GamesPageData{
		Games:      result.Games,
		Categories: categories,
		AllVibes:   vibes,
		Q:          req.filters.Query,
		Category:   req.filters.Category,
		Players:    req.filters.Players,
		Playtime:   req.filters.Playtime,
		Weight:     req.filters.Weight,
		Rating:     req.filters.Rating,
		Lang:       req.filters.Language,
		RecPlayers: req.filters.RecPlayers,
		Page:       result.Page,
		TotalPages: result.TotalPages,
		TotalCount: result.Total,
		PerPage:    result.PerPage,
	}, nil
}

func parseInt64Values(values []string) []int64 {
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func redirectToGames(w http.ResponseWriter, r *http.Request) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/games")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}
