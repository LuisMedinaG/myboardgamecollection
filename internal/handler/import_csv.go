package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/viewmodel"
)

// maxCSVUploadBytes caps CSV uploads. A BGG export is ~50 columns × ~200 bytes,
// so 5 MB comfortably fits even a 5,000-game collection.
const maxCSVUploadBytes = 5 << 20

// maxCSVPreviewRows is how many rows we display in the preview table. Anything
// larger is summarised but still imported in full when confirmed.
const maxCSVPreviewRows = 50

// HandleImportCSVPage renders the CSV upload page with instructions.
func (h *Handler) HandleImportCSVPage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	bggUsername, _ := h.Store.GetBGGUsername(userID)
	data := viewmodel.ImportCSVPageData{
		Enabled:     h.BGG != nil,
		BGGUsername: bggUsername,
	}
	if err := h.renderPage(w, r, "import_csv", "Import from CSV", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

// HandleImportCSVPreview parses an uploaded CSV and returns a preview partial
// listing the games that will be imported. No games are written to the database
// at this stage — the user must explicitly confirm via the import endpoint.
func (h *Handler) HandleImportCSVPreview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	rows, err := readUploadedCSV(w, r)
	if err != nil {
		renderCSVPreviewError(w, h, err.Error())
		return
	}
	if len(rows.Rows) == 0 {
		renderCSVPreviewError(w, h, "No valid games found in the CSV. Make sure the file has an \"objectid\" column.")
		return
	}

	owned, err := h.Store.OwnedBGGIDs(userID)
	if err != nil {
		slog.Error("OwnedBGGIDs (csv preview)", "userID", userID, "error", err)
		renderCSVPreviewError(w, h, "Could not load your existing collection. Try again.")
		return
	}

	data := viewmodel.ImportCSVPreviewData{
		TotalParsed: len(rows.Rows),
		SkippedRows: rows.SkippedRows,
	}
	var idsToImport []string
	for _, row := range rows.Rows {
		entry := viewmodel.CSVPreviewRow{
			BGGID:        row.BGGID,
			Name:         row.Name,
			AlreadyOwned: owned[row.BGGID],
		}
		if entry.Name == "" {
			entry.Name = fmt.Sprintf("BGG #%d", row.BGGID)
		}
		if entry.AlreadyOwned {
			data.OwnedCount++
		} else {
			data.NewCount++
			idsToImport = append(idsToImport, strconv.FormatInt(row.BGGID, 10))
		}
		if len(data.Rows) < maxCSVPreviewRows {
			data.Rows = append(data.Rows, entry)
		}
	}
	data.ImportIDsCSV = strings.Join(idsToImport, ",")

	if err := h.Renderer.Partial(w, "import_csv_preview", data); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}

// HandleImportCSV imports the BGG IDs the user confirmed from the preview step.
// The list is submitted as a hidden form field rather than re-uploading the CSV
// so the server never has to persist the file between requests.
func (h *Handler) HandleImportCSV(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.BGG == nil {
		renderCSVResultError(w, h, "BGG import is not configured on this server.")
		return
	}

	idsRaw := strings.TrimSpace(r.FormValue("bgg_ids"))
	if idsRaw == "" {
		renderCSVResultError(w, h, "No games selected to import. Upload a CSV and preview first.")
		return
	}

	ids := parseIDList(idsRaw)
	if len(ids) == 0 {
		renderCSVResultError(w, h, "Could not parse the list of games to import.")
		return
	}

	added, skipped, failed, err := h.BGG.ImportByBGGIDs(r.Context(), h.Store, ids, userID)
	if err != nil {
		renderCSVResultError(w, h, fmt.Sprintf("Import failed: %v", err))
		return
	}

	if err := h.Renderer.Partial(w, "import_csv_result", viewmodel.ImportCSVResultData{
		Added:   added,
		Skipped: skipped,
		Failed:  failed,
	}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}

// readUploadedCSV enforces the upload size cap, pulls the "csv_file" form file
// out of the multipart request, and runs it through the BGG CSV parser.
func readUploadedCSV(w http.ResponseWriter, r *http.Request) (bgg.CSVParseResult, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCSVUploadBytes)
	if err := r.ParseMultipartForm(maxCSVUploadBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return bgg.CSVParseResult{}, fmt.Errorf("file too large (max %d MB)", maxCSVUploadBytes>>20)
		}
		return bgg.CSVParseResult{}, fmt.Errorf("invalid upload")
	}

	file, header, err := r.FormFile("csv_file")
	if err != nil {
		return bgg.CSVParseResult{}, fmt.Errorf("no file uploaded")
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		return bgg.CSVParseResult{}, fmt.Errorf("file must have a .csv extension")
	}

	result, err := bgg.ParseCollectionCSV(file)
	if err != nil {
		return bgg.CSVParseResult{}, err
	}
	return result, nil
}

// parseIDList turns a comma-separated string of BGG IDs into a slice, dropping
// any empty/invalid entries silently. Used to read the hidden field that the
// preview partial sends back when the user confirms an import.
func parseIDList(raw string) []int64 {
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		out = append(out, id)
	}
	return out
}

func renderCSVPreviewError(w http.ResponseWriter, h *Handler, msg string) {
	if err := h.Renderer.Partial(w, "import_csv_preview", viewmodel.ImportCSVPreviewData{ErrMsg: msg}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}

func renderCSVResultError(w http.ResponseWriter, h *Handler, msg string) {
	if err := h.Renderer.Partial(w, "import_csv_result", viewmodel.ImportCSVResultData{ErrMsg: msg}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
}
