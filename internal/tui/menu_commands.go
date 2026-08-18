package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/special"
	"github.com/mvgrimes/mytui/internal/tui/components/menu"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func (m *Model) buildMenuCommands() []menu.Command {
	switch m.menu.Type {
	case core.MenuSaveFavorite:
		return []menu.Command{{
			Label: "Confirm Save (Prompt for name in Update)",
			Action: func() tea.Cmd {
				return nil
			},
		}}
	case core.MenuRunFavorite:
		var cmds []menu.Command
		m.menu.FavoriteNames = nil
		for name := range m.config.FavoriteQueries {
			m.menu.FavoriteNames = append(m.menu.FavoriteNames, name)
		}
		// Sort names for consistency
		for i := 0; i < len(m.menu.FavoriteNames); i++ {
			for j := i + 1; j < len(m.menu.FavoriteNames); j++ {
				if m.menu.FavoriteNames[i] > m.menu.FavoriteNames[j] {
					m.menu.FavoriteNames[i], m.menu.FavoriteNames[j] = m.menu.FavoriteNames[j], m.menu.FavoriteNames[i]
				}
			}
		}
		for _, name := range m.menu.FavoriteNames {
			n := name
			cmds = append(cmds, menu.Command{
				Label: n,
				Action: func() tea.Cmd {
					queryText := m.config.FavoriteQueries[n]
					queryText = strings.ReplaceAll(queryText, "\\n", "\n")
					m.query.Textarea.SetValue(queryText)
					m.menu.Show = false
					return nil
				},
			})
		}
		if len(cmds) == 0 {
			cmds = append(cmds, menu.Command{
				Label: "No favorites saved",
				Action: func() tea.Cmd {
					m.menu.Show = false
					return nil
				},
			})
		}
		return cmds
	default:
		return []menu.Command{
			{
				Label: "Status (\\s)",
				Action: func() tea.Cmd {
					m.specialOutput.Reset()
					special.Handle("\\s", m)
					m.addResultFromText(m.specialOutput.String(), "\\s")
					m.focus = core.FocusResults
					m.query.Textarea.Blur()
					return nil
				},
			},
			{
				Label: "Copy clipboard (unicode)",
				Action: func() tea.Cmd {
					m.copyToClipboard(m.currentFormat)
					return nil
				},
			},
			{
				Label: "Copy to clipboard (ascii)",
				Action: func() tea.Cmd {
					m.copyToClipboard(formatter.FormatTable)
					return nil
				},
			},
			{
				Label: "Copy clipboard (CSV)",
				Action: func() tea.Cmd {
					m.copyToClipboard(formatter.FormatCSV)
					return nil
				},
			},
			{
				Label: "Save query as favorite",
				Action: func() tea.Cmd {
					m.menu.Type = core.MenuSaveFavorite
					m.menu.Index = 0
					return nil
				},
			},
			{
				Label: "Run query as favorite",
				Action: func() tea.Cmd {
					m.menu.Type = core.MenuRunFavorite
					m.menu.Index = 0
					return nil
				},
			},
			{
				Label: "Exit",
				Action: func() tea.Cmd {
					return tea.Quit
				},
			},
		}
	}
}
