package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m Model) updateGlobalKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlK:
		m.updateSuggestions()
		if len(m.suggestions) > 0 {
			m.showSuggestions = true
			m.suggestionIndex = -1
		} else {
			m.showSuggestions = false
		}
		return m, nil, true
	case tea.KeyCtrlR:
		m.openHistorySearch()
		return m, nil, true
	case tea.KeyCtrlP:
		if m.focus == FocusQuery {
			if m.historyIndex > 0 {
				m.historyIndex--
				m.textarea.SetValue(m.history[m.historyIndex])
				m.textarea.CursorEnd()
				m.recalculateHeight()
			}
			return m, nil, true
		}
	case tea.KeyCtrlN:
		if m.focus == FocusQuery {
			if m.historyIndex < len(m.history)-1 {
				m.historyIndex++
				m.textarea.SetValue(m.history[m.historyIndex])
				m.textarea.CursorEnd()
				m.recalculateHeight()
			} else if m.historyIndex == len(m.history)-1 {
				m.historyIndex++
				m.textarea.Reset()
				m.recalculateHeight()
			}
			return m, nil, true
		}
	case tea.KeyCtrlC:
		return m, tea.Quit, true
	case tea.KeyCtrlD:
		if m.focus == FocusQuery && m.textarea.Value() == "" {
			return m, tea.Quit, true
		}
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
	case tea.KeyUp:
		if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
			m.textarea.CursorUp()
			return m, nil, true
		}
		if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
			if m.historyIndex > 0 {
				m.historyIndex--
				m.textarea.SetValue(m.history[m.historyIndex])
				m.textarea.CursorEnd()
				m.recalculateHeight()
			}
			return m, nil, true
		}
	case tea.KeyDown:
		if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
			m.textarea.CursorDown()
			return m, nil, true
		}
		if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
			if m.historyIndex < len(m.history)-1 {
				m.historyIndex++
				m.textarea.SetValue(m.history[m.historyIndex])
				m.textarea.CursorEnd()
				m.recalculateHeight()
			} else if m.historyIndex == len(m.history)-1 {
				m.historyIndex++
				m.textarea.Reset()
				m.recalculateHeight()
			}
			return m, nil, true
		}
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
