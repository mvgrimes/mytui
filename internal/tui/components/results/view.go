package results

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/tui/styles"
)

func renderResultHeader(r *core.Result, focused bool, width int) string {
	cleanQuery := strings.ReplaceAll(r.Query, "\n", " ")
	cleanQuery = strings.Join(strings.Fields(cleanQuery), " ")
	// Account for: icon (3), separators (3*3=9), timestamp (8), duration (~7), rows (~12)
	maxWidth := max(width-42, 10)
	if len(cleanQuery) > maxWidth {
		cleanQuery = cleanQuery[:maxWidth-3] + "..."
	}

	icon := " ▶ "
	if r.Expanded {
		icon = " ▼ "
	}

	count := 0
	if r.DbResult != nil {
		count = len(r.DbResult.Rows)
	}

	ts := r.Timestamp.Format("15:04:05")
	duration := fmt.Sprintf("%.2fs", r.Duration.Seconds())
	header := fmt.Sprintf("%s %-*s | %s | %s | %d rows", icon, maxWidth, cleanQuery, ts, duration, count)

	style := styles.HeaderStyle
	if focused {
		style = styles.HeaderFocusStyle
	}

	return style.Width(width).Render(header)
}

// highlightSelectedRow applies a highlight to the selected row in the formatted data.
// It handles ANSI-colored content properly.
func highlightSelectedRow(data string, selectedRow int, width int) string {
	if selectedRow < 0 {
		return data
	}

	lines := strings.Split(data, "\n")
	if selectedRow >= len(lines) {
		return data
	}

	// Strip ANSI codes to get raw text, then re-apply with background
	line := lines[selectedRow]
	// Preserve the original line content but add background highlight
	highlighted := styles.SelectedRowStyle.Width(width).Render(ansi.Strip(line))
	lines[selectedRow] = highlighted

	return strings.Join(lines, "\n")
}

func RenderPanel(m *Model, focus core.Focus, width int, availableHeight int) (string, int) {
	var resultsView []string
	for i, r := range m.Results {
		focused := focus == core.FocusResults && m.FocusedResultIndex == i
		resultsView = append(resultsView, renderResultHeader(r, focused, width))
		if r.Expanded {
			if r.FormattedHeader != "" {
				header := applyHorizontalOffset(r.FormattedHeader, r.XOffset, width)
				resultsView = append(resultsView, header)
			}

			viewContent := r.Viewport.View()
			if focused && r.DbResult != nil && len(r.DbResult.Rows) > 0 && r.SelectedRow >= 0 {
				viewportSelectedRow := r.SelectedRow - r.Viewport.YOffset()
				if viewportSelectedRow >= 0 && viewportSelectedRow < r.Viewport.Height() {
					viewContent = highlightSelectedRow(viewContent, viewportSelectedRow, width)
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

	visibleResultsStr := ""
	if len(resultsView) > 0 {
		fullResultsStr := strings.Join(resultsView, "\n")
		allResultLines := strings.Split(fullResultsStr, "\n")
		scrollOff := m.ScrollOffset
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
