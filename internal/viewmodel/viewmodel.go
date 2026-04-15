package viewmodel

import (
	"net/url"
	"strconv"

	"myboardgamecollection/internal/model"
)

// PageData wraps a title, the current user's BGG username, and page-specific
// data for full-page renders. User is empty on the login page.
type PageData struct {
	Title     string
	User      string // BGG username of the logged-in user; empty if not authenticated
	CSRFToken string // CSRF token for form/HTMX protection
	Data      any
}

// AuthPageData holds data for the login, signup, and change-password pages.
type AuthPageData struct {
	Error   string
	Success bool
}

// GamesPageData holds data for the games list page.
type GamesPageData struct {
	Games      []model.Game
	Categories []string
	AllVibes   []model.Vibe
	Q          string
	Category   string
	Players    string
	Playtime   string
	Weight     string
	Rating     string
	Lang       string
	RecPlayers string
	Page       int
	TotalPages int
	TotalCount int
	PerPage    int
}

// PageURL builds a /games URL that preserves all active filters and sets the
// given page number. Page 1 is omitted from the URL to keep links clean.
func (d GamesPageData) PageURL(page int) string {
	params := url.Values{}
	if d.Q != "" {
		params.Set("q", d.Q)
	}
	if d.Category != "" {
		params.Set("category", d.Category)
	}
	if d.Players != "" {
		params.Set("players", d.Players)
	}
	if d.Playtime != "" {
		params.Set("playtime", d.Playtime)
	}
	if d.Weight != "" {
		params.Set("weight", d.Weight)
	}
	if d.Rating != "" {
		params.Set("rating", d.Rating)
	}
	if d.Lang != "" {
		params.Set("lang", d.Lang)
	}
	if d.RecPlayers != "" {
		params.Set("rec_players", d.RecPlayers)
	}
	if page > 1 {
		params.Set("page", strconv.Itoa(page))
	}
	if d.PerPage > 0 && d.PerPage != 20 {
		params.Set("limit", strconv.Itoa(d.PerPage))
	}
	if len(params) == 0 {
		return "/games"
	}
	return "/games?" + params.Encode()
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
	Weight     string
	Rating     string
	Lang       string
	RecPlayers string
	ValidPlayers   []PlayerOption
	ValidPlaytimes []PlaytimeOption
	ValidWeights   []WeightOption
	ValidRatings   []RatingOption
	ValidLanguages []LanguageOption
	ValidRecPlayers []RecPlayersOption
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

// WeightOption represents a weight/complexity filter that has matching games.
type WeightOption struct {
	Value string
	Label string
}

// RatingOption represents a BGG rating filter that has matching games.
type RatingOption struct {
	Value string
	Label string
}

// LanguageOption represents a language dependence filter that has matching games.
type LanguageOption struct {
	Value string
	Label string
}

// RecPlayersOption represents a recommended player count filter that has matching games.
type RecPlayersOption struct {
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
	Username  string
	Enabled   bool
	CanSync   bool // false when the daily sync limit has been reached
	IsAdmin   bool
	SyncLimit int // max syncs allowed per day for this user
}

// ImportResultData holds data for the import result partial.
type ImportResultData struct {
	Count           int
	Updated         int
	CollectionItems int // rows returned by BGG collection API (owned filter)
	Username        string
	ErrMsg          string
}

// ImportCSVPageData holds data for the CSV import page.
type ImportCSVPageData struct {
	Enabled     bool   // false when no BGG client is configured server-side
	BGGUsername string // set when the user has a BGG username configured
}

// CSVPreviewRow is one row shown in the CSV import preview table.
type CSVPreviewRow struct {
	BGGID        int64
	Name         string
	AlreadyOwned bool
}

// ImportCSVPreviewData holds data for the CSV preview partial. It is rendered
// after a user uploads a file but before the actual import is confirmed.
type ImportCSVPreviewData struct {
	Rows         []CSVPreviewRow
	NewCount     int
	OwnedCount   int
	SkippedRows  int // rows in the file that had no usable BGG ID
	TotalParsed  int // unique BGG IDs successfully parsed from the file
	ImportIDsCSV string // comma-separated list of BGG IDs to send back on confirm
	ErrMsg       string
}

// ImportCSVResultData holds data for the CSV import result partial.
type ImportCSVResultData struct {
	Added   int
	Skipped int // already-owned games that were not re-fetched
	Failed  int // games whose metadata could not be fetched from BGG
	ErrMsg  string
}
