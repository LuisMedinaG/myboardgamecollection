package handler

import (
	"encoding/json"
	"net/http"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/model"
)

// requireAPIUserID reads the authenticated user's ID from context and writes a
// JSON 401 if absent. Use this instead of requireUserID in API handlers.
func (h *Handler) requireAPIUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	return id, true
}

// requireAPIID parses the {id} path value and writes a JSON 400 on failure.
func requireAPIID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := parseID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// writeAPIJSON writes an arbitrary JSON value without the {"data":...} wrapper.
// Used for paginated list responses that include top-level pagination fields.
func writeAPIJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ---------------------------------------------------------------------------
// Model → snake_case JSON converters
// ---------------------------------------------------------------------------

func gameToAPI(g model.Game) map[string]any {
	return map[string]any{
		"id":             g.ID,
		"bgg_id":         g.BGGID,
		"name":           g.Name,
		"description":    g.Description,
		"year_published": g.YearPublished,
		"image":          g.Image,
		"thumbnail":      g.Thumbnail,
		"min_players":    g.MinPlayers,
		"max_players":    g.MaxPlayers,
		"play_time":      g.PlayTime,
		"categories":     g.Categories,
		"mechanics":      g.Mechanics,
		"types":          g.Types,
		"weight":         g.Weight,
		"rules_url":      g.RulesURL,
		"vibes":          vibesToAPI(g.Vibes),
	}
}

func gamesToAPI(gs []model.Game) []map[string]any {
	out := make([]map[string]any, len(gs))
	for i, g := range gs {
		out[i] = gameToAPI(g)
	}
	return out
}

func vibeToAPI(v model.Vibe) map[string]any {
	return map[string]any{"id": v.ID, "name": v.Name}
}

func vibesToAPI(vs []model.Vibe) []map[string]any {
	out := make([]map[string]any, 0, len(vs))
	for _, v := range vs {
		out = append(out, vibeToAPI(v))
	}
	return out
}

func playerAidToAPI(a model.PlayerAid) map[string]any {
	return map[string]any{
		"id":       a.ID,
		"game_id":  a.GameID,
		"filename": a.Filename,
		"label":    a.Label,
	}
}

func playerAidsToAPI(as []model.PlayerAid) []map[string]any {
	out := make([]map[string]any, 0, len(as))
	for _, a := range as {
		out = append(out, playerAidToAPI(a))
	}
	return out
}
