package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderSuggestions() string {
	if len(m.suggestions) == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1)

	activeStyle := style.Copy().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00AAFF")).
		Bold(true)

	var b strings.Builder
	start := 0
	if m.suggestionIndex > 5 {
		start = m.suggestionIndex - 5
	}

	for i := start; i < len(m.suggestions) && i < start+10; i++ {
		s := m.suggestions[i]
		line := fmt.Sprintf("%-20s %s", s.Text, s.Description)
		if i == m.suggestionIndex {
			b.WriteString(activeStyle.Render(line) + "\n")
		} else {
			b.WriteString(style.Render(line) + "\n")
		}
	}
	if len(m.suggestions) > start+10 {
		b.WriteString(style.Render("...") + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(strings.TrimSuffix(b.String(), "\n"))
}

// computeSuggestionOffsets calculates the overlay offsets to place the suggestions popup
// near the cursor, just above and to the right, given the current layout.
func (m Model) computeSuggestionOffsets(resultsLines, fgHeight int) (int, int) {
	// Where the query area starts (number of lines in results view plus a header line)
	queryTop := resultsLines + 1
	// Cursor position within textarea
	curLine := m.textarea.Line()
	curCol := m.textarea.LineInfo().ColumnOffset
	// Account for line number gutter "NN | " which is width 5
	gutter := 5
	xOff := gutter + curCol + 1
	yOff := queryTop + curLine - fgHeight
	return xOff, yOff
}
