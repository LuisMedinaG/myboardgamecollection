package viewmodel

import "myboardgamecollection/internal/model"

// PageData wraps a title and arbitrary data for full-page renders.
type PageData struct {
	Title string
	Data  any
}

// GamesPageData holds data for the games list page.
type GamesPageData struct {
	Games      []model.Game
	Categories []string
	Q          string
	Category   string
	Players    string
	Playtime   string
}

// GameDetailData holds data for the game detail page.
type GameDetailData struct {
	Game model.Game
	Aids []model.PlayerAid
}

// GameEditData holds data for the game edit (vibe tagging) page.
type GameEditData struct {
	Game      model.Game
	AllVibes  []model.Vibe
	GameVibes map[int64]bool
}

// DiscoverPageData holds data for the discover page.
type DiscoverPageData struct {
	Vibes      []model.Vibe
	VibeID     int64
	VibeName   string
	Games      []model.Game
	Types      []string
	Categories []string
	Mechanics  []string
	Type       string
	Category   string
	Mechanic   string
	Players    string
	Playtime   string
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

// VibesPageData holds data for the vibe management page.
type VibesPageData struct {
	Vibes []model.Vibe
	Error string
}

// RulesPageData holds data for the rules page of a game.
type RulesPageData struct {
	Game       model.Game
	PlayerAids []model.PlayerAid
	EmbedURL   string
}

// PlayerAidsListData holds data for the player aids list partial.
type PlayerAidsListData struct {
	GameID int64
	Aids   []model.PlayerAid
}

// ImportPageData holds data for the BGG import page.
type ImportPageData struct {
	Username string
	Enabled  bool
}

// ImportResultData holds data for the import result partial.
type ImportResultData struct {
	Count   int
	Updated int
	ErrMsg  string
}
