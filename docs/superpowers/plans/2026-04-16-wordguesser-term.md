# WordGuesser Terminal Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a terminal Wordle client with Bubbletea that connects to the WordGuesser API, featuring tab-based navigation, animated floating bubbles, and Nix Flake packaging.

**Architecture:** Single Bubbletea program with a root model composing sub-models for each view (game, leaderboard, stats, history). The root owns tab navigation and the bubble animation layer. All files in `package main`.

**Tech Stack:** Go, Bubbletea, Lipgloss, Bubbles, Nix Flakes

---

### Task 1: Project Scaffolding — Go Module + Nix Flake + Minimal Main

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `flake.nix`

- [ ] **Step 1: Initialize Go module**

Run: `cd /home/rice/Music/wordguesser-term && go mod init github.com/rhysfuller/wordguesser-term`

- [ ] **Step 2: Add Bubbletea dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

- [ ] **Step 3: Write minimal main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	username := flag.String("username", "", "Set username for this session")
	flag.StringVar(username, "u", "", "Set username for this session (shorthand)")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	activeUsername := *username
	if activeUsername == "" && cfg != nil {
		activeUsername = cfg.Username
	}

	needsSetup := cfg == nil || cfg.APIKey == ""

	m := newApp(cfg, activeUsername, needsSetup)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Write flake.nix**

```nix
{
  description = "WordGuesser - Terminal Wordle client";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages = {
          wordguesser = pkgs.buildGoModule {
            pname = "wordguesser";
            version = "0.1.0";
            src = ./.;
            vendorHash = null; # Will be set after first build
          };
          default = self.packages.${system}.wordguesser;
        };
      }
    );
}
```

Note: `vendorHash` must be updated after the first `nix build` attempt. The build will fail and print the correct hash. Replace `null` with that hash.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go flake.nix
git commit -m "feat: scaffold Go module, main entry point, and Nix flake"
```

---

### Task 2: Config — Load and Save

**Files:**
- Create: `config.go`

- [ ] **Step 1: Write config.go**

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey   string `json:"api_key"`
	Username string `json:"username"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "wordguesser", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`

Note: This will fail until app.go exists (main.go references `newApp`). That's expected — we'll fix it in Task 4.

- [ ] **Step 3: Commit**

```bash
git add config.go
git commit -m "feat: add config load/save for API key and username"
```

---

### Task 3: API Client

**Files:**
- Create: `api.go`

- [ ] **Step 1: Write api.go**

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://rhysfuller.com"

type APIClient struct {
	secret   string
	username string
	http     *http.Client
}

func newAPIClient(secret, username string) *APIClient {
	return &APIClient{
		secret:   secret,
		username: username,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Response types

type GuessEntry struct {
	Word    string `json:"word"`
	Pattern string `json:"pattern"`
}

type GuessResponse struct {
	Pattern   string       `json:"pattern"`
	Guesses   []GuessEntry `json:"guesses"`
	Completed bool         `json:"completed"`
	Won       bool         `json:"won"`
	Word      string       `json:"word"`
	Error     string       `json:"error"`
}

type GameState struct {
	Username  string       `json:"username"`
	Date      string       `json:"date"`
	Guesses   []GuessEntry `json:"guesses"`
	Completed bool         `json:"completed"`
	Won       bool         `json:"won"`
}

type LeaderboardEntry struct {
	Username      string  `json:"username"`
	TotalPoints   int     `json:"total_points"`
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	AvgTries      float64 `json:"avg_tries"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

type LeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

