package main

import (
	"strings"
	"testing"
)

func sampleResults() *ResultsResponse {
	return &ResultsResponse{
		Date:   today(),
		Quote:  "The only way out is through.",
		Author: "Robert Frost",
		Results: []ResultEntry{
			{Username: "alice", Attempts: []int{2, 1}, Won: true, Score: 2, CurrentStreak: 3},
			{Username: "bob", Attempts: []int{3}, Won: false},
			{Username: "carol", Attempts: []int{1, 2, 3}, Won: false},
		},
	}
}

func TestUserCompleted(t *testing.T) {
	r := sampleResults()
	cases := []struct {
		username string
		want     bool
	}{
		{"alice", true}, // won
		{"ALICE", true}, // case-insensitive
		{"carol", true}, // used every attempt
		{"bob", false},  // still guessing
		{"dave", false}, // has not played
	}
	for _, c := range cases {
		if got := userCompleted(r, c.username); got != c.want {
			t.Errorf("userCompleted(%q) = %v, want %v", c.username, got, c.want)
		}
	}
	if userCompleted(nil, "alice") {
		t.Error("nil results should count as unfinished")
	}
}

func TestResultsView_HidesAuthorUntilPlayerFinishes(t *testing.T) {
	m := resultsModel{data: sampleResults(), username: "bob"}
	if strings.Contains(m.View(), "Robert Frost") {
		t.Fatalf("results leaked the author to an unfinished player:\n%s", m.View())
	}

	m.username = "alice"
	if !strings.Contains(m.View(), "Robert Frost") {
		t.Fatalf("expected a finished player to see the author:\n%s", m.View())
	}
}
