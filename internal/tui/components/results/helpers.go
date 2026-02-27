package results

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func AddResult(m *Model, result *db.Result, query string, format formatter.Format, width int, maxResults int, cfg *config.Config) {
	fullResult := formatter.FormatResult(result, format, cfg)

	// Split into header (first 3 lines) and data for pinned header
	header, data := splitTableHeaderAndData(fullResult, format)

	r := &core.Result{
		Query:           query,
		Timestamp:       time.Now(),
		Duration:        result.Duration,
		DisplaySize:     10,
		Expanded:        true,
		DbResult:        result,
		Formatted:       fullResult,
		FormattedHeader: header,
		FormattedData:   data,
		Viewport:        viewport.New(width, 7),
		Format:          format,
		SelectedRow:     0,
	}
	r.Viewport.SetContent(data)
	m.Results = append(m.Results, r)
	m.FocusedResultIndex = len(m.Results) - 1
	if len(m.Results) > maxResults {
		m.Results = m.Results[1:]
		m.FocusedResultIndex--
	}
}

func AddResultFromText(m *Model, text string, query string, width int, maxResults int) {
	r := &core.Result{
		Query:       query,
		Timestamp:   time.Now(),
		DisplaySize: 10,
		Expanded:    true,
		Formatted:   text,
		Viewport:    viewport.New(width, 10),
	}
	r.Viewport.SetContent(text)
	m.Results = append(m.Results, r)
	m.FocusedResultIndex = len(m.Results) - 1
	if len(m.Results) > maxResults {
		m.Results = m.Results[1:]
		m.FocusedResultIndex--
	}
}

// splitTableHeaderAndData splits a formatted table into header (first 3 lines) and data portions.
// For table formats (unicode, table), the first 3 lines are: top border, column headers, separator.
// For other formats (csv, tsv, vertical), no split is performed.
func splitTableHeaderAndData(formatted string, format formatter.Format) (header, data string) {
	// Only split for table formats that have a clear header structure
	if format != formatter.FormatUnicode && format != formatter.FormatTable {
		return "", formatted
	}

	lines := strings.Split(formatted, "\n")
	if len(lines) <= 3 {
		// Not enough lines to split, return as-is
		return "", formatted
	}

	// Header is first 3 lines (top border, headers, separator)
	header = strings.Join(lines[:3], "\n")
	// Data is the rest (data rows + bottom border)
	data = strings.Join(lines[3:], "\n")
	return header, data
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
// For table/unicode formats, columns are separated by |.
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

	// Find all column separator positions (|)
	var positions []int
	for i := 0; i < len(line); {
		r := rune(line[i])
		// Check for multi-byte | (U+2502, encoded as 3 bytes in UTF-8: E2 94 82)
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

// ensureSelectionVisible scrolls the viewport to keep the selected row visible
func ensureSelectionVisible(res *core.Result) {
	if res.SelectedRow < 0 {
		return
	}

	viewportTop := res.Viewport.YOffset
	viewportBottom := viewportTop + res.Viewport.Height - 1

	if res.SelectedRow < viewportTop {
		res.Viewport.SetYOffset(res.SelectedRow)
	} else if res.SelectedRow > viewportBottom {
		res.Viewport.SetYOffset(res.SelectedRow - res.Viewport.Height + 1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
