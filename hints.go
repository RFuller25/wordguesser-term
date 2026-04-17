package main

import (
	_ "embed"
	"sort"
	"strings"
)

//go:embed words.txt
var wordsRaw string

var wordList []string

func init() {
	for _, w := range strings.Split(strings.TrimSpace(wordsRaw), "\n") {
		w = strings.TrimSpace(strings.ToLower(w))
		if len([]rune(w)) == 5 {
			wordList = append(wordList, w)
		}
	}
}

// wordConstraints encodes everything the player knows about the secret word.
type wordConstraints struct {
	greens   [5]rune          // green: letter fixed at position (0 = unconstrained)
	excluded [5]map[rune]bool // yellow/gray: letter cannot be at this position
	minCount map[rune]int     // letter must appear at least this many times
	maxCount map[rune]int     // letter must appear at most this many times (set when a gray is seen)
}

func buildConstraints(guesses []GuessEntry) wordConstraints {
	c := wordConstraints{
		minCount: map[rune]int{},
		maxCount: map[rune]int{},
	}
	for i := range c.excluded {
		c.excluded[i] = map[rune]bool{}
	}

	for _, g := range guesses {
		word := []rune(strings.ToLower(g.Word))
		pat := g.Pattern

		// Count how many times each letter appears as green or yellow in this guess.
		// That count is a lower bound on occurrences in the secret word.
		gyCount := map[rune]int{}
		hasGray := map[rune]bool{}

		for i, ch := range word {
			switch rune(pat[i]) {
			case 'g', 'y':
				gyCount[ch]++
			case 'x':
				hasGray[ch] = true
			}
		}

		// Apply positional constraints.
		for i, ch := range word {
			switch rune(pat[i]) {
			case 'g':
				c.greens[i] = ch
			case 'y':
				c.excluded[i][ch] = true
			case 'x':
				// Gray means no extra copies beyond what green/yellow already established.
				// Exclude this position too (the letter isn't here).
				c.excluded[i][ch] = true
			}
		}

		// Propagate min counts.
		for ch, count := range gyCount {
			if c.minCount[ch] < count {
				c.minCount[ch] = count
			}
		}

		// When a gray is present for a letter, we know the exact count:
		// it equals the number of green+yellow occurrences (could be 0).
		for ch := range hasGray {
			count := gyCount[ch]
			if existing, ok := c.maxCount[ch]; !ok || existing > count {
				c.maxCount[ch] = count
			}
		}
	}

	return c
}

func matchesConstraints(word string, c wordConstraints) bool {
	runes := []rune(word)
	if len(runes) != 5 {
		return false
	}

	// Green positions must match exactly.
	for i, ch := range c.greens {
		if ch != 0 && runes[i] != ch {
			return false
		}
	}

	// Yellow/gray exclusions: letter cannot be at this position.
	for i, ch := range runes {
		if c.excluded[i][ch] {
			return false
		}
	}

	// Letter frequency constraints.
	counts := map[rune]int{}
	for _, ch := range runes {
		counts[ch]++
	}
	for ch, min := range c.minCount {
		if counts[ch] < min {
			return false
		}
	}
	for ch, max := range c.maxCount {
		if counts[ch] > max {
			return false
		}
	}

	return true
}

func uniqueLetterCount(word string) int {
	seen := map[rune]bool{}
	for _, ch := range word {
		seen[ch] = true
	}
	return len(seen)
}

// getHints returns up to 20 candidate words that fit the known constraints,
// sorted by most unique letters then alphabetically.
func getHints(guesses []GuessEntry) []string {
	c := buildConstraints(guesses)

	guessedWords := map[string]bool{}
	for _, g := range guesses {
		guessedWords[strings.ToLower(g.Word)] = true
	}

	var candidates []string
	for _, w := range wordList {
		if !guessedWords[w] && matchesConstraints(w, c) {
			candidates = append(candidates, w)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		ui := uniqueLetterCount(candidates[i])
		uj := uniqueLetterCount(candidates[j])
		if ui != uj {
			return ui > uj
		}
		return candidates[i] < candidates[j]
	})

	if len(candidates) > 20 {
		candidates = candidates[:20]
	}
	return candidates
}
