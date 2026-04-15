package main

import (
	"log/slog"
	"os"

	"myboardgamecollection/internal/bgg"

	"github.com/joho/godotenv"
)

type appConfig struct {
	Port          string
	DBPath        string
	DataDir       string
	SessionSecret string
}

func loadOptionalDotEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		return godotenv.Load()
	}
	return nil
}

func loadConfig() appConfig {
	cfg := appConfig{
		Port:          envOrDefault("PORT", "8080"),
		DBPath:        envOrDefault("DB_PATH", "games.db"),
		DataDir:       envOrDefault("DATA_DIR", "data"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
	}
	if cfg.SessionSecret == "" {
		slog.Warn("SESSION_SECRET is not set; using an insecure default; set it in production")
		cfg.SessionSecret = "dev-secret-change-me-in-production"
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func newBGGClientFromEnv() *bgg.Client {
	if token := os.Getenv("BGG_TOKEN"); token != "" {
		slog.Info("BGG auth: using token")
		return bgg.New(token)
	}
	if cookie := os.Getenv("BGG_COOKIE"); cookie != "" {
		slog.Info("BGG auth: using cookie")
		return bgg.NewWithCookies(cookie)
	}
	return nil
}
