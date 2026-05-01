package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#eab308"))
	grayStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1"))

	greenBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#22c55e")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
	yellowBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#eab308")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
	grayBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6b7280")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
	emptyBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#333333")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
	activeBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#555555")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
)

type letterState int

const (
	letterUnused letterState = iota
	letterAbsent
	letterWrongPos
	letterCorrect
)

type gameModel struct {
	client     *APIClient
	guesses    []GuessEntry
	input      string
	completed  bool
	won        bool
	skipped    bool
	revealWord string
	letters    map[rune]letterState
	loading    bool
	spinner    spinner.Model
	flashMsg   string
	flashUntil time.Time
	showHints  bool
}

type guessResultMsg struct {
	resp *GuessResponse
	err  error
}

type gameStateMsg struct {
	state   *GameState
	err     error
	skipped bool
}

type flashClearMsg struct{}

func newGameModel(client *APIClient) gameModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	return gameModel{
		client:  client,
		letters: make(map[rune]letterState),
		spinner: s,
	}
}

func (m gameModel) Init() tea.Cmd {
	return m.loadState
}

func (m *gameModel) loadState() tea.Msg {
	now := time.Now().In(chicagoTZ())
	date := now.Format("2006-01-02")
	wordData, err := m.client.GetWord(date)
	if err == nil && wordData.Skipped {
		return gameStateMsg{skipped: true}
	}
	state, err := m.client.GetGameState(m.client.username, date)
	return gameStateMsg{state: state, err: err}
}

func chicagoTZ() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.UTC
	}
	return loc
}

func (m gameModel) Update(msg tea.Msg) (gameModel, tea.Cmd) {
	switch msg := msg.(type) {
	case gameStateMsg:
		m.loading = false
		if msg.skipped {
			m.skipped = true
			return m, nil
		}
		if msg.err == nil {
			m.guesses = msg.state.Guesses
			m.completed = msg.state.Completed
			m.won = msg.state.Won
			m.updateLetterStates()
		}
		return m, nil

	case guessResultMsg:
		m.loading = false
		if msg.err != nil {
			m.flashMsg = msg.err.Error()
			m.flashUntil = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return flashClearMsg{} })
		}
		if msg.resp.Error == "invalid_word" {
			m.input = ""
			m.flashMsg = "Not a valid word"
			m.flashUntil = time.Now().Add(3 * time.Second)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return flashClearMsg{} })
		}
		m.guesses = msg.resp.Guesses
		m.completed = msg.resp.Completed
		m.won = msg.resp.Won
		m.input = ""
		if msg.resp.Completed {
			m.revealWord = msg.resp.Word
		}
		m.updateLetterStates()
		return m, nil

	case flashClearMsg:
		if time.Now().After(m.flashUntil) {
			m.flashMsg = ""
		}
		return m, nil

	case tea.KeyMsg:
		// Toggle hint panel with ? regardless of game state.
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if r == '?' {
					m.showHints = !m.showHints
					return m, nil
				}
			}
		}

		if m.completed || m.loading {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyEnter:
			if len(m.input) == 5 {
				m.loading = true
				guess := m.input
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					resp, err := m.client.SubmitGuess(strings.ToLower(guess))
					return guessResultMsg{resp: resp, err: err}
				})
			}
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if unicode.IsLetter(r) && len(m.input) < 5 {
					m.input += string(unicode.ToUpper(r))
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

func (m *gameModel) updateLetterStates() {
	for _, g := range m.guesses {
		for i, ch := range strings.ToUpper(g.Word) {
			pat := rune(g.Pattern[i])
			var state letterState
			switch pat {
			case 'g':
				state = letterCorrect
			case 'y':
				state = letterWrongPos
			case 'x':
				state = letterAbsent
			}
			if state > m.letters[ch] {
				m.letters[ch] = state
			}
		}
	}
}

func (m gameModel) InputEmpty() bool {
	return len(m.input) == 0
}

func (m gameModel) View() string {
	if m.skipped {
		return grayStyle.Render("No Wordle today — day called off. Streaks are safe, see you tomorrow!") + "\n"
	}

	var sb strings.Builder

	for i := 0; i < 6; i++ {
		row := m.renderRow(i)
		sb.WriteString(row + "\n")
	}

	sb.WriteString("\n")
	if m.loading {
		sb.WriteString(m.spinner.View() + " Submitting...\n")
	} else if m.flashMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(m.flashMsg) + "\n")
	} else if m.completed {
		if m.won {
			pts := 7 - len(m.guesses)
			sb.WriteString(greenStyle.Render(fmt.Sprintf("You won in %d/6! +%d points", len(m.guesses), pts)) + "\n")
		} else {
			sb.WriteString(grayStyle.Render(fmt.Sprintf("The word was: %s", strings.ToUpper(m.revealWord))) + "\n")
		}
	} else if m.showHints {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Guess %d/6 | ? to hide hints", len(m.guesses)+1)) + "\n")
	} else {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Guess %d/6 | Type a word and press Enter | ? for hints", len(m.guesses)+1)) + "\n")
	}

	sb.WriteString("\n")
	if m.showHints {
		sb.WriteString(m.renderHints())
	} else {
		sb.WriteString(m.renderKeyboard())
	}

	return sb.String()
}

