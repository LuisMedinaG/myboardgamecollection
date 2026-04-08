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
	"path/filepath"
	"syscall"
	"time"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/handler"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"

	"github.com/joho/godotenv"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFS embed.FS

func main() {
	// Use JSON structured logging; easy to query on Fly.io / any log aggregator.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			slog.Error("load .env failed", "error", err)
			os.Exit(1)
		}
	}

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	dbPath := "games.db"
	if p := os.Getenv("DB_PATH"); p != "" {
		dbPath = p
	}

	dataDir := "data"
	if p := os.Getenv("DATA_DIR"); p != "" {
		dataDir = p
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		slog.Warn("SESSION_SECRET is not set; using an insecure default — set it in production")
		sessionSecret = "dev-secret-change-me-in-production"
	}
	secret := []byte(sessionSecret)

	// Initialize store (database).
	s, err := store.New(dbPath)
	if err != nil {
		slog.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	if err := s.PopulateTaxonomy(); err != nil {
		slog.Warn("taxonomy migration failed", "error", err)
	}

	// Purge expired sessions at startup (best-effort, non-fatal).
	if err := s.DeleteExpiredSessions(); err != nil {
		slog.Warn("failed to purge expired sessions", "error", err)
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

	// Login rate limiter: 10 attempts per 15 minutes per IP.
	loginLimiter := httpx.NewLoginLimiter(10, 15*time.Minute)

	// Initialize renderer and handler.
	ren := render.New(templateFS)
	h := &handler.Handler{Store: s, Renderer: ren, BGG: bc, DataDir: dataDir, LoginLimiter: loginLimiter}

	// Ensure data directories.
	_ = os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755)
	_ = os.MkdirAll(filepath.Join(dataDir, "images"), 0o755)

	mux := http.NewServeMux()

	// Middleware helpers.
	auth := func(hf http.HandlerFunc) http.Handler {
		return httpx.Chain(hf, httpx.MethodGuard(http.MethodGet), httpx.RequireAuth(s, secret))
	}
	authPOST := func(hf http.HandlerFunc) http.Handler {
		return httpx.Chain(hf, httpx.MethodGuard(http.MethodPost), httpx.RequireAuth(s, secret), httpx.SameOrigin(), httpx.VerifyCSRF())
	}

	// Static files (embedded).
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Uploaded files (on disk).
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Join(dataDir, "uploads"))))
	mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		uploads.ServeHTTP(w, r)
	}))

	// Public routes (no auth required).
	mux.Handle("GET /login", httpx.Chain(http.HandlerFunc(h.HandleLoginPage), httpx.MethodGuard(http.MethodGet)))
	mux.Handle("POST /login", httpx.Chain(http.HandlerFunc(h.HandleLogin), httpx.MethodGuard(http.MethodPost), httpx.SameOrigin(), httpx.RateLimit(loginLimiter)))
	mux.Handle("GET /signup", httpx.Chain(http.HandlerFunc(h.HandleSignupPage), httpx.MethodGuard(http.MethodGet)))
	mux.Handle("POST /signup", httpx.Chain(http.HandlerFunc(h.HandleSignup), httpx.MethodGuard(http.MethodPost), httpx.SameOrigin(), httpx.RateLimit(loginLimiter)))
	mux.Handle("POST /logout", httpx.Chain(http.HandlerFunc(h.HandleLogout), httpx.MethodGuard(http.MethodPost), httpx.SameOrigin()))
	mux.HandleFunc("GET /images/{bgg_id}", h.HandleImage)

	// Authenticated routes.
	mux.Handle("GET /{$}", auth(h.HandleHome))
	mux.Handle("GET /games", auth(h.HandleGames))
	mux.Handle("GET /games/{id}", auth(h.HandleGameDetail))
	mux.Handle("POST /games/bulk-vibes", authPOST(h.HandleBulkVibeAssign))
	mux.Handle("POST /games/{id}/delete", authPOST(h.HandleGameDelete))

	mux.Handle("GET /games/{id}/edit", auth(h.HandleGameEdit))
	mux.Handle("POST /games/{id}/vibes", authPOST(h.HandleGameVibesSave))

	mux.Handle("GET /discover", auth(h.HandleDiscover))

	mux.Handle("GET /vibes", auth(h.HandleVibes))
	mux.Handle("POST /vibes", authPOST(h.HandleVibeCreate))
	mux.Handle("POST /vibes/batch-update", authPOST(h.HandleVibeBatchUpdate))
	mux.Handle("POST /vibes/{id}", authPOST(h.HandleVibeUpdate))
	mux.Handle("POST /vibes/{id}/delete", authPOST(h.HandleVibeDelete))

	mux.Handle("GET /games/{id}/rules", auth(h.HandleRules))
	mux.Handle("POST /games/{id}/rules/url", authPOST(h.HandleRulesURLUpdate))
	mux.Handle("POST /games/{id}/rules/upload", authPOST(h.HandlePlayerAidUpload))
	mux.Handle("POST /games/{id}/rules/aids/{aid_id}/delete", authPOST(h.HandlePlayerAidDelete))

	mux.Handle("GET /import", auth(h.HandleImport))
	mux.Handle("POST /import", authPOST(h.HandleImportSync))
	mux.Handle("POST /profile/bgg-username", authPOST(h.HandleSetBGGUsername))
	mux.Handle("GET /profile/change-password", auth(h.HandleChangePasswordPage))
	mux.Handle("POST /profile/change-password", authPOST(h.HandleChangePassword))

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

	// Periodic maintenance: purge expired sessions and stale rate-limit buckets.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.DeleteExpiredSessions(); err != nil {
					slog.Warn("session cleanup failed", "error", err)
				}
				loginLimiter.Cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Block until SIGINT or SIGTERM is received.
	<-ctx.Done()

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("stopped")
}
