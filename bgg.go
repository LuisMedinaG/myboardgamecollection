package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/fzerorubigd/gobgg"
)

var bggClient *gobgg.BGG

func initBGG(token string) {
	bggClient = gobgg.NewBGGClient(gobgg.SetAuthToken(token))
}

// fetchBGGCollection fetches a user's board game collection from BGG.
func fetchBGGCollection(ctx context.Context, username string) ([]CollectionEntry, error) {
	items, err := bggClient.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return nil, fmt.Errorf("fetching collection for %q: %w", username, err)
	}

	owned, _ := ownedBGGIDs()

	var out []CollectionEntry
	for _, item := range items {
		out = append(out, CollectionEntry{
			BGGID:        item.ID,
			Name:         item.Name,
			YearPublished: item.YearPublished,
			Thumbnail:    item.Thumbnail,
			AlreadyOwned: owned[item.ID],
		})
	}
	return out, nil
}

// importBGGGame fetches full details from BGG and creates a game in the collection.
func importBGGGame(ctx context.Context, bggID int64) (int64, error) {
	// Check if already owned
	if g, err := getGameByBGGID(bggID); err == nil {
		return g.ID, nil
	}

	things, err := bggClient.GetThings(ctx, gobgg.GetThingIDs(bggID))
	if err != nil {
		return 0, err
	}
	if len(things) == 0 {
		return 0, fmt.Errorf("game %d not found on BGG", bggID)
	}

	t := things[0]

	playTime, _ := strconv.Atoi(t.PlayTime)

	var cats []string
	for _, l := range t.Categories() {
		cats = append(cats, l.Name)
	}

	var mechs []string
	for _, l := range t.Mechanics() {
		mechs = append(mechs, l.Name)
	}

	game := Game{
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
	}

	return createGame(game)
}

// importBGGCollection imports all games from a user's BGG collection.
// Returns the number of new games imported.
func importBGGCollection(ctx context.Context, username string) (int, error) {
	items, err := bggClient.GetCollection(ctx, username, gobgg.SetCollectionTypes(gobgg.CollectionTypeOwn))
	if err != nil {
		return 0, fmt.Errorf("fetching collection for %q: %w", username, err)
	}

	imported := 0
	for _, item := range items {
		// Skip if already in local DB
		if _, err := getGameByBGGID(item.ID); err == nil {
			continue
		}

		things, err := bggClient.GetThings(ctx, gobgg.GetThingIDs(item.ID))
		if err != nil || len(things) == 0 {
			continue
		}

		t := things[0]
		playTime, _ := strconv.Atoi(t.PlayTime)

		var cats []string
		for _, l := range t.Categories() {
			cats = append(cats, l.Name)
		}
		var mechs []string
		for _, l := range t.Mechanics() {
			mechs = append(mechs, l.Name)
		}

		game := Game{
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
		}

		if _, err := createGame(game); err == nil {
			imported++
		}
	}

	return imported, nil
}
