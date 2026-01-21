package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderMenu() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1)

	activeStyle := style.Copy().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00AAFF")).
		Bold(true)

	var b strings.Builder

	title := " COMMANDS "
	switch m.menuType {
	case MenuSaveFavorite:
		title = " SAVE FAVORITE "
	case MenuRunFavorite:
		title = " RUN FAVORITE "
	}

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Render(title))
	b.WriteString("\n\n")

	if m.menuType == MenuSaveFavorite {
		b.WriteString("Enter name for favorite:\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30).
			Render(m.favoriteInput + "_"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Enter: Save • Esc: Cancel"))
	} else {
		commands := m.GetCommands()
		for i, cmd := range commands {
			line := cmd.Label
			if i == m.menuIndex {
				b.WriteString(activeStyle.Render(line) + "\n")
			} else {
				b.WriteString(style.Render(line) + "\n")
			}
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(1, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(b.String())
}
