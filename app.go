package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabGame tab = iota
	tabLeaderboard
	tabStats
	tabHistory
)

var tabNames = []string{"1 Game", "2 Leaderboard", "3 Stats", "4 History"}

type tickMsg time.Time

type appModel struct {
	activeTab   tab
	game        gameModel
	leaderboard leaderboardModel
	stats       statsModel
	history     historyModel
	setup       setupModel
	needsSetup  bool
	bubbleField bubbleField
	config      *Config
	username    string
	width       int
	height      int
	client      *APIClient
}

func newApp(cfg *Config, username string, needsSetup bool) appModel {
	m := appModel{
		config:     cfg,
		username:   username,
		needsSetup: needsSetup,
	}

	if needsSetup {
		m.setup = newSetupModel(username)
	} else {
		m.client = newAPIClient(cfg.APIKey, username)
		m.game = newGameModel(m.client)
		m.leaderboard = newLeaderboardModel(m.client, username)
		m.stats = newStatsModel(m.client, username)
		m.history = newHistoryModel(m.client)
	}

	return m
}

func (m appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.tickCmd(),
	}
	if m.needsSetup {
		cmds = append(cmds, m.setup.Init())
	} else {
		cmds = append(cmds, m.game.Init())
	}
	return tea.Batch(cmds...)
}

func (m appModel) tickCmd() tea.Cmd {
	return tea.Tick(60*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.bubbleField.resize(msg.Width, msg.Height)
		return m, nil

	case tickMsg:
		m.bubbleField.update()
		cmds = append(cmds, m.tickCmd())

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

		if m.needsSetup {
			var cmd tea.Cmd
			m.setup, cmd = m.setup.Update(msg)
			if m.setup.done {
				m.needsSetup = false
				m.config = m.setup.config
				m.username = m.config.Username
				m.client = newAPIClient(m.config.APIKey, m.username)
				m.game = newGameModel(m.client)
				m.leaderboard = newLeaderboardModel(m.client, m.username)
				m.stats = newStatsModel(m.client, m.username)
				m.history = newHistoryModel(m.client)
				return m, tea.Batch(m.tickCmd(), m.game.Init())
			}
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Tab switching
		switched := false
		switch msg.Type {
		case tea.KeyTab:
			m.activeTab = (m.activeTab + 1) % 4
			switched = true
		case tea.KeyShiftTab:
			m.activeTab = (m.activeTab + 3) % 4
			switched = true
		case tea.KeyRunes:
			if m.activeTab != tabGame || m.game.InputEmpty() {
				switch string(msg.Runes) {
				case "1":
					m.activeTab = tabGame
					switched = true
				case "2":
					m.activeTab = tabLeaderboard
					switched = true
				case "3":
					m.activeTab = tabStats
					switched = true
				case "4":
					m.activeTab = tabHistory
					switched = true
				}
			}
		}

		if switched {
			switch m.activeTab {
			case tabLeaderboard:
				cmds = append(cmds, m.leaderboard.Fetch())
			case tabStats:
				cmds = append(cmds, m.stats.Fetch())
			case tabHistory:
				cmds = append(cmds, m.history.Fetch())
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Delegate to active view
	if !m.needsSetup {
		var cmd tea.Cmd
		switch m.activeTab {
		case tabGame:
			m.game, cmd = m.game.Update(msg)
		case tabLeaderboard:
			m.leaderboard, cmd = m.leaderboard.Update(msg)
		case tabStats:
			m.stats, cmd = m.stats.Update(msg)
		case tabHistory:
			m.history, cmd = m.history.Update(msg)
		}
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.setup, cmd = m.setup.Update(msg)
		if m.setup.done {
			m.needsSetup = false
			m.config = m.setup.config
			m.username = m.config.Username
			m.client = newAPIClient(m.config.APIKey, m.username)
			m.game = newGameModel(m.client)
			m.leaderboard = newLeaderboardModel(m.client, m.username)
			m.stats = newStatsModel(m.client, m.username)
			m.history = newHistoryModel(m.client)
			cmds = append(cmds, m.game.Init())
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	if m.width == 0 {
		return ""
	}

	if m.bubbleField.width == 0 {
		m.bubbleField = newBubbleField(m.width, m.height)
	}

	bg := m.bubbleField.view()

	var fg string
	if m.needsSetup {
		fg = m.setup.View()
	} else {
		fg = m.renderTabBar() + "\n\n"
		switch m.activeTab {
		case tabGame:
			fg += m.game.View()
		case tabLeaderboard:
			fg += m.leaderboard.View()
		case tabStats:
			fg += m.stats.View()
		case tabHistory:
			fg += m.history.View()
		}
		fg += "\n" + dimStyle.Render("Tab/1-4: switch views | Ctrl+C: quit")
	}

	return overlay(bg, fg, m.width, m.height)
}

func (m appModel) renderTabBar() string {
	var tabs []string
	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#6366f1")).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 2)

	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabs = append(tabs, activeStyle.Render(name))
		} else {
			tabs = append(tabs, inactiveStyle.Render(name))
		}
	}

	usernameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Padding(0, 1)

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	tabBar += usernameStyle.Render("  " + m.username)

	border := lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#333333"))

	return border.Render(tabBar)
}

func overlay(bg, fg string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	fgStart := (height - len(fgLines)) / 2
	if fgStart < 0 {
		fgStart = 0
	}

	result := make([]string, len(bgLines))
	copy(result, bgLines)

	for i, line := range fgLines {
		row := fgStart + i
		if row >= 0 && row < len(result) && strings.TrimSpace(line) != "" {
			lineWidth := lipgloss.Width(line)
			pad := (width - lineWidth) / 2
			if pad < 0 {
				pad = 0
			}
			result[row] = strings.Repeat(" ", pad) + line
		}
	}

	return strings.Join(result, "\n")
}
