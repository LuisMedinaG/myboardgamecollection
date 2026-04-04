package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleVibes(w http.ResponseWriter, r *http.Request) {
	vibes, _ := h.Store.AllVibes()
	if err := h.Renderer.Page(w, "vibes", "Manage Vibes", viewmodel.VibesPageData{Vibes: vibes}); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleVibeCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if _, err := h.Store.CreateVibe(name); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			h.renderVibesWithError(w, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleVibeUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateVibe(id, name); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			h.renderVibesWithError(w, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleVibeDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.Store.DeleteVibe(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func (h *Handler) HandleGameEdit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	game, err := h.Store.GetGame(id)
	if err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	vibes, _ := h.Store.AllVibes()
	gameVibeList, _ := h.Store.VibesForGame(id)
	gvMap := make(map[int64]bool)
	for _, v := range gameVibeList {
		gvMap[v.ID] = true
	}
	data := viewmodel.GameEditData{Game: game, AllVibes: vibes, GameVibes: gvMap}
	if err := h.Renderer.Page(w, "game_edit", "Edit — "+game.Name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleGameVibesSave(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
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
	if err := h.Store.SetGameVibes(id, vibeIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d", id), http.StatusSeeOther)
}

func (h *Handler) renderVibesWithError(w http.ResponseWriter, errMsg string) {
	vibes, _ := h.Store.AllVibes()
	h.Renderer.Page(w, "vibes", "Manage Vibes", viewmodel.VibesPageData{Vibes: vibes, Error: errMsg})
}
