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
	bgg  *gobgg.BGG
	http *http.Client
}

// New creates a new BGG client with the given auth token.
func New(token string) *Client {
	token = strings.TrimSpace(token)
	httpClient := &http.Client{Transport: &authTransport{base: http.DefaultTransport, token: token}}
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetAuthToken(token), gobgg.SetClient(httpClient)), http: httpClient}
}

// NewWithCookies creates a new BGG client using a raw cookie string
// (e.g. "bggusername=foo; bggpassword=bar; SessionID=xyz").
func NewWithCookies(raw string) *Client {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	cookies := parseCookieString(raw)
	httpClient := &http.Client{Transport: &authTransport{base: http.DefaultTransport, cookies: cookies}}
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetCookies("", cookies), gobgg.SetClient(httpClient)), http: httpClient}
}

// authTransport attaches auth cookies, a bearer token, and a User-Agent to every
// outgoing request. gobgg only wires cookies into a handful of endpoints
// (notably not /xmlapi2/thing), and BGG now returns HTTP 401 for unauthenticated
// /thing requests, which gobgg surfaces as "XML decoding failed: EOF". Doing it
// at the transport layer guarantees every call is authenticated.
type authTransport struct {
	base    http.RoundTripper
	cookies []*http.Cookie
	token   string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "myboardgamecollection/1.0 (+https://github.com/LuisMedinaG/myboardgamecollection)")
	}
	// Token is the primary auth strategy; cookies are a workaround for local
	// dev when no token is available. Only fall back to cookies if no token.
	if t.token != "" {
		if req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+t.token)
		}
	} else {
		for _, c := range t.cookies {
			req.AddCookie(c)
		}
	}
	return t.base.RoundTrip(req)
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
		things, fetchErr := c.getThingsWithRetry(ctx, item.ID)
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

// getThingsWithRetry calls the BGG thing API. BGG returns HTTP 202 while it
// builds the response, in which case we retry with a short backoff. Any other
// non-200 status (401/403/5xx) fails immediately with a descriptive error so
// users aren't left waiting on a silent retry loop. We cap at a handful of
// attempts so a single-game import can't hang for minutes.
func (c *Client) getThingsWithRetry(ctx context.Context, id int64) ([]gobgg.ThingResult, error) {
	const maxAttempts = 5
	delay := 500 * time.Millisecond

	for attempt := 1; ; attempt++ {
		status, err := c.probeThingStatus(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("probing /thing for bgg_id=%d: %w", id, err)
		}

		switch {
		case status == http.StatusOK:
			things, err := c.bgg.GetThings(ctx, gobgg.GetThingIDs(id))
			if err != nil {
				return nil, fmt.Errorf("decoding /thing for bgg_id=%d: %w", id, err)
			}
			if len(things) == 0 {
				return nil, fmt.Errorf("empty thing response for bgg_id=%d", id)
			}
			return things, nil
		case status == http.StatusAccepted:
			// BGG is still building the response; retry.
		case status == http.StatusUnauthorized, status == http.StatusForbidden:
			return nil, fmt.Errorf("BGG returned HTTP %d for /thing?id=%d — check BGG_TOKEN (or BGG_COOKIE in dev)", status, id)
		default:
			return nil, fmt.Errorf("BGG returned HTTP %d for /thing?id=%d", status, id)
		}

		if attempt >= maxAttempts {
			return nil, fmt.Errorf("BGG /thing?id=%d still 202 after %d attempts", id, attempt)
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		delay *= 2
		if delay > 4*time.Second {
			delay = 4 * time.Second
		}
	}
}

// probeThingStatus issues a lightweight GET against BGG's /thing endpoint and
// returns the HTTP status code. We use a GET (not HEAD — BGG's xmlapi2 doesn't
// reliably honor HEAD) and close the body immediately without decoding. This
// lets us distinguish 202 (queued) from 401/403 (auth) without parsing XML.
func (c *Client) probeThingStatus(ctx context.Context, id int64) (int, error) {
	u := fmt.Sprintf("https://boardgamegeek.com/xmlapi2/thing?id=%d&stats=1", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
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
