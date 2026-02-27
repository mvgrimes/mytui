package menu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

type UpdateDeps struct {
	OnSaveFavorite func(name string) tea.Cmd
}

func Update(m *Model, msg tea.KeyMsg, cmds []Command, deps UpdateDeps) (bool, tea.Cmd) {
	if !m.Show {
		return false, nil
	}

	if m.Type == core.MenuSaveFavorite {
		switch msg.Type {
		case tea.KeyEnter:
			cmd := deps.OnSaveFavorite(m.FavoriteInput)
			m.Show = false
			m.FavoriteInput = ""
			m.Filter = ""
			return true, cmd
		case tea.KeyEsc:
			m.Show = false
			m.FavoriteInput = ""
			m.Filter = ""
			return true, nil
		case tea.KeyBackspace:
			if len(m.FavoriteInput) > 0 {
				m.FavoriteInput = m.FavoriteInput[:len(m.FavoriteInput)-1]
			}
			return true, nil
		case tea.KeyRunes:
			m.FavoriteInput += string(msg.Runes)
			return true, nil
		}
		if msg.String() == "ctrl+ " || msg.String() == "ctrl+space" || msg.Type == tea.KeyCtrlAt {
			m.Show = false
			m.FavoriteInput = ""
			m.Filter = ""
			return true, nil
		}
		return true, nil
	}

	switch msg.String() {
	case "up", "k":
		filtered := FilteredCommands(m, cmds)
		if len(filtered) > 0 {
			if m.Index > 0 {
				m.Index--
			} else {
				m.Index = len(filtered) - 1
			}
		}
	case "down", "j":
		filtered := FilteredCommands(m, cmds)
		if len(filtered) > 0 {
			if m.Index < len(filtered)-1 {
				m.Index++
			} else {
				m.Index = 0
			}
		}
	case "enter":
		oldType := m.Type
		filtered := FilteredCommands(m, cmds)
		if len(filtered) == 0 {
			return true, nil
		}
		if m.Index < 0 || m.Index >= len(filtered) {
			m.Index = 0
		}
		cmd := filtered[m.Index].Action()
		if m.Type == oldType {
			m.Show = false
			m.Filter = ""
		}
		return true, cmd
	case "esc", "ctrl+ ", "ctrl+space":
		m.Show = false
		m.Filter = ""
		return true, nil
	}
	if msg.Type == tea.KeyCtrlAt {
		m.Show = false
		m.Filter = ""
		return true, nil
	}
	if msg.Type == tea.KeyBackspace {
		if len(m.Filter) > 0 {
			m.Filter = m.Filter[:len(m.Filter)-1]
		}
		m.Index = 0
		return true, nil
	}
	if msg.Type == tea.KeyRunes {
		m.Filter += string(msg.Runes)
		m.Index = 0
		return true, nil
	}

	return true, nil
}
