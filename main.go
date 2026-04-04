package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static
var staticFiles embed.FS

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	dbPath := "games.db"
	if p := os.Getenv("DB_PATH"); p != "" {
		dbPath = p
	}

	// BGG token is optional — app works without it (seed data + manual use)
	if token := os.Getenv("BGG_TOKEN"); token != "" {
		initBGG(token)
	}

	initTemplates()

	if err := initDB(dbPath); err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := seedIfEmpty(); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	// Ensure uploads directory
	_ = os.MkdirAll("data/uploads", 0o755)

	mux := http.NewServeMux()

	// Static files (embedded)
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Uploaded files (on disk)
	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir("data/uploads")))
	mux.Handle("GET /uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		uploads.ServeHTTP(w, r)
	}))

	// Routes
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /games", handleGames)
	mux.HandleFunc("GET /games/{id}", handleGameDetail)
	mux.HandleFunc("POST /games/{id}/delete", handleGameDelete)

	// Rules
	mux.HandleFunc("GET /games/{id}/rules", handleRules)
	mux.HandleFunc("POST /games/{id}/rules/url", handleRulesURLUpdate)
	mux.HandleFunc("POST /games/{id}/rules/upload", handlePlayerAidUpload)
	mux.HandleFunc("POST /games/{id}/rules/aids/{aid_id}/delete", handlePlayerAidDelete)

	// BGG Import
	mux.HandleFunc("GET /import", handleImport)
	mux.HandleFunc("POST /import", handleImportSync)

	log.Printf("Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
