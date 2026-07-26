package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type historyModel struct {
	resultsStore  *resultsStore
	statsStore    *statsStore
	date          time.Time
	results       *ResultsResponse
	loading       bool
	spinner       spinner.Model
	errMsg        string
	cursor        int
	viewingPlayer bool
	streakUsername string
	streak         *UserStatsResponse
}

type historyMsg struct {
	resp *ResultsResponse
	err  error
}

type historyStreakMsg struct {
	username string
	stats    *UserStatsResponse
	err      error
}

func newHistoryModel(store *resultsStore, statsStore *statsStore) historyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return historyModel{
		resultsStore: store,
		statsStore:   statsStore,
		date:         time.Now().In(chicagoTZ()),
		spinner:      s,
	}
}

func (m historyModel) Init() tea.Cmd {
	return nil
}

func (m *historyModel) Fetch() tea.Cmd {
	m.loading = true
	date := m.date.Format("2006-01-02")
	today := time.Now().In(chicagoTZ()).Format("2006-01-02")
	store := m.resultsStore
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := store.Get(date, isFinalDate(date, today))
		return historyMsg{resp: resp, err: err}
	})
}

func (m historyModel) Update(msg tea.Msg) (historyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case historyMsg:
		m.loading = false
		m.cursor = 0
		m.viewingPlayer = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.results = nil
		} else {
			m.results = msg.resp
			m.errMsg = ""
		}
		return m, nil

	case historyStreakMsg:
		if msg.err == nil {
			m.streakUsername = msg.username
			m.streak = msg.stats
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		if m.viewingPlayer {
			if msg.Type == tea.KeyEsc {
				m.viewingPlayer = false
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyLeft:
			m.date = m.date.AddDate(0, 0, -1)
			return m, m.Fetch()
		case tea.KeyRight:
			tomorrow := m.date.AddDate(0, 0, 1)
			today := time.Now().In(chicagoTZ())
			if !tomorrow.After(today) {
				m.date = tomorrow
				return m, m.Fetch()
			}
		case tea.KeyRunes:
			if string(msg.Runes) == "t" {
				today := time.Now().In(chicagoTZ())
				if m.date.Format("2006-01-02") != today.Format("2006-01-02") {
					m.date = today
					return m, m.Fetch()
				}
			}
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.results != nil && m.cursor < len(m.results.Results)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			if m.results != nil && len(m.results.Results) > 0 {
				m.viewingPlayer = true
				username := m.results.Results[m.cursor].Username
				statsStore := m.statsStore
				return m, func() tea.Msg {
					stats, err := statsStore.Get(username)
					return historyStreakMsg{username: username, stats: stats, err: err}
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

func (m historyModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("History") + "\n\n")

	dateStr := m.date.Format("2006-01-02")
	dayName := m.date.Format("Monday")
	sb.WriteString(fmt.Sprintf("  ◀  %s (%s)  ▶\n", accentStyle.Bold(true).Render(dateStr), dayName))
	sb.WriteString(dimStyle.Render("  ← / → to change date | ↑ / ↓ select player | Enter view board | t: today") + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading results...\n")
		return sb.String()
	}

	if m.errMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if m.results == nil || len(m.results.Results) == 0 {
		sb.WriteString(dimStyle.Render("No results for this date.") + "\n")
		return sb.String()
	}

	if m.viewingPlayer {
		return m.renderPlayerBoard()
	}

	if m.results.Word != "" {
		sb.WriteString(fmt.Sprintf("  Word: %s\n\n", greenStyle.Bold(true).Render(strings.ToUpper(m.results.Word))))
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	header := fmt.Sprintf("  %-14s %8s %8s %-12s", "Player", "Guesses", "Solved", "Status")
	sb.WriteString(headerStyle.Render(header) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 47)) + "\n")

	for i, r := range m.results.Results {
		solved := "✗"
		solvedStyle := grayStyle
		if r.Solved {
			solved = "✓"
			solvedStyle = greenStyle
		}

		status := "In progress"
		if r.Completed {
			if r.Solved {
				status = fmt.Sprintf("%d/6", r.NumGuesses)
			} else {
				status = "Failed"
			}
		}

		solvedCell := fmt.Sprintf("%8s", solved)
		line := fmt.Sprintf("  %-14s %8d %s %-12s",
			r.Username, r.NumGuesses, solvedStyle.Render(solvedCell), status)

		if i == m.cursor {
			sb.WriteString(accentStyle.Bold(true).Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}

func (m historyModel) renderPlayerBoard() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)
	r := m.results.Results[m.cursor]

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("History") + "\n\n")
	sb.WriteString(fmt.Sprintf("  %s\n", accentStyle.Bold(true).Render(r.Username)))

	if m.results.Word != "" {
		sb.WriteString(fmt.Sprintf("  Word: %s\n", greenStyle.Bold(true).Render(strings.ToUpper(m.results.Word))))
	}

	status := "In progress"
	if r.Completed {
		if r.Solved {
			status = fmt.Sprintf("Solved in %d/6", r.NumGuesses)
		} else {
			status = "Failed"
		}
	}
	sb.WriteString(dimStyle.Render("  "+status) + "\n")

	if m.streakUsername == r.Username && m.streak != nil {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  Current streak: %d", m.streak.CurrentStreak)) + "\n")
	}
	sb.WriteString("\n")

	grid := renderGuessGrid(r.Guesses, r.Patterns, 6)
	for _, line := range strings.Split(grid, "\n") {
		sb.WriteString("  " + line + "\n")
	}

	sb.WriteString("\n" + dimStyle.Render("  Esc to go back") + "\n")
	return sb.String()
}
