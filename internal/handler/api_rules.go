package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"myboardgamecollection/internal/model"
)

func (h *Handler) HandleAPIUpdateRulesURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}

	var body struct {
		RulesURL string `json:"rules_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateRulesURL(body.RulesURL); err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if _, err := h.Store.GetGame(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	if err := h.Store.UpdateGameRulesURL(id, body.RulesURL, userID); err != nil {
		slog.Error("UpdateGameRulesURL", "gameID", id, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"game_id":   id,
		"rules_url": body.RulesURL,
	})
}

func (h *Handler) HandleAPIUploadPlayerAid(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPlayerAidUploadBytes)
	if err := r.ParseMultipartForm(maxPlayerAidUploadBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "file too large")
			return
		}
		writeAPIError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	if _, err := h.Store.GetGame(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	filename, err := h.savePlayerAidFile(file)
	if err != nil {
		if err == errUnsupportedPlayerAidType {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("HandleAPIUploadPlayerAid: savePlayerAidFile", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	label := sanitizePlayerAidLabel(r.FormValue("label"), header.Filename)

	aidID, err := h.Store.CreatePlayerAid(id, filename, label)
	if err != nil {
		slog.Error("CreatePlayerAid", "gameID", id, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeAPIData(w, http.StatusCreated, playerAidToAPI(model.PlayerAid{
		ID:       aidID,
		GameID:   id,
		Filename: filename,
		Label:    label,
	}))
}

func (h *Handler) HandleAPIDeletePlayerAid(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}
	id, ok := requireAPIID(w, r)
	if !ok {
		return
	}

	aidID, err := parseAidID(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid aid_id")
		return
	}

	if _, err := h.Store.GetGame(id, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, "game not found")
		return
	}

	aid, err := h.Store.GetPlayerAid(aidID)
	if err != nil || aid.GameID != id {
		writeAPIError(w, http.StatusNotFound, "player aid not found")
		return
	}

	_ = os.Remove(filepath.Join(h.DataDir, "uploads", aid.Filename))

	if err := h.Store.DeletePlayerAid(aidID); err != nil {
		slog.Error("DeletePlayerAid", "aidID", aidID, "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
