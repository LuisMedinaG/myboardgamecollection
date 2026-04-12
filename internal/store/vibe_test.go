package store

import (
	"testing"

	"myboardgamecollection/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddVibesToGamesIgnoresDuplicateIDs(t *testing.T) {
	s := newTestStore(t)

	userID, err := s.RegisterUser("vibe-owner", "pass", "", "")
	require.NoError(t, err)

	gameOneID, err := s.CreateGame(model.Game{BGGID: 101, Name: "First Game"}, userID)
	require.NoError(t, err)
	gameTwoID, err := s.CreateGame(model.Game{BGGID: 102, Name: "Second Game"}, userID)
	require.NoError(t, err)

	vibeID, err := s.CreateVibe("Cozy", userID)
	require.NoError(t, err)

	err = s.AddVibesToGames(userID, []int64{gameOneID, gameOneID, gameTwoID}, []int64{vibeID, vibeID})
	require.NoError(t, err)

	gameOneVibes, err := s.VibesForGame(gameOneID)
	require.NoError(t, err)
	require.Len(t, gameOneVibes, 1)
	assert.Equal(t, vibeID, gameOneVibes[0].ID)

	gameTwoVibes, err := s.VibesForGame(gameTwoID)
	require.NoError(t, err)
	require.Len(t, gameTwoVibes, 1)
	assert.Equal(t, vibeID, gameTwoVibes[0].ID)
}

func TestSetGameVibesIgnoresDuplicateIDs(t *testing.T) {
	s := newTestStore(t)

	userID, err := s.RegisterUser("set-vibes-owner", "pass", "", "")
	require.NoError(t, err)

	gameID, err := s.CreateGame(model.Game{BGGID: 201, Name: "Only Game"}, userID)
	require.NoError(t, err)

	firstVibeID, err := s.CreateVibe("Cozy", userID)
	require.NoError(t, err)
	secondVibeID, err := s.CreateVibe("Thinky", userID)
	require.NoError(t, err)

	err = s.SetGameVibes(userID, gameID, []int64{firstVibeID, firstVibeID, secondVibeID})
	require.NoError(t, err)

	vibes, err := s.VibesForGame(gameID)
	require.NoError(t, err)
	require.Len(t, vibes, 2)
	assert.ElementsMatch(t, []int64{firstVibeID, secondVibeID}, []int64{vibes[0].ID, vibes[1].ID})
}
