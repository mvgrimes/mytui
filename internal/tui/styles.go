package tui

import "github.com/charmbracelet/lipgloss"

var (
	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF00")).
			Padding(0, 1).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Background(lipgloss.Color("#222222"))

	headerFocusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0055FF")).
				Background(lipgloss.Color("#CCCCCC")).
				Bold(true)
)
