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

// bggRPS is the maximum requests-per-second this process will send to BGG. BGG
// aggressively rate-limits the /xmlapi2 and /api endpoints and returns HTTP 429
// once you exceed roughly a few requests per second. 2/s is well under that
// and still syncs a ~120-game collection in about a minute.
const bggRPS = 2

// New creates a new BGG client with the given auth token.
func New(token string) *Client {
	token = strings.TrimSpace(token)
	httpClient := newHTTPClient(&authTransport{base: http.DefaultTransport, token: token})
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetAuthToken(token), gobgg.SetClient(httpClient))}
}

// NewWithCookies creates a new BGG client using a raw cookie string
// (e.g. "bggusername=foo; bggpassword=bar; SessionID=xyz").
func NewWithCookies(raw string) *Client {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	cookies := parseCookieString(raw)
	httpClient := newHTTPClient(&authTransport{base: http.DefaultTransport, cookies: cookies})
	return &Client{bgg: gobgg.NewBGGClient(gobgg.SetCookies("", cookies), gobgg.SetClient(httpClient))}
}

// newHTTPClient wraps an authTransport in a throttling/429-aware transport and
// returns an http.Client configured for BGG.
func newHTTPClient(auth *authTransport) *http.Client {
	return &http.Client{Transport: &throttledTransport{
		base:    auth,
		tick:    time.NewTicker(time.Second / bggRPS),
		maxRetry: 3,
	}}
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

// throttledTransport paces outgoing requests to BGG and transparently retries
// HTTP 429 responses, honoring the Retry-After header when present. This keeps
// rate-limit handling out of every individual endpoint caller (and gobgg never
// needs to know about it).
type throttledTransport struct {
	base     http.RoundTripper
	tick     *time.Ticker
	maxRetry int
}

func (t *throttledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		// Pace outgoing requests.
		select {
		case <-t.tick.C:
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}

		resp, err := t.base.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests || attempt >= t.maxRetry {
			return resp, nil
		}

		// Got 429 — drain & close body, sleep, retry.
		wait := parseRetryAfter(resp.Header.Get("Retry-After"))
		if wait <= 0 {
			wait = time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		}
		resp.Body.Close()
		slog.Warn("bgg rate limited; backing off", "attempt", attempt+1, "wait", wait, "url", req.URL.Path)
		select {
		case <-time.After(wait):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}
}

// parseRetryAfter parses an HTTP Retry-After header (seconds or HTTP date).
// Returns 0 if unparseable.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// parseCookieString parses a raw Cookie header value into []*http.Cookie.
func parseCookieString(raw string) []*http.Cookie {
	header := http.Header{"Cookie": {raw}}
	req := http.Request{Header: header}
	return req.Cookies()
}

// ImportCollection syncs a user's BGG collection. When fullRefresh is false
// (the normal sync path) only games not already in the user's collection are
// fetched from BGG. When fullRefresh is true every owned item is re-fetched
// and its metadata updated — intended for admins and future paid tiers.
func (c *Client) ImportCollection(ctx context.Context, s *store.Store, username string, userID int64, fullRefresh bool) (added, updated, collectionCount int, err error) {
	items, err := c.bgg.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("fetching collection for %q: %w", username, err)
	}
	collectionCount = len(items)

	owned, err := s.OwnedBGGIDs(userID)
	if err != nil {
		return 0, 0, collectionCount, fmt.Errorf("loading owned IDs: %w", err)
	}

	// Normal sync: only fetch games we don't already own (0 /thing calls when
	// nothing changed). Full refresh: fetch every item to update metadata.
	var idsToFetch []int64
	for _, item := range items {
		if fullRefresh || !owned[item.ID] {
			idsToFetch = append(idsToFetch, item.ID)
		}
	}

	if len(idsToFetch) == 0 {
		return 0, 0, collectionCount, nil
	}

	// Batch requests in groups of bggThingBatchSize — gobgg/BGG allows up to
	// 20 IDs per /thing call, so we cut request count by up to 20x.
	var firstThingErr error
	thingFailures := 0
	for _, batch := range chunkIDs(idsToFetch, bggThingBatchSize) {
		things, fetchErr := c.getThingsWithRetry(ctx, batch...)
		if fetchErr != nil {
			if firstThingErr == nil {
				firstThingErr = fetchErr
			}
			thingFailures += len(batch)
			slog.Warn("bgg thing fetch failed", "batch_size", len(batch), "error", fetchErr)
			continue
		}
		for _, t := range things {
			game := thingToGame(t)
			if owned[t.ID] {
				if updateErr := s.UpdateGame(game, userID); updateErr == nil {
					updated++
				}
			} else {
				if _, createErr := s.CreateGame(game, userID); createErr == nil {
					added++
				}
			}
		}
	}

	if thingFailures == len(idsToFetch) {
		return added, updated, collectionCount, fmt.Errorf("could not load game details from BGG for any of %d collection item(s); last error: %w", len(idsToFetch), firstThingErr)
	}

	return added, updated, collectionCount, nil
}

// bggThingBatchSize is the max number of IDs to request per /thing call.
// gobgg enforces a hard limit of 20.
const bggThingBatchSize = 20

// chunkIDs splits ids into contiguous chunks of at most size.
func chunkIDs(ids []int64, size int) [][]int64 {
	if size <= 0 {
		return [][]int64{ids}
	}
	var out [][]int64
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[i:end])
	}
	return out
}

// getThingsWithRetry calls the BGG thing API for one or more IDs. Rate-limiting
// and 429 backoff are handled by the throttledTransport, so this function only
// needs to retry when BGG returns an empty (queued) response.
func (c *Client) getThingsWithRetry(ctx context.Context, ids ...int64) ([]gobgg.ThingResult, error) {
	const maxAttempts = 4
	delay := 500 * time.Millisecond

	for attempt := 1; ; attempt++ {
		things, err := c.bgg.GetThings(ctx, gobgg.GetThingIDs(ids...))
		if err == nil && len(things) > 0 {
			return things, nil
		}
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "eof") {
			return nil, fmt.Errorf("fetching /thing for bgg_ids=%v: %w", ids, err)
		}

		if attempt >= maxAttempts {
			if err != nil {
				return nil, fmt.Errorf("fetching /thing for bgg_ids=%v after %d attempts: %w", ids, attempt, err)
			}
			return nil, fmt.Errorf("empty /thing response for bgg_ids=%v after %d attempts", ids, attempt)
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		delay *= 2
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
		Weight:        t.AverageWeight,
	}
}
