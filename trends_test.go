package main

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestScoreOf(t *testing.T) {
	cases := []struct {
		r    ResultEntry
		want int
	}{
		{ResultEntry{Solved: true, NumGuesses: 3}, 4},
		{ResultEntry{Solved: true, NumGuesses: 1}, 6},
		{ResultEntry{Solved: false, NumGuesses: 6}, 0},
	}
	for _, c := range cases {
		if got := scoreOf(c.r); got != c.want {
			t.Errorf("scoreOf(%+v) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestBuildDayStats_SkipsMissingDays(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Results: []ResultEntry{
			{Username: "alice", Solved: true, Completed: true, NumGuesses: 3},
			{Username: "bob", Solved: false, Completed: true, NumGuesses: 6},
		}},
		"2026-01-03": nil,
	}
	days := buildDayStats([]string{"2026-01-01", "2026-01-02", "2026-01-03"}, byDate)
	if len(days) != 1 {
		t.Fatalf("expected 1 day (missing days skipped), got %d: %+v", len(days), days)
	}
	d := days[0]
	if d.date != "2026-01-01" || d.playerCount != 2 {
		t.Fatalf("unexpected day stat: %+v", d)
	}
	if d.avgScore != 2.0 { // (4 + 0) / 2
		t.Fatalf("expected avgScore 2.0, got %.2f", d.avgScore)
	}
}

func TestBuildDayStats_ExcludesIncompleteFromAverage(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Results: []ResultEntry{
			{Username: "alice", Solved: true, Completed: true, NumGuesses: 2},
			{Username: "bob", Completed: false},
		}},
	}
	days := buildDayStats([]string{"2026-01-01"}, byDate)
	if len(days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(days))
	}
	if days[0].avgScore != 5.0 {
		t.Fatalf("expected avgScore 5.0 (only alice's completed game counted), got %.2f", days[0].avgScore)
	}
	if days[0].playerCount != 2 {
		t.Fatalf("expected playerCount 2 (both listed regardless of completion), got %d", days[0].playerCount)
	}
}

func TestRollingPlayerAverages_TrailingSevenDaysOnlyPlayedDays(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Results: []ResultEntry{{Username: "alice", Solved: true, Completed: true, NumGuesses: 2}}}, // score 5
		"2026-01-02": {Results: []ResultEntry{{Username: "alice", Solved: true, Completed: true, NumGuesses: 4}}}, // score 3
		"2026-01-10": {Results: []ResultEntry{{Username: "alice", Solved: true, Completed: true, NumGuesses: 6}}}, // score 1, >7 days after 01-02
	}
	dates := []string{"2026-01-01", "2026-01-02", "2026-01-10"}
	points := rollingPlayerAverages(dates, byDate)
	alice := points["alice"]
	if len(alice) != 3 {
		t.Fatalf("expected 3 points for alice, got %d: %+v", len(alice), alice)
	}
	if alice[0].value != 5.0 {
		t.Fatalf("expected first point avg 5.0, got %.2f", alice[0].value)
	}
	if alice[1].value != 4.0 { // (5+3)/2, both within trailing 7 days of 01-02
		t.Fatalf("expected second point avg 4.0, got %.2f", alice[1].value)
	}
	if alice[2].value != 1.0 { // window reset: 01-10 is >7 days after 01-02
		t.Fatalf("expected third point avg 1.0, got %.2f", alice[2].value)
	}
}

func TestRollingPlayerAverages_MergesUsernamesDifferingOnlyByCase(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Results: []ResultEntry{{Username: "Sintfoap", Solved: true, Completed: true, NumGuesses: 2}}}, // score 5
		"2026-01-02": {Results: []ResultEntry{{Username: "sintfoap", Solved: true, Completed: true, NumGuesses: 4}}}, // score 3
	}
	dates := []string{"2026-01-01", "2026-01-02"}
	points := rollingPlayerAverages(dates, byDate)
	if len(points) != 1 {
		t.Fatalf("expected a single merged player series, got %d: %+v", len(points), points)
	}
	merged, ok := points["sintfoap"]
	if !ok {
		t.Fatalf("expected series keyed by lowercase username, got keys %v", pointKeys(points))
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 points for the merged player, got %d", len(merged))
	}
	if merged[1].value != 4.0 { // (5+3)/2, both within trailing 7 days
		t.Fatalf("expected merged rolling avg 4.0, got %.2f", merged[1].value)
	}
}

