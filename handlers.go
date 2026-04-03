package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const newItemSentinel = "__new__"

func formDropdowns() (genres, subgenres []string) {
	genres, _ = distinctGenres()
	subgenres, _ = distinctSubgenres()
	return
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func formatSubgenre(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		parts[i] = capitalize(p)
	}
	return strings.Join(parts, " ")
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func parseGameForm(r *http.Request) (Game, error) {
	if err := r.ParseForm(); err != nil {
		return Game{}, err
	}

	minP, _ := strconv.Atoi(r.FormValue("min_players"))
	maxP, _ := strconv.Atoi(r.FormValue("max_players"))
	playtime, _ := strconv.Atoi(r.FormValue("playtime"))

	if minP < 1 {
		minP = 1
	}
	if maxP < minP {
		maxP = minP
	}
	if playtime < 1 {
		playtime = 30
	}

	genre := strings.TrimSpace(strings.ToLower(r.FormValue("genre")))
	if genre == newItemSentinel {
		genre = strings.TrimSpace(strings.ToLower(r.FormValue("genre_new")))
	}

	subgenre := strings.TrimSpace(strings.ToLower(r.FormValue("subgenre")))
	if subgenre == newItemSentinel {
		subgenre = strings.TrimSpace(strings.ToLower(r.FormValue("subgenre_new")))
	}

	return Game{
		Name:       strings.TrimSpace(r.FormValue("name")),
		Genre:      genre,
		Subgenre:   subgenre,
		MinPlayers: minP,
		MaxPlayers: maxP,
		Playtime:   playtime,
		QuickRef:   strings.TrimSpace(r.FormValue("quickref")),
		RulesURL:   strings.TrimSpace(r.FormValue("rules_url")),
	}, nil
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Handlers

func handleHome(w http.ResponseWriter, r *http.Request) {
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	home(count).Render(r.Context(), w)
}

func handleGames(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	players := r.URL.Query().Get("players")
	playtime := r.URL.Query().Get("playtime")

	games, err := filterGames(genre, players, playtime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	genres, _ := distinctGenres()

	data := GamesPageData{
		Games:    games,
		Genres:   genres,
		Genre:    genre,
		Players:  players,
		Playtime: playtime,
	}

	if isHTMX(r) {
		gamesResult(data.Games).Render(r.Context(), w)
		return
	}
	gamesPage(data).Render(r.Context(), w)
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
	gameDetail(game).Render(r.Context(), w)
}

func handleGameNew(w http.ResponseWriter, r *http.Request) {
	genres, subgenres := formDropdowns()
	gameForm(Game{MinPlayers: 2, MaxPlayers: 4, Playtime: 30}, genres, subgenres, false).Render(r.Context(), w)
}

func handleGameCreate(w http.ResponseWriter, r *http.Request) {
	g, err := parseGameForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if g.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	id, err := createGame(g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d", id), http.StatusSeeOther)
}

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
	genres, subgenres := formDropdowns()
	gameForm(game, genres, subgenres, true).Render(r.Context(), w)
}

func handleGameUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	g, err := parseGameForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if g.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	g.ID = id

	if err := updateGame(g); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/games/%d", id), http.StatusSeeOther)
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
