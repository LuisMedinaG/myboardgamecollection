package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Handlers

func handleHome(w http.ResponseWriter, r *http.Request) {
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	renderPage(w, "home", "Home", count)
}

func handleGames(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")

	games, err := filterGames(category, players, playtime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, _ := distinctCategories()

	data := GamesPageData{
		Games:      games,
		Categories: categories,
		Category:   category,
		Players:    players,
		Playtime:   playtime,
	}

	if isHTMX(r) {
		renderPartial(w, "games_result", data.Games)
		return
	}
	renderPage(w, "games", "My Games", data)
}

func handleGameDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	game, err := getGame(id)
	if err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	aids, _ := getPlayerAids(id)
	renderPage(w, "game_detail", game.Name, GameDetailData{Game: game, Aids: aids})
}

func handleGameDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := deleteGame(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/games")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/games", http.StatusSeeOther)
}

// BGG import

func handleImport(w http.ResponseWriter, r *http.Request) {
	username := getConfig("bgg_username")
	renderPage(w, "import", "Import Collection", username)
}

func handleImportSync(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	_ = setConfig("bgg_username", username)

	count, err := importBGGCollection(r.Context(), username)
	if err != nil {
		renderPartial(w, "import_result", ImportResultData{Count: 0, ErrMsg: fmt.Sprintf("Import failed: %v", err)})
		return
	}

	renderPartial(w, "import_result", ImportResultData{Count: count})
}

// Rules

func handleRules(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	game, err := getGame(id)
	if err != nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	aids, _ := getPlayerAids(id)

	data := RulesPageData{
		Game:       game,
		PlayerAids: aids,
		EmbedURL:   driveEmbedURL(game.RulesURL),
	}
	renderPage(w, "rules", data.Game.Name+" — Rules", data)
}

func handleRulesURLUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	rulesURL := strings.TrimSpace(r.FormValue("rules_url"))
	if err := updateGameRulesURL(id, rulesURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		game, _ := getGame(id)
		aids, _ := getPlayerAids(id)
		data := RulesPageData{
			Game:       game,
			PlayerAids: aids,
			EmbedURL:   driveEmbedURL(game.RulesURL),
		}
		renderPartial(w, "rules_content", data)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", id), http.StatusSeeOther)
}

// Player aid upload

func handlePlayerAidUpload(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("player_aid")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("game_%d_%d%s", id, time.Now().UnixMilli(), ext)

	// Save file
	uploadDir := filepath.Join("data", "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		http.Error(w, "failed to create upload directory", http.StatusInternalServerError)
		return
	}

	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	if _, err := createPlayerAid(id, filename, label); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		aids, _ := getPlayerAids(id)
		renderPartial(w, "player_aids_list", PlayerAidsListData{GameID: id, Aids: aids})
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", id), http.StatusSeeOther)
}

func handlePlayerAidDelete(w http.ResponseWriter, r *http.Request) {
	aidID, err := strconv.ParseInt(r.PathValue("aid_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid aid id", http.StatusBadRequest)
		return
	}

	aid, err := getPlayerAid(aidID)
	if err != nil {
		http.Error(w, "player aid not found", http.StatusNotFound)
		return
	}

	// Delete file
	_ = os.Remove(filepath.Join("data", "uploads", aid.Filename))

	if err := deletePlayerAid(aidID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		aids, _ := getPlayerAids(aid.GameID)
		renderPartial(w, "player_aids_list", PlayerAidsListData{GameID: aid.GameID, Aids: aids})
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", aid.GameID), http.StatusSeeOther)
}

// driveEmbedURL converts a Google Drive sharing URL to an embeddable preview URL.
var driveFileIDRegex = regexp.MustCompile(`/d/([a-zA-Z0-9_-]+)`)

func driveEmbedURL(url string) string {
	if url == "" {
		return ""
	}
	matches := driveFileIDRegex.FindStringSubmatch(url)
	if len(matches) < 2 {
		return ""
	}
	return fmt.Sprintf("https://drive.google.com/file/d/%s/preview", matches[1])
}