func pointKeys(m map[string][]playerPoint) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestBuildDistribution(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Results: []ResultEntry{
			{Solved: true, Completed: true, NumGuesses: 3},
			{Solved: true, Completed: true, NumGuesses: 3},
			{Solved: false, Completed: true, NumGuesses: 6},
			{Completed: false},
		}},
	}
	dist := buildDistribution([]string{"2026-01-01"}, byDate)
	want := map[string]int{"1": 0, "2": 0, "3": 2, "4": 0, "5": 0, "6": 0, "Fail": 1}
	if len(dist) != 7 {
		t.Fatalf("expected 7 buckets, got %d", len(dist))
	}
	for _, b := range dist {
		if b.count != want[b.label] {
			t.Errorf("bucket %s: got %d, want %d", b.label, b.count, want[b.label])
		}
	}
}

func TestHardestWord_PicksLowestSolveRate(t *testing.T) {
	byDate := map[string]*ResultsResponse{
		"2026-01-01": {Word: "crane", Results: []ResultEntry{
			{Solved: true, Completed: true, NumGuesses: 3},
			{Solved: true, Completed: true, NumGuesses: 3},
		}},
		"2026-01-02": {Word: "zymic", Results: []ResultEntry{
			{Solved: false, Completed: true, NumGuesses: 6},
			{Solved: true, Completed: true, NumGuesses: 6},
		}},
	}
	word, rate, ok := hardestWord([]string{"2026-01-01", "2026-01-02"}, byDate)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if word != "ZYMIC" || rate != 0.5 {
		t.Fatalf("expected ZYMIC at 0.5, got %s at %.2f", word, rate)
	}
}

func TestHardestWord_NoDataReturnsNotOK(t *testing.T) {
	_, _, ok := hardestWord([]string{"2026-01-01"}, map[string]*ResultsResponse{})
	if ok {
		t.Fatal("expected ok=false with no data")
	}
}

func TestBestDay_PicksHighestAvgScore(t *testing.T) {
	days := []dayStat{
		{date: "2026-01-01", avgScore: 3.0},
		{date: "2026-01-02", avgScore: 5.5},
		{date: "2026-01-03", avgScore: 4.0},
	}
	date, score, ok := bestDay(days)
	if !ok || date != "2026-01-02" || score != 5.5 {
		t.Fatalf("expected 2026-01-02 at 5.5, got %s at %.2f (ok=%v)", date, score, ok)
	}
}

func TestBestDay_EmptyReturnsNotOK(t *testing.T) {
	_, _, ok := bestDay(nil)
	if ok {
		t.Fatal("expected ok=false for empty input")
	}
}

func TestWindowDates_ReturnsChronologicalRange(t *testing.T) {
	end := time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)
	dates := windowDates(end)
	if len(dates) != trendsWindowDays {
		t.Fatalf("expected %d dates, got %d", trendsWindowDays, len(dates))
	}
	if dates[len(dates)-1] != "2026-01-30" {
		t.Fatalf("expected last date to be window end, got %s", dates[len(dates)-1])
	}
	if dates[0] != "2026-01-01" {
		t.Fatalf("expected first date 29 days before end, got %s", dates[0])
	}
}

func TestTrendsModel_LeftShiftsWindowBack(t *testing.T) {
	m := newTrendsModel(&resultsStore{fetch: func(string) (*ResultsResponse, error) { return &ResultsResponse{}, nil }, mem: newMemCache[*ResultsResponse]()}, "alice")
	start := m.windowEnd
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatal("expected a fetch cmd")
	}
	if !m2.windowEnd.Equal(start.AddDate(0, 0, -1)) {
		t.Fatal("expected windowEnd shifted back one day")
	}
}

func TestTrendsModel_RightBlockedPastToday(t *testing.T) {
	m := newTrendsModel(&resultsStore{fetch: func(string) (*ResultsResponse, error) { return &ResultsResponse{}, nil }, mem: newMemCache[*ResultsResponse]()}, "alice")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd != nil {
		t.Fatal("expected no cmd when windowEnd is already today")
	}
}

