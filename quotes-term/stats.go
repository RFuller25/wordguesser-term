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
	client := m.client
	username := m.username
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := client.GetUserStats(username)
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
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Stats — %s", m.username)) + "\n\n")

	if m.loading {
		sb.WriteString(m.spinner.View() + " Loading stats...\n")
		return sb.String()
	}

	if m.errMsg != "" {
		sb.WriteString(redStyle.Render("Error: "+m.errMsg) + "\n")
		return sb.String()
	}

	if m.stats == nil {
		sb.WriteString(dimStyle.Render("No games played yet.") + "\n")
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
	if s.Played > 0 {
		winPct = float64(s.Won) / float64(s.Played) * 100
	}

	cards := []string{
		cardStyle.Render(labelStyle.Render("Played") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.Played))),
		cardStyle.Render(labelStyle.Render("Won") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.Won))),
		cardStyle.Render(labelStyle.Render("Win %") + "\n" + valueStyle.Render(fmt.Sprintf("%.0f%%", winPct))),
	}

	cards2 := []string{
		cardStyle.Render(labelStyle.Render("Current Streak") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.CurrentStreak))),
		cardStyle.Render(labelStyle.Render("Best Streak") + "\n" + valueStyle.Render(fmt.Sprintf("%d", s.BestStreak))),
	}

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cards...) + "\n\n")
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cards2...) + "\n")

	return sb.String()
}
