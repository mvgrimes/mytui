package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateModalKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if updated, cmd, handled := m.updateRowDetailModal(msg); handled {
		return updated, cmd, true
	}
	if updated, cmd, handled := m.updateCopyMenu(msg); handled {
		return updated, cmd, true
	}
	if updated, cmd, handled := m.updateHistorySearch(msg); handled {
		return updated, cmd, true
	}
	if updated, cmd, handled := m.updateMenu(msg); handled {
		return updated, cmd, true
	}
	return m, nil, false
}

func (m Model) updateRowDetailModal(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showRowDetail {
		return m, nil, false
	}

	switch msg.String() {
	case "q", "esc":
		m.showRowDetail = false
	case "j", "down":
		m.rowDetailViewport.LineDown(1)
	case "k", "up":
		m.rowDetailViewport.LineUp(1)
	case "g":
		m.rowDetailViewport.GotoTop()
	case "G":
		m.rowDetailViewport.GotoBottom()
	}

	return m, nil, true
}

func (m Model) updateCopyMenu(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showCopyMenu {
		return m, nil, false
	}

	switch msg.String() {
	case "j", "down":
		if m.copyMenuIndex < 5 {
			m.copyMenuIndex++
		}
	case "k", "up":
		if m.copyMenuIndex > 0 {
			m.copyMenuIndex--
		}
	case "enter":
		m.copyRowToClipboard(CopyFormat(m.copyMenuIndex))
		m.focus = FocusQuery
		m.UpdateCursorStyle()
		return m, m.textarea.Focus(), true
	case "esc":
		m.showCopyMenu = false
	}

	return m, nil, true
}

func (m Model) updateHistorySearch(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showHistorySearch {
		return m, nil, false
	}

	indices := m.filteredHistoryIndices()
	switch msg.Type {
	case tea.KeyEsc:
		m.showHistorySearch = false
		return m, m.textarea.Focus(), true
	case tea.KeyEnter:
		if len(indices) > 0 && m.historySearchIndex >= 0 && m.historySearchIndex < len(indices) {
			query := m.history[indices[m.historySearchIndex]]
			m.textarea.SetValue(query)
			m.textarea.CursorEnd()
		}
		m.showHistorySearch = false
		m.showSuggestions = false
		return m, m.textarea.Focus(), true
	case tea.KeyUp:
		if m.historySearchIndex < len(indices)-1 {
			m.historySearchIndex++
		}
		m.historySearchClampScroll(m.historyListHeight())
		return m, nil, true
	case tea.KeyDown:
		if m.historySearchIndex > 0 {
			m.historySearchIndex--
		}
		m.historySearchClampScroll(m.historyListHeight())
		return m, nil, true
	case tea.KeyBackspace:
		if len(m.historySearchFilter) > 0 {
			m.historySearchFilter = m.historySearchFilter[:len(m.historySearchFilter)-1]
			m.historySearchIndex = 0
			m.historySearchScroll = 0
		}
		return m, nil, true
	case tea.KeyRunes:
		m.historySearchFilter += string(msg.Runes)
		m.historySearchIndex = 0
		m.historySearchScroll = 0
		return m, nil, true
	}

	if msg.Type == tea.KeySpace {
		m.historySearchFilter += " "
		m.historySearchIndex = 0
		m.historySearchScroll = 0
	}

	return m, nil, true
}

func (m Model) updateMenu(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if !m.showMenu {
		return m, nil, false
	}

	if m.menuType == MenuSaveFavorite {
		switch msg.Type {
		case tea.KeyEnter:
			m.SaveFavorite(m.favoriteInput)
			m.showMenu = false
			m.favoriteInput = ""
			m.menuFilter = ""
			return m, nil, true
		case tea.KeyEsc:
			m.showMenu = false
			m.favoriteInput = ""
			m.menuFilter = ""
			return m, nil, true
		case tea.KeyBackspace:
			if len(m.favoriteInput) > 0 {
				m.favoriteInput = m.favoriteInput[:len(m.favoriteInput)-1]
			}
			return m, nil, true
		case tea.KeyRunes:
			m.favoriteInput += string(msg.Runes)
			return m, nil, true
		}
		if msg.String() == "ctrl+ " || msg.String() == "ctrl+space" || msg.Type == tea.KeyCtrlAt {
			m.showMenu = false
			m.favoriteInput = ""
			m.menuFilter = ""
			return m, nil, true
		}
		return m, nil, true
	}

	switch msg.String() {
	case "up", "k":
		cmds := m.filteredMenuCommands()
		if len(cmds) > 0 {
			if m.menuIndex > 0 {
				m.menuIndex--
			} else {
				m.menuIndex = len(cmds) - 1
			}
		}
	case "down", "j":
		cmds := m.filteredMenuCommands()
		if len(cmds) > 0 {
			if m.menuIndex < len(cmds)-1 {
				m.menuIndex++
			} else {
				m.menuIndex = 0
			}
		}
	case "enter":
		oldType := m.menuType
		cmds := m.filteredMenuCommands()
		if len(cmds) == 0 {
			return m, nil, true
		}
		if m.menuIndex < 0 || m.menuIndex >= len(cmds) {
			m.menuIndex = 0
		}
		cmd := cmds[m.menuIndex].Action(&m)
		if m.menuType == oldType {
			m.showMenu = false
			m.menuFilter = ""
		}
		return m, cmd, true
	case "esc", "ctrl+ ", "ctrl+space":
		m.showMenu = false
		m.menuFilter = ""
		return m, nil, true
	}
	if msg.Type == tea.KeyCtrlAt {
		m.showMenu = false
		m.menuFilter = ""
		return m, nil, true
	}
	if msg.Type == tea.KeyBackspace {
		if len(m.menuFilter) > 0 {
			m.menuFilter = m.menuFilter[:len(m.menuFilter)-1]
		}
		m.menuIndex = 0
		return m, nil, true
	}
	if msg.Type == tea.KeyRunes {
		m.menuFilter += string(msg.Runes)
		m.menuIndex = 0
		return m, nil, true
	}

	return m, nil, true
}