type UserStatsResponse struct {
	Username      string  `json:"username"`
	TotalPoints   int     `json:"total_points"`
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	AvgTries      float64 `json:"avg_tries"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

type ResultEntry struct {
	Username   string   `json:"username"`
	Guesses    []string `json:"guesses"`
	Patterns   []string `json:"patterns"`
	Solved     bool     `json:"solved"`
	Completed  bool     `json:"completed"`
	NumGuesses int      `json:"num_guesses"`
}

type ResultsResponse struct {
	Date    string        `json:"date"`
	Word    string        `json:"word"`
	Results []ResultEntry `json:"results"`
}

// API methods

func (c *APIClient) getJSON(path string, params url.Values, result any) error {
	u := baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Wordle-Secret", c.secret)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, result)
}

func (c *APIClient) SubmitGuess(guess string) (*GuessResponse, error) {
	payload := map[string]string{
		"secret_key": c.secret,
		"username":   c.username,
		"guess":      guess,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/wordle/guess/", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != 200 {
		// Try to parse error from body
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result GuessResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) GetGameState(username, date string) (*GameState, error) {
	var state GameState
	params := url.Values{"username": {username}, "date": {date}}
	err := c.getJSON("/api/wordle/state/", params, &state)
	return &state, err
}

func (c *APIClient) GetLeaderboard() (*LeaderboardResponse, error) {
	var lb LeaderboardResponse
	err := c.getJSON("/api/wordle/leaderboard/", nil, &lb)
	return &lb, err
}

func (c *APIClient) GetUserStats(username string) (*UserStatsResponse, error) {
	var stats UserStatsResponse
	params := url.Values{"username": {username}}
	err := c.getJSON("/api/wordle/user-stats/", params, &stats)
	return &stats, err
}

func (c *APIClient) GetResults(date string) (*ResultsResponse, error) {
	var results ResultsResponse
	params := url.Values{"date": {date}}
	err := c.getJSON("/api/wordle/results/", params, &results)
	return &results, err
}
```

- [ ] **Step 2: Commit**

```bash
git add api.go
git commit -m "feat: add API client for all WordGuesser endpoints"
```

---

### Task 4: Bubble Animation System

**Files:**
- Create: `bubbles.go`

- [ ] **Step 1: Write bubbles.go**

```go
package main

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type bubble struct {
	x, y   float64
	speedY float64
	wobble float64
	phase  float64
	char   string
	color  lipgloss.Style
}

type bubbleField struct {
	bubbles    []bubble
	width      int
	height     int
	targetCount int
	tick       int
}

var bubbleChars = []string{"·", "•", "○", "◦", "°", "◯"}

var bubbleColors = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#818cf8")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#7c3aed")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#3b82f6")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#8b5cf6")),
}

func newBubbleField(width, height int) bubbleField {
	bf := bubbleField{
		width:       width,
		height:      height,
		targetCount: 20,
	}
	// Spawn initial bubbles at random positions
	for i := 0; i < bf.targetCount; i++ {
		bf.bubbles = append(bf.bubbles, bf.spawnBubble(true))
	}
	return bf
}

func (bf *bubbleField) spawnBubble(randomY bool) bubble {
	y := float64(bf.height)
	if randomY {
		y = rand.Float64() * float64(bf.height)
	}
	return bubble{
		x:      rand.Float64() * float64(bf.width),
		y:      y,
		speedY: 0.3 + rand.Float64()*1.2,
		wobble: 0.5 + rand.Float64()*2.0,
		phase:  rand.Float64() * math.Pi * 2,
		char:   bubbleChars[rand.Intn(len(bubbleChars))],
		color:  bubbleColors[rand.Intn(len(bubbleColors))],
	}
}

func (bf *bubbleField) update() {
	bf.tick++
	alive := bf.bubbles[:0]
	for i := range bf.bubbles {
		b := &bf.bubbles[i]
		b.y -= b.speedY
		b.x += math.Sin(b.phase+float64(bf.tick)*0.05) * b.wobble * 0.15
		if b.y > -1 {
			alive = append(alive, *b)
		}
	}
	bf.bubbles = alive

	// Spawn new bubbles to maintain target count
	for len(bf.bubbles) < bf.targetCount {
		bf.bubbles = append(bf.bubbles, bf.spawnBubble(false))
	}
}

func (bf *bubbleField) resize(width, height int) {
	bf.width = width
	bf.height = height
}

func (bf *bubbleField) view() string {
	if bf.width == 0 || bf.height == 0 {
		return ""
	}

	// Build a grid of characters
	grid := make([][]string, bf.height)
	for i := range grid {
		grid[i] = make([]string, bf.width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	// Place bubbles on the grid
	for _, b := range bf.bubbles {
		ix := int(math.Round(b.x))
		iy := int(math.Round(b.y))
		if ix >= 0 && ix < bf.width && iy >= 0 && iy < bf.height {
			grid[iy][ix] = b.color.Render(b.char)
		}
	}

	// Render grid to string
	var sb strings.Builder
	for i, row := range grid {
		for _, cell := range row {
			sb.WriteString(cell)
		}
		if i < len(grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
```

- [ ] **Step 2: Commit**

```bash
git add bubbles.go
git commit -m "feat: add floating bubble animation system"
```

---

### Task 5: Setup Flow — First-Run API Key and Username Entry

**Files:**
- Create: `setup.go`

- [ ] **Step 1: Write setup.go**

```go
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

	// Update active input
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
```

- [ ] **Step 2: Commit**

```bash
git add setup.go
git commit -m "feat: add first-run setup flow for API key and username"
```

---

### Task 6: Game View — Grid, Keyboard, and Guess Input

**Files:**
- Create: `game.go`

- [ ] **Step 1: Write game.go**

```go
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
	revealWord string
	letters    map[rune]letterState
	loading    bool
	spinner    spinner.Model
	flashMsg   string
	flashUntil time.Time
	width      int
}

type guessResultMsg struct {
	resp *GuessResponse
	err  error
}

type gameStateMsg struct {
	state *GameState
	err   error
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
		if msg.err == nil {
			m.guesses = msg.state.Guesses
			m.completed = msg.state.Completed
			m.won = msg.state.Won
			m.updateLetterStates()
		}
		// 404 means no game yet — that's fine
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
			// Only upgrade state, never downgrade
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
	var sb strings.Builder

	// Grid
	for i := 0; i < 6; i++ {
		row := m.renderRow(i)
		sb.WriteString(row + "\n")
	}

	// Status line
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
	} else {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("Guess %d/6 | Type a word and press Enter", len(m.guesses)+1)) + "\n")
	}

	// Keyboard
	sb.WriteString("\n")
	sb.WriteString(m.renderKeyboard())

	return sb.String()
}

func (m gameModel) renderRow(row int) string {
	cells := make([]string, 5)

	if row < len(m.guesses) {
		// Submitted row
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
		// Current input row
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
		// Empty row
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
```

- [ ] **Step 2: Commit**

```bash
git add game.go
git commit -m "feat: add game view with grid, keyboard, and guess submission"
```

---

### Task 7: Leaderboard View

**Files:**
- Create: `leaderboard.go`

- [ ] **Step 1: Write leaderboard.go**

```go
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

	// Header
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
```

- [ ] **Step 2: Commit**

```bash
git add leaderboard.go
git commit -m "feat: add leaderboard view with player rankings"
```

---

### Task 8: Stats View

**Files:**
- Create: `stats.go`

- [ ] **Step 1: Write stats.go**

```go
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
```

- [ ] **Step 2: Commit**

```bash
git add stats.go
git commit -m "feat: add personal stats view with stat cards"
```

---

### Task 9: History View

**Files:**
- Create: `history.go`

- [ ] **Step 1: Write history.go**

```go
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

	// Date selector
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

	// Word reveal
	if m.results.Word != "" {
		sb.WriteString(fmt.Sprintf("  Word: %s\n\n", greenStyle.Bold(true).Render(strings.ToUpper(m.results.Word))))
	}

	// Results table
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
```

- [ ] **Step 2: Commit**

```bash
git add history.go
git commit -m "feat: add history view with date navigation"
```

---

### Task 10: Root App Model — Tab Bar, View Routing, Bubble Background

**Files:**
- Create: `app.go`

- [ ] **Step 1: Write app.go**

```go
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
		// Global keys
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
				return m, m.game.Init()
			}
			return m, cmd
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
				cmd := m.leaderboard.Fetch()
				cmds = append(cmds, cmd)
			case tabStats:
				cmd := m.stats.Fetch()
				cmds = append(cmds, cmd)
			case tabHistory:
				cmd := m.history.Fetch()
				cmds = append(cmds, cmd)
			}
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
			return m, nil
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

	// Initialize bubble field on first render
	if m.bubbleField.width == 0 {
		m.bubbleField = newBubbleField(m.width, m.height)
	}

	// Render bubble background
	bg := m.bubbleField.view()

	// Render foreground
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

	// Center foreground content
	fgStyled := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(fg)

	// Overlay foreground on bubble background
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		fgStyled,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	) + "\x1b[0m" // Reset any trailing styles

	// Simple overlay: just place fg centered over bg
	_ = bg
	return fgStyled
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

// Place foreground text over the bubble background
func overlay(bg, fg string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	// Pad to height
	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	// Center the foreground vertically
	fgStart := (height - len(fgLines)) / 2
	if fgStart < 0 {
		fgStart = 0
	}

	result := make([]string, len(bgLines))
	copy(result, bgLines)

	for i, line := range fgLines {
		row := fgStart + i
		if row >= 0 && row < len(result) && strings.TrimSpace(line) != "" {
			// Center horizontally
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
```

- [ ] **Step 2: Verify the project compiles**

Run: `go build -o wordguesser .`

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "feat: add root app model with tab navigation and bubble overlay"
```

---

### Task 11: Integration — Compile, Fix, and Run

This task is for fixing any compilation errors and ensuring everything works together.

- [ ] **Step 1: Build the project**

Run: `cd /home/rice/Music/wordguesser-term && go build -o wordguesser .`

Fix any compilation errors that arise. Common issues:
- Import paths
- Method signature mismatches between files
- Missing type assertions

- [ ] **Step 2: Run the binary**

Run: `./wordguesser --help`

Verify the help output shows the `--username` flag.

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve compilation issues and wire everything together"
```

---

### Task 12: Nix Flake — Set vendorHash and Verify Build

- [ ] **Step 1: Attempt nix build**

Run: `cd /home/rice/Music/wordguesser-term && nix build .#wordguesser 2>&1`

This will fail with a hash mismatch. Copy the correct hash from the error output.

- [ ] **Step 2: Update vendorHash in flake.nix**

Replace `vendorHash = null;` with the correct hash from the error output, e.g.:
```nix
vendorHash = "sha256-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";
```

- [ ] **Step 3: Rebuild**

Run: `nix build .#wordguesser`

- [ ] **Step 4: Test the flake binary**

Run: `./result/bin/wordguesser --help`

- [ ] **Step 5: Commit**

```bash
git add flake.nix flake.lock
git commit -m "feat: finalize Nix flake with correct vendorHash"
```

---

### Task 13: Polish — Bubble Overlay and Final Touches

- [ ] **Step 1: Fix the bubble overlay in app.go**

The `View()` method in `app.go` has a dead code path. Replace the `View()` method with a clean implementation that uses the `overlay` function:

```go
func (m appModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Initialize bubble field on first render
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
```

- [ ] **Step 2: Build and manually test**

Run: `go build -o wordguesser . && ./wordguesser`

Verify:
- Bubbles animate in the background
- Setup flow appears on first run (or game view if already configured)
- Tab switching works
- Ctrl+C quits

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "fix: clean up bubble overlay rendering"
```
