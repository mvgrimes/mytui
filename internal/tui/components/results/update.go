package results

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

type UpdateDeps struct {
	Focus              core.Focus
	FocusedResultIndex int
	Width              int
	Config             *config.Config
	ConnExecute        func(string) (*db.Result, error)
	OpenRowDetail      func(res *core.Result)
	OpenCopyMenu       func()
	OpenRowInEditor    func(res *core.Result) tea.Cmd
	SetFocus           func(core.Focus)
	SetFocusedResult   func(int)
	SetQueryText       func(string)
	TextareaFocus      func() tea.Cmd
	RecalculateHeight  func()
}

func UpdateKey(m *Model, msg tea.KeyMsg, deps UpdateDeps) (bool, tea.Cmd) {
	if deps.Focus != core.FocusResults || deps.FocusedResultIndex < 0 || deps.FocusedResultIndex >= len(m.Results) {
		return false, nil
	}

	res := m.Results[deps.FocusedResultIndex]

	if res.SearchActive {
		switch msg.Type {
		case tea.KeyRunes:
			res.SearchInput += string(msg.Runes)
			if match := findMatchingRow(res, res.SearchInput, 0, true); match >= 0 {
				res.SelectedRow = match
				ensureSelectionVisible(res)
			}
		case tea.KeyBackspace:
			if len(res.SearchInput) > 0 {
				res.SearchInput = res.SearchInput[:len(res.SearchInput)-1]
			}
			if res.SearchInput != "" {
				if match := findMatchingRow(res, res.SearchInput, 0, true); match >= 0 {
					res.SelectedRow = match
					ensureSelectionVisible(res)
				}
			}
		case tea.KeyEnter:
			res.SearchQuery = res.SearchInput
			res.SearchActive = false
		case tea.KeyEsc:
			res.SelectedRow = res.PreSearchRow
			res.SearchInput = ""
			res.SearchActive = false
			ensureSelectionVisible(res)
		}
		return true, nil
	}

	totalRows := 0
	if res.DbResult != nil {
		totalRows = len(res.DbResult.Rows)
	}

	if m.VimPendingKey == "g" && msg.String() == "g" {
		m.VimPendingKey = ""
		if totalRows > 0 {
			res.SelectedRow = 0
			ensureSelectionVisible(res)
		} else {
			res.Viewport.GotoTop()
		}
		return true, nil
	}
	if m.VimPendingKey == "g" && msg.String() != "g" {
		m.VimPendingKey = ""
	}

	switch msg.String() {
	case "/":
		if totalRows > 0 {
			res.SearchActive = true
			res.SearchInput = ""
			res.PreSearchRow = res.SelectedRow
		}
		return true, nil
	case "n":
		if res.SearchQuery != "" && totalRows > 0 {
			if match := findMatchingRow(res, res.SearchQuery, res.SelectedRow+1, true); match >= 0 {
				res.SelectedRow = match
				ensureSelectionVisible(res)
			}
		}
		return true, nil
	case "N":
		if res.SearchQuery != "" && totalRows > 0 {
			start := res.SelectedRow - 1
			if start < 0 {
				start = totalRows - 1
			}
			if match := findMatchingRow(res, res.SearchQuery, start, false); match >= 0 {
				res.SelectedRow = match
				ensureSelectionVisible(res)
			}
		}
		return true, nil
	case "q", "esc":
		deps.SetFocus(core.FocusQuery)
		deps.SetFocusedResult(-1)
		return true, deps.TextareaFocus()
	case "j", "down":
		if totalRows > 0 && res.SelectedRow < totalRows-1 {
			res.SelectedRow++
			ensureSelectionVisible(res)
		} else if totalRows > 0 {
			scrollBottomBorderIntoView(res)
		} else if totalRows == 0 {
			res.Viewport.LineDown(1)
		}
		return true, nil
	case "k", "up":
		if totalRows > 0 && res.SelectedRow > 0 {
			res.SelectedRow--
			ensureSelectionVisible(res)
		} else if totalRows == 0 {
			res.Viewport.LineUp(1)
		}
		return true, nil
	case "h", "left":
		res.Viewport.ScrollLeft(5)
		res.XOffset -= 5
		if res.XOffset < 0 {
			res.XOffset = 0
		}
		return true, nil
	case "l", "right":
		res.Viewport.ScrollRight(5)
		res.XOffset += 5
		contentWidth := maxContentWidth(res.Formatted)
		maxOffset := contentWidth - deps.Width
		if maxOffset < 0 {
			maxOffset = 0
		}
		if res.XOffset > maxOffset {
			res.XOffset = maxOffset
		}
		return true, nil
	case "0", "^":
		if res.XOffset > 0 {
			res.Viewport.ScrollLeft(res.XOffset)
			res.XOffset = 0
		}
		return true, nil
	case "$":
		contentWidth := maxContentWidth(res.Formatted)
		maxOffset := contentWidth - deps.Width
		if maxOffset < 0 {
			maxOffset = 0
		}
		if res.XOffset < maxOffset {
			res.Viewport.ScrollRight(maxOffset - res.XOffset)
			res.XOffset = maxOffset
		}
		return true, nil
	case "w":
		colOffset := nextColumnBoundary(res.FormattedHeader, res.XOffset, true)
		delta := colOffset - res.XOffset
		if delta > 0 {
			res.Viewport.ScrollRight(delta)
			res.XOffset = colOffset
		}
		return true, nil
	case "b":
		colOffset := nextColumnBoundary(res.FormattedHeader, res.XOffset, false)
		delta := res.XOffset - colOffset
		if delta > 0 {
			res.Viewport.ScrollLeft(delta)
			res.XOffset = colOffset
		}
		return true, nil
	case "g":
		m.VimPendingKey = "g"
		return true, nil
	case "G":
		if totalRows > 0 {
			res.SelectedRow = totalRows - 1
			ensureSelectionVisible(res)
			scrollBottomBorderIntoView(res)
		} else {
			res.Viewport.GotoBottom()
		}
		return true, nil
	case "ctrl+u":
		if totalRows > 0 {
			halfPage := res.Viewport.Height / 2
			res.SelectedRow -= halfPage
			if res.SelectedRow < 0 {
				res.SelectedRow = 0
			}
			ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewUp()
		}
		return true, nil
	case "ctrl+d":
		if totalRows > 0 {
			halfPage := res.Viewport.Height / 2
			res.SelectedRow += halfPage
			pastLastRow := res.SelectedRow >= totalRows
			if res.SelectedRow >= totalRows {
				res.SelectedRow = totalRows - 1
			}
			ensureSelectionVisible(res)
			if pastLastRow {
				scrollBottomBorderIntoView(res)
			}
		} else {
			res.Viewport.HalfViewDown()
		}
		return true, nil
	case "pgup":
		if totalRows > 0 {
			res.SelectedRow -= res.Viewport.Height
			if res.SelectedRow < 0 {
				res.SelectedRow = 0
			}
			ensureSelectionVisible(res)
		} else {
			res.Viewport.HalfViewUp()
			res.Viewport.HalfViewUp()
		}
		return true, nil
	case "pgdown":
		if totalRows > 0 {
			res.SelectedRow += res.Viewport.Height
			pastLastRow := res.SelectedRow >= totalRows
			if res.SelectedRow >= totalRows {
				res.SelectedRow = totalRows - 1
			}
			ensureSelectionVisible(res)
			if pastLastRow {
				scrollBottomBorderIntoView(res)
			}
		} else {
			res.Viewport.HalfViewDown()
			res.Viewport.HalfViewDown()
		}
		return true, nil
	case "enter":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			deps.OpenRowDetail(res)
		}
		return true, nil
	case "y":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			deps.OpenCopyMenu()
		}
		return true, nil
	case "v":
		if res.SelectedRow >= 0 && res.DbResult != nil && len(res.DbResult.Rows) > 0 {
			return true, deps.OpenRowInEditor(res)
		}
		return true, nil
	case "e":
		res.Expanded = true
		deps.RecalculateHeight()
		return true, nil
	case "c":
		res.Expanded = false
		deps.RecalculateHeight()
		return true, nil
	case "+":
		res.DisplaySize += 2
		deps.RecalculateHeight()
		return true, nil
	case "-":
		if res.DisplaySize > 2 {
			res.DisplaySize -= 2
		}
		deps.RecalculateHeight()
		return true, nil
	case "d":
		m.Results = append(m.Results[:deps.FocusedResultIndex], m.Results[deps.FocusedResultIndex+1:]...)
		if len(m.Results) == 0 {
			deps.SetFocus(core.FocusQuery)
			deps.SetFocusedResult(-1)
			return true, deps.TextareaFocus()
		}
		if deps.FocusedResultIndex >= len(m.Results) {
			deps.SetFocusedResult(len(m.Results) - 1)
		}
		deps.RecalculateHeight()
		return true, nil
	case "r":
		deps.SetQueryText(res.Query)
		deps.SetFocus(core.FocusQuery)
		deps.SetFocusedResult(-1)
		return true, deps.TextareaFocus()
	case "R":
		newResult, err := deps.ConnExecute(res.Query)
		if err == nil {
			res.DbResult = newResult
			res.Formatted = formatter.FormatResult(newResult, res.Format, deps.Config)
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
		return true, nil
	}

	return false, nil
}
