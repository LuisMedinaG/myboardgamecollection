package handler

import "net/http"

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	count := h.Store.GameCount(userID)
	if err := h.renderPage(w, r, "home", "Home", count); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
