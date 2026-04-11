package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"myboardgamecollection/internal/viewmodel"
)

const maxPlayerAidUploadBytes = 10 << 20

func (h *Handler) HandleRules(w http.ResponseWriter, r *http.Request) {
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
	aids, err := h.Store.GetPlayerAids(id)
	if err != nil {
		slog.Error("GetPlayerAids", "gameID", id, "error", err)
	}

	data := viewmodel.RulesPageData{
		Game:       game,
		PlayerAids: aids,
		EmbedURL:   driveEmbedURL(game.RulesURL),
	}
	if err := h.renderPage(w, r, "rules", game.Name+" — Rules", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleRulesURLUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	rulesURL := strings.TrimSpace(r.FormValue("rules_url"))
	if err := validateRulesURL(rulesURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateGameRulesURL(id, rulesURL, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		game, err := h.Store.GetGame(id, userID)
		if err != nil {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		aids, err := h.Store.GetPlayerAids(id)
		if err != nil {
			slog.Error("GetPlayerAids", "gameID", id, "error", err)
		}
		data := viewmodel.RulesPageData{
			Game:       game,
			PlayerAids: aids,
			EmbedURL:   driveEmbedURL(game.RulesURL),
		}
		if err := h.Renderer.Partial(w, "rules_content", data); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", id), http.StatusSeeOther)
}

func (h *Handler) HandlePlayerAidUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	// Verify game ownership before accepting the upload.
	if _, err := h.Store.GetGame(id, userID); err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPlayerAidUploadBytes)
	if err := r.ParseMultipartForm(maxPlayerAidUploadBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid multipart upload", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("player_aid")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename, err := h.savePlayerAidFile(file)
	if err != nil {
		if err == errUnsupportedPlayerAidType {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	label := sanitizePlayerAidLabel(r.FormValue("label"), header.Filename)

	if _, err := h.Store.CreatePlayerAid(id, filename, label); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		aids, err := h.Store.GetPlayerAids(id)
		if err != nil {
			slog.Error("GetPlayerAids", "gameID", id, "error", err)
		}
		if err := h.Renderer.Partial(w, "player_aids_list", viewmodel.PlayerAidsListData{GameID: id, Aids: aids}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", id), http.StatusSeeOther)
}

func (h *Handler) HandlePlayerAidDelete(w http.ResponseWriter, r *http.Request) {
	aidID, err := parseAidID(r)
	if err != nil {
		http.Error(w, "invalid aid id", http.StatusBadRequest)
		return
	}
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	aid, err := h.Store.GetPlayerAid(aidID)
	if err != nil {
		http.Error(w, "player aid not found", http.StatusNotFound)
		return
	}
	// Verify the parent game belongs to this user.
	if _, err := h.Store.GetGame(aid.GameID, userID); err != nil {
		http.Error(w, "player aid not found", http.StatusNotFound)
		return
	}

	_ = os.Remove(filepath.Join(h.DataDir, "uploads", aid.Filename))

	if err := h.Store.DeletePlayerAid(aidID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		aids, err := h.Store.GetPlayerAids(aid.GameID)
		if err != nil {
			slog.Error("GetPlayerAids", "gameID", aid.GameID, "error", err)
		}
		if err := h.Renderer.Partial(w, "player_aids_list", viewmodel.PlayerAidsListData{GameID: aid.GameID, Aids: aids}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", aid.GameID), http.StatusSeeOther)
}
