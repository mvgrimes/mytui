package modals

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mvgrimes/mytui/internal/tui/styles"
)

// RenderHistorySearch renders the full-screen FZF-style history search modal.
func RenderHistorySearch(m *HistorySearchModel, history []string, timestamps []string, width int, height int) string {
	modalWidth := width - 4
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Inner width (inside border+padding)
	innerWidth := modalWidth - 6 // border(2) + padding(2*2)
	if innerWidth < 30 {
		innerWidth = 30
	}

	indices := filteredHistoryIndices(history, m.Filter)
	total := len(history)
	matched := len(indices)

	// Compute the list height: modal height minus fixed chrome lines.
	// Fixed lines: title(1) + blank(1) + separator(1) + count(1) + blank(1) +
	//              preview_header(1) + preview lines(3) + blank(1) + filter(1) + help(1) = 12
	modalHeight := height - 4
	if modalHeight < 10 {
		modalHeight = 10
	}
	chromeLines := 12
	listHeight := modalHeight - chromeLines
	if listHeight < 3 {
		listHeight = 3
	}

	// Clamp index and scroll without mutating receiver (copy values)
	selIdx := m.Index
	scroll := m.Scroll
	if len(indices) == 0 {
		selIdx = -1
		scroll = 0
	} else {
		if selIdx < 0 {
			selIdx = 0
		}
		if selIdx >= len(indices) {
			selIdx = len(indices) - 1
		}
		if selIdx < scroll {
			scroll = selIdx
		}
		if selIdx >= scroll+listHeight {
			scroll = selIdx - listHeight + 1
		}
		if scroll < 0 {
			scroll = 0
		}
	}

	// Build list lines
	var listLines []string
	end := scroll + listHeight
	if end > len(indices) {
		end = len(indices)
	}
	for i := end - 1; i >= scroll; i-- {
		histIdx := indices[i]
		ts := ""
		if histIdx < len(timestamps) {
			ts = timestamps[histIdx]
		}
		query := history[histIdx]
		// Replace newlines with spaces for single-line display
		query = strings.ReplaceAll(query, "\n", " ")

		prefix := "  "
		var line string
		if ts != "" {
			line = fmt.Sprintf("%s %s", ts, query)
		} else {
			line = query
		}
		// Truncate to fit
		maxLineWidth := innerWidth - len(prefix)
		if maxLineWidth < 1 {
			maxLineWidth = 1
		}
		if len(line) > maxLineWidth {
			line = line[:maxLineWidth]
		}

		if i == selIdx {
			line = styles.HistorySelectedStyle.Width(innerWidth).Render("> " + line)
		} else {
			line = styles.HistoryItemStyle.Width(innerWidth).Render(prefix + line)
		}
		listLines = append(listLines, line)
	}
	// Pad list to listHeight so layout is stable
	for len(listLines) < listHeight {
		listLines = append(listLines, strings.Repeat(" ", innerWidth))
	}

	// Separator
	separator := strings.Repeat("─", innerWidth)
	separatorLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(separator)

	// Count
	countLine := styles.HistoryCountStyle.Render(fmt.Sprintf("%d/%d", matched, total))

	// Preview: full text of the selected entry
	previewQuery := ""
	if selIdx >= 0 && selIdx < len(indices) {
		previewQuery = history[indices[selIdx]]
	}
	// Wrap preview to innerWidth, show up to 3 lines.
	// Split on existing newlines first, then wrap each segment.
	var previewLines []string
	for _, segment := range strings.Split(previewQuery, "\n") {
		previewLines = append(previewLines, wrapText(segment, innerWidth)...)
		if len(previewLines) >= 3 {
			break
		}
	}
	for len(previewLines) < 3 {
		previewLines = append(previewLines, "")
	}
	if len(previewLines) > 3 {
		previewLines = previewLines[:3]
	}
	previewRendered := make([]string, len(previewLines))
	for i, pl := range previewLines {
		previewRendered[i] = styles.HistoryPreviewStyle.Width(innerWidth).Render(pl)
	}

	// Filter input
	filterLine := styles.HistoryFilterStyle.Render("> " + m.Filter + "_")

	// Help
	helpLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).
		Render("Up/Down: navigate  Enter: select  Esc: cancel  type to fuzzy filter")

	title := styles.HistoryTitleStyle.Render("HISTORY SEARCH")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(listLines, "\n"),
		separatorLine,
		countLine,
		"",
		strings.Join(previewRendered, "\n"),
		"",
		filterLine,
		helpLine,
	)

	return styles.HistoryBorderStyle.
		Width(innerWidth + 4). // +4 for padding(2*2)
		Render(content)
}

// wrapText wraps text to the given width, splitting on spaces where possible.
func wrapText(text string, width int) []string {
	if text == "" || width <= 0 {
		return []string{""}
	}
	var lines []string
	for len(text) > width {
		// Try to break on a space
		breakAt := width
		for breakAt > 0 && text[breakAt-1] != ' ' {
			breakAt--
		}
		if breakAt == 0 {
			breakAt = width
		}
		lines = append(lines, text[:breakAt])
		text = strings.TrimLeft(text[breakAt:], " ")
	}
	lines = append(lines, text)
	return lines
}
