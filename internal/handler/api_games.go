package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"myboardgamecollection/internal/store"
)

// HandleAPIListGames returns a paginated, filtered list of the user's games.
//
// GET /api/v1/games
// Query: q, category, players, playtime, page, limit
func (h *Handler) HandleAPIListGames(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")
	weight := r.URL.Query().Get("weight")
	page, limit := parsePagination(r)

	games, total, err := h.Store.FilterGames(q, category, players, playtime, weight, page, limit, userID)
	if err != nil {
		slog.Error("HandleAPIListGames: FilterGames", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	categories, err := h.Store.DistinctCategories(userID)
	if err != nil {
		slog.Error("HandleAPIListGames: DistinctCategories", "error", err)
		categories = []string{}
	}

	h.populateGameVibes(games)

	writeAPIJSON(w, http.StatusOK, map[string]any{
		"data":       gamesToAPI(games),
		"total":      total,
		"page":       page,
		"per_page":   limit,
		"categories": categories,
	})
}

// HandleAPIGetGame returns a single game with its vibes and player aids.
//
// GET /api/v1/games/{id}
func (h *Handler) HandleAPIGetGame(w http.ResponseWriter, r *http.Request) {
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	game, err := h.Store.GetGame(id, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	vibes, err := h.Store.VibesForGame(game.ID)
	if err != nil {
		slog.Error("HandleAPIGetGame: VibesForGame", "error", err)
	}

	aids, err := h.Store.GetPlayerAids(game.ID)
	if err != nil {
		slog.Error("HandleAPIGetGame: GetPlayerAids", "error", err)
	}

	resp := gameToAPI(game)
	resp["vibes"] = vibesToAPI(vibes)
	resp["player_aids"] = playerAidsToAPI(aids)

	writeAPIData(w, http.StatusOK, resp)
}

// HandleAPIDeleteGame removes a game from the user's collection.
//
// DELETE /api/v1/games/{id}
func (h *Handler) HandleAPIDeleteGame(w http.ResponseWriter, r *http.Request) {
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	// Verify the game exists and belongs to the user.
	if _, err := h.Store.GetGame(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	if err := h.Store.DeleteGame(id, userID); err != nil {
		slog.Error("HandleAPIDeleteGame: DeleteGame", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleAPISetGameVibes replaces all vibe associations for a game.
//
// POST /api/v1/games/{id}/vibes
// Body: {"vibe_ids": [1, 2, 3]}
func (h *Handler) HandleAPISetGameVibes(w http.ResponseWriter, r *http.Request) {
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		VibeIDs []int64 `json:"vibe_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, err := h.Store.GetGame(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	if err := h.Store.SetGameVibes(userID, id, body.VibeIDs); err != nil {
		if store.IsOwnershipError(err) {
			writeAPIError(w, http.StatusNotFound, "one or more vibes were not found")
			return
		}
		slog.Error("HandleAPISetGameVibes: SetGameVibes", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"game_id":  id,
		"vibe_ids": body.VibeIDs,
	})
}

// HandleAPIBulkVibes adds vibes to multiple games at once.
//
// POST /api/v1/games/bulk-vibes
// Body: {"game_ids": [1,2,3], "vibe_ids": [4,5]}
func (h *Handler) HandleAPIBulkVibes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		GameIDs []int64 `json:"game_ids"`
		VibeIDs []int64 `json:"vibe_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.GameIDs) == 0 || len(body.VibeIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, "game_ids and vibe_ids required")
		return
	}

	if err := h.Store.AddVibesToGames(userID, body.GameIDs, body.VibeIDs); err != nil {
		if store.IsOwnershipError(err) {
			writeAPIError(w, http.StatusNotFound, "one or more games or vibes were not found")
			return
		}
		slog.Error("HandleAPIBulkVibes: AddVibesToGames", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"updated": len(body.GameIDs),
	})
}
