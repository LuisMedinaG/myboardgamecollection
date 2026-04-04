package main

import "fmt"

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

// PlayerAid represents an uploaded player aid image for a game.
type PlayerAid struct {
	ID       int64
	GameID   int64
	Filename string
	Label    string
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
