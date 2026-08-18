package suggestions

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/mvgrimes/mytui/internal/vim"
)

type UpdateDeps struct {
	FocusQuery        bool
	VimState          *vim.VimState
	Textarea          *textarea.Model
	RecalculateHeight func()
	UpdateSuggestions func()
	ApplySuggestion   func()
	ShouldOpenOnEdit  func() bool
}

func UpdateKey(m *Model, msg tea.KeyPressMsg, deps UpdateDeps) (bool, tea.Cmd) {
	if !m.Show {
		return false, nil
	}
	if (msg.Code == tea.KeyEnter || msg.Code == tea.KeyKpEnter) && msg.Mod.Contains(tea.ModAlt) {
		return false, nil
	}

	consumed := false
	switch msg.String() {
	case "up", "shift+tab":
		if len(m.Items) > 0 {
			if m.Index > 0 {
				m.Index--
			} else {
				m.Index = len(m.Items) - 1
			}
		}
		consumed = true
	case "down":
		if len(m.Items) > 0 {
			if m.Index < len(m.Items)-1 {
				if m.Index < 0 {
					m.Index = 0
				} else {
					m.Index++
				}
			} else {
				m.Index = 0
			}
		}
		consumed = true
	case "tab":
		if len(m.Items) > 0 {
			if m.Index < 0 {
				m.Index = 0
			} else if m.Index < len(m.Items)-1 {
				m.Index++
			} else {
				m.Index = 0
			}
		}
		consumed = true
	case "enter":
		if m.Index >= 0 && m.Index < len(m.Items) {
			deps.ApplySuggestion()
			m.Show = false
			consumed = true
		}
	case "space":
		if m.Index >= 0 && m.Index < len(m.Items) {
			deps.ApplySuggestion()
			m.Index = -1
			*deps.Textarea, _ = deps.Textarea.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
			deps.UpdateSuggestions()
			consumed = true
		}
	case ";":
		if m.Index >= 0 && m.Index < len(m.Items) {
			deps.ApplySuggestion()
			m.Show = false
			*deps.Textarea, _ = deps.Textarea.Update(tea.KeyPressMsg{Code: ';', Text: ";"})
			consumed = true
		}
	case ",":
		if m.Index >= 0 && m.Index < len(m.Items) {
			deps.ApplySuggestion()
			m.Show = false
			*deps.Textarea, _ = deps.Textarea.Update(tea.KeyPressMsg{Code: ',', Text: ","})
			consumed = true
		}
	case "esc":
		m.Index = -1
		m.Show = false
		consumed = true
	default:
		if deps.FocusQuery && deps.VimState.Mode == vim.InsertMode {
			oldVal := deps.Textarea.Value()
			var tiCmd tea.Cmd
			*deps.Textarea, tiCmd = deps.Textarea.Update(msg)
			if deps.Textarea.Value() != oldVal {
				deps.RecalculateHeight()
				deps.UpdateSuggestions()
				if deps.ShouldOpenOnEdit() && len(m.Items) > 0 {
					if m.Index >= len(m.Items) {
						m.Index = -1
					}
					m.Show = true
				} else {
					m.Show = false
				}
			}
			return true, tiCmd
		}
	}

	if consumed {
		return true, nil
	}

	return false, nil
}
