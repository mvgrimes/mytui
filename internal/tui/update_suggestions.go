package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m Model) updateSuggestionsKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showSuggestions {
		return m, nil, false
	}

	consumed := false
	switch msg.String() {
	case "up", "shift+tab":
		if len(m.suggestions) > 0 {
			if m.suggestionIndex > 0 {
				m.suggestionIndex--
			} else {
				m.suggestionIndex = len(m.suggestions) - 1
			}
		}
		consumed = true
	case "down":
		if len(m.suggestions) > 0 {
			if m.suggestionIndex < len(m.suggestions)-1 {
				if m.suggestionIndex < 0 {
					m.suggestionIndex = 0
				} else {
					m.suggestionIndex++
				}
			} else {
				m.suggestionIndex = 0
			}
		}
		consumed = true
	case "tab":
		if len(m.suggestions) > 0 {
			if m.suggestionIndex < 0 {
				m.suggestionIndex = 0
			} else if m.suggestionIndex < len(m.suggestions)-1 {
				m.suggestionIndex++
			} else {
				m.suggestionIndex = 0
			}
		}
		consumed = true
	case "enter":
		if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
			m.applySuggestion()
			m.showSuggestions = false
			consumed = true
		}
	case " ":
		if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
			m.applySuggestion()
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
			m.updateSuggestions()
			consumed = true
		}
	case ";":
		if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
			m.applySuggestion()
			m.showSuggestions = false
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
			consumed = true
		}
	case ",":
		if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
			m.applySuggestion()
			m.showSuggestions = false
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{','}})
			consumed = true
		}
	case "esc":
		if m.suggestionIndex >= 0 {
			m.suggestionIndex = -1
		} else {
			m.showSuggestions = false
		}
		consumed = true
	default:
		if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
			oldVal := m.textarea.Value()
			var tiCmd tea.Cmd
			m.textarea, tiCmd = m.textarea.Update(msg)
			if m.textarea.Value() != oldVal {
				m.recalculateHeight()
				m.updateSuggestions()
				if m.shouldOpenSuggestionsOnEdit() && len(m.suggestions) > 0 {
					if m.suggestionIndex >= len(m.suggestions) {
						m.suggestionIndex = -1
					}
					m.showSuggestions = true
				} else {
					m.showSuggestions = false
				}
			}
			return m, tiCmd, true
		}
	}

	if consumed {
		return m, nil, true
	}

	return m, nil, false
}
