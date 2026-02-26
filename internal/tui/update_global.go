package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateGlobalKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit, true
	case tea.KeyTab:
		if m.focus == FocusQuery {
			if len(m.results) > 0 {
				m.focus = FocusResults
				m.focusedResult = len(m.results) - 1
				m.textarea.Blur()
				m.ensureFocusedResultVisible()
			}
			return m, nil, true
		}
		if m.focusedResult > 0 {
			m.focusedResult--
			m.ensureFocusedResultVisible()
		} else {
			m.focus = FocusQuery
			m.focusedResult = -1
			return m, m.textarea.Focus(), true
		}
		return m, nil, true
	case tea.KeyShiftTab:
		if m.focus == FocusQuery {
			if len(m.results) > 0 {
				m.focus = FocusResults
				m.focusedResult = 0
				m.textarea.Blur()
				m.ensureFocusedResultVisible()
			}
			return m, nil, true
		}
		if m.focusedResult < len(m.results)-1 {
			m.focusedResult++
			m.ensureFocusedResultVisible()
		} else {
			m.focus = FocusQuery
			m.focusedResult = -1
			return m, m.textarea.Focus(), true
		}
		return m, nil, true
	default:
		if msg.String() == "ctrl+ " || msg.String() == "ctrl+space" {
			m.showMenu = true
			m.menuIndex = 0
			m.menuType = MenuMain
			m.menuFilter = ""
			return m, nil, true
		}
	}

	return m, nil, false
}
