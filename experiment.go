package main

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	blueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6"))
	blueBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3b82f6")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	redBorder  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#ef4444")).Width(3).Height(1).Align(lipgloss.Center, lipgloss.Center)
)

type experimentModel struct {
	rows      [7]string
	cursor    int
	showHints bool
	letters   map[rune]letterState
}

func newExperimentModel() experimentModel {
	return experimentModel{
		cursor:  0,
		letters: make(map[rune]letterState),
	}
}

func (m experimentModel) Init() tea.Cmd { return nil }

func (m experimentModel) Update(msg tea.Msg) (experimentModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if key.Type == tea.KeyRunes {
		for _, r := range key.Runes {
			if r == '?' {
				m.showHints = !m.showHints
				return m, nil
			}
		}
	}

	switch key.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < 6 {
			m.cursor++
		}
	case tea.KeyBackspace:
		if len(m.rows[m.cursor]) > 0 {
			m.rows[m.cursor] = m.rows[m.cursor][:len(m.rows[m.cursor])-1]
		}
	case tea.KeyEnter:
		if m.cursor < 6 {
			m.cursor++
		}
	case tea.KeyRunes:
		for _, r := range key.Runes {
			if unicode.IsLetter(r) && len(m.rows[m.cursor]) < 5 {
				m.rows[m.cursor] += string(unicode.ToUpper(r))
			}
		}
	}

	m.recomputeLetters()
	return m, nil
}

func (m *experimentModel) recomputeLetters() {
	m.letters = make(map[rune]letterState)
	target := m.rows[0]
	if len(target) != 5 {
		return
	}
	for i := 1; i < 7; i++ {
		guess := m.rows[i]
		if len(guess) != 5 {
			continue
		}
		pat := computePattern(guess, target)
		for j, ch := range guess {
			var s letterState
			switch pat[j] {
			case 'g':
				s = letterCorrect
			case 'y':
				s = letterWrongPos
			case 'x':
				s = letterAbsent
			}
			if s > m.letters[ch] {
				m.letters[ch] = s
			}
		}
	}
}

// computePattern: standard Wordle scoring. 'g' green, 'y' yellow, 'x' gray.
func computePattern(guess, target string) string {
	g := []rune(strings.ToUpper(guess))
	t := []rune(strings.ToUpper(target))
	pat := []byte("xxxxx")
	counts := map[rune]int{}
	for i := 0; i < 5; i++ {
		if g[i] == t[i] {
			pat[i] = 'g'
		} else {
			counts[t[i]]++
		}
	}
	for i := 0; i < 5; i++ {
		if pat[i] == 'g' {
			continue
		}
		if counts[g[i]] > 0 {
			pat[i] = 'y'
			counts[g[i]]--
		}
	}
	return string(pat)
}

func isValidWord(w string) bool {
	if len(w) != 5 {
		return false
	}
	lw := strings.ToLower(w)
	for _, v := range wordList {
		if v == lw {
			return true
		}
	}
	return false
}

func (m experimentModel) InputEmpty() bool {
	for _, r := range m.rows {
		if len(r) > 0 {
			return false
		}
	}
	return true
}

func (m experimentModel) View() string {
	var sb strings.Builder

	for i := 0; i < 7; i++ {
		sb.WriteString(m.renderRow(i) + "\n")
	}

	sb.WriteString("\n")
	target := m.rows[0]
	if len(target) != 5 {
		sb.WriteString(dimStyle.Render("Top row: target word | ↑/↓ to move | Backspace to edit | ? for hints") + "\n")
	} else if !isValidWord(target) {
		sb.WriteString(redStyle.Render("Target not in word list (still scored)") + "\n")
	} else {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Target: %s | row %d", target, m.cursor)) + "\n")
	}

	sb.WriteString("\n")
	if m.showHints {
		sb.WriteString(m.renderHints())
	} else {
		sb.WriteString(m.renderKeyboard())
	}

	return sb.String()
}

func (m experimentModel) renderRow(row int) string {
	word := m.rows[row]
	active := row == m.cursor
	cells := make([]string, 5)

	if row == 0 {
		for i := 0; i < 5; i++ {
			var ch string
			if i < len(word) {
				ch = string(word[i])
			} else if i == len(word) && active {
				ch = dimStyle.Render("_")
			} else {
				ch = " "
			}
			border := blueBorder
			if !active && len(word) > 0 && i < len(word) {
				cells[i] = border.Render(blueStyle.Bold(true).Render(ch))
			} else if i < len(word) {
				cells[i] = border.Render(blueStyle.Bold(true).Render(ch))
			} else {
				cells[i] = border.Render(ch)
			}
		}
		return lipgloss.JoinHorizontal(lipgloss.Center, cells...)
	}

	target := m.rows[0]
	canScore := len(target) == 5 && len(word) == 5
	wordInvalid := len(word) == 5 && !isValidWord(word)

	var pat string
	if canScore {
		pat = computePattern(word, target)
	}

	for i := 0; i < 5; i++ {
		var ch string
		if i < len(word) {
			ch = string(word[i])
		} else if i == len(word) && active {
			ch = dimStyle.Render("_")
		} else {
			ch = " "
		}

		if i < len(word) && canScore {
			switch pat[i] {
			case 'g':
				cells[i] = greenBorder.Render(greenStyle.Bold(true).Render(ch))
			case 'y':
				cells[i] = yellowBorder.Render(yellowStyle.Bold(true).Render(ch))
			default:
				cells[i] = grayBorder.Render(grayStyle.Render(ch))
			}
			if wordInvalid {
				switch pat[i] {
				case 'g':
					cells[i] = redBorder.Render(greenStyle.Bold(true).Render(ch))
				case 'y':
					cells[i] = redBorder.Render(yellowStyle.Bold(true).Render(ch))
				default:
					cells[i] = redBorder.Render(grayStyle.Render(ch))
				}
			}
		} else if i < len(word) {
			if wordInvalid {
				cells[i] = redBorder.Render(redStyle.Bold(true).Render(ch))
			} else {
				cells[i] = activeBorder.Render(brightStyle.Bold(true).Render(ch))
			}
		} else if active {
			cells[i] = activeBorder.Render(ch)
		} else {
			cells[i] = emptyBorder.Render(" ")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, cells...)
}

func (m experimentModel) renderKeyboard() string {
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

func (m experimentModel) renderHints() string {
	target := m.rows[0]
	if len(target) != 5 {
		return dimStyle.Render("Set target word (top row) to see hints.")
	}

	var guesses []GuessEntry
	for i := 1; i < 7; i++ {
		w := m.rows[i]
		if len(w) != 5 {
			continue
		}
		guesses = append(guesses, GuessEntry{
			Word:    strings.ToLower(w),
			Pattern: computePattern(w, target),
		})
	}

	hints := getHints(guesses)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1"))
	wordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	var sb strings.Builder
	if len(hints) == 0 {
		sb.WriteString(dimStyle.Render("No matching words found.") + "\n")
		return sb.String()
	}

	sb.WriteString(headerStyle.Render(fmt.Sprintf("Possible words (%d shown)", len(hints))) + "\n\n")

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
