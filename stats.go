package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type statsModel struct {
	client   *APIClient
	stats    *UserStatsResponse
	loading  bool
	spinner  spinner.Model
	errMsg   string
	username string
}

type statsMsg struct {
	resp *UserStatsResponse
	err  error
}

func newStatsModel(client *APIClient, username string) statsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return statsModel{
		client:   client,
		spinner:  s,
		username: username,
	}
}

func (m statsModel) Init() tea.Cmd {
	return nil
}

func (m *statsModel) Fetch() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := m.client.GetUserStats(m.username)
		return statsMsg{resp: resp, err: err}
	})
}

func (m statsModel) Update(msg tea.Msg) (statsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case statsMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.stats = msg.resp
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

func (m statsModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Stats — %s", m.username)) + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading stats...\n")
		return sb.String()
	}

	if m.errMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if m.stats == nil {
		sb.WriteString(dimStyle.Render("No stats available.") + "\n")
		return sb.String()
	}

	s := m.stats
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Padding(0, 2).
		Width(20).
		Align(lipgloss.Center)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))

	winPct := 0.0
	if s.GamesPlayed > 0 {
		winPct = float64(s.GamesWon) / float64(s.GamesPlayed) * 100
	}

	cards := []string{
		cardStyle.Render(labelStyle.Render("Games Played") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.GamesPlayed))),
		cardStyle.Render(labelStyle.Render("Games Won") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.GamesWon))),
		cardStyle.Render(labelStyle.Render("Win %") + "\n" + valueStyle.Render(fmt.Sprintf("%.0f%%", winPct))),
		cardStyle.Render(labelStyle.Render("Avg Tries") + "\n" + valueStyle.Render(fmt.Sprintf("%.1f", s.AvgTries))),
	}

	cards2 := []string{
		cardStyle.Render(labelStyle.Render("Current Streak") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.CurrentStreak))),
		cardStyle.Render(labelStyle.Render("Best Streak") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.BestStreak))),
		cardStyle.Render(labelStyle.Render("Total Points") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.TotalPoints))),
	}

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cards...) + "\n\n")
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cards2...) + "\n")

	return sb.String()
}
