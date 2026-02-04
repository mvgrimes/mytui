package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Background(lipgloss.Color("#222222"))

	headerFocusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0A0A0A")).
				Background(lipgloss.Color("#FAB283")).
				Bold(true)
)
