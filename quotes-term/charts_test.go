package main

import (
	"strings"
	"testing"
)

func TestRollingWinAverages(t *testing.T) {
	history := []HistoryGame{
		{Won: true, Attempts: 1},
		{Won: false, Attempts: 3}, // losses are skipped
		{Won: true, Attempts: 3},
		{Won: true, Attempts: 2},
	}
	got := rollingWinAverages(history, 2)
	want := []float64{1, 2, 2.5}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if all := rollingWinAverages(history, 0); all[2] != 2 {
		t.Errorf("window 0 should average everything, got %v", all)
	}
}

func TestRenderDistribution(t *testing.T) {
	out := renderDistribution([]int{1, 3, 0, 0})
	for _, want := range []string{"1 ", "Fail", "3 (75%)", "1 (25%)"} {
		if !strings.Contains(out, want) {
			t.Errorf("distribution missing %q:\n%s", want, out)
		}
	}
	if renderDistribution(nil) != "" {
		t.Error("malformed distribution should render nothing")
	}
}

func TestRenderRollingSparkline(t *testing.T) {
	if renderRollingSparkline(nil, 10) != "" {
		t.Error("no wins should render nothing")
	}
	out := renderRollingSparkline([]HistoryGame{{Won: true, Attempts: 1}, {Won: true, Attempts: 3}}, 10)
	if !strings.Contains(out, "2.00") {
		t.Errorf("expected latest rolling average 2.00, got %q", out)
	}
}

func TestStatsViewShowsFirstTryStreaks(t *testing.T) {
	m := statsModel{username: "alice", stats: &UserStatsResponse{
		Played: 5, Won: 4, FirstTryWins: 3, FirstTryPct: 60,
		CurrentFirstTryStreak: 2, BestFirstTryStreak: 7,
		AttemptDist: []int{1, 3, 1, 0},
		History:     []HistoryGame{{Won: true, Attempts: 1}},
	}}
	view := m.View()
	for _, want := range []string{"1st-Try Streak", "Best 1st-Try", "Attempts to win", "Rolling avg"} {
		if !strings.Contains(view, want) {
			t.Errorf("stats view missing %q", want)
		}
	}
}
