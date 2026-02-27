package results

import (
	"strings"

	"github.com/mvgrimes/mytui/internal/tui/core"
)

// rowMatchesQuery checks if any cell in the given row contains the query (case-insensitive).
func rowMatchesQuery(res *core.Result, rowIdx int, query string) bool {
	if res.DbResult == nil || rowIdx < 0 || rowIdx >= len(res.DbResult.Rows) {
		return false
	}
	lowerQuery := strings.ToLower(query)
	for _, cell := range core.InterfaceSliceToStrings(res.DbResult.Rows[rowIdx]) {
		if strings.Contains(strings.ToLower(cell), lowerQuery) {
			return true
		}
	}
	return false
}

// findMatchingRow searches for the next row matching query starting from startRow.
// If forward is true, searches downward (wrapping around); otherwise searches upward.
// Returns the matching row index, or -1 if no match is found.
func findMatchingRow(res *core.Result, query string, startRow int, forward bool) int {
	if res.DbResult == nil || query == "" {
		return -1
	}
	totalRows := len(res.DbResult.Rows)
	if totalRows == 0 {
		return -1
	}

	for i := 0; i < totalRows; i++ {
		var idx int
		if forward {
			idx = (startRow + i) % totalRows
		} else {
			idx = (startRow - i + totalRows) % totalRows
		}
		if rowMatchesQuery(res, idx, query) {
			return idx
		}
	}
	return -1
}
