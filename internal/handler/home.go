package handler

import "net/http"

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	count := h.Store.GameCount(userID)
	if err := h.Renderer.Page(w, "home", "Home", count, h.currentUsername(r)); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
