package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"myboardgamecollection/internal/httpx"
	"myboardgamecollection/internal/store"
)

func ensureDataDirs(dataDir string) {
	_ = os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755)
	_ = os.MkdirAll(filepath.Join(dataDir, "images"), 0o755)
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           httpx.Chain(handler, httpx.SecurityHeaders()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func startServer(server *http.Server, port string) {
	slog.Info("listening", "addr", "http://localhost:"+port)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()
}

func startMaintenance(ctx context.Context, s *store.Store, limiter *httpx.LoginLimiter) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.DeleteExpiredSessions(); err != nil {
					slog.Warn("session cleanup failed", "error", err)
				}
				limiter.Cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func shutdownServer(server *http.Server) {
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("stopped")
}
