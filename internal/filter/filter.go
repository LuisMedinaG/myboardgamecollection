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
