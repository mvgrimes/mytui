package results

import "strings"

// ComputeAvailableHeight returns the number of lines available for the results list.
func ComputeAvailableHeight(queryValue string, placeholder string, height int) int {
	queryAreaHeight := len(strings.Split(queryValue, "\n"))
	if queryAreaHeight < 3 {
		queryAreaHeight = 3
	}
	if queryValue == "" {
		queryAreaHeight = len(strings.Split(placeholder, "\n"))
		if queryAreaHeight < 3 {
			queryAreaHeight = 3
		}
	}
	// overhead: 1 (qHeader) + queryAreaHeight + 1 (helpText) + 1 (statusLine)
	overhead := 3 + queryAreaHeight
	available := height - overhead
	if available < 0 {
		return 0
	}
	return available
}

// TotalResultsHeight returns the total number of rendered lines across all results.
func TotalResultsHeight(m *Model) int {
	total := 0
	for _, r := range m.Results {
		total++ // result header line
		if r.Expanded {
			if r.FormattedHeader != "" {
				total += strings.Count(r.FormattedHeader, "\n") + 1
			}
			total += r.Viewport.Height()
		}
	}
	return total
}

// ScrollToBottom adjusts ScrollOffset so the newest result is visible.
func ScrollToBottom(m *Model, available int) {
	total := TotalResultsHeight(m)
	if total > available {
		m.ScrollOffset = total - available
	} else {
		m.ScrollOffset = 0
	}
}

// ClampScrollOffset ensures ScrollOffset stays within valid bounds.
func ClampScrollOffset(m *Model, available int) {
	total := TotalResultsHeight(m)
	maxOffset := total - available
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.ScrollOffset > maxOffset {
		m.ScrollOffset = maxOffset
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
}

// EnsureFocusedResultVisible adjusts ScrollOffset so the focused result is visible.
func EnsureFocusedResultVisible(m *Model, available int) {
	if m.FocusedResultIndex < 0 || m.FocusedResultIndex >= len(m.Results) {
		return
	}

	// Compute the top y-position of the focused result in the full results rendering.
	top := 0
	for i, r := range m.Results {
		if i == m.FocusedResultIndex {
			break
		}
		top++ // result header line
		if r.Expanded {
			if r.FormattedHeader != "" {
				top += strings.Count(r.FormattedHeader, "\n") + 1
			}
			top += r.Viewport.Height()
		}
	}

	r := m.Results[m.FocusedResultIndex]
	height := 1 // result header line
	if r.Expanded {
		if r.FormattedHeader != "" {
			height += strings.Count(r.FormattedHeader, "\n") + 1
		}
		height += r.Viewport.Height()
	}
	bottom := top + height

	// Scroll up if result is above the visible area.
	if top < m.ScrollOffset {
		m.ScrollOffset = top
	}
	// Scroll down if result extends below the visible area.
	if bottom > m.ScrollOffset+available {
		m.ScrollOffset = bottom - available
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
}
