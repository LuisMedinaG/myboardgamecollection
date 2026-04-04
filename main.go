package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/handler"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFS embed.FS

func main() {
	// Use JSON structured logging; easy to query on Fly.io / any log aggregator.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	dbPath := "games.db"
	if p := os.Getenv("DB_PATH"); p != "" {
		dbPath = p
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if (adminUsername == "") != (adminPassword == "") {
		slog.Error("ADMIN_USERNAME and ADMIN_PASSWORD must either both be set or both be empty")
		os.Exit(1)
	}

	// Initialize store (database).
	s, err := store.New(dbPath)
	if err != nil {
		slog.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	if err := s.SeedIfEmpty(); err != nil {
		slog.Warn("seed failed", "error", err)
	}
	if err := s.PopulateTaxonomy(); err != nil {
		slog.Warn("taxonomy migration failed", "error", err)
	}

	// Initialize BGG client (optional): token takes priority, then cookie.
	var bc *bgg.Client
	if token := os.Getenv("BGG_TOKEN"); token != "" {
		bc = bgg.New(token)
		slog.Info("BGG auth: using token")
	} else if cookie := os.Getenv("BGG_COOKIE"); cookie != "" {
		bc = bgg.NewWithCookies(cookie)
		slog.Info("BGG auth: using cookie")
	}

	// Initialize renderer and handler.
	ren := render.New(templateFS)
	h := &handler.Handler{Store: s, Renderer: ren, BGG: bc}

	// Ensure data directories.
	_ = os.MkdirAll("data/uploads", 0o755)
	_ = os.MkdirAll("data/images", 0o755)

	mux := http.NewServeMux()
	adminGET := func(hf http.HandlerFunc) http.Handler {
		return httpx.Chain(http.HandlerFunc(hf), httpx.MethodGuard(http.MethodGet), httpx.AdminAuth(adminUsername, adminPassword))
	}
	adminPOST := func(hf http.HandlerFunc) http.Handler {
		return httpx.Chain(http.HandlerFunc(hf), httpx.MethodGuard(http.MethodPost), httpx.AdminAuth(adminUsername, adminPassword), httpx.SameOrigin())
	}

	// Static files (embedded).
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Uploaded files (on disk).
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir("data/uploads")))
	mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		uploads.ServeHTTP(w, r)
	}))

	// Routes.
	mux.HandleFunc("GET /{$}", h.HandleHome)
	mux.HandleFunc("GET /games", h.HandleGames)
	mux.HandleFunc("GET /games/{id}", h.HandleGameDetail)
	mux.Handle("POST /games/bulk-vibes", adminPOST(h.HandleBulkVibeAssign))
	mux.Handle("POST /games/{id}/delete", adminPOST(h.HandleGameDelete))

	mux.Handle("GET /games/{id}/edit", adminGET(h.HandleGameEdit))
	mux.Handle("POST /games/{id}/vibes", adminPOST(h.HandleGameVibesSave))

	mux.HandleFunc("GET /discover", h.HandleDiscover)

	mux.Handle("GET /vibes", adminGET(h.HandleVibes))
	mux.Handle("POST /vibes", adminPOST(h.HandleVibeCreate))
	mux.Handle("POST /vibes/batch-update", adminPOST(h.HandleVibeBatchUpdate))
	mux.Handle("POST /vibes/{id}", adminPOST(h.HandleVibeUpdate))
	mux.Handle("POST /vibes/{id}/delete", adminPOST(h.HandleVibeDelete))

	mux.HandleFunc("GET /games/{id}/rules", h.HandleRules)
	mux.Handle("POST /games/{id}/rules/url", adminPOST(h.HandleRulesURLUpdate))
	mux.Handle("POST /games/{id}/rules/upload", adminPOST(h.HandlePlayerAidUpload))
	mux.Handle("POST /games/{id}/rules/aids/{aid_id}/delete", adminPOST(h.HandlePlayerAidDelete))

	mux.Handle("GET /import", adminGET(h.HandleImport))
	mux.Handle("POST /import", adminPOST(h.HandleImportSync))

	mux.HandleFunc("GET /images/{bgg_id}", h.HandleImage)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpx.Chain(mux, httpx.SecurityHeaders()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// Run the server in a goroutine so we can wait for a shutdown signal.
	slog.Info("listening", "addr", "http://localhost:"+port)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until SIGINT or SIGTERM is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("stopped")
}
