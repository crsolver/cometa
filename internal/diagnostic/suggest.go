package diagnostic

import (
	"sort"
	"unicode/utf8"
)

// Suggest returns the candidate closest to name, or "" when none is similar
// enough to be a plausible typo.
func Suggest(name string, candidates []string) string {
	limit := 1
	if n := utf8.RuneCountInString(name); n <= 2 {
		return ""
	} else if n >= 8 {
		limit = 3
	} else if n >= 4 {
		limit = 2
	}
	sorted := append([]string(nil), candidates...)
	sort.Strings(sorted)
	best, bestDistance := "", limit+1
	for _, candidate := range sorted {
		if candidate == name || candidate == "" || candidate[0] == 0 {
			continue
		}
		if d := distance(name, candidate); d < bestDistance {
			best, bestDistance = candidate, d
		}
	}
	return best
}

// Hint formats a "did you mean" suffix, or "" when there is no suggestion.
func Hint(name string, candidates []string) string {
	if s := Suggest(name, candidates); s != "" {
		return "; ¿quisiste decir \"" + s + "\"?"
	}
	return ""
}

// distance is the Damerau-Levenshtein distance over runes, so a swapped pair
// of letters counts as one edit.
func distance(a, b string) int {
	x, y := []rune(a), []rune(b)
	rows := make([][]int, len(x)+1)
	for i := range rows {
		rows[i] = make([]int, len(y)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(x); i++ {
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && x[i-1] == y[j-2] && x[i-2] == y[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}
	return rows[len(x)][len(y)]
}
