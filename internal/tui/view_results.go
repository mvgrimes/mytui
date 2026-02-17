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

// nextColumnBoundary finds the next column separator position in the header line.
// For table/unicode formats, columns are separated by │ or |.
// If forward is true, finds the next boundary after currentOffset; otherwise finds the previous one.
func nextColumnBoundary(header string, currentOffset int, forward bool) int {
	if header == "" {
		return currentOffset
	}
	// Use the second line (column headers) if available, otherwise use first line
	lines := strings.Split(header, "\n")
	line := lines[0]
	if len(lines) > 1 {
		line = lines[1]
	}

	// Find all column separator positions (│ or |)
	var positions []int
	for i := 0; i < len(line); {
		r := rune(line[i])
		// Check for multi-byte │ (U+2502, encoded as 3 bytes in UTF-8: E2 94 82)
		if i+2 < len(line) && line[i] == 0xE2 && line[i+1] == 0x94 && line[i+2] == 0x82 {
			positions = append(positions, i)
			i += 3
			continue
		}
		if r == '|' {
			positions = append(positions, i)
		}
		i++
	}

	if len(positions) == 0 {
		return currentOffset
	}

	if forward {
		for _, p := range positions {
			if p > currentOffset {
				return p
			}
		}
		// Already past last separator
		return currentOffset
	}

	// backward
	for i := len(positions) - 1; i >= 0; i-- {
		if positions[i] < currentOffset {
			return positions[i]
		}
	}
	return 0
}
