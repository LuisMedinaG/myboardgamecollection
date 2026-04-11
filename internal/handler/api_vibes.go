package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// HandleAPIListVibes returns all vibes owned by the authenticated user.
//
// GET /api/v1/vibes
func (h *Handler) HandleAPIListVibes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	vibes, err := h.Store.AllVibes(userID)
	if err != nil {
		slog.Error("HandleAPIListVibes: AllVibes", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, vibesToAPI(vibes))
}

// HandleAPICreateVibe creates a new vibe.
//
// POST /api/v1/vibes
// Body: {"name": "Chill"}
func (h *Handler) HandleAPICreateVibe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		writeAPIError(w, http.StatusBadRequest, "name too long (max 100 characters)")
		return
	}

	id, err := h.Store.CreateVibe(name, userID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeAPIError(w, http.StatusConflict, "vibe already exists")
			return
		}
		slog.Error("HandleAPICreateVibe: CreateVibe", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusCreated, map[string]any{"id": id, "name": name})
}

// HandleAPIUpdateVibe renames an existing vibe.
//
// PUT /api/v1/vibes/{id}
// Body: {"name": "New Name"}
func (h *Handler) HandleAPIUpdateVibe(w http.ResponseWriter, r *http.Request) {
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeAPIError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		writeAPIError(w, http.StatusBadRequest, "name too long (max 100 characters)")
		return
	}

	if _, err := h.Store.GetVibe(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "vibe not found")
		return
	}

	if err := h.Store.UpdateVibe(id, name, userID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeAPIError(w, http.StatusConflict, "vibe already exists")
			return
		}
		slog.Error("HandleAPIUpdateVibe: UpdateVibe", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{"id": id, "name": name})
}

// HandleAPIDeleteVibe removes a vibe.
//
// DELETE /api/v1/vibes/{id}
func (h *Handler) HandleAPIDeleteVibe(w http.ResponseWriter, r *http.Request) {
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	if err := h.Store.DeleteVibe(id, userID); err != nil {
		slog.Error("HandleAPIDeleteVibe: DeleteVibe", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
