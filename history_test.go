package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func sampleResults() *ResultsResponse {
	return &ResultsResponse{
		Date: "2026-07-25",
		Word: "crane",
		Results: []ResultEntry{
			{Username: "alice", Guesses: []string{"board", "crane"}, Patterns: []string{"xyxxx", "ggggg"}, Solved: true, Completed: true, NumGuesses: 2},
			{Username: "bob", Guesses: []string{"crane"}, Patterns: []string{"ggggg"}, Solved: true, Completed: true, NumGuesses: 1},
		},
	}
}

func TestHistoryModel_CursorMovesWithinBounds(t *testing.T) {
	m := historyModel{results: sampleResults()}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.cursor != 1 {
		t.Fatalf("expected cursor 1 after Down, got %d", m2.cursor)
	}

	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m3.cursor != 1 {
		t.Fatalf("expected cursor clamped at 1 (last row), got %d", m3.cursor)
	}

	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m4.cursor != 0 {
		t.Fatalf("expected cursor 0 after Up, got %d", m4.cursor)
	}

	m5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m5.cursor != 0 {
		t.Fatalf("expected cursor clamped at 0, got %d", m5.cursor)
	}
}

func TestHistoryModel_EnterEntersBoardMode(t *testing.T) {
	m := historyModel{results: sampleResults()}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m2.viewingPlayer {
		t.Fatal("expected viewingPlayer true after Enter")
	}
}

func TestHistoryModel_EnterNoOpWhenNoResults(t *testing.T) {
	m := historyModel{results: &ResultsResponse{Results: nil}}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m2.viewingPlayer {
		t.Fatal("expected viewingPlayer to stay false with no results")
	}
}

func TestHistoryModel_FetchResetsCursorAndViewMode(t *testing.T) {
	m := historyModel{results: sampleResults(), cursor: 1, viewingPlayer: true}

	m2, _ := m.Update(historyMsg{resp: sampleResults()})
	if m2.cursor != 0 {
		t.Fatalf("expected cursor reset to 0 on fetch, got %d", m2.cursor)
	}
	if m2.viewingPlayer {
		t.Fatal("expected viewingPlayer reset to false on fetch")
	}
}

func TestHistoryModel_EscExitsBoardMode(t *testing.T) {
	m := historyModel{results: sampleResults(), viewingPlayer: true}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m2.viewingPlayer {
		t.Fatal("expected viewingPlayer false after Esc")
	}
}

func TestHistoryModel_ArrowsIgnoredInBoardMode(t *testing.T) {
	m := historyModel{results: sampleResults(), viewingPlayer: true, cursor: 0}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.cursor != 0 {
		t.Fatalf("expected cursor unchanged in board mode, got %d", m2.cursor)
	}
	if !m2.viewingPlayer {
		t.Fatal("expected viewingPlayer to remain true")
	}
}

func TestHistoryModel_RowsHaveConsistentVisibleWidth(t *testing.T) {
	m := historyModel{results: &ResultsResponse{
		Results: []ResultEntry{
			{Username: "loneword", NumGuesses: 6, Solved: false, Completed: true},    // status "Failed"
			{Username: "twowords", NumGuesses: 3, Solved: true, Completed: true},     // status "3/6"
			{Username: "threewords", NumGuesses: 1, Solved: false, Completed: false}, // status "In progress"
		},
	}}
	view := m.View()

	var widths []int
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "loneword") || strings.Contains(line, "twowords") || strings.Contains(line, "threewords") {
			widths = append(widths, lipgloss.Width(line))
		}
	}
	if len(widths) != 3 {
		t.Fatalf("expected to find 3 player rows, found %d in:\n%s", len(widths), view)
	}
	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Fatalf("expected all row widths equal, got %v", widths)
		}
	}
}

func TestHistoryModel_RenderPlayerBoardShowsWord(t *testing.T) {
	m := historyModel{
		viewingPlayer: true,
		results: &ResultsResponse{
			Word: "crane",
			Results: []ResultEntry{
				{Username: "alice", Guesses: []string{"board"}, Patterns: []string{"xxxxx"}},
			},
		},
	}
	out := m.View()
	if !strings.Contains(out, "CRANE") {
		t.Fatalf("expected board view to show the word CRANE, got:\n%s", out)
	}
}

func TestHistoryModel_JumpToToday(t *testing.T) {
	m := historyModel{
		results: &ResultsResponse{Results: []ResultEntry{{Username: "alice"}}},
		date:    time.Now().In(chicagoTZ()).AddDate(0, 0, -10),
	}
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if cmd == nil {
		t.Fatal("expected a fetch cmd to be returned")
	}
	today := time.Now().In(chicagoTZ()).Format("2006-01-02")
	if m2.date.Format("2006-01-02") != today {
		t.Fatalf("expected date to jump to today (%s), got %s", today, m2.date.Format("2006-01-02"))
	}
}

