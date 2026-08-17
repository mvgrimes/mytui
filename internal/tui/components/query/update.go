package query

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/vim"
)

type UpdateDeps struct {
	Focus             core.Focus
	VimState          *vim.VimState
	Suggestions       *suggestions.Model
	Completer         *completion.Completer
	OpenHistorySearch func()
	RecalculateHeight func()
	ExecuteQuery      func(string) tea.Cmd
}

func UpdateKey(m *Model, msg tea.KeyMsg, deps UpdateDeps) (bool, tea.Cmd) {
	if deps.Focus != core.FocusQuery {
		return false, nil
	}

	switch msg.Type {
	case tea.KeyCtrlK:
		m.UpdateSuggestions(deps.Completer, deps.Suggestions)
		if len(deps.Suggestions.Items) > 0 {
			deps.Suggestions.Show = true
			deps.Suggestions.Index = -1
		} else {
			deps.Suggestions.Show = false
		}
		return true, nil
	case tea.KeyCtrlR:
		deps.OpenHistorySearch()
		return true, nil
	case tea.KeyCtrlP:
		if m.HistoryIndex > 0 {
			m.HistoryIndex--
			m.Textarea.SetValue(m.History[m.HistoryIndex])
			m.Textarea.CursorEnd()
			deps.RecalculateHeight()
		}
		return true, nil
	case tea.KeyCtrlN:
		if m.HistoryIndex < len(m.History)-1 {
			m.HistoryIndex++
			m.Textarea.SetValue(m.History[m.HistoryIndex])
			m.Textarea.CursorEnd()
			deps.RecalculateHeight()
		} else if m.HistoryIndex == len(m.History)-1 {
			m.HistoryIndex++
			m.Textarea.Reset()
			deps.RecalculateHeight()
		}
		return true, nil
	case tea.KeyCtrlL:
		m.Textarea.Reset()
		m.HistoryIndex = len(m.History)
		deps.Suggestions.Show = false
		deps.RecalculateHeight()
		return true, nil
	case tea.KeyCtrlD:
		if m.Textarea.Value() == "" {
			return true, tea.Quit
		}
	case tea.KeyUp:
		m.Textarea.CursorUp()
		return true, nil
	case tea.KeyDown:
		m.Textarea.CursorDown()
		return true, nil
	}

	if deps.VimState.Mode == vim.NormalMode {
		handled, cmd := updateNormalMode(m, msg, deps)
		return handled, cmd
	}

	return updateInsertMode(m, msg, deps)
}

