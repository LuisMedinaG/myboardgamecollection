package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	canSync, err := h.Store.CanSync(userID)
	if err != nil {
		slog.Error("CanSync", "userID", userID, "error", err)
	}
	data := viewmodel.ImportPageData{
		Username: httpx.UsernameFromContext(r.Context()),
		Enabled:  h.BGG != nil,
		CanSync:  canSync,
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

	canSync, err := h.Store.CanSync(userID)
	if err != nil {
		slog.Error("CanSync", "userID", userID, "error", err)
	}
	if !canSync {
		renderImportError(w, r, h, "You have already synced today. Please try again tomorrow.")
		return
	}

	username := httpx.UsernameFromContext(r.Context())
	added, updated, collCount, err := h.BGG.ImportCollection(r.Context(), h.Store, username, userID)
	if err != nil {
		renderImportError(w, r, h, fmt.Sprintf("Import failed: %v", err))
		return
	}

	if err := h.Store.RecordSync(userID); err != nil {
		slog.Error("RecordSync", "userID", userID, "error", err)
	}

	if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{
		Count: added, Updated: updated, CollectionItems: collCount, Username: username,
	}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}

func renderImportError(w http.ResponseWriter, r *http.Request, h *Handler, msg string) {
	if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{ErrMsg: msg}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}
