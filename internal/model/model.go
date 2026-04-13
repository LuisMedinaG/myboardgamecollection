package model

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
	Types              string  // BGG subdomain (e.g. "Family Games, Strategy Games")
	Weight             float64 // BGG average weight (complexity): 1.0=lightest, 5.0=heaviest
	Rating             float64 // BGG community average rating (0.0 = unknown)
	LanguageDependence int     // BGG language dependence poll level: 0=unknown, 1=no text … 5=unplayable
	RecommendedPlayers string  // comma-separated counts recommended by BGG community, e.g. "1,2,3"
	RulesURL           string  // Google Drive link to rulebook PDF
	Vibes              []Vibe  // populated on demand; nil when not fetched
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

// ThumbnailURL returns the URL to use for the game thumbnail in templates.
// When a BGGID is available it routes through the local image proxy/cache so
// external URLs are never sent directly to browsers (avoids CSP issues and
// provides resilience against upstream URL changes).
func (g Game) ThumbnailURL() string {
	if g.BGGID > 0 {
		return fmt.Sprintf("/images/%d", g.BGGID)
	}
	return g.Thumbnail
}

// BestPlayerCount returns the first (most-recommended) player count from
// RecommendedPlayers, or an empty string when not set.
func (g Game) BestPlayerCount() string {
	if g.RecommendedPlayers == "" {
		return ""
	}
	parts := strings.SplitN(g.RecommendedPlayers, ",", 2)
	return strings.TrimSpace(parts[0])
}

// LanguageDependenceLabel returns a short label for the BGG language-dependence
// poll level (1–5). Returns an empty string for level 0 (unknown/not set).
func (g Game) LanguageDependenceLabel() string {
	switch g.LanguageDependence {
	case 1:
		return "No necessary in-game text"
	case 2:
		return "Some necessary text"
	case 3:
		return "Moderate in-game text"
	case 4:
		return "Extensive use of text"
	case 5:
		return "Unplayable in another language"
	default:
		return ""
	}
}

// LanguageDependenceSub returns a one-line description for the language-dependence
// level, used as a subtitle beneath LanguageDependenceLabel.
func (g Game) LanguageDependenceSub() string {
	switch g.LanguageDependence {
	case 1:
		return "Can be played in any language."
	case 2:
		return "A few words that can be memorized."
	case 3:
		return "Some text needed, but manageable."
	case 4:
		return "Difficult to play without proficiency."
	case 5:
		return "Reading proficiency required."
	default:
		return ""
	}
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

var vibeColorClasses = [...]string{
	"vibe-color-0",
	"vibe-color-1",
	"vibe-color-2",
	"vibe-color-3",
	"vibe-color-4",
	"vibe-color-5",
	"vibe-color-6",
	"vibe-color-7",
}

// ColorClass returns a stable CSS class for this vibe based on its ID.
func (v Vibe) ColorClass() string {
	return vibeColorClasses[v.ID%int64(len(vibeColorClasses))]
}

// CollectionEntry holds a game from a user's BGG collection.
type CollectionEntry struct {
	BGGID         int64
	Name          string
	YearPublished int
	Thumbnail     string
	AlreadyOwned  bool
}

// User represents a registered user of the application.
type User struct {
	ID          int64
	Username    string
	BGGUsername string
	Email       string
}
