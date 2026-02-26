package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m Model) updateQueryKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.focus != FocusQuery {
		return m, nil, false
	}

	if m.vimState.Mode == vim.NormalMode {
		return m.updateQueryNormalMode(msg)
	}

	return m.updateQueryInsertMode(msg)
}

func (m Model) updateQueryNormalMode(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	keyStr := msg.String()

	if m.vimPendingKey == "f" || m.vimPendingKey == "F" {
		if len(keyStr) == 1 {
			targetChar := rune(keyStr[0])
			forward := m.vimPendingKey == "f"
			m.findCharInLine(targetChar, forward)
			m.lastFindChar = targetChar
			m.lastFindForward = forward
		}
		m.vimPendingKey = ""
		return m, nil, true
	}

	if m.vimPendingKey == "r" {
		if len(keyStr) == 1 {
			text := m.textarea.Value()
			pos := m.cursorPosition()
			if pos < len(text) {
				newText := text[:pos] + keyStr + text[pos+1:]
				m.textarea.SetValue(newText)
				m.textarea.SetCursor(pos)
			}
		}
		m.vimPendingKey = ""
		return m, nil, true
	}

	if m.vimPendingKey == "ci" || m.vimPendingKey == "di" {
		if keyStr == "w" {
			m.deleteInnerWord()
			if m.vimPendingKey == "ci" {
				m.vimState.Mode = vim.InsertMode
				m.vimPendingKey = ""
				return m, m.textarea.Focus(), true
			}
		}
		m.vimPendingKey = ""
		return m, nil, true
	}

	if m.vimPendingKey == "g" {
		switch keyStr {
		case "i":
			m.insertSQLTemplate(sqlTemplateInsert, sqlOffsetInsert)
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		case "s":
			m.insertSQLTemplate(sqlTemplateSelect, sqlOffsetSelect)
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		case "d":
			m.insertSQLTemplate(sqlTemplateDelete, sqlOffsetDelete)
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		case "c":
			m.insertSQLTemplate(sqlTemplateCreate, sqlOffsetCreate)
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		case "f":
			m.jumpToFieldsPosition()
			m.vimPendingKey = ""
			return m, nil, true
		case "t":
			m.jumpToTablePosition()
			m.vimPendingKey = ""
			return m, nil, true
		case "w":
			m.jumpToWherePosition()
			m.vimPendingKey = ""
			return m, nil, true
		}
		m.vimPendingKey = ""
		return m, nil, true
	}

	switch keyStr {
	case "i":
		if m.vimPendingKey == "c" || m.vimPendingKey == "d" {
			m.vimPendingKey = m.vimPendingKey + "i"
			return m, nil, true
		}
		m.vimState.Mode = vim.InsertMode
		return m, m.textarea.Focus(), true
	case "a":
		m.vimState.Mode = vim.InsertMode
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
		return m, m.textarea.Focus(), true
	case "A":
		m.vimState.Mode = vim.InsertMode
		m.textarea.CursorEnd()
		return m, m.textarea.Focus(), true
	case "I":
		m.vimState.Mode = vim.InsertMode
		m.textarea.CursorStart()
		return m, m.textarea.Focus(), true
	case "h":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
	case "l":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	case "j":
		m.textarea.CursorDown()
	case "k":
		m.textarea.CursorUp()
	case "w":
		if m.vimPendingKey == "d" {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
			m.vimPendingKey = ""
		} else if m.vimPendingKey == "c" {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		} else {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
		}
	case "b":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
	case "0", "^":
		if m.vimPendingKey == "d" {
			m.deleteToLineStart()
			m.vimPendingKey = ""
			return m, nil, true
		} else if m.vimPendingKey == "c" {
			m.deleteToLineStart()
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		}
		m.textarea.CursorStart()
	case "$":
		if m.vimPendingKey == "d" {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.vimPendingKey = ""
			return m, nil, true
		} else if m.vimPendingKey == "c" {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		}
		m.textarea.CursorEnd()
	case "o":
		m.vimState.Mode = vim.InsertMode
		m.textarea.CursorEnd()
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, m.textarea.Focus(), true
	case "O":
		m.vimState.Mode = vim.InsertMode
		m.textarea.CursorStart()
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.textarea.CursorUp()
		return m, m.textarea.Focus(), true
	case "D":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	case "C":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
		m.vimState.Mode = vim.InsertMode
		return m, m.textarea.Focus(), true
	case "d":
		if m.vimPendingKey == "d" {
			m.textarea.CursorStart()
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.vimPendingKey = ""
		} else {
			m.vimPendingKey = "d"
		}
		return m, nil, true
	case "c":
		if m.vimPendingKey == "c" {
			m.textarea.CursorStart()
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.vimState.Mode = vim.InsertMode
			m.vimPendingKey = ""
			return m, m.textarea.Focus(), true
		}
		m.vimPendingKey = "c"
		return m, nil, true
	case "x":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	case "r":
		m.vimPendingKey = "r"
		return m, nil, true
	case ";":
		if m.lastFindChar != 0 {
			m.findCharInLine(m.lastFindChar, m.lastFindForward)
		}
	case "u":
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	case "f":
		m.vimPendingKey = "f"
		return m, nil, true
	case "F":
		m.vimPendingKey = "F"
		return m, nil, true
	case "g":
		m.vimPendingKey = "g"
		return m, nil, true
	default:
		m.vimPendingKey = ""
	}
	if keyStr != "d" && keyStr != "c" && keyStr != "f" && keyStr != "F" && keyStr != "i" && keyStr != "g" && keyStr != "r" {
		m.vimPendingKey = ""
	}
	return m, nil, true
}

func (m Model) updateQueryInsertMode(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.vimPendingKey == "ctrl+x" {
		switch msg.String() {
		case "i":
			m.insertSQLTemplate(sqlTemplateInsert, sqlOffsetInsert)
		case "s":
			m.insertSQLTemplate(sqlTemplateSelect, sqlOffsetSelect)
		case "d":
			m.insertSQLTemplate(sqlTemplateDelete, sqlOffsetDelete)
		case "c":
			m.insertSQLTemplate(sqlTemplateCreate, sqlOffsetCreate)
		case "f":
			m.jumpToFieldsPosition()
		case "t":
			m.jumpToTablePosition()
		case "w":
			m.jumpToWherePosition()
		}
		m.vimPendingKey = ""
		return m, nil, true
	}

	if m.vimPendingKey == "insert-j" {
		m.vimPendingKey = ""
		if msg.String() == "k" {
			m.vimState.Mode = vim.NormalMode
			return m, nil, true
		}
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if msg.Type == tea.KeyCtrlX {
		m.vimPendingKey = "ctrl+x"
		return m, nil, true
	}

	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'j' {
		m.vimPendingKey = "insert-j"
		return m, nil, true
	}

	if msg.Type == tea.KeyEsc {
		m.vimState.Mode = vim.NormalMode
		return m, nil, true
	}
	if msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlJ {
		if msg.Alt || msg.Type == tea.KeyCtrlJ {
			var tiCmd tea.Cmd
			m.textarea, tiCmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.recalculateHeight()
			return m, tiCmd, true
		}
		query := m.textarea.Value()
		if strings.TrimSpace(query) != "" {
			m.showSuggestions = false
			updated, cmd := m.executeQuery(query)
			return updated, cmd, true
		}
	}

	return m, nil, false
}
