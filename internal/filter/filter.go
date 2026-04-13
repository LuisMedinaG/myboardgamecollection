package filter

import (
	"sort"
	"strings"

	"myboardgamecollection/internal/model"
	"myboardgamecollection/internal/viewmodel"
)

// PlayerCondition returns a SQL condition for the given player filter value.
// The prefix is prepended to column names (e.g. "g." for aliased queries, or "" for unaliased).
func PlayerCondition(players, prefix string) string {
	switch players {
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
}

// PlaytimeCondition returns a SQL condition for the given playtime filter value.
func PlaytimeCondition(playtime, prefix string) string {
	switch playtime {
	case "short":
		return prefix + "play_time < 30"
	case "medium":
		return prefix + "play_time >= 30 AND " + prefix + "play_time <= 60"
	case "long":
		return prefix + "play_time > 60"
	default:
		return ""
	}
}

// ValidPlayerOptions returns player filter options that match at least one game.
func ValidPlayerOptions(games []model.Game) []viewmodel.PlayerOption {
	type def struct {
		value string
		label string
		match func(model.Game) bool
	}
	all := []def{
		{"1", "Solo", func(g model.Game) bool { return g.MinPlayers <= 1 }},
		{"2", "Up to 2", func(g model.Game) bool { return g.MinPlayers <= 2 }},
		{"3", "Up to 3", func(g model.Game) bool { return g.MinPlayers <= 3 }},
		{"4", "Up to 4", func(g model.Game) bool { return g.MinPlayers <= 4 }},
		{"5plus", "5+", func(g model.Game) bool { return g.MaxPlayers >= 5 }},
	}
	var opts []viewmodel.PlayerOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, viewmodel.PlayerOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// ValidPlaytimeOptions returns playtime filter options that match at least one game.
func ValidPlaytimeOptions(games []model.Game) []viewmodel.PlaytimeOption {
	type def struct {
		value string
		label string
		match func(model.Game) bool
	}
	all := []def{
		{"short", "< 30 min", func(g model.Game) bool { return g.PlayTime < 30 }},
		{"medium", "30–60 min", func(g model.Game) bool { return g.PlayTime >= 30 && g.PlayTime <= 60 }},
		{"long", "> 60 min", func(g model.Game) bool { return g.PlayTime > 60 }},
	}
	var opts []viewmodel.PlaytimeOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, viewmodel.PlaytimeOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// WeightCondition returns a SQL condition for the given weight filter value.
// BGG weight scale: 1.0–2.0 = Light, 2.0–3.0 = Medium, 3.0–5.0 = Heavy.
func WeightCondition(weight, prefix string) string {
	switch weight {
	case "light":
		return prefix + "weight > 0 AND " + prefix + "weight < 2.0"
	case "medium":
		return prefix + "weight >= 2.0 AND " + prefix + "weight < 3.0"
	case "heavy":
		return prefix + "weight >= 3.0"
	default:
		return ""
	}
}

// ValidWeightOptions returns weight filter options that match at least one game.
func ValidWeightOptions(games []model.Game) []viewmodel.WeightOption {
	type def struct {
		value string
		label string
		match func(model.Game) bool
	}
	all := []def{
		{"light", "Light (< 2)", func(g model.Game) bool { return g.Weight > 0 && g.Weight < 2.0 }},
		{"medium", "Medium (2–3)", func(g model.Game) bool { return g.Weight >= 2.0 && g.Weight < 3.0 }},
		{"heavy", "Heavy (3+)", func(g model.Game) bool { return g.Weight >= 3.0 }},
	}
	var opts []viewmodel.WeightOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, viewmodel.WeightOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// RatingCondition returns a SQL condition for the given BGG average rating filter.
// Values: "good" = ≥6.0, "great" = ≥7.0, "excellent" = ≥8.0.
func RatingCondition(rating, prefix string) string {
	switch rating {
	case "good":
		return prefix + "rating >= 6.0"
	case "great":
		return prefix + "rating >= 7.0"
	case "excellent":
		return prefix + "rating >= 8.0"
	default:
		return ""
	}
}

// LanguageCondition returns a SQL condition for the given language dependence filter.
// BGG scale: 1=No necessary in-game text, 2=Some text, 3=Moderate, 4=Extensive, 5=Unplayable.
func LanguageCondition(lang, prefix string) string {
	switch lang {
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
}

// RecommendedPlayersCondition returns a SQL condition that matches games where
// the given player count appears in the recommended_players column.
// The stored value is a comma-separated list of plain numbers (e.g. "1,2,3").
// A sentinel-comma wrapping technique is used to avoid substring false-positives.
// Only digit characters are accepted; any other input returns "".
func RecommendedPlayersCondition(count, prefix string) string {
	if count == "" {
		return ""
	}
	for _, ch := range count {
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	// ',1,2,3,' LIKE '%,2,%' — wrapping both sides with commas prevents
	// matching "2" inside "12" or "22".
	return "',' || " + prefix + "recommended_players || ',' LIKE '%," + count + ",%'"
}

// ValidRatingOptions returns rating filter options that match at least one game.
func ValidRatingOptions(games []model.Game) []viewmodel.RatingOption {
	type def struct {
		value string
		label string
		match func(model.Game) bool
	}
	all := []def{
		{"good", "Good (6+)", func(g model.Game) bool { return g.Rating >= 6.0 }},
		{"great", "Great (7+)", func(g model.Game) bool { return g.Rating >= 7.0 }},
		{"excellent", "Excellent (8+)", func(g model.Game) bool { return g.Rating >= 8.0 }},
	}
	var opts []viewmodel.RatingOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, viewmodel.RatingOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// ValidLanguageOptions returns language dependence filter options that match at least one game.
func ValidLanguageOptions(games []model.Game) []viewmodel.LanguageOption {
	type def struct {
		value string
		label string
		match func(model.Game) bool
	}
	all := []def{
		{"free", "Language-free", func(g model.Game) bool { return g.LanguageDependence == 1 }},
		{"low", "Low dependence", func(g model.Game) bool { return g.LanguageDependence == 2 }},
		{"moderate", "Moderate dependence", func(g model.Game) bool { return g.LanguageDependence == 3 }},
		{"high", "High dependence", func(g model.Game) bool { return g.LanguageDependence >= 4 }},
	}
	var opts []viewmodel.LanguageOption
	for _, o := range all {
		for _, g := range games {
			if o.match(g) {
				opts = append(opts, viewmodel.LanguageOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// ValidRecPlayersOptions returns recommended-player-count options that match at least one game.
func ValidRecPlayersOptions(games []model.Game) []viewmodel.RecPlayersOption {
	type def struct {
		value string
		label string
	}
	all := []def{
		{"1", "Solo (1P)"},
		{"2", "2 Players"},
		{"3", "3 Players"},
		{"4", "4 Players"},
		{"5", "5 Players"},
	}
	var opts []viewmodel.RecPlayersOption
	for _, o := range all {
		for _, g := range games {
			if containsCount(g.RecommendedPlayers, o.value) {
				opts = append(opts, viewmodel.RecPlayersOption{Value: o.value, Label: o.label})
				break
			}
		}
	}
	return opts
}

// containsCount checks whether the comma-separated counts string contains the
// exact token n (e.g. "2" in "1,2,3" → true, "2" in "1,12,3" → false).
func containsCount(counts, n string) bool {
	return strings.Contains(","+counts+",", ","+n+",")
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
