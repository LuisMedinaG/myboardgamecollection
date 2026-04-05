package bgg

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	token = strings.TrimSpace(token)
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetAuthToken(token))}
}

// NewWithCookies creates a new BGG client using a raw cookie string
// (e.g. "bggusername=foo; bggpassword=bar; SessionID=xyz").
func NewWithCookies(raw string) *Client {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	cookies := parseCookieString(raw)
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetCookies("", cookies))}
}

// parseCookieString parses a raw Cookie header value into []*http.Cookie.
func parseCookieString(raw string) []*http.Cookie {
	header := http.Header{"Cookie": {raw}}
	req := http.Request{Header: header}
	return req.Cookies()
}

// ImportCollection syncs all games from a user's BGG collection into the given
// userID's data. Returns the number of new games added, games updated, and how
// many rows BGG returned in the collection response (before per-game fetches).
func (c *Client) ImportCollection(ctx context.Context, s *store.Store, username string, userID int64) (added, updated, collectionCount int, err error) {
	items, err := c.bgg.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetching collection for %q: %w", username, err)
	}
	collectionCount = len(items)

	owned, err := s.OwnedBGGIDs(userID)
	if err != nil {
		return 0, 0, collectionCount, fmt.Errorf("loading owned IDs: %w", err)
	}

	var firstThingErr error
	thingFailures := 0
	for _, item := range items {
		things, fetchErr := getThingsWithRetry(ctx, c.bgg, item.ID)
		if fetchErr != nil {
			if firstThingErr == nil {
				firstThingErr = fetchErr
			}
			thingFailures++
			slog.Warn("bgg thing fetch failed", "bgg_id", item.ID, "name", item.Name, "error", fetchErr)
			continue
		}

		game := thingToGame(things[0])
		if owned[item.ID] {
			if updateErr := s.UpdateGame(game, userID); updateErr == nil {
				updated++
			}
		} else {
			if _, createErr := s.CreateGame(game, userID); createErr == nil {
				added++
			}
		}
	}

	if collectionCount > 0 && thingFailures == collectionCount {
		return added, updated, collectionCount, fmt.Errorf("could not load game details from BGG for any of %d collection item(s); last error: %w", collectionCount, firstThingErr)
	}

	return added, updated, collectionCount, nil
}

// getThingsWithRetry calls the BGG thing API with the same backoff strategy as
// gobgg's collection client. The thing endpoint can return HTTP 202 or an
// empty payload while BGG builds the response; gobgg.GetThings does not retry.
func getThingsWithRetry(ctx context.Context, bggc *gobgg.BGG, id int64) ([]gobgg.ThingResult, error) {
	delay := time.Second
	for i := 1; ; i++ {
		things, err := bggc.GetThings(ctx, gobgg.GetThingIDs(id))
		if err != nil {
			return nil, err
		}
		if len(things) > 0 {
			return things, nil
		}

		if i >= 25 {
			return nil, fmt.Errorf("empty thing response for bgg_id=%d after %d attempts", id, i)
		}

		delay += time.Duration(i) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
