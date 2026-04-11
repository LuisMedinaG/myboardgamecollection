package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"myboardgamecollection/internal/store"
	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleGames(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")
	page, limit := parsePagination(r)

	games, total, err := h.Store.FilterGames(q, category, players, playtime, page, limit, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	categories, err := h.Store.DistinctCategories(userID)
	if err != nil {
		slog.Error("DistinctCategories", "userID", userID, "error", err)
	}
	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("AllVibes", "userID", userID, "error", err)
	}

	h.populateGameVibes(games)

	data := viewmodel.GamesPageData{
		Games:      games,
		Categories: categories,
		AllVibes:   vibes,
		Q:          q,
		Category:   category,
		Players:    players,
		Playtime:   playtime,
		Page:       page,
		TotalPages: totalPages,
		TotalCount: total,
		PerPage:    limit,
	}

	if isHTMX(r) {
		if err := h.Renderer.Partial(w, "games_result", data); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	if err := h.renderPage(w, r, "games", "My Games", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGameDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	game, err := h.Store.GetGame(id, userID)
	if err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	aids, err := h.Store.GetPlayerAids(id)
	if err != nil {
		slog.Error("GetPlayerAids", "gameID", id, "error", err)
	}
	if err := h.renderPage(w, r, "game_detail", game.Name, viewmodel.GameDetailData{Game: game, Aids: aids}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleBulkVibeAssign(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var gameIDs []int64
	for _, v := range r.Form["game_ids"] {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			gameIDs = append(gameIDs, id)
		}
	}
	var vibeIDs []int64
	for _, v := range r.Form["vibes"] {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			vibeIDs = append(vibeIDs, id)
		}
	}
	if len(gameIDs) == 0 || len(vibeIDs) == 0 {
		http.Error(w, "select at least one game and one vibe", http.StatusBadRequest)
		return
	}
	if err := h.Store.AddVibesToGames(userID, gameIDs, vibeIDs); err != nil {
		if store.IsOwnershipError(err) {
			http.Error(w, "one or more selected games or vibes were not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

func (h *Handler) HandleGameDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteGame(id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/games")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}
