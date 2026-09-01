package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

const maxAttempts = 3

type playModel struct {
	client            *APIClient
	username          string
	quote             string
	options           []string
	author            string
	attempts          []int
	attemptsRemaining int
	cursor            int
	completed         bool
	won               bool
	score             int
	currentStreak     int
	bestStreak        int
	brokenStreak      int
	loading           bool
	spinner           spinner.Model
	errMsg            string
	flashMsg          string
	flashUntil        time.Time
}

type playStateMsg struct {
	state *StateResponse
	err   error
}

type playTodayMsg struct {
	today *TodayResponse
	err   error
}

type playResultsMsg struct {
	results *ResultsResponse
	err     error
}

type playGuessMsg struct {
	resp *GuessResponse
	err  error
}

type playFlashClearMsg struct{}

func newPlayModel(client *APIClient, username string) playModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return playModel{
		client:            client,
		username:          username,
		attemptsRemaining: maxAttempts,
		loading:           true,
		spinner:           s,
	}
}

func chicagoTZ() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.UTC
	}
	return loc
}

// today returns the current date, Chicago time, as "YYYY-MM-DD" — the
// display/query date used across all quotes-term tabs.
func today() string {
	return time.Now().In(chicagoTZ()).Format("2006-01-02")
}

func (m playModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadState)
}

func (m *playModel) loadState() tea.Msg {
	state, err := m.client.GetState(m.username, today())
	return playStateMsg{state: state, err: err}
}

func (m *playModel) fetchToday() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		t, err := client.GetToday(today())
		return playTodayMsg{today: t, err: err}
	}
}

func (m *playModel) fetchResults() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		r, err := client.GetResults(today())
		return playResultsMsg{results: r, err: err}
	}
}

// AwaitingGuess reports whether the play tab currently wants raw number-key
// input for an option pick, so the app model can suppress number-key tab
// switching while a guess is in progress.
func (m playModel) AwaitingGuess() bool {
	return !m.loading && !m.completed && len(m.options) > 0
}

func (m playModel) Update(msg tea.Msg) (playModel, tea.Cmd) {
	switch msg := msg.(type) {
	case playStateMsg:
		if msg.err != nil {
			m.loading = false
			m.errMsg = msg.err.Error()
			return m, nil
		}
		if msg.state == nil {
			return m, m.fetchToday()
		}
		m.completed = msg.state.Completed
		m.won = msg.state.Won
		m.attempts = msg.state.Attempts
		m.score = msg.state.Score
		if m.completed {
			return m, m.fetchResults()
		}
		return m, m.fetchToday()

	case playTodayMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.quote = msg.today.Quote
		m.options = msg.today.Options
		m.attemptsRemaining = maxAttempts - len(m.attempts)
		return m, nil

	case playResultsMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.quote = msg.results.Quote
		m.author = msg.results.Author
		return m, nil

	case playGuessMsg:
		m.loading = false
		if msg.err != nil {
			m.flashMsg = msg.err.Error()
			m.flashUntil = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return playFlashClearMsg{} })
		}
		m.attempts = msg.resp.Attempts
		m.attemptsRemaining = msg.resp.AttemptsRemaining
		m.completed = msg.resp.Completed
		m.won = msg.resp.Won
		if m.completed {
			m.score = msg.resp.Score
			m.currentStreak = msg.resp.CurrentStreak
			m.bestStreak = msg.resp.BestStreak
			m.brokenStreak = msg.resp.BrokenStreak
			m.loading = true
			return m, m.fetchResults()
		}
		return m, nil

	case playFlashClearMsg:
		if time.Now().After(m.flashUntil) {
			m.flashMsg = ""
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading || m.completed || len(m.options) == 0 {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			return m.submitGuess(m.cursor + 1)
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if r >= '1' && r <= '9' {
					n := int(r - '0')
					if n <= len(m.options) {
						return m.submitGuess(n)
					}
				}
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m playModel) submitGuess(guess int) (playModel, tea.Cmd) {
	m.loading = true
	m.flashMsg = ""
	client := m.client
	date := today()
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := client.SubmitGuess(guess, date)
		return playGuessMsg{resp: resp, err: err}
	})
}

// formatAttemptsRemaining renders the remaining-guesses hint shown under the
// options list.
func formatAttemptsRemaining(n int) string {
	if n <= 0 {
		return "No guesses remaining"
	}
	if n == 1 {
		return "1 guess remaining"
	}
	return fmt.Sprintf("%d guesses remaining", n)
}

// optionLabel maps a 0-based option index to its display letter (A-E). It
// returns "?" for an out-of-range index rather than panicking, since it's
// used purely for rendering.
func optionLabel(index int) string {
	if index < 0 || index > 4 {
		return "?"
	}
	return string(rune('A' + index))
}

func (m playModel) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Quotes") + "\n\n")

	if m.errMsg != "" {
		sb.WriteString(redStyle.Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if m.loading && m.quote == "" {
		sb.WriteString(m.spinner.View() + " Loading...\n")
		return sb.String()
	}

	if m.quote != "" {
		sb.WriteString(brightStyle.Render(fmt.Sprintf("\"%s\"", m.quote)) + "\n\n")
	}

	if m.completed {
		if m.author != "" {
			sb.WriteString(dimStyle.Render("— "+m.author) + "\n\n")
		}
		if m.won {
			sb.WriteString(greenStyle.Render(fmt.Sprintf("You got it in %d/%d! +%d points", len(m.attempts), maxAttempts, m.score)) + "\n")
		} else {
			sb.WriteString(redStyle.Render("Out of guesses.") + "\n")
		}
		if m.currentStreak > 0 || m.bestStreak > 0 {
			sb.WriteString(dimStyle.Render(fmt.Sprintf("Streak: %d (best %d)", m.currentStreak, m.bestStreak)) + "\n")
		}
		return sb.String()
	}

	for i, opt := range m.options {
		label := fmt.Sprintf("%s) %s", optionLabel(i), opt)
		if i == m.cursor {
			sb.WriteString(accentStyle.Bold(true).Render("> "+label) + "\n")
		} else {
			sb.WriteString("  " + label + "\n")
		}
	}

	sb.WriteString("\n")
	if m.loading {
		sb.WriteString(m.spinner.View() + " Submitting...\n")
	} else if m.flashMsg != "" {
		sb.WriteString(redStyle.Render(m.flashMsg) + "\n")
	} else {
		hint := fmt.Sprintf("%s | ↑↓+Enter or press 1-%d to guess", formatAttemptsRemaining(m.attemptsRemaining), len(m.options))
		sb.WriteString(dimStyle.Render(hint) + "\n")
	}

	return sb.String()
}
