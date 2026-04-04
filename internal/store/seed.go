package store

import (
	"fmt"

	"myboardgamecollection/internal/model"
)

func (s *Store) seedDefaultVibes() error {
	defaults := []string{"Party", "Family Dinner", "Light Friend Night", "Heavy Euro", "Strangers Meeting"}
	for _, name := range defaults {
		_, _ = s.db.Exec("INSERT OR IGNORE INTO vibes (name) VALUES (?)", name)
	}
	return nil
}

// SeedIfEmpty populates the games table with sample data when it is empty.
func (s *Store) SeedIfEmpty() error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seeds := []model.Game{
		{
			BGGID: 13, Name: "Catan", YearPublished: 1995,
			Description: "Collect and trade resources to build up the island of Catan in this modern classic. Players try to be the dominant force on the island by building settlements, cities, and roads. On each turn dice are rolled to determine what resources the island produces.",
			Image:      "https://picsum.photos/seed/bgg13/400/400",
			Thumbnail:  "https://picsum.photos/seed/bgg13/200/200",
			MinPlayers: 3, MaxPlayers: 4, PlayTime: 90,
			Categories: "Economic, Negotiation",
			Mechanics:  "Dice Rolling, Hand Management, Network and Route Building, Resource Management, Trading",
			Types:      "Family Games, Strategy Games",
		},
		{
			BGGID: 178900, Name: "Codenames", YearPublished: 2015,
			Description: "Give one-word clues to help your team identify secret agents. Two rival spymasters know the secret identities of 25 agents. Their teammates know the agents only by their codenames.",
			Image:      "https://picsum.photos/seed/bgg178900/400/400",
			Thumbnail:  "https://picsum.photos/seed/bgg178900/200/200",
			MinPlayers: 2, MaxPlayers: 8, PlayTime: 15,
			Categories: "Card Game, Deduction, Party Game, Word Game",
			Mechanics:  "Communication Limits, Push Your Luck, Team-Based Game",
			Types:      "Family Games, Party Games",
		},
		{
			BGGID: 70323, Name: "King of Tokyo", YearPublished: 2011,
			Description: "Mutant monsters, gigantic robots, and strange aliens battle to become the King of Tokyo. Roll dice, smash your opponents, and claim the city!",
			Image:      "https://picsum.photos/seed/bgg70323/400/400",
			Thumbnail:  "https://picsum.photos/seed/bgg70323/200/200",
			MinPlayers: 2, MaxPlayers: 6, PlayTime: 30,
			Categories: "Dice, Fighting, Science Fiction",
			Mechanics:  "Dice Rolling, King of the Hill, Player Elimination, Press Your Luck",
			Types:      "Family Games",
		},
		{
			BGGID: 174430, Name: "Gloomhaven", YearPublished: 2017,
			Description: "Vanquish monsters with strategic cardplay in a persistent legacy campaign. Players take on the role of wandering adventurers with their own special set of skills in this tactical combat game.",
			Image:      "https://picsum.photos/seed/bgg174430/400/400",
			Thumbnail:  "https://picsum.photos/seed/bgg174430/200/200",
			MinPlayers: 1, MaxPlayers: 4, PlayTime: 120,
			Categories: "Adventure, Exploration, Fantasy, Fighting, Miniatures",
			Mechanics:  "Action Queue, Campaign, Cooperative Game, Grid Movement, Hand Management, Modular Board",
			Types:      "Strategy Games, Thematic Games",
			RulesURL:   "https://drive.google.com/file/d/1pPpSCCFWOaNUPe2GqXkzsLjjOF6KC2Bi/view",
		},
		{
			BGGID: 167791, Name: "Terraforming Mars", YearPublished: 2016,
			Description: "Compete with rival CEOs to make Mars habitable and build your corporate empire. Initiate huge projects to raise the temperature, oxygen level, and ocean coverage until the environment is livable.",
			Image:      "https://picsum.photos/seed/bgg167791/400/400",
			Thumbnail:  "https://picsum.photos/seed/bgg167791/200/200",
			MinPlayers: 1, MaxPlayers: 5, PlayTime: 120,
			Categories: "Economic, Industry, Science Fiction, Territory Building",
			Mechanics:  "Card Drafting, End Game Bonuses, Hand Management, Hexagonal Grid, Tile Placement, Variable Player Powers",
			Types:      "Strategy Games",
		},
	}

	for _, g := range seeds {
		if _, err := s.CreateGame(g); err != nil {
			return fmt.Errorf("seed %q: %w", g.Name, err)
		}
	}
	return nil
}
