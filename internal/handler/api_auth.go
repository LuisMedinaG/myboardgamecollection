package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"myboardgamecollection/internal/httpx"
)

// HandleAPILogin authenticates a user and returns a short-lived JWT access token
// plus a long-lived opaque refresh token.
//
// POST /api/v1/auth/login
// Body: {"username":"...","password":"..."}
func (h *Handler) HandleAPILogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Username == "" || body.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "username and password required")
		return
	}

	userID, err := h.Store.AuthenticateUser(body.Username, body.Password)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	_, isAdmin, err := h.Store.GetUserInfo(userID)
	if err != nil {
		slog.Error("HandleAPILogin: GetUserInfo", "user_id", userID, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	accessToken, err := httpx.GenerateAccessToken(userID, body.Username, isAdmin, h.JWTSecret)
	if err != nil {
		slog.Error("HandleAPILogin: GenerateAccessToken", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	refreshToken, err := h.Store.CreateAPIRefreshToken(userID)
	if err != nil {
		slog.Error("HandleAPILogin: CreateAPIRefreshToken", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    900, // 15 minutes in seconds
	})
}

// HandleAPIRefresh issues a new access token using a valid refresh token.
//
// POST /api/v1/auth/refresh
// Body: {"refresh_token":"..."}
func (h *Handler) HandleAPIRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.RefreshToken == "" {
		writeAPIError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	userID, username, isAdmin, err := h.Store.ValidateAPIRefreshToken(body.RefreshToken)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	accessToken, err := httpx.GenerateAccessToken(userID, username, isAdmin, h.JWTSecret)
	if err != nil {
		slog.Error("HandleAPIRefresh: GenerateAccessToken", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"expires_in":   900,
	})
}

// HandleAPILogout revokes a refresh token.
//
// POST /api/v1/auth/logout
// Body: {"refresh_token":"..."}
func (h *Handler) HandleAPILogout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Best-effort delete — ignore not-found errors.
	if body.RefreshToken != "" {
		if err := h.Store.DeleteAPIRefreshToken(body.RefreshToken); err != nil {
			slog.Error("HandleAPILogout: DeleteAPIRefreshToken", "error", err)
		}
	}
	writeAPIData(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleAPIPing is a protected endpoint that confirms JWT auth is working.
//
// GET /api/v1/ping
func (h *Handler) HandleAPIPing(w http.ResponseWriter, r *http.Request) {
	username := h.currentUsername(r)
	writeAPIData(w, http.StatusOK, map[string]any{
		"pong":     true,
		"username": username,
	})
}

// writeAPIData writes a JSON success response: {"data": v}
func writeAPIData(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
}

// writeAPIError writes a JSON error response: {"error": msg}
func writeAPIError(w http.ResponseWriter, status int, msg string) {
	httpx.WriteJSONError(w, status, msg)
}
