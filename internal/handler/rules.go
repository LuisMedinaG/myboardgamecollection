package handler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"myboardgamecollection/internal/viewmodel"
)

const maxPlayerAidUploadBytes = 10 << 20

func (h *Handler) HandleRules(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	game, err := h.Store.GetGame(id)
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
	if err := h.Renderer.Page(w, "rules", game.Name+" — Rules", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleRulesURLUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r)
	if !ok {
		return
	}
	rulesURL := strings.TrimSpace(r.FormValue("rules_url"))
	if err := validateRulesURL(rulesURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateGameRulesURL(id, rulesURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		game, err := h.Store.GetGame(id)
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

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	ext, ok := allowedImageExtension(contentType)
	if !ok {
		http.Error(w, "unsupported file type; upload PNG, JPEG, GIF, or WebP", http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	reader := io.MultiReader(bytes.NewReader(buffer[:n]), file)

	filename := fmt.Sprintf("game_%d_%d%s", id, time.Now().UnixMilli(), ext)

	uploadDir := filepath.Join("data", "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, reader); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

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

	aid, err := h.Store.GetPlayerAid(aidID)
	if err != nil {
		http.Error(w, "player aid not found", http.StatusNotFound)
		return
	}

	_ = os.Remove(filepath.Join("data", "uploads", aid.Filename))

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
