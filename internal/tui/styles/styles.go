package styles

import "charm.land/lipgloss/v2"

var (
	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Background(lipgloss.Color("#222222"))

	HeaderFocusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0A0A0A")).
				Background(lipgloss.Color("#FAB283")).
				Bold(true)

	// Selected row highlight style - subtle dark blue background
	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2A2A3A"))

	// Row detail modal styles
	RowDetailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	RowDetailTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	RowDetailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	RowDetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	// Copy menu styles
	CopyMenuBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	CopyMenuTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	CopyMenuItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA"))

	CopyMenuSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#444466")).
				Bold(true)

	// History search modal styles
	HistoryBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00AAFF")).
				Background(lipgloss.Color("#1A1A1A")).
				Padding(1, 2)

	HistoryTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF")).
				Bold(true)

	HistoryItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA")).
				Background(lipgloss.Color("#1A1A1A"))

	HistorySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#1F4080")).
				Bold(true)

	HistoryCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	HistoryPreviewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC")).
				Background(lipgloss.Color("#222233"))

	HistoryFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#333333")).
				Padding(0, 1)
)
