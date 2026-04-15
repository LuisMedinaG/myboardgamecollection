package filter

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/viewmodel"
)

// FilterDef defines a filter with its SQL condition builder and validation.
type FilterDef struct {
	Name        string
	Column      string
	Condition   func(value string, prefix string) string
	ValidValues map[string]string // value -> label
}

// Filters contains all available game filters.
var Filters = map[string]FilterDef{
	"players": {
		Name:   "players",
		Column: "min_players",
		Condition: func(value, prefix string) string {
			switch value {
			case "1":
				return prefix + "min_players <= 1"
			case "2":
				return prefix + "min_players <= 2"
			case "2only":
				return prefix + "min_players = 2 AND " + prefix + "max_players = 2"
			case "3":
				return prefix + "min_players <= 3"
			case "4":
				return prefix + "min_players <= 4"
			case "5plus":
				return prefix + "max_players >= 5"
			default:
				return ""
			}
		},
		ValidValues: map[string]string{
			"1":     "Solo",
			"2":     "Up to 2",
			"3":     "Up to 3",
			"4":     "Up to 4",
			"5plus": "5+",
		},
	},
	"playtime": {
		Name:   "playtime",
		Column: "play_time",
		Condition: func(value, prefix string) string {
			switch value {
			case "short":
				return prefix + "play_time < 30"
			case "medium":
				return prefix + "play_time >= 30 AND " + prefix + "play_time <= 60"
			case "long":
				return prefix + "play_time > 60"
			default:
				return ""
			}
		},
		ValidValues: map[string]string{
			"short":  "< 30 min",
			"medium": "30–60 min",
			"long":   "> 60 min",
		},
	},
	"weight": {
		Name:   "weight",
		Column: "weight",
		Condition: func(value, prefix string) string {
			switch value {
			case "light":
				return prefix + "weight > 0 AND " + prefix + "weight < 2.0"
			case "medium":
				return prefix + "weight >= 2.0 AND " + prefix + "weight < 3.0"
			case "heavy":
				return prefix + "weight >= 3.0"
			default:
				return ""
			}
		},
		ValidValues: map[string]string{
			"light":  "Light (< 2)",
			"medium": "Medium (2–3)",
			"heavy":  "Heavy (3+)",
		},
	},
	"rating": {
		Name:   "rating",
		Column: "rating",
		Condition: func(value, prefix string) string {
			switch value {
			case "good":
				return prefix + "rating >= 6.0"
			case "great":
				return prefix + "rating >= 7.0"
			case "excellent":
				return prefix + "rating >= 8.0"
			default:
				return ""
			}
		},
		ValidValues: map[string]string{
			"good":      "Good (6+)",
			"great":     "Great (7+)",
			"excellent": "Excellent (8+)",
		},
	},
	"lang": {
		Name:   "language_dependence",
		Column: "language_dependence",
		Condition: func(value, prefix string) string {
			switch value {
			case "free":
				return prefix + "language_dependence = 1"
			case "low":
				return prefix + "language_dependence = 2"
			case "moderate":
				return prefix + "language_dependence = 3"
			case "high":
				return prefix + "language_dependence >= 4"
			default:
				return ""
			}
		},
		ValidValues: map[string]string{
			"free":     "Language-free",
			"low":      "Low dependence",
			"moderate": "Moderate dependence",
			"high":     "High dependence",
		},
	},
	"rec_players": {
		Name:   "recommended_players",
		Column: "recommended_players",
		Condition: func(value, prefix string) string {
			if value == "" {
				return ""
			}
			for _, ch := range value {
				if ch < '0' || ch > '9' {
					return ""
				}
			}
			return fmt.Sprintf("',' || %srecommended_players || ',' LIKE '%%,%s,%%'", prefix, value)
		},
		ValidValues: map[string]string{
			"1": "Solo (1P)",
			"2": "2 Players",
			"3": "3 Players",
			"4": "4 Players",
			"5": "5 Players",
		},
	},
}

// QueryBuilder builds SQL WHERE clauses from filter parameters.
type QueryBuilder struct {
	filters map[string]FilterDef
}

// NewQueryBuilder creates a new query builder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{filters: Filters}
}

