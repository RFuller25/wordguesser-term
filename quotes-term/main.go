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
