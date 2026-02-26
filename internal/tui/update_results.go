package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/formatter"
)

func (m Model) updateResultsKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.focus != FocusResults || m.focusedResult < 0 || m.focusedResult >= len(m.results) {
		return m, nil, false
	}

	res := m.results[m.focusedResult]

	if res.SearchActive {
		switch msg.Type {
		case tea.KeyRunes:
			res.SearchInput += string(msg.Runes)
			if match := findMatchingRow(res, res.SearchInput, 0, true); match >= 0 {
				res.SelectedRow = match
				m.ensureSelectionVisible(res)
			}
		case tea.KeyBackspace:
			if len(res.SearchInput) > 0 {
				res.SearchInput = res.SearchInput[:len(res.SearchInput)-1]
			}
			if res.SearchInput != "" {
				if match := findMatchingRow(res, res.SearchInput, 0, true); match >= 0 {
					res.SelectedRow = match
					m.ensureSelectionVisible(res)
				}
			}
		case tea.KeyEnter:
			res.SearchQuery = res.SearchInput
			res.SearchActive = false
		case tea.KeyEsc:
			res.SelectedRow = res.PreSearchRow
			res.SearchInput = ""
			res.SearchActive = false
			m.ensureSelectionVisible(res)
		}
		return m, nil, true
	}

	totalRows := 0
	if res.DbResult != nil {
		totalRows = len(res.DbResult.Rows)
	}

	if m.vimPendingKey == "g" && msg.String() == "g" {
		m.vimPendingKey = ""
		if totalRows > 0 {
			res.SelectedRow = 0
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.GotoTop()
		}
		return m, nil, true
	}
	if m.vimPendingKey == "g" && msg.String() != "g" {
		m.vimPendingKey = ""
	}

	switch msg.String() {
	case "/":
		if totalRows > 0 {
			res.SearchActive = true
			res.SearchInput = ""
			res.PreSearchRow = res.SelectedRow
		}
		return m, nil, true
	case "n":
		if res.SearchQuery != "" && totalRows > 0 {
			if match := findMatchingRow(res, res.SearchQuery, res.SelectedRow+1, true); match >= 0 {
				res.SelectedRow = match
				m.ensureSelectionVisible(res)
			}
		}
		return m, nil, true
	case "N":
		if res.SearchQuery != "" && totalRows > 0 {
			start := res.SelectedRow - 1
			if start < 0 {
				start = totalRows - 1
			}
			if match := findMatchingRow(res, res.SearchQuery, start, false); match >= 0 {
				res.SelectedRow = match
				m.ensureSelectionVisible(res)
			}
		}
		return m, nil, true
	case "q", "esc":
		m.focus = FocusQuery
		m.focusedResult = -1
		return m, m.textarea.Focus(), true
	case "j", "down":
		if totalRows > 0 && res.SelectedRow < totalRows-1 {
			res.SelectedRow++
			m.ensureSelectionVisible(res)
		} else if totalRows == 0 {
			res.Viewport.LineDown(1)
		}
		return m, nil, true
	case "k", "up":
		if totalRows > 0 && res.SelectedRow > 0 {
			res.SelectedRow--
			m.ensureSelectionVisible(res)
		} else if totalRows == 0 {
			res.Viewport.LineUp(1)
		}
		return m, nil, true
	case "h", "left":
		res.Viewport.ScrollLeft(5)
		res.XOffset -= 5
		if res.XOffset < 0 {
			res.XOffset = 0
		}
		return m, nil, true
	case "l", "right":
		res.Viewport.ScrollRight(5)
		res.XOffset += 5
		contentWidth := maxContentWidth(res.Formatted)
		maxOffset := contentWidth - m.width
		if maxOffset < 0 {
			maxOffset = 0
		}
		if res.XOffset > maxOffset {
			res.XOffset = maxOffset
		}
		return m, nil, true
	case "0", "^":
		if res.XOffset > 0 {
			res.Viewport.ScrollLeft(res.XOffset)
			res.XOffset = 0
		}
		return m, nil, true
	case "$":
		contentWidth := maxContentWidth(res.Formatted)
		maxOffset := contentWidth - m.width
		if maxOffset < 0 {
			maxOffset = 0
		}
		if res.XOffset < maxOffset {
			res.Viewport.ScrollRight(maxOffset - res.XOffset)
			res.XOffset = maxOffset
		}
		return m, nil, true
	case "w":
		colOffset := nextColumnBoundary(res.FormattedHeader, res.XOffset, true)
		delta := colOffset - res.XOffset
		if delta > 0 {
			res.Viewport.ScrollRight(delta)
			res.XOffset = colOffset
		}
		return m, nil, true
	case "b":
		colOffset := nextColumnBoundary(res.FormattedHeader, res.XOffset, false)
		delta := res.XOffset - colOffset
		if delta > 0 {
			res.Viewport.ScrollLeft(delta)
			res.XOffset = colOffset
		}
		return m, nil, true
	case "g":
		m.vimPendingKey = "g"
		return m, nil, true
	case "G":
		if totalRows > 0 {
			res.SelectedRow = totalRows - 1
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.GotoBottom()
		}
		return m, nil, true
	case "ctrl+u":
		if totalRows > 0 {
			halfPage := res.Viewport.Height / 2
			res.SelectedRow -= halfPage
			if res.SelectedRow < 0 {
				res.SelectedRow = 0
			}
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewUp()
		}
		return m, nil, true
	case "ctrl+d":
		if totalRows > 0 {
			halfPage := res.Viewport.Height / 2
			res.SelectedRow += halfPage
			if res.SelectedRow >= totalRows {
				res.SelectedRow = totalRows - 1
			}
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewDown()
		}
		return m, nil, true
	case "pgup":
		if totalRows > 0 {
			res.SelectedRow -= res.Viewport.Height
			if res.SelectedRow < 0 {
				res.SelectedRow = 0
			}
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewUp()
			res.Viewport.HalfViewUp()
		}
		return m, nil, true
	case "pgdown":
		if totalRows > 0 {
			res.SelectedRow += res.Viewport.Height
			if res.SelectedRow >= totalRows {
				res.SelectedRow = totalRows - 1
			}
			m.ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewDown()
			res.Viewport.HalfViewDown()
		}
		return m, nil, true
	case "enter":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			m.openRowDetailModal(res)
		}
		return m, nil, true
	case "y":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			m.showCopyMenu = true
			m.copyMenuIndex = 0
		}
		return m, nil, true
	case "v":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			return m, m.openRowInEditor(res), true
		}
		return m, nil, true
	case "e":
		res.Expanded = true
		m.recalculateHeight()
		return m, nil, true
	case "c":
		res.Expanded = false
		m.recalculateHeight()
		return m, nil, true
	case "+":
		res.DisplaySize += 2
		m.recalculateHeight()
		return m, nil, true
	case "-":
		if res.DisplaySize > 2 {
			res.DisplaySize -= 2
		}
		m.recalculateHeight()
		return m, nil, true
	case "d":
		m.results = append(m.results[:m.focusedResult], m.results[m.focusedResult+1:]...)
		if len(m.results) == 0 {
			m.focus = FocusQuery
			m.focusedResult = -1
			return m, m.textarea.Focus(), true
		}
		if m.focusedResult >= len(m.results) {
			m.focusedResult = len(m.results) - 1
		}
		m.recalculateHeight()
		return m, nil, true
	case "r":
		m.textarea.SetValue(res.Query)
		m.focus = FocusQuery
		m.focusedResult = -1
		return m, m.textarea.Focus(), true
	case "R":
		newResult, err := m.conn.ExecuteQuery(res.Query)
		if err == nil {
			res.DbResult = newResult
			res.Formatted = formatter.FormatResult(newResult, res.Format, m.config)
			res.FormattedHeader, res.FormattedData = splitTableHeaderAndData(res.Formatted, res.Format)
			res.Viewport.SetContent(res.FormattedData)
			if res.XOffset > 0 {
				res.Viewport.ScrollLeft(res.XOffset)
				res.XOffset = 0
			}
			res.Viewport.GotoTop()
			res.Timestamp = time.Now()
			res.Duration = newResult.Duration
		}
		return m, nil, true
	}

	return m, nil, false
}
