package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"myboardgamecollection/internal/bgg"
	"myboardgamecollection/internal/handler"
	"myboardgamecollection/internal/render"
	"myboardgamecollection/internal/store"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFS embed.FS

func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	dbPath := "games.db"
	if p := os.Getenv("DB_PATH"); p != "" {
		dbPath = p
	}

	// Initialize store (database).
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer s.Close()

	if err := s.SeedIfEmpty(); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	// Initialize BGG client (optional).
	var bc *bgg.Client
	if token := os.Getenv("BGG_TOKEN"); token != "" {
		bc = bgg.New(token)
	}

	// Initialize renderer and handler.
	ren := render.New(templateFS)
	h := &handler.Handler{Store: s, Renderer: ren, BGG: bc}

	// Ensure uploads directory.
	_ = os.MkdirAll("data/uploads", 0o755)

	mux := http.NewServeMux()

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
	mux.HandleFunc("POST /games/{id}/delete", h.HandleGameDelete)

	mux.HandleFunc("GET /games/{id}/edit", h.HandleGameEdit)
	mux.HandleFunc("POST /games/{id}/vibes", h.HandleGameVibesSave)

	mux.HandleFunc("GET /discover", h.HandleDiscover)

	mux.HandleFunc("GET /vibes", h.HandleVibes)
	mux.HandleFunc("POST /vibes", h.HandleVibeCreate)
	mux.HandleFunc("POST /vibes/{id}", h.HandleVibeUpdate)
	mux.HandleFunc("POST /vibes/{id}/delete", h.HandleVibeDelete)

	mux.HandleFunc("GET /games/{id}/rules", h.HandleRules)
	mux.HandleFunc("POST /games/{id}/rules/url", h.HandleRulesURLUpdate)
	mux.HandleFunc("POST /games/{id}/rules/upload", h.HandlePlayerAidUpload)
	mux.HandleFunc("POST /games/{id}/rules/aids/{aid_id}/delete", h.HandlePlayerAidDelete)

	mux.HandleFunc("GET /import", h.HandleImport)
	mux.HandleFunc("POST /import", h.HandleImportSync)

	log.Printf("Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