func (m gameModel) renderRow(row int) string {
	cells := make([]string, 5)

	if row < len(m.guesses) {
		word := strings.ToUpper(m.guesses[row].Word)
		pattern := m.guesses[row].Pattern
		for i := 0; i < 5; i++ {
			ch := string(word[i])
			switch rune(pattern[i]) {
			case 'g':
				cells[i] = greenBorder.Render(greenStyle.Bold(true).Render(ch))
			case 'y':
				cells[i] = yellowBorder.Render(yellowStyle.Bold(true).Render(ch))
			case 'x':
				cells[i] = grayBorder.Render(grayStyle.Render(ch))
			}
		}
	} else if row == len(m.guesses) && !m.completed {
		for i := 0; i < 5; i++ {
			if i < len(m.input) {
				cells[i] = activeBorder.Render(brightStyle.Bold(true).Render(string(m.input[i])))
			} else if i == len(m.input) {
				cells[i] = activeBorder.Render(dimStyle.Render("_"))
			} else {
				cells[i] = emptyBorder.Render(" ")
			}
		}
	} else {
		for i := 0; i < 5; i++ {
			cells[i] = emptyBorder.Render(" ")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, cells...)
}

func (m gameModel) renderKeyboard() string {
	rows := []string{"QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM"}
	var lines []string

	for _, row := range rows {
		var keys []string
		for _, ch := range row {
			style := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Width(1).
				Height(0).
				Align(lipgloss.Center)

			letter := string(ch)
			switch m.letters[ch] {
			case letterCorrect:
				style = style.BorderForeground(lipgloss.Color("#22c55e"))
				letter = greenStyle.Bold(true).Render(letter)
			case letterWrongPos:
				style = style.BorderForeground(lipgloss.Color("#eab308"))
				letter = yellowStyle.Bold(true).Render(letter)
			case letterAbsent:
				style = style.BorderForeground(lipgloss.Color("#333333"))
				letter = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Strikethrough(true).Render(letter)
			default:
				style = style.BorderForeground(lipgloss.Color("#444444"))
				letter = dimStyle.Render(letter)
			}
			keys = append(keys, style.Render(letter))
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, keys...))
	}

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func (m gameModel) renderHints() string {
	hints := getHints(m.guesses)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	wordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	var sb strings.Builder

	if len(hints) == 0 {
		sb.WriteString(dimStyle.Render("No matching words found.") + "\n")
		return sb.String()
	}

	sb.WriteString(headerStyle.Render(fmt.Sprintf("Possible words (%d shown)", len(hints))) + "\n\n")

	// Render in 4 columns, colouring letters by known state.
	cols := 4
	for i, w := range hints {
		upper := strings.ToUpper(w)
		var styled strings.Builder
		for _, ch := range upper {
			switch m.letters[ch] {
			case letterCorrect:
				styled.WriteString(greenStyle.Render(string(ch)))
			case letterWrongPos:
				styled.WriteString(yellowStyle.Render(string(ch)))
			default:
				styled.WriteString(wordStyle.Render(string(ch)))
			}
		}
		cell := lipgloss.NewStyle().Width(8).Render(styled.String())
		sb.WriteString(cell)
		if (i+1)%cols == 0 {
			sb.WriteString("\n")
		}
	}
	if len(hints)%cols != 0 {
		sb.WriteString("\n")
	}

	return sb.String()
}