func TestTrendsModel_PanelCyclingWraps(t *testing.T) {
	m := trendsModel{}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.panel != 1 {
		t.Fatalf("expected panel 1, got %d", m.panel)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.panel != 0 {
		t.Fatalf("expected panel wrapped to 0, got %d", m.panel)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if m.panel != 2 {
		t.Fatalf("expected panel wrapped back to 2, got %d", m.panel)
	}
}

func TestTrendsModel_EnterTogglesPlayerModeOnlyOnPlayerPanel(t *testing.T) {
	m := trendsModel{panel: 0}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.playerMode != 0 {
		t.Fatal("expected no toggle on Overview panel")
	}
	m.panel = 1
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.playerMode != 1 {
		t.Fatal("expected toggle to all-overlay mode")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.playerMode != 0 {
		t.Fatal("expected toggle back to single mode")
	}
}

func TestTrendsModel_PlayerCursorOnlyMovesInSingleModeOnPlayerPanel(t *testing.T) {
	m := trendsModel{panel: 1, playerMode: 0, playerNames: []string{"alice", "bob", "carol"}}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.playerCursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.playerCursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.playerCursor != 2 {
		t.Fatalf("expected cursor clamped at 2, got %d", m.playerCursor)
	}
	m.playerMode = 1
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m2.playerCursor != 2 {
		t.Fatal("expected no cursor change in all-overlay mode")
	}
}

func TestTrendsModel_FetchBuildsStatsAndUsesAllWindowDates(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	store := &resultsStore{
		fetch: func(date string) (*ResultsResponse, error) {
			mu.Lock()
			calls[date]++
			mu.Unlock()
			return &ResultsResponse{Date: date, Word: "crane", Results: []ResultEntry{
				{Username: "alice", Solved: true, Completed: true, NumGuesses: 3},
			}}, nil
		},
		mem: newMemCache[*ResultsResponse](),
	}
	m := newTrendsModel(store, "alice")
	cmd := m.Fetch()
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) < 2 {
		t.Fatalf("expected a batch of at least 2 cmds, got %T", cmd())
	}
	tm, ok := batch[1]().(trendsMsg)
	if !ok {
		t.Fatal("expected second batched cmd to produce a trendsMsg")
	}
	if len(tm.days) == 0 {
		t.Fatal("expected non-empty day stats")
	}
	if len(calls) != trendsWindowDays {
		t.Fatalf("expected %d distinct dates fetched, got %d", trendsWindowDays, len(calls))
	}
}

func TestTrendsModel_ViewRendersWithoutPanicking(t *testing.T) {
	m := trendsModel{
		width: 100, height: 40,
		days: []dayStat{{date: "2026-01-01", avgScore: 4.2, playerCount: 3}},
		perPlayer: map[string][]playerPoint{
			"alice": {{date: "2026-01-01", value: 4.0}},
		},
		playerNames:  []string{"alice"},
		distribution: []guessBucket{{label: "3", count: 2}, {label: "Fail", count: 1}},
	}
	for panel := 0; panel < 3; panel++ {
		m.panel = panel
		out := m.View()
		if out == "" {
			t.Fatalf("expected non-empty View() for panel %d", panel)
		}
	}

	m.panel = 1
	m.playerMode = 0
	if out := m.View(); !strings.Contains(out, "alice") {
		t.Fatalf("expected single-player panel to show username, got:\n%s", out)
	}
	m.playerMode = 1
	if out := m.View(); out == "" {
		t.Fatal("expected non-empty View() in all-overlay mode")
	}
}

func TestHardestWordSkipsTodayUntilPlayerFinishes(t *testing.T) {
	today := time.Now().In(chicagoTZ()).Format("2006-01-02")
	yesterday := time.Now().In(chicagoTZ()).AddDate(0, 0, -1).Format("2006-01-02")
	store := &resultsStore{
		fetch: func(date string) (*ResultsResponse, error) {
			switch date {
			case today:
				// Nobody has solved today's word, so it is the hardest so far.
				return &ResultsResponse{Date: date, Word: "zebra", Results: []ResultEntry{
					{Username: "bob", Completed: false, NumGuesses: 1},
					{Username: "alice", Solved: false, Completed: true, NumGuesses: 6},
				}}, nil
			case yesterday:
				return &ResultsResponse{Date: date, Word: "crane", Results: []ResultEntry{
					{Username: "bob", Solved: true, Completed: true, NumGuesses: 3},
				}}, nil
			}
			return &ResultsResponse{Date: date}, nil
		},
		mem: newMemCache[*ResultsResponse](),
	}

	hardestFor := func(username string) string {
		m := newTrendsModel(store, username)
		batch := m.Fetch()().(tea.BatchMsg)
		return batch[1]().(trendsMsg).hardestWord
	}

	if got := hardestFor("bob"); got != "CRANE" {
		t.Fatalf("expected today's word withheld from an unfinished player, got %q", got)
	}
	if got := hardestFor("alice"); got != "ZEBRA" {
		t.Fatalf("expected today's word for a finished player, got %q", got)
	}
}
