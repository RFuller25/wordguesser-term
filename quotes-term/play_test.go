package main

import "testing"

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