// BuildWhereClause builds a WHERE clause and args from URL parameters.
func (qb *QueryBuilder) BuildWhereClause(params url.Values, prefix string) (string, []any) {
	var conditions []string
	var args []any

	for name, def := range qb.filters {
		if values, exists := params[name]; exists && len(values) > 0 {
			value := values[0]
			if condition := def.Condition(value, prefix); condition != "" {
				conditions = append(conditions, condition)
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// GetValidOptions returns valid filter options that match at least one game.
func (qb *QueryBuilder) GetValidOptions(games []model.Game, filterName string) []viewmodel.FilterOption {
	def, exists := qb.filters[filterName]
	if !exists {
		return nil
	}

	var options []viewmodel.FilterOption
	for value, label := range def.ValidValues {
		for _, game := range games {
			if qb.matchesFilter(game, filterName, value) {
				options = append(options, viewmodel.FilterOption{Value: value, Label: label})
				break
			}
		}
	}
	return options
}

// matchesFilter checks if a game matches a specific filter value.
func (qb *QueryBuilder) matchesFilter(game model.Game, filterName, value string) bool {
	switch filterName {
	case "players":
		switch value {
		case "1":
			return game.MinPlayers <= 1
		case "2":
			return game.MinPlayers <= 2
		case "2only":
			return game.MinPlayers == 2 && game.MaxPlayers == 2
		case "3":
			return game.MinPlayers <= 3
		case "4":
			return game.MinPlayers <= 4
		case "5plus":
			return game.MaxPlayers >= 5
		}
	case "playtime":
		switch value {
		case "short":
			return game.PlayTime < 30
		case "medium":
			return game.PlayTime >= 30 && game.PlayTime <= 60
		case "long":
			return game.PlayTime > 60
		}
	case "weight":
		switch value {
		case "light":
			return game.Weight > 0 && game.Weight < 2.0
		case "medium":
			return game.Weight >= 2.0 && game.Weight < 3.0
		case "heavy":
			return game.Weight >= 3.0
		}
	case "rating":
		switch value {
		case "good":
			return game.Rating >= 6.0
		case "great":
			return game.Rating >= 7.0
		case "excellent":
			return game.Rating >= 8.0
		}
	case "lang":
		switch value {
		case "free":
			return game.LanguageDependence == 1
		case "low":
			return game.LanguageDependence == 2
		case "moderate":
			return game.LanguageDependence == 3
		case "high":
			return game.LanguageDependence >= 4
		}
	case "rec_players":
		return strings.Contains(","+game.RecommendedPlayers+",", ","+value+",")
	}
	return false
}

// Legacy functions for backward compatibility
func PlayerCondition(players, prefix string) string {
	return Filters["players"].Condition(players, prefix)
}

func PlaytimeCondition(playtime, prefix string) string {
	return Filters["playtime"].Condition(playtime, prefix)
}

func WeightCondition(weight, prefix string) string {
	return Filters["weight"].Condition(weight, prefix)
}

func RatingCondition(rating, prefix string) string {
	return Filters["rating"].Condition(rating, prefix)
}

func LanguageCondition(lang, prefix string) string {
	return Filters["lang"].Condition(lang, prefix)
}

func RecommendedPlayersCondition(count, prefix string) string {
	return Filters["rec_players"].Condition(count, prefix)
}

// Valid*Options functions for backward compatibility
func ValidPlayerOptions(games []model.Game) []viewmodel.PlayerOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "players")
	result := make([]viewmodel.PlayerOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.PlayerOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

func ValidPlaytimeOptions(games []model.Game) []viewmodel.PlaytimeOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "playtime")
	result := make([]viewmodel.PlaytimeOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.PlaytimeOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

func ValidWeightOptions(games []model.Game) []viewmodel.WeightOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "weight")
	result := make([]viewmodel.WeightOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.WeightOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

func ValidRatingOptions(games []model.Game) []viewmodel.RatingOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "rating")
	result := make([]viewmodel.RatingOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.RatingOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

func ValidLanguageOptions(games []model.Game) []viewmodel.LanguageOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "lang")
	result := make([]viewmodel.LanguageOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.LanguageOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

func ValidRecPlayersOptions(games []model.Game) []viewmodel.RecPlayersOption {
	qb := NewQueryBuilder()
	opts := qb.GetValidOptions(games, "rec_players")
	result := make([]viewmodel.RecPlayersOption, len(opts))
	for i, opt := range opts {
		result[i] = viewmodel.RecPlayersOption{Value: opt.Value, Label: opt.Label}
	}
	return result
}

// ExtractField collects unique comma-separated values from a game field, sorted.
func ExtractField(games []model.Game, field func(model.Game) string) []string {
	seen := make(map[string]bool)
	for _, g := range games {
		for _, v := range strings.Split(field(g), ", ") {
			v = strings.TrimSpace(v)
			if v != "" {
				seen[v] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}

// SplitDedupSort splits comma-separated values, deduplicates, and sorts them.
func SplitDedupSort(values []string) []string {
	seen := make(map[string]bool)
	for _, raw := range values {
		for _, v := range strings.Split(raw, ", ") {
			v = strings.TrimSpace(v)
			if v != "" {
				seen[v] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}
