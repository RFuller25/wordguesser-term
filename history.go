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
	client  *APIClient
	date    time.Time
	results *ResultsResponse
	loading bool
	spinner spinner.Model
	errMsg  string
}

type historyMsg struct {
	resp *ResultsResponse
	err  error
}

func newHistoryModel(client *APIClient) historyModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return historyModel{
		client:  client,
		date:    time.Now().In(chicagoTZ()),
		spinner: s,
	}
}

func (m historyModel) Init() tea.Cmd {
	return nil
}

func (m *historyModel) Fetch() tea.Cmd {
	m.loading = true
	date := m.date.Format("2006-01-02")
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := m.client.GetResults(date)
		return historyMsg{resp: resp, err: err}
	})
}

func (m historyModel) Update(msg tea.Msg) (historyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case historyMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.results = nil
		} else {
			m.results = msg.resp
			m.errMsg = ""
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
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
	sb.WriteString(dimStyle.Render("  ← / → to change date") + "\n\n")

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

	if m.results.Word != "" {
		sb.WriteString(fmt.Sprintf("  Word: %s\n\n", greenStyle.Bold(true).Render(strings.ToUpper(m.results.Word))))
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	header := fmt.Sprintf("  %-14s %8s %8s %8s", "Player", "Guesses", "Solved", "Status")
	sb.WriteString(headerStyle.Render(header) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", 45)) + "\n")

	for _, r := range m.results.Results {
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

		line := fmt.Sprintf("  %-14s %8d %8s %8s",
			r.Username, r.NumGuesses, solvedStyle.Render(solved), status)
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
