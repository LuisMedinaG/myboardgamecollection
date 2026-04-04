package bgg

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/store"

	"github.com/fzerorubigd/gobgg"
)

// Client wraps the BoardGameGeek API client.
type Client struct {
	bgg *gobgg.BGG
}

// New creates a new BGG client with the given auth token.
func New(token string) *Client {
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetAuthToken(token))}
}

// NewWithCookies creates a new BGG client using a raw cookie string
// (e.g. "bggusername=foo; bggpassword=bar; SessionID=xyz").
func NewWithCookies(raw string) *Client {
	cookies := parseCookieString(raw)
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetCookies("", cookies))}
}

// parseCookieString parses a raw Cookie header value into []*http.Cookie.
func parseCookieString(raw string) []*http.Cookie {
	header := http.Header{"Cookie": {raw}}
	req := http.Request{Header: header}
	return req.Cookies()
}

// ImportCollection imports all games from a user's BGG collection.
// Returns the number of new games imported.
func (c *Client) ImportCollection(ctx context.Context, s *store.Store, username string) (int, error) {
	items, err := c.bgg.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return 0, fmt.Errorf("fetching collection for %q: %w", username, err)
	}

	imported := 0
	for _, item := range items {
		if _, err := s.GetGameByBGGID(item.ID); err == nil {
			continue // already owned
		}

		things, err := c.bgg.GetThings(ctx, gobgg.GetThingIDs(item.ID))
		if err != nil || len(things) == 0 {
			continue
		}

		game := thingToGame(things[0])
		if _, err := s.CreateGame(game); err == nil {
			imported++
		}
	}

	return imported, nil
}

// thingToGame converts a BGG API thing into a Game model.
func thingToGame(t gobgg.ThingResult) model.Game {
	playTime, _ := strconv.Atoi(t.PlayTime)

	var cats []string
	for _, l := range t.Categories() {
		cats = append(cats, l.Name)
	}
	var mechs []string
	for _, l := range t.Mechanics() {
		mechs = append(mechs, l.Name)
	}
	var types []string
	for _, l := range t.GetLinkByName("boardgamesubdomain") {
		types = append(types, l.Name)
	}

	return model.Game{
		BGGID:         t.ID,
		Name:          t.Name,
		Description:   t.Description,
		YearPublished: t.YearPublished,
		Image:         t.Image,
		Thumbnail:     t.Thumbnail,
		MinPlayers:    t.MinPlayers,
		MaxPlayers:    t.MaxPlayers,
		PlayTime:      playTime,
		Categories:    strings.Join(cats, ", "),
		Mechanics:     strings.Join(mechs, ", "),
		Types:         strings.Join(types, ", "),
	}
}
