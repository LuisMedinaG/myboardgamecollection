package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"myboardgamecollection/internal/viewmodel"
)

func (h *Handler) HandleImport(w http.ResponseWriter, r *http.Request) {
	data := viewmodel.ImportPageData{
		Username: h.Store.GetConfig("bgg_username"),
		Enabled:  h.BGG != nil,
	}
	if err := h.Renderer.Page(w, "import", "Import Collection", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handler) HandleImportSync(w http.ResponseWriter, r *http.Request) {
	if h.BGG == nil {
		if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{ErrMsg: "BGG import is not configured on this server."}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	if err := h.Store.SetConfig("bgg_username", username); err != nil {
		slog.Error("SetConfig bgg_username", "error", err)
	}

	added, updated, err := h.BGG.ImportCollection(r.Context(), h.Store, username)
	if err != nil {
		if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{ErrMsg: fmt.Sprintf("Import failed: %v", err)}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}

	if err := h.Renderer.Partial(w, "import_result", viewmodel.ImportResultData{Count: added, Updated: updated}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}
