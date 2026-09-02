package main

import (
	"strings"
	"testing"
)

func TestFormatAttemptsRemaining(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "No guesses remaining"},
		{1, "1 guess remaining"},
		{2, "2 guesses remaining"},
		{3, "3 guesses remaining"},
		{-1, "No guesses remaining"},
	}
	for _, c := range cases {
		got := formatAttemptsRemaining(c.n)
		if got != c.want {
			t.Errorf("formatAttemptsRemaining(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestOptionLabel(t *testing.T) {
	cases := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{4, "E"},
		{5, "?"},
		{-1, "?"},
	}
	for _, c := range cases {
		got := optionLabel(c.index)
		if got != c.want {
			t.Errorf("optionLabel(%d) = %q, want %q", c.index, got, c.want)
		}
	}
}

func TestGuessedWrong(t *testing.T) {
	m := playModel{attempts: []int{2, 4}}
	for i, want := range []bool{false, true, false, true, false} {
		if got := m.guessedWrong(i); got != want {
			t.Errorf("guessedWrong(%d) = %v, want %v", i, got, want)
		}
	}

	// The winning pick is the final attempt of a won game, never a wrong one.
	won := playModel{attempts: []int{2, 4}, won: true}
	if won.guessedWrong(3) {
		t.Error("expected the winning option not to be marked wrong")
	}
	if !won.guessedWrong(1) {
		t.Error("expected an earlier miss to still be marked wrong")
	}
}

func TestPlayView_MarksIncorrectGuesses(t *testing.T) {
	m := playModel{
		quote:             "The only way out is through.",
		options:           []string{"Robert Frost", "Maya Angelou", "James Baldwin"},
		attempts:          []int{2},
		attemptsRemaining: 2,
		cursor:            0,
	}

	lines := strings.Split(m.View(), "\n")
	var optionLines []string
	for _, l := range lines {
		if strings.Contains(l, ") ") {
			optionLines = append(optionLines, l)
		}
	}
	if len(optionLines) != 3 {
		t.Fatalf("expected 3 option lines, got %d:\n%s", len(optionLines), m.View())
	}
	if !strings.Contains(optionLines[1], "✗") {
		t.Errorf("expected the guessed option to be marked wrong, got %q", optionLines[1])
	}
	for _, i := range []int{0, 2} {
		if strings.Contains(optionLines[i], "✗") {
			t.Errorf("expected option %d unmarked, got %q", i, optionLines[i])
		}
	}
}
