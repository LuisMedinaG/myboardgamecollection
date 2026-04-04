package handler

import "net/http"

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	count := h.Store.GameCount()
	if err := h.Renderer.Page(w, "home", "Home", count); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
