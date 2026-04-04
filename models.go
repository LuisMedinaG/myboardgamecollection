package main

import (
	"fmt"
	"strings"
)

// Game represents a board game in the collection, imported from BGG.
type Game struct {
	ID            int64
	BGGID         int64
	Name          string
	Description   string
	YearPublished int
	Image         string
	Thumbnail     string
	MinPlayers    int
	MaxPlayers    int
	PlayTime      int
	Categories    string
	Mechanics     string
	Types         string // BGG subdomain (e.g. "Family Games, Strategy Games")
	RulesURL      string // Google Drive link to rulebook PDF
}

// PlayersStr returns a formatted player count string.
func (g Game) PlayersStr() string {
	if g.MinPlayers == g.MaxPlayers {
		return fmt.Sprintf("%d", g.MinPlayers)
	}
	return fmt.Sprintf("%d\u2013%d", g.MinPlayers, g.MaxPlayers)
}

// PlaytimeStr returns a formatted playtime string.
func (g Game) PlaytimeStr() string {
	if g.PlayTime >= 60 {
		h := g.PlayTime / 60
		m := g.PlayTime % 60
		if m == 0 {
			return fmt.Sprintf("%d hr", h)
		}
		return fmt.Sprintf("%d hr %d min", h, m)
	}
	return fmt.Sprintf("%d min", g.PlayTime)
}

// BGGURL returns the BoardGameGeek URL for this game.
func (g Game) BGGURL() string {
	return fmt.Sprintf("https://boardgamegeek.com/boardgame/%d", g.BGGID)
}

// Tagline returns the first sentence of the description as a one-liner.
func (g Game) Tagline() string {
	d := strings.TrimSpace(g.Description)
	if d == "" {
		return ""
	}
	if idx := strings.Index(d, ". "); idx != -1 && idx < 200 {
		return d[:idx+1]
	}
	if idx := strings.Index(d, "."); idx != -1 && idx < 200 {
		return d[:idx+1]
	}
	if len(d) > 150 {
		return d[:150] + "..."
	}
	return d
}

// PlayerAid represents an uploaded player aid image for a game.
type PlayerAid struct {
	ID       int64
	GameID   int64
	Filename string
	Label    string
}

// Vibe represents a mood/occasion tag for games.
type Vibe struct {
	ID   int64
	Name string
}

// DiscoverPageData holds data for the discover page.
type DiscoverPageData struct {
	Vibes      []Vibe
	VibeID     int64
	VibeName   string
	Games      []Game
	Types      []string
	Categories []string
	Mechanics  []string
	Type           string
	Category       string
	Mechanic       string
	Players        string
	Playtime       string
	ValidPlayers   []PlayerOption
	ValidPlaytimes []PlaytimeOption
}

// PlayerOption represents a player count filter that has matching games.
type PlayerOption struct {
	Value string
	Label string
}

// PlaytimeOption represents a playtime filter that has matching games.
type PlaytimeOption struct {
	Value string
	Label string
}

// GameEditData holds data for the game edit (vibe tagging) page.
type GameEditData struct {
	Game      Game
	AllVibes  []Vibe
	GameVibes map[int64]bool
}

// VibesPageData holds data for the vibe management page.
type VibesPageData struct {
	Vibes []Vibe
	Error string
}

// GamesPageData holds data for the games list page.
type GamesPageData struct {
	Games      []Game
	Categories []string
	Category   string
	Players    string
	Playtime   string
}

// CollectionEntry holds a game from a user's BGG collection.
type CollectionEntry struct {
	BGGID         int64
	Name          string
	YearPublished int
	Thumbnail     string
	AlreadyOwned  bool
}

// RulesPageData holds data for the rules page of a game.
type RulesPageData struct {
	Game       Game
	PlayerAids []PlayerAid
	EmbedURL   string // Google Drive embed URL for PDF viewer
}

// GameDetailData holds data for the game detail page.
type GameDetailData struct {
	Game Game
	Aids []PlayerAid
}

// PlayerAidsListData holds data for the player aids list partial.
type PlayerAidsListData struct {
	GameID int64
	Aids   []PlayerAid
}

// ImportResultData holds data for the import result partial.
type ImportResultData struct {
	Count  int
	ErrMsg string
}

// ImportPageData holds data for the BGG import page.
type ImportPageData struct {
	Username string
	Enabled  bool
}
