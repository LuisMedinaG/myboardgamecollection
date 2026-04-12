package handler

import (
	"log/slog"
	"net/http"
	"strconv"
)

func (h *Handler) HandleAPIDiscover(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()

	vibeIDStr := q.Get("vibe_id")
	if vibeIDStr == "" {
		writeAPIError(w, http.StatusBadRequest, "vibe_id is required")
		return
	}
	vibeID, err := strconv.ParseInt(vibeIDStr, 10, 64)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid vibe_id")
		return
	}

	vibe, err := h.Store.GetVibe(vibeID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "vibe not found")
		return
	}

	typ := q.Get("type")
	category := q.Get("category")
	mechanic := q.Get("mechanic")
	players := q.Get("players")
	playtime := q.Get("playtime")
	weight := q.Get("weight")

	games, err := h.Store.FilterGamesByVibe(vibeID, typ, category, mechanic, players, playtime, weight, userID)
	if err != nil {
		slog.Error("FilterGamesByVibe", "vibeID", vibeID, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIJSON(w, http.StatusOK, map[string]any{
		"data":  gamesToAPI(games),
		"total": len(games),
		"vibe":  vibeToAPI(vibe),
	})
}
