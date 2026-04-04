package handler

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// imageHTTPClient has a finite timeout so a slow upstream never hangs a request.
var imageHTTPClient = &http.Client{Timeout: 10 * time.Second}

// HandleImage serves a game thumbnail from a local disk cache.
// On cache miss it downloads from the BGG URL stored in the database,
// saves the result atomically, then serves the cached file.
// Route: GET /images/{bgg_id}
func (h *Handler) HandleImage(w http.ResponseWriter, r *http.Request) {
	bggID, err := strconv.ParseInt(r.PathValue("bgg_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cachePath := filepath.Join("data", "images", strconv.FormatInt(bggID, 10))

	// Serve from cache when available.
	if f, err := os.Open(cachePath); err == nil {
		defer f.Close()
		serveImageFile(w, r, f)
		return
	}

	// Cache miss — look up the upstream URL in the database.
	game, err := h.Store.GetGameByBGGID(bggID)
	if err != nil || game.Thumbnail == "" {
		http.NotFound(w, r)
		return
	}

	// Download the image.
	resp, err := imageHTTPClient.Get(game.Thumbnail)
	if err != nil || resp.StatusCode != http.StatusOK {
		if err == nil {
			resp.Body.Close()
		}
		http.NotFound(w, r)
		return
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Join("data", "images"), 0o755); err != nil {
		slog.Error("image cache mkdir", "error", err)
		http.NotFound(w, r)
		return
	}

	// Write to a temp file and rename atomically so concurrent requests never
	// see a partially-written cache file.
	tmp := cachePath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		slog.Error("image cache create", "error", err)
		http.NotFound(w, r)
		return
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		slog.Error("image cache write", "error", err)
		http.NotFound(w, r)
		return
	}
	f.Close()

	// os.Rename is atomic on POSIX: a concurrent rename just means one writer
	// wins and the other's file is overwritten — both are identical content.
	if err := os.Rename(tmp, cachePath); err != nil {
		os.Remove(tmp)
		slog.Error("image cache rename", "error", err)
		http.NotFound(w, r)
		return
	}

	// Serve the freshly cached file.
	cached, err := os.Open(cachePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer cached.Close()
	serveImageFile(w, r, cached)
}

// serveImageFile detects the content type from the first 512 bytes,
// then uses http.ServeContent for range/ETag/If-Modified-Since support.
func serveImageFile(w http.ResponseWriter, r *http.Request, f *os.File) {
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ct := http.DetectContentType(buf[:n])

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400") // browsers cache for 1 day
	http.ServeContent(w, r, "", fi.ModTime(), f)
}
