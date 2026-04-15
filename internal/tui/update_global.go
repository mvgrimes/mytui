package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/tui/components/results"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func (m *Model) updateGlobalKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return true, tea.Quit
	case tea.KeyTab:
		if m.focus == core.FocusQuery {
			if len(m.results.Results) > 0 {
				m.focus = core.FocusResults
				m.results.FocusedResultIndex = len(m.results.Results) - 1
				m.query.Textarea.Blur()
				available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
				results.EnsureFocusedResultVisible(&m.results, available)
			}
			return true, nil
		}
		if m.results.FocusedResultIndex > 0 {
			m.results.FocusedResultIndex--
			available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
			results.EnsureFocusedResultVisible(&m.results, available)
		} else {
			m.focus = core.FocusQuery
			m.results.FocusedResultIndex = -1
			return true, m.query.Textarea.Focus()
		}
		return true, nil
	case tea.KeyShiftTab:
		if m.focus == core.FocusQuery {
			if len(m.results.Results) > 0 {
				m.focus = core.FocusResults
				m.results.FocusedResultIndex = 0
				m.query.Textarea.Blur()
				available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
				results.EnsureFocusedResultVisible(&m.results, available)
			}
			return true, nil
		}
		if m.results.FocusedResultIndex < len(m.results.Results)-1 {
			m.results.FocusedResultIndex++
			available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
			results.EnsureFocusedResultVisible(&m.results, available)
		} else {
			m.focus = core.FocusQuery
			m.results.FocusedResultIndex = -1
			return true, m.query.Textarea.Focus()
		}
		return true, nil
	default:
		// Ctrl+Space is commonly sent as NUL (Ctrl+@). Bubble Tea reports that as KeyCtrlAt.
		if msg.Type == tea.KeyCtrlAt || msg.String() == "ctrl+ " || msg.String() == "ctrl+space" || msg.String() == "ctrl+@" {
			m.menu.Show = true
			m.menu.Index = 0
			m.menu.Type = core.MenuMain
			m.menu.Filter = ""
			return true, nil
		}
	}

	return false, nil
}
