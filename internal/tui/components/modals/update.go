package modals

import tea "charm.land/bubbletea/v2"

type HistoryDeps struct {
	History            []string
	Timestamps         []string
	ListHeight         int
	SetQueryText       func(string)
	TextareaFocus      func() tea.Cmd
	SetShowSuggestions func(bool)
}

type CopyMenuDeps struct {
	OnCopy func(formatIndex int) tea.Cmd
}

func UpdateRowDetail(m *RowDetailModel, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	if !m.Show {
		return false, nil
	}

	switch msg.String() {
	case "q", "esc":
		m.Show = false
	case "j", "down", "k", "up", "pgup", "pgdown":
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		return true, cmd
	case "h", "left":
		m.Viewport.ScrollLeft(5)
	case "l", "right":
		m.Viewport.ScrollRight(5)
	case "0", "^":
		m.Viewport.ScrollLeft(1 << 20)
	case "$":
		m.Viewport.ScrollRight(1 << 20)
	case "g":
		m.Viewport.GotoTop()
	case "G":
		m.Viewport.GotoBottom()
	}

	return true, nil
}

func UpdateCopyMenu(m *CopyMenuModel, msg tea.KeyPressMsg, deps CopyMenuDeps) (bool, tea.Cmd) {
	if !m.Show {
		return false, nil
	}

	switch msg.String() {
	case "j", "down":
		if m.Index < 5 {
			m.Index++
		}
	case "k", "up":
		if m.Index > 0 {
			m.Index--
		}
	case "enter":
		cmd := deps.OnCopy(m.Index)
		m.Show = false
		return true, cmd
	case "esc":
		m.Show = false
	}

	return true, nil
}

func UpdateHistorySearch(m *HistorySearchModel, msg tea.KeyPressMsg, deps HistoryDeps) (bool, tea.Cmd) {
	if !m.Show {
		return false, nil
	}

	indices := filteredHistoryIndices(deps.History, m.Filter)
	switch msg.Code {
	case tea.KeyEsc:
		m.Show = false
		return true, deps.TextareaFocus()
	case tea.KeyEnter, tea.KeyKpEnter:
		if len(indices) > 0 && m.Index >= 0 && m.Index < len(indices) {
			query := deps.History[indices[m.Index]]
			deps.SetQueryText(query)
		}
		m.Show = false
		deps.SetShowSuggestions(false)
		return true, deps.TextareaFocus()
	case tea.KeyUp:
		if m.Index < len(indices)-1 {
			m.Index++
		}
		HistorySearchClampScroll(m, deps.History, deps.ListHeight)
		return true, nil
	case tea.KeyDown:
		if m.Index > 0 {
			m.Index--
		}
		HistorySearchClampScroll(m, deps.History, deps.ListHeight)
		return true, nil
	case tea.KeyBackspace:
		if len(m.Filter) > 0 {
			m.Filter = m.Filter[:len(m.Filter)-1]
			m.Index = 0
			m.Scroll = 0
		}
		return true, nil
	}

	if msg.Text != "" {
		m.Filter += msg.Text
		m.Index = 0
		m.Scroll = 0
	}

	return true, nil
}