func updateNormalMode(m *Model, msg tea.KeyMsg, deps UpdateDeps) (bool, tea.Cmd) {
	keyStr := msg.String()

	if m.VimPendingKey == "f" || m.VimPendingKey == "F" {
		if len(keyStr) == 1 {
			targetChar := rune(keyStr[0])
			forward := m.VimPendingKey == "f"
			m.findCharInLine(targetChar, forward)
			m.LastFindChar = targetChar
			m.LastFindForward = forward
		}
		m.VimPendingKey = ""
		return true, nil
	}

	if m.VimPendingKey == "r" {
		if len(keyStr) == 1 {
			text := m.Textarea.Value()
			pos := m.CursorPosition()
			if pos < len(text) {
				newText := text[:pos] + keyStr + text[pos+1:]
				m.Textarea.SetValue(newText)
				m.Textarea.SetCursor(pos)
			}
		}
		m.VimPendingKey = ""
		return true, nil
	}

	if m.VimPendingKey == "ci" || m.VimPendingKey == "di" {
		if keyStr == "w" {
			m.deleteInnerWord()
			if m.VimPendingKey == "ci" {
				deps.VimState.Mode = vim.InsertMode
				m.VimPendingKey = ""
				return true, m.Textarea.Focus()
			}
		}
		m.VimPendingKey = ""
		return true, nil
	}

	if m.VimPendingKey == "g" {
		switch keyStr {
		case "i":
			m.insertSQLTemplate(core.SQLTemplateInsert, core.SQLOffsetInsert)
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		case "s":
			m.insertSQLTemplate(core.SQLTemplateSelect, core.SQLOffsetSelect)
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		case "d":
			m.insertSQLTemplate(core.SQLTemplateDelete, core.SQLOffsetDelete)
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		case "c":
			m.insertSQLTemplate(core.SQLTemplateCreate, core.SQLOffsetCreate)
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		case "f":
			m.jumpToFieldsPosition()
			m.VimPendingKey = ""
			return true, nil
		case "t":
			m.jumpToTablePosition()
			m.VimPendingKey = ""
			return true, nil
		case "w":
			m.jumpToWherePosition()
			m.VimPendingKey = ""
			return true, nil
		}
		m.VimPendingKey = ""
		return true, nil
	}

	switch keyStr {
	case "enter":
		query := m.Textarea.Value()
		if strings.TrimSpace(query) != "" {
			deps.Suggestions.Show = false
			return true, deps.ExecuteQuery(query)
		}
		return true, nil
	case "i":
		if m.VimPendingKey == "c" || m.VimPendingKey == "d" {
			m.VimPendingKey = m.VimPendingKey + "i"
			return true, nil
		}
		deps.VimState.Mode = vim.InsertMode
		return true, m.Textarea.Focus()
	case "a":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
		return true, m.Textarea.Focus()
	case "A":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorEnd()
		return true, m.Textarea.Focus()
	case "I":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorStart()
		return true, m.Textarea.Focus()
	case "h":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
	case "l":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	case "j":
		m.Textarea.CursorDown()
	case "k":
		m.Textarea.CursorUp()
	case "w":
		if m.VimPendingKey == "d" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
			m.VimPendingKey = ""
		} else if m.VimPendingKey == "c" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		} else {
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
		}
	case "b":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
	case "0", "^":
		if m.VimPendingKey == "d" {
			m.deleteToLineStart()
			m.VimPendingKey = ""
			return true, nil
		} else if m.VimPendingKey == "c" {
			m.deleteToLineStart()
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		}
		m.Textarea.CursorStart()
	case "$":
		if m.VimPendingKey == "d" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.VimPendingKey = ""
			return true, nil
		} else if m.VimPendingKey == "c" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		}
		m.Textarea.CursorEnd()
	case "o":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorEnd()
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return true, m.Textarea.Focus()
	case "O":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorStart()
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.Textarea.CursorUp()
		return true, m.Textarea.Focus()
	case "D":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	case "C":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
		deps.VimState.Mode = vim.InsertMode
		return true, m.Textarea.Focus()
	case "d":
		if m.VimPendingKey == "d" {
			m.Textarea.CursorStart()
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			m.VimPendingKey = ""
		} else {
			m.VimPendingKey = "d"
		}
		return true, nil
	case "c":
		if m.VimPendingKey == "c" {
			m.Textarea.CursorStart()
			m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		}
		m.VimPendingKey = "c"
		return true, nil
	case "x":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	case "r":
		m.VimPendingKey = "r"
		return true, nil
	case ";":
		if m.LastFindChar != 0 {
			m.findCharInLine(m.LastFindChar, m.LastFindForward)
		}
	case "u":
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	case "f":
		m.VimPendingKey = "f"
		return true, nil
	case "F":
		m.VimPendingKey = "F"
		return true, nil
	case "g":
		m.VimPendingKey = "g"
		return true, nil
	default:
		m.VimPendingKey = ""
	}
	if keyStr != "d" && keyStr != "c" && keyStr != "f" && keyStr != "F" && keyStr != "i" && keyStr != "g" && keyStr != "r" {
		m.VimPendingKey = ""
	}
	return true, nil
}

func updateInsertMode(m *Model, msg tea.KeyMsg, deps UpdateDeps) (bool, tea.Cmd) {
	if m.VimPendingKey == "ctrl+x" {
		switch msg.String() {
		case "i":
			m.insertSQLTemplate(core.SQLTemplateInsert, core.SQLOffsetInsert)
		case "s":
			m.insertSQLTemplate(core.SQLTemplateSelect, core.SQLOffsetSelect)
		case "d":
			m.insertSQLTemplate(core.SQLTemplateDelete, core.SQLOffsetDelete)
		case "c":
			m.insertSQLTemplate(core.SQLTemplateCreate, core.SQLOffsetCreate)
		case "f":
			m.jumpToFieldsPosition()
		case "t":
			m.jumpToTablePosition()
		case "w":
			m.jumpToWherePosition()
		}
		m.VimPendingKey = ""
		return true, nil
	}

	if m.VimPendingKey == "insert-j" {
		m.VimPendingKey = ""
		if msg.String() == "k" {
			deps.VimState.Mode = vim.NormalMode
			return true, nil
		}
		m.Textarea, _ = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	if msg.Type == tea.KeyCtrlX {
		m.VimPendingKey = "ctrl+x"
		return true, nil
	}

	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'j' {
		m.VimPendingKey = "insert-j"
		return true, nil
	}

	if msg.Type == tea.KeyEsc {
		deps.VimState.Mode = vim.NormalMode
		return true, nil
	}
	if msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlJ {
		if msg.Alt || msg.Type == tea.KeyCtrlJ {
			var tiCmd tea.Cmd
			m.Textarea, tiCmd = m.Textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
			deps.RecalculateHeight()
			return true, tiCmd
		}
		query := m.Textarea.Value()
		if strings.TrimSpace(query) != "" {
			deps.Suggestions.Show = false
			cmd := deps.ExecuteQuery(query)
			return true, cmd
		}
	}

	return false, nil
}
