package service

import (
	"context"
	"fmt"

	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/store"
)

type GameService struct {
	store *store.Store
}

func NewGameService(store *store.Store) *GameService {
	return &GameService{store: store}
}

type GameFilters struct {
	Query      string
	Category   string
	Players    string
	Playtime   string
	Weight     string
	Rating     string
	Language   string
	RecPlayers string
}

type GameListResult struct {
	Games      []model.Game
	Total      int
	Page       int
	TotalPages int
	PerPage    int
}

func (gs *GameService) ListGames(ctx context.Context, userID int64, filters GameFilters, page, limit int) (*GameListResult, error) {
	games, total, err := gs.store.FilterGames(
		filters.Query,
		filters.Category,
		filters.Players,
		filters.Playtime,
		filters.Weight,
		filters.Rating,
		filters.Language,
		filters.RecPlayers,
		page,
		limit,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("filter games: %w", err)
	}

	if err := gs.populateGameVibes(games); err != nil {
		return nil, err
	}

	return &GameListResult{
		Games:      games,
		Total:      total,
		Page:       page,
		TotalPages: pageCount(total, limit),
		PerPage:    limit,
	}, nil
}

func (gs *GameService) GetGame(ctx context.Context, gameID, userID int64) (*model.Game, error) {
	game, err := gs.store.GetGame(gameID, userID)
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}
	return &game, nil
}

func (gs *GameService) DeleteGame(ctx context.Context, gameID, userID int64) error {
	if err := gs.store.DeleteGame(gameID, userID); err != nil {
		return fmt.Errorf("delete game: %w", err)
	}
	return nil
}

func (gs *GameService) AssignVibesToGames(ctx context.Context, userID int64, gameIDs, vibeIDs []int64) error {
	if len(gameIDs) == 0 || len(vibeIDs) == 0 {
		return fmt.Errorf("select at least one game and one vibe")
	}

	if err := gs.store.AddVibesToGames(userID, gameIDs, vibeIDs); err != nil {
		if store.IsOwnershipError(err) {
			return fmt.Errorf("one or more selected games or vibes were not found")
		}
		return fmt.Errorf("assign vibes to games: %w", err)
	}
	return nil
}

func (gs *GameService) GetCategories(ctx context.Context, userID int64) ([]string, error) {
	categories, err := gs.store.DistinctCategories(userID)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	return categories, nil
}

func (gs *GameService) GetAllVibes(ctx context.Context, userID int64) ([]model.Vibe, error) {
	vibes, err := gs.store.AllVibes(userID)
	if err != nil {
		return nil, fmt.Errorf("get all vibes: %w", err)
	}
	return vibes, nil
}

func (gs *GameService) populateGameVibes(games []model.Game) error {
	if len(games) == 0 {
		return nil
	}

	gameIDs := make([]int64, len(games))
	for i, game := range games {
		gameIDs[i] = game.ID
	}

	gameVibes, err := gs.store.VibesForGames(gameIDs)
	if err != nil {
		return fmt.Errorf("populate game vibes: %w", err)
	}

	for i := range games {
		games[i].Vibes = gameVibes[games[i].ID]
	}
	return nil
}

func pageCount(total, limit int) int {
	pages := (total + limit - 1) / limit
	if pages < 1 {
		return 1
	}
	return pages
}
