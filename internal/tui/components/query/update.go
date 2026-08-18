package query

import (
	"strings"

	tea "charm.land/bubbletea/v2"
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

func UpdateKey(m *Model, msg tea.KeyPressMsg, deps UpdateDeps) (bool, tea.Cmd) {
	if deps.Focus != core.FocusQuery {
		return false, nil
	}

	switch msg.String() {
	case "ctrl+k":
		m.UpdateSuggestions(deps.Completer, deps.Suggestions)
		if len(deps.Suggestions.Items) > 0 {
			deps.Suggestions.Show = true
			deps.Suggestions.Index = -1
		} else {
			deps.Suggestions.Show = false
		}
		return true, nil
	case "ctrl+r":
		deps.OpenHistorySearch()
		return true, nil
	case "ctrl+p":
		m.historyPrevious(deps.RecalculateHeight)
		return true, nil
	case "ctrl+n":
		m.historyNext(deps.RecalculateHeight)
		return true, nil
	case "ctrl+l":
		m.Textarea.Reset()
		m.HistoryIndex = len(m.History)
		deps.Suggestions.Show = false
		deps.RecalculateHeight()
		return true, nil
	case "ctrl+d":
		if m.Textarea.Value() == "" {
			return true, tea.Quit
		}
	}

	switch msg.Code {
	case tea.KeyUp:
		if m.Textarea.Line() == 0 {
			m.historyPrevious(deps.RecalculateHeight)
		} else {
			m.Textarea.CursorUp()
		}
		return true, nil
	case tea.KeyDown:
		if m.Textarea.Line() == m.Textarea.LineCount()-1 {
			m.historyNext(deps.RecalculateHeight)
		} else {
			m.Textarea.CursorDown()
		}
		return true, nil
	}

	if deps.VimState.Mode == vim.NormalMode {
		handled, cmd := updateNormalMode(m, msg, deps)
		return handled, cmd
	}

	return updateInsertMode(m, msg, deps)
}

func (m *Model) historyPrevious(recalculateHeight func()) {
	if m.HistoryIndex > 0 {
		m.HistoryIndex--
		m.Textarea.SetValue(m.History[m.HistoryIndex])
		m.Textarea.CursorEnd()
		recalculateHeight()
	}
}

func (m *Model) historyNext(recalculateHeight func()) {
	if m.HistoryIndex < len(m.History)-1 {
		m.HistoryIndex++
		m.Textarea.SetValue(m.History[m.HistoryIndex])
		m.Textarea.CursorEnd()
		recalculateHeight()
	} else if m.HistoryIndex == len(m.History)-1 {
		m.HistoryIndex++
		m.Textarea.Reset()
		recalculateHeight()
	}
}

func updateNormalMode(m *Model, msg tea.KeyPressMsg, deps UpdateDeps) (bool, tea.Cmd) {
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
				m.Textarea.SetCursorColumn(pos)
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
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyRight})
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
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	case "l":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	case "j":
		m.Textarea.CursorDown()
	case "k":
		m.Textarea.CursorUp()
	case "w":
		if m.VimPendingKey == "d" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModAlt})
			m.VimPendingKey = ""
		} else if m.VimPendingKey == "c" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModAlt})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		} else {
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
		}
	case "b":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
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
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
			m.VimPendingKey = ""
			return true, nil
		} else if m.VimPendingKey == "c" {
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		}
		m.Textarea.CursorEnd()
	case "o":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorEnd()
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		return true, m.Textarea.Focus()
	case "O":
		deps.VimState.Mode = vim.InsertMode
		m.Textarea.CursorStart()
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m.Textarea.CursorUp()
		return true, m.Textarea.Focus()
	case "D":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	case "C":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
		deps.VimState.Mode = vim.InsertMode
		return true, m.Textarea.Focus()
	case "d":
		if m.VimPendingKey == "d" {
			m.Textarea.CursorStart()
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
			m.VimPendingKey = ""
		} else {
			m.VimPendingKey = "d"
		}
		return true, nil
	case "c":
		if m.VimPendingKey == "c" {
			m.Textarea.CursorStart()
			m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
			deps.VimState.Mode = vim.InsertMode
			m.VimPendingKey = ""
			return true, m.Textarea.Focus()
		}
		m.VimPendingKey = "c"
		return true, nil
	case "x":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	case "r":
		m.VimPendingKey = "r"
		return true, nil
	case ";":
		if m.LastFindChar != 0 {
			m.findCharInLine(m.LastFindChar, m.LastFindForward)
		}
	case "u":
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
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

func updateInsertMode(m *Model, msg tea.KeyPressMsg, deps UpdateDeps) (bool, tea.Cmd) {
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
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	if msg.String() == "ctrl+x" {
		m.VimPendingKey = "ctrl+x"
		return true, nil
	}

	if msg.Text == "j" {
		m.VimPendingKey = "insert-j"
		return true, nil
	}

	if msg.Code == tea.KeyEsc {
		deps.VimState.Mode = vim.NormalMode
		return true, nil
	}
	if msg.Code == tea.KeyEnter || msg.Code == tea.KeyKpEnter || msg.String() == "ctrl+j" {
		if msg.Mod.Contains(tea.ModAlt) || msg.String() == "ctrl+j" {
			var tiCmd tea.Cmd
			m.Textarea, tiCmd = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
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
