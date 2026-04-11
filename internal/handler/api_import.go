package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"myboardgamecollection/internal/httpx"
)

// HandleAPIImportSync triggers a BGG collection sync for the authenticated user.
//
// POST /api/v1/import/sync
// Body: {"full_refresh": bool}
func (h *Handler) HandleAPIImportSync(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		FullRefresh bool `json:"full_refresh"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	isAdmin := httpx.IsAdminFromContext(r.Context())
	limit := syncLimitRegular
	if isAdmin {
		limit = syncLimitAdmin
	}

	canSync, err := h.Store.CanSync(userID, limit)
	if err != nil {
		slog.Error("HandleAPIImportSync: CanSync", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !canSync {
		writeAPIError(w, http.StatusTooManyRequests, fmt.Sprintf("sync limit reached (%d per day)", limit))
		return
	}

	bggUsername, err := h.Store.GetBGGUsername(userID)
	if err != nil {
		slog.Error("HandleAPIImportSync: GetBGGUsername", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if bggUsername == "" {
		writeAPIError(w, http.StatusUnprocessableEntity, "BGG username not set")
		return
	}

	if h.BGG == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "BGG import is not configured")
		return
	}

	fullRefresh := isAdmin && body.FullRefresh

	added, updated, total, err := h.BGG.ImportCollection(r.Context(), h.Store, bggUsername, userID, fullRefresh)
	if err != nil {
		slog.Error("HandleAPIImportSync: ImportCollection", "error", err)
		writeAPIError(w, http.StatusBadGateway, "BGG import failed")
		return
	}

	if err := h.Store.RecordSync(userID); err != nil {
		slog.Error("HandleAPIImportSync: RecordSync", "error", err)
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"added":   added,
		"updated": updated,
		"total":   total,
	})
}

// HandleAPIImportCSVPreview parses an uploaded CSV and returns a preview of the
// games it contains, flagging which ones are already in the user's collection.
//
// POST /api/v1/import/csv/preview
// Multipart: field "csv_file" (max 5 MB)
func (h *Handler) HandleAPIImportCSVPreview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	result, err := readUploadedCSV(w, r)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "too large"):
			writeAPIError(w, http.StatusRequestEntityTooLarge, msg)
		case strings.Contains(msg, "no file"):
			writeAPIError(w, http.StatusBadRequest, msg)
		default:
			writeAPIError(w, http.StatusUnprocessableEntity, msg)
		}
		return
	}

	if len(result.Rows) == 0 {
		writeAPIError(w, http.StatusUnprocessableEntity, "no valid games found in CSV")
		return
	}

	owned, err := h.Store.OwnedBGGIDs(userID)
	if err != nil {
		slog.Error("HandleAPIImportCSVPreview: OwnedBGGIDs", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "internal error")
		return
	}

	limit := maxCSVPreviewRows
	if len(result.Rows) < limit {
		limit = len(result.Rows)
	}

	rows := make([]map[string]any, limit)
	for i := 0; i < limit; i++ {
		row := result.Rows[i]
		rows[i] = map[string]any{
			"bgg_id":        row.BGGID,
			"name":          row.Name,
			"already_owned": owned[row.BGGID],
		}
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"rows":        rows,
		"total_rows":  len(result.Rows),
		"preview_limit": maxCSVPreviewRows,
	})
}

// HandleAPIImportCSV imports games by their BGG IDs.
//
// POST /api/v1/import/csv
// Body: {"bgg_ids": [123, 456]}
func (h *Handler) HandleAPIImportCSV(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireAPIUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		BGGIDs []int64 `json:"bgg_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.BGGIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, "bgg_ids required")
		return
	}

	if h.BGG == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "BGG import is not configured")
		return
	}

	userID, _ := httpx.UserIDFromContext(r.Context())

	added, _, failed, err := h.BGG.ImportByBGGIDs(r.Context(), h.Store, body.BGGIDs, userID)
	if err != nil {
		slog.Error("HandleAPIImportCSV: ImportByBGGIDs", "error", err)
		writeAPIError(w, http.StatusBadGateway, "BGG import failed")
		return
	}

	writeAPIData(w, http.StatusOK, map[string]any{
		"imported": added,
		"failed":   failed,
	})
}
