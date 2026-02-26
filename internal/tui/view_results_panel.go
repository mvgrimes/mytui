package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderResultsPanel() (string, int) {
	var resultsView []string
	for i, r := range m.results {
		focused := m.focus == FocusResults && m.focusedResult == i
		resultsView = append(resultsView, m.renderResultHeader(r, focused))
		if r.Expanded {
			if r.FormattedHeader != "" {
				header := applyHorizontalOffset(r.FormattedHeader, r.XOffset, m.width)
				resultsView = append(resultsView, header)
			}

			viewContent := r.Viewport.View()
			if focused && r.DbResult != nil && len(r.DbResult.Rows) > 0 && r.SelectedRow >= 0 {
				viewportSelectedRow := r.SelectedRow - r.Viewport.YOffset
				if viewportSelectedRow >= 0 && viewportSelectedRow < r.Viewport.Height {
					viewContent = highlightSelectedRow(viewContent, viewportSelectedRow, m.width)
				}
			}
			if focused && r.SearchActive {
				vcLines := strings.Split(viewContent, "\n")
				if len(vcLines) > 1 {
					vcLines[len(vcLines)-1] = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("/ " + r.SearchInput + "▏")
					viewContent = strings.Join(vcLines, "\n")
				}
			}
			resultsView = append(resultsView, viewContent)
		}
	}

	availableHeight := m.computeAvailableHeight()
	visibleResultsStr := ""
	if len(resultsView) > 0 {
		fullResultsStr := strings.Join(resultsView, "\n")
		allResultLines := strings.Split(fullResultsStr, "\n")
		scrollOff := m.resultsScrollOffset
		if scrollOff > len(allResultLines) {
			scrollOff = len(allResultLines)
		}
		end := scrollOff + availableHeight
		if end > len(allResultLines) {
			end = len(allResultLines)
		}
		if scrollOff < end {
			visibleResultsStr = strings.Join(allResultLines[scrollOff:end], "\n")
		}
	}

	visibleResultLines := 0
	if visibleResultsStr != "" {
		visibleResultLines = strings.Count(visibleResultsStr, "\n") + 1
	}

	return visibleResultsStr, visibleResultLines
}
