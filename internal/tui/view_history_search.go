package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	fzfalgo "github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// filteredHistoryIndices returns indices into m.history that match the current
// historySearchFilter using fuzzy matching (fzf algorithm).
// With no filter: newest-first order. With a filter: sorted by score descending
// (best match at index 0, which appears at the bottom of the display).
func (m *Model) filteredHistoryIndices() []int {
	if m.historySearchFilter == "" {
		out := make([]int, len(m.history))
		for i := range out {
			out[i] = len(m.history) - 1 - i
		}
		return out
	}

	// fzf requires the pattern to be lowercased when caseSensitive=false.
	pattern := []rune(strings.ToLower(m.historySearchFilter))
	slab := util.MakeSlab(100*1024, 2048)

	type scored struct {
		idx   int
		score int
	}
	var matches []scored

	for i := len(m.history) - 1; i >= 0; i-- {
		chars := util.ToChars([]byte(m.history[i]))
		result, _ := fzfalgo.FuzzyMatchV2(false, false, true, &chars, pattern, false, slab)
		if result.Start >= 0 {
			matches = append(matches, scored{idx: i, score: result.Score})
		}
	}

	// Sort by score descending so the best match is at index 0 (bottom of list).
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	out := make([]int, len(matches))
	for i, s := range matches {
		out[i] = s.idx
	}
	return out
}

// openHistorySearch opens the history search modal positioned at the most
// recent entry.
func (m *Model) openHistorySearch() {
	m.historySearchFilter = ""
	m.historySearchScroll = 0
	m.historySearchIndex = 0 // 0 = most recent entry in filtered list
	m.showHistorySearch = true
}

// historySearchClampScroll adjusts historySearchScroll so that the selected
// row is visible inside the list area.
func (m *Model) historySearchClampScroll(listHeight int) {
	indices := m.filteredHistoryIndices()
	if len(indices) == 0 {
		m.historySearchScroll = 0
		return
	}
	if m.historySearchIndex < 0 {
		m.historySearchIndex = 0
	}
	if m.historySearchIndex >= len(indices) {
		m.historySearchIndex = len(indices) - 1
	}
	if m.historySearchIndex < m.historySearchScroll {
		m.historySearchScroll = m.historySearchIndex
	}
	if m.historySearchIndex >= m.historySearchScroll+listHeight {
		m.historySearchScroll = m.historySearchIndex - listHeight + 1
	}
	if m.historySearchScroll < 0 {
		m.historySearchScroll = 0
	}
}

// renderHistorySearch renders the full-screen FZF-style history search modal.
func (m Model) renderHistorySearch() string {
	modalWidth := m.width - 4
	if modalWidth < 40 {
		modalWidth = 40
	}

	// Inner width (inside border+padding)
	innerWidth := modalWidth - 6 // border(2) + padding(2*2)
	if innerWidth < 30 {
		innerWidth = 30
	}

	indices := m.filteredHistoryIndices()
	total := len(m.history)
	matched := len(indices)

	// Compute the list height: modal height minus fixed chrome lines.
	// Fixed lines: title(1) + blank(1) + separator(1) + count(1) + blank(1) +
	//              preview_header(1) + preview lines(3) + blank(1) + filter(1) + help(1) = 12
	modalHeight := m.height - 4
	if modalHeight < 10 {
		modalHeight = 10
	}
	chromeLines := 12
	listHeight := modalHeight - chromeLines
	if listHeight < 3 {
		listHeight = 3
	}

	// Clamp index and scroll without mutating receiver (copy values)
	selIdx := m.historySearchIndex
	scroll := m.historySearchScroll
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
		if histIdx < len(m.historyTimestamps) {
			ts = m.historyTimestamps[histIdx]
		}
		query := m.history[histIdx]
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
			line = historySelectedStyle.Width(innerWidth).Render("> " + line)
		} else {
			line = historyItemStyle.Width(innerWidth).Render(prefix + line)
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
	countLine := historyCountStyle.Render(fmt.Sprintf("%d/%d", matched, total))

	// Preview: full text of the selected entry
	previewQuery := ""
	if selIdx >= 0 && selIdx < len(indices) {
		previewQuery = m.history[indices[selIdx]]
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
		previewRendered[i] = historyPreviewStyle.Width(innerWidth).Render(pl)
	}

	// Filter input
	filterLine := historyFilterStyle.Render("> " + m.historySearchFilter + "_")

	// Help
	helpLine := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).
		Render("Up/Down: navigate  Enter: select  Esc: cancel  type to fuzzy filter")

	title := historyTitleStyle.Render("HISTORY SEARCH")

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

	return historyBorderStyle.
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
