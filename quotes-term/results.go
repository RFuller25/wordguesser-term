package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type resultsModel struct {
	client   *APIClient
	data     *ResultsResponse
	loading  bool
	spinner  spinner.Model
	errMsg   string
	username string
}

type resultsMsg struct {
	resp *ResultsResponse
	err  error
}

func newResultsModel(client *APIClient, username string) resultsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return resultsModel{
		client:   client,
		spinner:  s,
		username: username,
	}
}

func (m resultsModel) Init() tea.Cmd {
	return nil
}

func (m *resultsModel) Fetch() tea.Cmd {
	m.loading = true
	client := m.client
	date := today()
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := client.GetResults(date)
		return resultsMsg{resp: resp, err: err}
	})
}

func (m resultsModel) Update(msg tea.Msg) (resultsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case resultsMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.data = msg.resp
			m.errMsg = ""
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// userCompleted reports whether username has finished today's quote (won or
// used every attempt) according to resp. A nil resp, or a user with no entry
// in it, counts as unfinished — the author stays hidden until they play.
func userCompleted(resp *ResultsResponse, username string) bool {
	if resp == nil {
		return false
	}
	for _, r := range resp.Results {
		if strings.EqualFold(r.Username, username) {
			return r.Won || len(r.Attempts) >= maxAttempts
		}
	}
	return false
}

func (m resultsModel) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Today's Results") + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading results...\n")
		return sb.String()
	}

	if m.errMsg != "" {
		sb.WriteString(redStyle.Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if m.data == nil || m.data.Quote == "" {
		sb.WriteString(dimStyle.Render("No quote set for today yet.") + "\n")
		return sb.String()
	}

	sb.WriteString(brightStyle.Render(fmt.Sprintf("\"%s\"", m.data.Quote)) + "\n")
	if userCompleted(m.data, m.username) {
		sb.WriteString(dimStyle.Render("— "+m.data.Author) + "\n\n")
	} else {
		sb.WriteString(dimStyle.Render("— hidden until you finish today's quote") + "\n\n")
	}

	if len(m.data.Results) == 0 {
		sb.WriteString(dimStyle.Render("No one has played yet.") + "\n")
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	header := fmt.Sprintf("  %-14s %8s %6s %6s %6s", "Player", "Attempts", "Won", "Score", "Streak")
	sb.WriteString(headerStyle.Render(header) + "\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("─", 50)) + "\n")

	for _, r := range m.data.Results {
		won := "no"
		if r.Won {
			won = "yes"
		}
		line := fmt.Sprintf("  %-14s %8d %6s %6d %6d", r.Username, len(r.Attempts), won, r.Score, r.CurrentStreak)
		if strings.EqualFold(r.Username, m.username) {
			sb.WriteString(accentStyle.Bold(true).Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}
