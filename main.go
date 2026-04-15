package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myboardgamecollection/internal/handler"
	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/service"
	"myboardgamecollection/internal/store"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := loadOptionalDotEnv(); err != nil {
		slog.Error("load .env failed", "error", err)
		os.Exit(1)
	}

	cfg := loadConfig()
	sessionKey := []byte(cfg.SessionSecret)

	s, err := store.New(cfg.DBPath)
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

	bc := newBGGClientFromEnv()
	loginLimiter := httpx.NewLoginLimiter(10, 15*time.Minute)

	ren := render.New(templateFS)
	gameService := service.NewGameService(s)
	gameHandler := handler.NewGameHandler(gameService, s, ren)
	h := &handler.Handler{Store: s, Renderer: ren, BGG: bc, DataDir: cfg.DataDir, LoginLimiter: loginLimiter, JWTSecret: cfg.SessionSecret}
	ensureDataDirs(cfg.DataDir)

	mux := http.NewServeMux()
	registerRoutes(mux, routeDeps{
		store:         s,
		legacy:        h,
		games:         gameHandler,
		loginLimiter:  loginLimiter,
		sessionSecret: cfg.SessionSecret,
		sessionKey:    sessionKey,
		dataDir:       cfg.DataDir,
		staticFiles:   staticFiles,
	})

	server := newHTTPServer(cfg.Port, mux)
	startServer(server, cfg.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startMaintenance(ctx, s, loginLimiter)

	<-ctx.Done()
	shutdownServer(server)
}
