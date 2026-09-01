package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab int

const (
	tabPlay tab = iota
	tabLeaderboard
	tabStats
	tabResults
)

const tabCount = 4

var tabNames = []string{"1 Play", "2 Leaderboard", "3 Stats", "4 Results"}

type tickMsg time.Time

type appModel struct {
	activeTab   tab
	play        playModel
	leaderboard leaderboardModel
	stats       statsModel
	results     resultsModel
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
		m.play = newPlayModel(m.client, username)
		m.leaderboard = newLeaderboardModel(m.client, username)
		m.stats = newStatsModel(m.client, username)
		m.results = newResultsModel(m.client, username)
	}

	return m
}

// startSession (re)builds the client and every tab model from m.config and
// m.username. Called once at construction (implicitly, via newApp) and once
// more after setup completes.
func (m *appModel) startSession() tea.Cmd {
	m.client = newAPIClient(m.config.APIKey, m.username)
	m.play = newPlayModel(m.client, m.username)
	m.leaderboard = newLeaderboardModel(m.client, m.username)
	m.stats = newStatsModel(m.client, m.username)
	m.results = newResultsModel(m.client, m.username)
	return m.play.Init()
}

func (m appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tickCmd()}
	if m.needsSetup {
		cmds = append(cmds, m.setup.Init())
	} else {
		cmds = append(cmds, m.play.Init())
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
				sessionCmd := m.startSession()
				return m, tea.Batch(m.tickCmd(), sessionCmd)
			}
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switched := false
		switch msg.Type {
		case tea.KeyTab:
			m.activeTab = (m.activeTab + 1) % tabCount
			switched = true
		case tea.KeyShiftTab:
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			switched = true
		case tea.KeyRunes:
			blocked := m.activeTab == tabPlay && m.play.AwaitingGuess()
			if !blocked {
				switch string(msg.Runes) {
				case "1":
					m.activeTab = tabPlay
					switched = true
				case "2":
					m.activeTab = tabLeaderboard
					switched = true
				case "3":
					m.activeTab = tabStats
					switched = true
				case "4":
					m.activeTab = tabResults
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
			case tabResults:
				cmds = append(cmds, m.results.Fetch())
			}
			return m, tea.Batch(cmds...)
		}
	}

	if !m.needsSetup {
		var cmd tea.Cmd
		switch m.activeTab {
		case tabPlay:
			m.play, cmd = m.play.Update(msg)
		case tabLeaderboard:
			m.leaderboard, cmd = m.leaderboard.Update(msg)
		case tabStats:
			m.stats, cmd = m.stats.Update(msg)
		case tabResults:
			m.results, cmd = m.results.Update(msg)
		}
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.setup, cmd = m.setup.Update(msg)
		if m.setup.done {
			m.needsSetup = false
			m.config = m.setup.config
			m.username = m.config.Username
			cmds = append(cmds, m.startSession())
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
		case tabPlay:
			fg += m.play.View()
		case tabLeaderboard:
			fg += m.leaderboard.View()
		case tabStats:
			fg += m.stats.View()
		case tabResults:
			fg += m.results.View()
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
