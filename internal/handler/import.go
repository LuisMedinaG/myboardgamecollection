package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/viewmodel"
)

const (
	syncLimitRegular = 1
	// syncLimitAdmin is effectively unlimited — admins are trusted to bulk-test
	// imports without hitting the daily cap. CanSync still counts usage so we
	// can see admin activity, but the ceiling is high enough to never trip.
	syncLimitAdmin = 100
)

func (h *Handler) HandleImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	limit := syncLimit(r)
	isAdmin := httpx.IsAdminFromContext(r.Context())
	canSync, err := h.Store.CanSync(userID, limit)
	if err != nil {
		slog.Error("CanSync", "userID", userID, "error", err)
	}
	bggUsername, _ := h.Store.GetBGGUsername(userID)
	data := viewmodel.ImportPageData{
		Username:  bggUsername,
		Enabled:   h.BGG != nil && bggUsername != "",
		CanSync:   canSync,
		IsAdmin:   isAdmin,
		SyncLimit: limit,
	}
	if err := h.renderPage(w, r, "import", "Sync Collection", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleImportSync(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	if h.BGG == nil {
		renderImportError(w, r, h, "BGG import is not configured on this server.")
		return
	}

	limit := syncLimit(r)
	isAdmin := httpx.IsAdminFromContext(r.Context())
	canSync, err := h.Store.CanSync(userID, limit)
	if err != nil {
		slog.Error("CanSync", "userID", userID, "error", err)
	}
	if !canSync {
		msg := fmt.Sprintf("Daily sync limit reached (%d). Come back tomorrow to sync again.", limit)
		renderImportError(w, r, h, msg)
		return
	}

	bggUsername, _ := h.Store.GetBGGUsername(userID)
	if bggUsername == "" {
		renderImportError(w, r, h, "No BGG username set. Add your BoardGameGeek username in your profile to sync.")
		return
	}
	// Full refresh re-fetches metadata for every owned game. Gated to admins
	// for now; a future tier system will open it to paid users (see #51).
	fullRefresh := isAdmin && r.FormValue("full_refresh") == "1"
	added, updated, collCount, err := h.BGG.ImportCollection(r.Context(), h.Store, bggUsername, userID, fullRefresh)
	if err != nil {
		renderImportError(w, r, h, fmt.Sprintf("Import failed: %v", err))
		return
	}

	if err := h.Store.RecordSync(userID); err != nil {
		slog.Error("RecordSync", "userID", userID, "error", err)
	}

	if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{
		Count: added, Updated: updated, CollectionItems: collCount, Username: bggUsername,
	}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}

// HandleSetBGGUsername lets a logged-in user set or update their BGG username.
func (h *Handler) HandleSetBGGUsername(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	bgg := strings.TrimSpace(r.FormValue("bgg_username"))
	if err := h.Store.SetBGGUsername(userID, bgg); err != nil {
		slog.Error("SetBGGUsername", "userID", userID, "error", err)
		http.Error(w, "failed to update BGG username", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/import", http.StatusSeeOther)
}

func renderImportError(w http.ResponseWriter, r *http.Request, h *Handler, msg string) {
	if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{ErrMsg: msg}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}
