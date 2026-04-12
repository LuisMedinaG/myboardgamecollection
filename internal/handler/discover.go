package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"myboardgamecollection/internal/filter"
	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	vibeIDStr := r.URL.Query().Get("vibe")

	// No vibe selected — show vibe grid.
	if vibeIDStr == "" {
		h.renderDiscoverGrid(w, r, userID)
		return
	}

	vibeID, err := strconv.ParseInt(vibeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid vibe", http.StatusBadRequest)
		return
	}

	vibe, err := h.Store.GetVibe(vibeID, userID)
	if err != nil {
		http.Error(w, "vibe not found", http.StatusNotFound)
		return
	}

	typ := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	mechanic := r.URL.Query().Get("mechanic")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")
	weight := r.URL.Query().Get("weight")

	games, err := h.Store.FilterGamesByVibe(vibeID, typ, category, mechanic, players, playtime, weight, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := buildDiscoverData(vibe, vibeID, games, typ, category, mechanic, players, playtime, weight)

	if isHTMX(r) {
		if err := h.Renderer.Partial(w, "discover_result", data); err != nil {
			http.Error(w, "render error", http.StatusInternalServerError)
		}
		return
	}
	if err := h.renderPage(w, r, "discover", "Discover — "+vibe.Name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderDiscoverGrid(w http.ResponseWriter, r *http.Request, userID int64) {
	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("AllVibes", "error", err)
	}
	data := viewmodel.DiscoverPageData{Vibes: vibes}
	if isHTMX(r) {
		if err := h.Renderer.Partial(w, "discover_result", data); err != nil {
			http.Error(w, "render error", http.StatusInternalServerError)
		}
		return
	}
	if err := h.renderPage(w, r, "discover", "Discover", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func buildDiscoverData(vibe model.Vibe, vibeID int64, games []model.Game, typ, category, mechanic, players, playtime, weight string) viewmodel.DiscoverPageData {
	return viewmodel.DiscoverPageData{
		VibeID:         vibeID,
		VibeName:       vibe.Name,
		Games:          games,
		Types:          filter.ExtractField(games, func(g model.Game) string { return g.Types }),
		Categories:     filter.ExtractField(games, func(g model.Game) string { return g.Categories }),
		Mechanics:      filter.ExtractField(games, func(g model.Game) string { return g.Mechanics }),
		Type:           typ,
		Category:       category,
		Mechanic:       mechanic,
		Players:        players,
		Playtime:       playtime,
		Weight:         weight,
		ValidPlayers:   filter.ValidPlayerOptions(games),
		ValidPlaytimes: filter.ValidPlaytimeOptions(games),
		ValidWeights:   filter.ValidWeightOptions(games),
	}
}
