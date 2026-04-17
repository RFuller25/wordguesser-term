# WordGuesser Terminal Client — Design Spec

A terminal-based client for the WordGuesser API (a Wordle clone) built with Go and the Bubbletea TUI framework. Installable via Nix Flakes as the `wordguesser` command.

## Overview

The client connects to the WordGuesser API at `https://rhysfuller.com` and provides a full-featured Wordle experience in the terminal with animated floating bubbles, tab-based navigation across four views, and persistent configuration.

## Architecture

Single Bubbletea program with a root model that composes sub-models. The root model owns:

- Tab bar rendering and navigation (Tab key or number keys 1-4)
- Floating bubble animation system (renders behind all views)
- Active view routing (delegates Update/View to the current sub-model)

### File Structure

```
wordguesser-term/
├── main.go              # Entry point, flag parsing, config loading
├── app.go               # Root model — tab bar, bubble animation, view routing
├── config.go            # Config load/save (~/.config/wordguesser/config.json)
├── api.go               # HTTP client for all API endpoints
├── game.go              # Game view — grid, keyboard, input handling
├── leaderboard.go       # Leaderboard view — table of players + stats
├── stats.go             # Personal stats view
├── history.go           # Browse results by date
├── bubbles.go           # Floating bubble animation system
├── setup.go             # First-run flow — API key + username entry
├── go.mod / go.sum
├── flake.nix            # Nix flake for installation
└── flake.lock
```

All files in `package main`. Each file has a single responsibility.

## Configuration

