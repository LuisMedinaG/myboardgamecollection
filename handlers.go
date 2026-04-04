package main

import (
	"bytes"
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
	if err := renderPage(w, "home", "Home", count); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
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
		if err := renderPartial(w, "games_result", data.Games); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	if err := renderPage(w, "games", "My Games", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
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
	if err := renderPage(w, "game_detail", game.Name, GameDetailData{Game: game, Aids: aids}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
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
	data := ImportPageData{
		Username: getConfig("bgg_username"),
		Enabled:  isBGGConfigured(),
	}
	if err := renderPage(w, "import", "Import Collection", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
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
		if err := renderPartial(w, "import_result", ImportResultData{Count: 0, ErrMsg: fmt.Sprintf("Import failed: %v", err)}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}

	if err := renderPartial(w, "import_result", ImportResultData{Count: count}); err != nil {
		http.Error(w, "failed to render partial", http.StatusInternalServerError)
	}
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
	if err := renderPage(w, "rules", data.Game.Name+" — Rules", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
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
		if err := renderPartial(w, "rules_content", data); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
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

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	ext, ok := allowedImageExtension(contentType)
	if !ok {
		http.Error(w, "unsupported file type; upload PNG, JPEG, GIF, or WebP", http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	reader := io.MultiReader(bytes.NewReader(buffer[:n]), file)

	// Generate unique filename using the detected content type.
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

	if _, err := io.Copy(dst, reader); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	if _, err := createPlayerAid(id, filename, label); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isHTMX(r) {
		aids, _ := getPlayerAids(id)
		if err := renderPartial(w, "player_aids_list", PlayerAidsListData{GameID: id, Aids: aids}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
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
		if err := renderPartial(w, "player_aids_list", PlayerAidsListData{GameID: aid.GameID, Aids: aids}); err != nil {
			http.Error(w, "failed to render partial", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d/rules", aid.GameID), http.StatusSeeOther)
}

// Discover

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	vibeIDStr := r.URL.Query().Get("vibe")

	// Step 1: no vibe selected — show vibe grid
	if vibeIDStr == "" {
		vibes, _ := allVibes()
		data := DiscoverPageData{Vibes: vibes}
		if isHTMX(r) {
			if err := renderPartial(w, "discover_result", data); err != nil {
				http.Error(w, "render error", http.StatusInternalServerError)
			}
			return
		}
		if err := renderPage(w, "discover", "Discover", data); err != nil {
			http.Error(w, "render error", http.StatusInternalServerError)
		}
		return
	}

	vibeID, err := strconv.ParseInt(vibeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid vibe", http.StatusBadRequest)
		return
	}

	vibe, err := getVibe(vibeID)
	if err != nil {
		http.Error(w, "vibe not found", http.StatusNotFound)
		return
	}

	typ := r.URL.Query().Get("type")
	category := r.URL.Query().Get("category")
	mechanic := r.URL.Query().Get("mechanic")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")

	games, err := filterGamesByVibe(vibeID, typ, category, mechanic, players, playtime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DiscoverPageData{
		VibeID:         vibeID,
		VibeName:       vibe.Name,
		Games:          games,
		Types:          typesForGames(games),
		Categories:     categoriesForGames(games),
		Mechanics:      mechanicsForGames(games),
		Type:           typ,
		Category:       category,
		Mechanic:       mechanic,
		Players:        players,
		Playtime:       playtime,
		ValidPlayers:   validPlayerOptions(games),
		ValidPlaytimes: validPlaytimeOptions(games),
	}

	if isHTMX(r) {
		if err := renderPartial(w, "discover_result", data); err != nil {
			http.Error(w, "render error", http.StatusInternalServerError)
		}
		return
	}
	if err := renderPage(w, "discover", "Discover — "+vibe.Name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// Game edit (vibe tagging)

func handleGameEdit(w http.ResponseWriter, r *http.Request) {
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
	vibes, _ := allVibes()
	gameVibeList, _ := vibesForGame(id)
	gvMap := make(map[int64]bool)
	for _, v := range gameVibeList {
		gvMap[v.ID] = true
	}
	data := GameEditData{Game: game, AllVibes: vibes, GameVibes: gvMap}
	if err := renderPage(w, "game_edit", "Edit — "+game.Name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func handleGameVibesSave(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var vibeIDs []int64
	for _, v := range r.Form["vibes"] {
		vid, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			vibeIDs = append(vibeIDs, vid)
		}
	}
	if err := setGameVibes(id, vibeIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d", id), http.StatusSeeOther)
}

// Vibe management

func handleVibes(w http.ResponseWriter, r *http.Request) {
	vibes, _ := allVibes()
	if err := renderPage(w, "vibes", "Manage Vibes", VibesPageData{Vibes: vibes}); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func handleVibeCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if _, err := createVibe(name); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			renderVibesWithError(w, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func handleVibeUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := updateVibe(id, name); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			renderVibesWithError(w, fmt.Sprintf("A vibe named %q already exists.", name))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func renderVibesWithError(w http.ResponseWriter, errMsg string) {
	vibes, _ := allVibes()
	renderPage(w, "vibes", "Manage Vibes", VibesPageData{Vibes: vibes, Error: errMsg})
}

func handleVibeDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := deleteVibe(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/vibes", http.StatusSeeOther)
}

func validPlayerOptions(games []Game) []PlayerOption {
	all := []struct {
		value string
		label string
		match func(Game) bool
	}{
		{"1", "Solo", func(g Game) bool { return g.MinPlayers <= 1 }},
		{"2", "Up to 2", func(g Game) bool { return g.MinPlayers <= 2 }},
		{"3", "Up to 3", func(g Game) bool { return g.MinPlayers <= 3 }},
		{"4", "Up to 4", func(g Game) bool { return g.MinPlayers <= 4 }},
		{"5plus", "5+", func(g Game) bool { return g.MaxPlayers >= 5 }},
	}
	var opts []PlayerOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, PlayerOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

func validPlaytimeOptions(games []Game) []PlaytimeOption {
	all := []struct {
		value string
		label string
		match func(Game) bool
	}{
		{"short", "< 30 min", func(g Game) bool { return g.PlayTime < 30 }},
		{"medium", "30–60 min", func(g Game) bool { return g.PlayTime >= 30 && g.PlayTime <= 60 }},
		{"long", "> 60 min", func(g Game) bool { return g.PlayTime > 60 }},
	}
	var opts []PlaytimeOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, PlaytimeOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

func allowedImageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
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
