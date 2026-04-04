package handler

import (
	"log/slog"
	"net/http"

	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleGames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")

	games, err := h.Store.FilterGames(q, category, players, playtime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, _ := h.Store.DistinctCategories()

	data := viewmodel.GamesPageData{
		Games:      games,
		Categories: categories,
		Q:          q,
		Category:   category,
		Players:    players,
		Playtime:   playtime,
	}

	if isHTMX(r) {
		if err := h.Renderer.Partial(w, "games_result", data.Games); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	if err := h.Renderer.Page(w, "games", "My Games", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGameDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	game, err := h.Store.GetGame(id)
	if err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	aids, err := h.Store.GetPlayerAids(id)
	if err != nil {
		slog.Error("GetPlayerAids", "gameID", id, "error", err)
	}
	if err := h.Renderer.Page(w, "game_detail", game.Name, viewmodel.GameDetailData{Game: game, Aids: aids}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGameDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteGame(id); err != nil {
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
