package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) renderResultHeader(r *Result, focused bool) string {
	cleanQuery := strings.ReplaceAll(r.Query, "\n", " ")
	cleanQuery = strings.Join(strings.Fields(cleanQuery), " ")
	// Account for: icon (3), separators (3*3=9), timestamp (8), duration (~7), rows (~12)
	maxWidth := max(m.width-42, 10)
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

	style := headerStyle
	if focused {
		style = headerFocusStyle
	}

	return style.Width(m.width).Render(header)
}

// maxContentWidth returns the maximum line width in a multi-line string.
// Uses ANSI-aware width calculation to properly handle color codes.
func maxContentWidth(s string) int {
	maxWidth := 0
	for _, line := range strings.Split(s, "\n") {
		w := ansi.StringWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

// applyHorizontalOffset shifts a multi-line string horizontally by the given offset.
// This is used to sync the pinned header with the viewport's horizontal scroll.
// Uses ANSI-aware truncation to properly handle color codes.
func applyHorizontalOffset(s string, offset, width int) string {
	if offset == 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	result := make([]string, len(lines))

	for i, line := range lines {
		// Use ANSI-aware truncation from the left
		if offset > 0 {
			line = ansi.TruncateLeft(line, offset, "")
		}
		// Pad to width if needed (using ANSI-aware width calculation)
		lineWidth := ansi.StringWidth(line)
		if lineWidth < width {
			line = line + strings.Repeat(" ", width-lineWidth)
		}
		result[i] = line
	}

	return strings.Join(result, "\n")
}
