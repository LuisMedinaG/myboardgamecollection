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

	if err := initDB(dbPath); err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := seedIfEmpty(); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Routes
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /games", handleGames)
	mux.HandleFunc("GET /games/new", handleGameNew)
	mux.HandleFunc("POST /games", handleGameCreate)
	mux.HandleFunc("GET /games/{id}", handleGameDetail)
	mux.HandleFunc("GET /games/{id}/edit", handleGameEdit)
	mux.HandleFunc("POST /games/{id}", handleGameUpdate)
	mux.HandleFunc("POST /games/{id}/delete", handleGameDelete)

	log.Printf("Listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