Stored at `~/.config/wordguesser/config.json` (uses Go's `os.UserConfigDir()`, XDG-compatible, works on NixOS).

```json
{
  "api_key": "the-shared-secret",
  "username": "rhys"
}
```

- On first run, if no config exists, the setup screen prompts for API key and username.
- The `--username` flag overrides the stored username for the session (does not persist the override).
- The API key is required and has no flag override — it must be entered through the setup flow.

## CLI Interface

```
wordguesser [flags]

Flags:
  --username, -u   Set username for this session
  --help, -h       Show help
```

If `--username` is not provided and no username is in the config, the setup screen prompts for it.

## Views

### 1. Game View (Tab 1)

The primary Wordle gameplay screen.

**Layout (top to bottom):**
- Tab bar (shared, rendered by root)
- Wordle grid: 6 rows x 5 columns of bordered boxes
- Status line: "Guess N/6 | Type a word and press Enter"
- Letter keyboard: QWERTY layout showing letter states
- Footer: keybind hints

**Grid cells:**
- Empty: dim border, no content
- Current input: bright border, white letter, blinking cursor on the next empty cell
- Submitted — correct position: green border and letter
- Submitted — wrong position: yellow border and letter
- Submitted — not in word: gray border and letter

**Keyboard display:**
- Unused letters: dim default color
- Green letters: green colored
- Yellow letters: yellow colored
- Gray (eliminated) letters: dim with strikethrough

**Input handling:**
- A-Z keys: append letter to current guess (max 5)
- Backspace: remove last letter
- Enter: submit guess to `POST /api/wordle/guess/`
- Number keys 1-4: switch tabs, only processed when current input is empty (0 letters typed). If letters are typed, number keys are ignored (they aren't valid Wordle input anyway).
- Tab/Shift+Tab: always switch tabs regardless of input state (these can't conflict with word typing)

**API interaction:**
- On view load: `GET /api/wordle/state/` to restore any in-progress game
- On Enter: `POST /api/wordle/guess/` with the 5-letter word
- Invalid word response: show "Not a valid word" flash message, clear current input
- Game complete: show win/loss message with the word revealed, points earned

**Error states:**
- "Game already completed" — show final state with result
- "No guesses remaining" — show loss state
- "No word for today" — show message, disable input
- Network errors — show error message with retry hint

### 2. Leaderboard View (Tab 2)

Shows all players ranked by total points.

**Layout:**
- Tab bar
- Table with columns: Rank, Username, Points, Games, Wins, Avg Tries, Streak, Best Streak
- Current user's row highlighted

**Data source:** `GET /api/wordle/leaderboard/`

Fetches fresh data each time the tab is selected.

### 3. Stats View (Tab 3)

Shows the current user's personal statistics.

**Layout:**
- Tab bar
- Stat cards: Games Played, Games Won, Win %, Avg Tries, Current Streak, Best Streak, Total Points

**Data source:** `GET /api/wordle/user-stats/?username=<username>`

### 4. History View (Tab 4)

Browse results for any date.

**Layout:**
- Tab bar
- Date selector: shows current date, left/right arrow keys to change date
- Results table for that date: Username, Guesses, Solved, Num Guesses
- The word for that date (shown since these are completed games)

**Data source:** `GET /api/wordle/results/?date=<date>`

**Input handling:**
- Left arrow: previous day
- Right arrow: next day
- Today's date is the default and the maximum (can't go forward)

## Floating Bubbles Animation

Lively animated bubbles rendered behind all views by the root model.

**Bubble properties:**
- Position (x, y float)
- Size: small (1 char), medium (2 char), large (3 char) — using Unicode circles/dots
- Speed: varied vertical drift (0.3-1.5 cells per tick)
- Horizontal wobble: slight sinusoidal drift
- Opacity: varied using different Unicode characters (dim dots vs bright circles)
- Color: palette of soft purples, blues, and indigos via Lipgloss

**Behavior:**
- 15-25 bubbles on screen at any time
- Bubbles spawn at random x positions along the bottom
- Drift upward and despawn when they leave the top
- New bubbles spawn to maintain target count
- Tick rate: ~60ms for smooth animation
- Bubbles render behind view content using Lipgloss layering (Place over the bubble field)

**Characters used:** `·`, `•`, `○`, `◯`, `◦`, `°` for varied sizes and visual weight.

## Setup Flow

On first run (no config file exists):

1. Show a welcome screen with the app title
2. Prompt for API key (text input, masked with `*`)
3. Prompt for username (text input, visible)
4. Validate by calling `GET /api/wordle/leaderboard/` with the provided key
5. On success: save config, transition to game view
6. On failure: show error, allow retry

The bubble animation plays during setup too.

## Tab Navigation

- Tab key: cycle to next tab (wraps around)
- Shift+Tab: cycle to previous tab
- Number keys 1-4: jump directly to tab (only when game input is empty)
- Active tab: highlighted with accent color background
- Inactive tabs: dim text

## Styling

All styling via Lipgloss (Bubbletea's styling library).

- Dark background theme (terminal default background)
- Accent color: indigo/purple (`#6366f1`)
- Green: `#22c55e` (correct)
- Yellow: `#eab308` (wrong position)
- Gray: `#6b7280` (not in word)
- Borders: rounded Lipgloss borders on grid cells
- Tab bar: top-border style, active tab with accent background

## Nix Flake

The `flake.nix` provides:
- A package `wordguesser` that builds the Go binary
- A default package pointing to `wordguesser`
- Works with `nix run github:user/wordguesser-term` and `nix profile install github:user/wordguesser-term`

Uses `buildGoModule` from nixpkgs. The `vendorHash` will be set after the first build with dependencies resolved.

## Dependencies

Go modules:
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/lipgloss` — Styling
- `github.com/charmbracelet/bubbles` — UI components (text input, spinner, table, viewport)

No other external dependencies. HTTP client uses the Go standard library.

## Error Handling

- Network errors: display inline error message with the HTTP status or connection error, auto-dismiss after 3 seconds
- API errors (400/403/404): parse error body and display meaningful message
- Config file errors: fall back to setup flow
- Invalid input: ignore invalid keystrokes silently

## Data Flow

```
User Input → Root Model
  ├── Tab navigation keys → Root handles directly, switches active view
  ├── Quit (Ctrl+C) → Root handles, exits program
  └── All other keys → Delegated to active view's Update()

Active View Update() → API calls (blocking, via Cmd)
  └── API Response → View updates state → View() re-renders

Tick (60ms) → Root updates bubble positions → Root View() re-renders background
```

API calls are made as Bubbletea Cmds (async). The view shows a loading spinner while waiting for responses.
