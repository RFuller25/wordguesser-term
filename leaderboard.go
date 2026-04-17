package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type leaderboardModel struct {
	client   *APIClient
	entries  []LeaderboardEntry
	loading  bool
	spinner  spinner.Model
	errMsg   string
	username string
}

type leaderboardMsg struct {
	resp *LeaderboardResponse
	err  error
}

func newLeaderboardModel(client *APIClient, username string) leaderboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return leaderboardModel{
		client:   client,
		spinner:  s,
		username: username,
	}
}

func (m leaderboardModel) Init() tea.Cmd {
	return nil
}

func (m *leaderboardModel) Fetch() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := m.client.GetLeaderboard()
		return leaderboardMsg{resp: resp, err: err}
	})
}

func (m leaderboardModel) Update(msg tea.Msg) (leaderboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case leaderboardMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.entries = msg.resp.Leaderboard
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

func (m leaderboardModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Leaderboard") + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading leaderboard...\n")
		return sb.String()
	}

	if m.errMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if len(m.entries) == 0 {
		sb.WriteString(dimStyle.Render("No players yet.") + "\n")
		return sb.String()
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888"))
	header := fmt.Sprintf("  %-4s %-14s %6s %6s %5s %6s %6s %6s",
		"#", "Player", "Points", "Games", "Wins", "Avg", "Streak", "Best")
	sb.WriteString(headerStyle.Render(header) + "\n")
	sb.WriteString(dimStyle.Render(strings.Repeat("─", 70)) + "\n")

	for i, e := range m.entries {
		line := fmt.Sprintf("  %-4d %-14s %6d %6d %5d %6.1f %6d %6d",
			i+1, e.Username, e.TotalPoints, e.GamesPlayed, e.GamesWon,
			e.AvgTries, e.CurrentStreak, e.BestStreak)

		if strings.EqualFold(e.Username, m.username) {
			sb.WriteString(accentStyle.Bold(true).Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	return sb.String()
}
