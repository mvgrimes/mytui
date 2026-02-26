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

	// Selected row highlight style - subtle dark blue background
	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2A2A3A"))

	// Row detail modal styles
	rowDetailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	rowDetailTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	rowDetailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	rowDetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	// Copy menu styles
	copyMenuBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	copyMenuTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	copyMenuItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA"))

	copyMenuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#444466")).
				Bold(true)

	// History search modal styles
	historyBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	historyTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	historyItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA")).
				Background(lipgloss.Color("#1A1A1A"))

	historySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#1F4080")).
				Bold(true)

	historyCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	historyPreviewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC")).
				Background(lipgloss.Color("#222233"))

	historyFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#333333")).
				Padding(0, 1)
)
