package main

import "github.com/charmbracelet/lipgloss"

var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	brightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6366f1")).MarginBottom(1)
)
