package main

import "fmt"

// Game represents a board game in the collection.
type Game struct {
	ID         int64
	Name       string
	Genre      string
	Subgenre   string
	MinPlayers int
	MaxPlayers int
	Playtime   int
	QuickRef   string
	RulesURL   string
}

// PlayersStr returns a formatted player count string.
func (g Game) PlayersStr() string {
	if g.MinPlayers == g.MaxPlayers {
		return fmt.Sprintf("%d", g.MinPlayers)
	}
	return fmt.Sprintf("%d–%d", g.MinPlayers, g.MaxPlayers)
}

// PlaytimeStr returns a formatted playtime string.
func (g Game) PlaytimeStr() string {
	return fmt.Sprintf("%d min", g.Playtime)
}

// GamesPageData holds data for the games list page.
type GamesPageData struct {
	Games    []Game
	Genres   []string
	Genre    string
	Players  string
	Playtime string
}
