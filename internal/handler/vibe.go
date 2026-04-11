package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"myboardgamecollection/internal/store"
	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleVibes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("AllVibes", "error", err)
	}
	if err := h.renderPage(w, r, "vibes", "Manage Vibes", viewmodel.VibesPageData{Vibes: vibes}); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleVibeCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(name) > 100 {
		http.Error(w, "name too long (max 100 characters)", http.StatusBadRequest)
		return
	}
	if _, err := h.Store.CreateVibe(name, userID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			h.renderVibesWithError(w, r, userID, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleVibeUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(name) > 100 {
		http.Error(w, "name too long (max 100 characters)", http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateVibe(id, name, userID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			h.renderVibesWithError(w, r, userID, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleVibeBatchUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	raw := r.FormValue("updates")
	if raw == "" {
		http.Error(w, "no updates provided", http.StatusBadRequest)
		return
	}

	var updates map[string]string
	if err := json.Unmarshal([]byte(raw), &updates); err != nil {
		http.Error(w, "invalid updates", http.StatusBadRequest)
		return
	}

	for idStr, name := range updates {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		name = strings.TrimSpace(name)
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if len(name) > 100 {
			http.Error(w, "name too long (max 100 characters)", http.StatusBadRequest)
			return
		}
		if err := h.Store.UpdateVibe(id, name, userID); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				h.renderVibesWithError(w, r, userID, fmt.Sprintf("A vibe named %q already exists.", name))
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleVibeDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteVibe(id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleGameEdit(w http.ResponseWriter, r *http.Request) {
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
	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("AllVibes", "error", err)
	}
	gameVibeList, err := h.Store.VibesForGame(id)
	if err != nil {
		slog.Error("VibesForGame", "gameID", id, "error", err)
	}
	gvMap := make(map[int64]bool)
	for _, v := range gameVibeList {
		gvMap[v.ID] = true
	}
	data := viewmodel.GameEditData{Game: game, AllVibes: vibes, GameVibes: gvMap}
	if err := h.renderPage(w, r, "game_edit", "Edit — "+game.Name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGameVibesSave(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	// Verify game belongs to user before mutating vibes.
	if _, err := h.Store.GetGame(id, userID); err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var vibeIDs []int64
	for _, v := range r.Form["vibes"] {
		vid, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			vibeIDs = append(vibeIDs, vid)
		}
	}
	if err := h.Store.SetGameVibes(userID, id, vibeIDs); err != nil {
		if store.IsOwnershipError(err) {
			http.Error(w, "one or more selected vibes were not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d", id), http.StatusSeeOther)
}

func (h *Handler) renderVibesWithError(w http.ResponseWriter, r *http.Request, userID int64, errMsg string) {
	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("AllVibes", "error", err)
	}
	h.renderPage(w, r, "vibes", "Manage Vibes", viewmodel.VibesPageData{Vibes: vibes, Error: errMsg})
}
