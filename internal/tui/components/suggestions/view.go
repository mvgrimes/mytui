package suggestions

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

func Render(m *Model) string {
	if len(m.Items) == 0 {
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
	if m.Index > 5 {
		start = m.Index - 5
	}

	for i := start; i < len(m.Items) && i < start+10; i++ {
		s := m.Items[i]
		line := fmt.Sprintf("%-20s %s", s.Text, s.Description)
		if i == m.Index {
			b.WriteString(activeStyle.Render(line) + "\n")
		} else {
			b.WriteString(style.Render(line) + "\n")
		}
	}
	if len(m.Items) > start+10 {
		b.WriteString(style.Render("...") + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(strings.TrimSuffix(b.String(), "\n"))
}

// ComputeOffsets calculates the overlay offsets to place the suggestions popup
// near the cursor. Prefers placing above the cursor, but if there's not enough space,
// places below the cursor instead.
func ComputeOffsets(resultsLines, fgHeight int, ta *textarea.Model) (int, int) {
	// Where the query area starts (number of lines in results view plus a header line)
	queryTop := resultsLines + 1
	// Cursor position within textarea
	curLine := ta.Line()
	curCol := ta.LineInfo().ColumnOffset
	// Account for line number gutter "NN | " which is width 5
	gutter := 5
	xOff := gutter + curCol + 1

	// Try to place above the cursor
	yOff := queryTop + curLine - fgHeight

	// If not enough space above (would go off-screen or overlap with top),
	// place below the cursor instead
	if yOff < 0 {
		yOff = queryTop + curLine + 1
	}

	return xOff, yOff
}