func TestHistoryModel_JumpToTodayNoOpWhenAlreadyToday(t *testing.T) {
	m := historyModel{
		results: &ResultsResponse{Results: []ResultEntry{{Username: "alice"}}},
		date:    time.Now().In(chicagoTZ()),
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	if cmd != nil {
		t.Fatal("expected no cmd when already on today's date")
	}
}

func TestHistoryModel_EnterTriggersStreakFetch(t *testing.T) {
	var gotUsername string
	store := &statsStore{
		fetch: func(username string) (*UserStatsResponse, error) {
			gotUsername = username
			return &UserStatsResponse{Username: username, CurrentStreak: 5}, nil
		},
		mem: newMemCache[*UserStatsResponse](),
	}
	m := historyModel{results: sampleResults(), statsStore: store}

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m2.viewingPlayer {
		t.Fatal("expected viewingPlayer true")
	}
	if cmd == nil {
		t.Fatal("expected a streak-fetch cmd")
	}
	msg := cmd()
	sm, ok := msg.(historyStreakMsg)
	if !ok {
		t.Fatalf("expected historyStreakMsg, got %T", msg)
	}
	if sm.username != "alice" || sm.stats.CurrentStreak != 5 {
		t.Fatalf("unexpected streak msg: %+v", sm)
	}
	if gotUsername != "alice" {
		t.Fatalf("expected fetch called with alice, got %q", gotUsername)
	}
}

func TestHistoryModel_HistoryStreakMsgUpdatesState(t *testing.T) {
	m := historyModel{results: sampleResults(), viewingPlayer: true}
	m2, _ := m.Update(historyStreakMsg{username: "alice", stats: &UserStatsResponse{CurrentStreak: 9}})
	if m2.streakUsername != "alice" || m2.streak == nil || m2.streak.CurrentStreak != 9 {
		t.Fatalf("expected streak state updated, got %+v", m2)
	}
}

func TestHistoryModel_RenderPlayerBoardShowsStreak(t *testing.T) {
	m := historyModel{
		results:        sampleResults(),
		viewingPlayer:  true,
		cursor:         0,
		streakUsername: "alice",
		streak:         &UserStatsResponse{CurrentStreak: 12},
	}
	out := m.View()
	if !strings.Contains(out, "Current streak: 12") {
		t.Fatalf("expected streak line, got:\n%s", out)
	}
}

// todayResults returns today's (Chicago) results with alice finished and bob
// still playing.
func todayResults() *ResultsResponse {
	return &ResultsResponse{
		Date: time.Now().In(chicagoTZ()).Format("2006-01-02"),
		Word: "crane",
		Results: []ResultEntry{
			{Username: "alice", Guesses: []string{"board", "crane"}, Patterns: []string{"xyxxx", "ggggg"}, Solved: true, Completed: true, NumGuesses: 2},
			{Username: "bob", Guesses: []string{"slate"}, Patterns: []string{"xxxxy"}, Completed: false, NumGuesses: 1},
		},
	}
}

func TestHistoryModel_TodayWordHiddenUntilPlayerFinishes(t *testing.T) {
	m := historyModel{date: time.Now().In(chicagoTZ()), results: todayResults(), username: "bob"}

	if !m.spoilersHidden() {
		t.Fatal("expected spoilers hidden for a player still in progress")
	}
	if strings.Contains(strings.ToUpper(m.renderWordLine()), "CRANE") {
		t.Fatalf("word line leaked the answer: %q", m.renderWordLine())
	}

	m.username = "alice"
	if m.spoilersHidden() {
		t.Fatal("expected spoilers shown for a player who finished")
	}
	if !strings.Contains(strings.ToUpper(m.renderWordLine()), "CRANE") {
		t.Fatalf("expected the answer for a finished player, got %q", m.renderWordLine())
	}
}

func TestHistoryModel_TodayWordHiddenWhenPlayerHasNoEntry(t *testing.T) {
	m := historyModel{date: time.Now().In(chicagoTZ()), results: todayResults(), username: "carol"}
	if !m.spoilersHidden() {
		t.Fatal("expected spoilers hidden for a player who has not played today")
	}
}

func TestHistoryModel_PastDateAlwaysRevealsWord(t *testing.T) {
	m := historyModel{
		date:     time.Now().In(chicagoTZ()).AddDate(0, 0, -1),
		results:  sampleResults(),
		username: "bob",
	}
	if m.spoilersHidden() {
		t.Fatal("expected past dates to be revealed regardless of today's game")
	}
	if !strings.Contains(strings.ToUpper(m.renderWordLine()), "CRANE") {
		t.Fatalf("expected past date to show the word, got %q", m.renderWordLine())
	}
}

func TestHistoryModel_PlayerBoardMasksLettersUntilPlayerFinishes(t *testing.T) {
	m := historyModel{
		date:          time.Now().In(chicagoTZ()),
		results:       todayResults(),
		username:      "bob",
		viewingPlayer: true,
		cursor:        0, // alice, who has solved it
	}

	board := strings.ToUpper(m.renderPlayerBoard())
	for _, ch := range []string{"C", "R", "A", "N", "E", "B", "O", "D"} {
		if strings.Contains(board, "│ "+ch+" │") {
			t.Fatalf("player board leaked letter %s:\n%s", ch, board)
		}
	}

	m.username = "alice"
	revealed := strings.ToUpper(m.renderPlayerBoard())
	if !strings.Contains(revealed, "│ C │") {
		t.Fatalf("expected a finished player to see the letters:\n%s", revealed)
	}
}

func TestUserCompleted(t *testing.T) {
	if userCompleted(nil, "alice") {
		t.Fatal("nil results should count as unfinished")
	}
	r := todayResults()
	if !userCompleted(r, "ALICE") {
		t.Fatal("expected case-insensitive match for a finished player")
	}
	if userCompleted(r, "bob") {
		t.Fatal("expected bob to count as unfinished")
	}
}
