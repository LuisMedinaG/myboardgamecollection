package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// HandleAPIGetProfile returns the authenticated user's profile info.
//
// GET /api/v1/profile
func (h *Handler) HandleAPIGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	username := h.currentUsername(r)

	bggUsername, err := h.Store.GetBGGUsername(userID)
	if err != nil {
		slog.Error("HandleAPIGetProfile: GetBGGUsername", "error", err)
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"username":     username,
		"bgg_username": bggUsername,
	})
}

// HandleAPISetBGGUsername updates the user's BGG username.
//
// PUT /api/v1/profile/bgg-username
// Body: {"bgg_username": "boardgamer42"}
func (h *Handler) HandleAPISetBGGUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		BGGUsername string `json:"bgg_username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	bggUsername := strings.TrimSpace(body.BGGUsername)

	if err := h.Store.SetBGGUsername(userID, bggUsername); err != nil {
		slog.Error("HandleAPISetBGGUsername: SetBGGUsername", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"bgg_username": bggUsername,
	})
}

// HandleAPIChangePassword changes the user's password after verifying the current one.
//
// PUT /api/v1/profile/password
// Body: {"current_password": "old", "new_password": "newpass123"}
func (h *Handler) HandleAPIChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		writeAPIError(w, http.StatusBadRequest, "current_password and new_password required")
		return
	}
	if len(body.NewPassword) < 8 {
		writeAPIError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	if err := h.Store.ChangePassword(userID, body.CurrentPassword, body.NewPassword); err != nil {
		if strings.Contains(err.Error(), "current password") {
			writeAPIError(w, http.StatusForbidden, "current password incorrect")
			return
		}
		slog.Error("HandleAPIChangePassword: ChangePassword", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
