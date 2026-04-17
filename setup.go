package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type setupStep int

const (
	stepAPIKey setupStep = iota
	stepUsername
	stepValidating
	stepError
)

type setupModel struct {
	step        setupStep
	apiKeyInput textinput.Model
	userInput   textinput.Model
	spinner     spinner.Model
	errMsg      string
	done        bool
	config      *Config
}

type setupDoneMsg struct {
	config *Config
}

type setupErrMsg struct {
	err error
}

func newSetupModel(existingUsername string) setupModel {
	apiKey := textinput.New()
	apiKey.Placeholder = "Enter your API key"
	apiKey.EchoMode = textinput.EchoPassword
	apiKey.EchoCharacter = '*'
	apiKey.Focus()
	apiKey.Width = 40

	user := textinput.New()
	user.Placeholder = "Enter your username"
	user.Width = 40
	if existingUsername != "" {
		user.SetValue(existingUsername)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1"))

	return setupModel{
		step:        stepAPIKey,
		apiKeyInput: apiKey,
		userInput:   user,
		spinner:     s,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			switch m.step {
			case stepAPIKey:
				if m.apiKeyInput.Value() == "" {
					return m, nil
				}
				m.step = stepUsername
				m.userInput.Focus()
				return m, textinput.Blink
			case stepUsername:
				if m.userInput.Value() == "" {
					return m, nil
				}
				m.step = stepValidating
				apiKey := m.apiKeyInput.Value()
				username := m.userInput.Value()
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					client := newAPIClient(apiKey, username)
					_, err := client.GetLeaderboard()
					if err != nil {
						return setupErrMsg{err: err}
					}
					cfg := &Config{APIKey: apiKey, Username: username}
					if err := saveConfig(cfg); err != nil {
						return setupErrMsg{err: fmt.Errorf("failed to save config: %w", err)}
					}
					return setupDoneMsg{config: cfg}
				})
			case stepError:
				m.step = stepAPIKey
				m.apiKeyInput.Focus()
				m.errMsg = ""
				return m, textinput.Blink
			}
		}

	case setupDoneMsg:
		m.done = true
		m.config = msg.config
		return m, nil

	case setupErrMsg:
		m.step = stepError
		m.errMsg = msg.err.Error()
		return m, nil

	case spinner.TickMsg:
		if m.step == stepValidating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	switch m.step {
	case stepAPIKey:
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	case stepUsername:
		m.userInput, cmd = m.userInput.Update(msg)
	}
	return m, cmd
}

func (m setupModel) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#6366f1")).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginBottom(2)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ef4444")).
		Bold(true)

	var s string
	s += titleStyle.Render("Welcome to WordGuesser!") + "\n"
	s += subtitleStyle.Render("Set up your account to start playing.") + "\n\n"

	switch m.step {
	case stepAPIKey:
		s += "API Key:\n"
		s += m.apiKeyInput.View() + "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Press Enter to continue")

	case stepUsername:
		s += "API Key: ****\n\n"
		s += "Username:\n"
		s += m.userInput.View() + "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Press Enter to continue")

	case stepValidating:
		s += "API Key: ****\n"
		s += "Username: " + m.userInput.Value() + "\n\n"
		s += m.spinner.View() + " Validating..."

	case stepError:
		s += errorStyle.Render("Error: "+m.errMsg) + "\n\n"
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Press Enter to try again")
	}

	return s
}
