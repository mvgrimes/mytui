package tui

import (
	"fmt"
	"strings"
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
