package menu

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

// FilteredCommands returns the command list filtered by the current Filter.
func FilteredCommands(m *Model, cmds []Command) []Command {
	if m.Filter == "" {
		return cmds
	}
	q := strings.ToLower(m.Filter)
	out := make([]Command, 0, len(cmds))
	for _, c := range cmds {
		if strings.Contains(strings.ToLower(c.Label), q) {
			out = append(out, c)
		}
	}
	return out
}

func Render(m *Model, cmds []Command) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1)

	activeStyle := style.
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00AAFF")).
		Bold(true)

	var b strings.Builder

	title := " COMMANDS "
	switch m.Type {
	case core.MenuSaveFavorite:
		title = " SAVE FAVORITE "
	case core.MenuRunFavorite:
		title = " RUN FAVORITE "
	}

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Render(title))
	b.WriteString("\n\n")

	if m.Type == core.MenuSaveFavorite {
		b.WriteString("Enter name for favorite:\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30).
			Render(m.FavoriteInput + "_"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Enter: Save • Esc: Cancel"))
	} else {
		// Filter input line
		b.WriteString("Filter: ")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30).
			Render(m.Filter + "_"))
		b.WriteString("\n\n")

		commands := FilteredCommands(m, cmds)
		for i, cmd := range commands {
			line := cmd.Label
			if i == m.Index {
				b.WriteString(activeStyle.Render(line) + "\n")
			} else {
				b.WriteString(style.Render(line) + "\n")
			}
		}
		if len(commands) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("No matches") + "\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(1, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(b.String())
}
