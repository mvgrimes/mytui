package modals

import (
	"sort"
	"strings"

	fzfalgo "github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// filteredHistoryIndices returns indices into history that match the current
// filter using fuzzy matching (fzf algorithm).
// With no filter: newest-first order. With a filter: sorted by score descending
// (best match at index 0, which appears at the bottom of the display).
func filteredHistoryIndices(history []string, filter string) []int {
	if filter == "" {
		out := make([]int, len(history))
		for i := range out {
			out[i] = len(history) - 1 - i
		}
		return out
	}

	// fzf requires the pattern to be lowercased when caseSensitive=false.
	pattern := []rune(strings.ToLower(filter))
	slab := util.MakeSlab(100*1024, 2048)

	type scored struct {
		idx   int
		score int
	}
	var matches []scored

	for i := len(history) - 1; i >= 0; i-- {
		chars := util.ToChars([]byte(history[i]))
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

// OpenHistorySearch opens the history search modal positioned at the most
// recent entry.
func OpenHistorySearch(m *HistorySearchModel) {
	m.Filter = ""
	m.Scroll = 0
	m.Index = 0 // 0 = most recent entry in filtered list
	m.Show = true
}

// HistorySearchClampScroll adjusts Scroll so that the selected row is visible.
func HistorySearchClampScroll(m *HistorySearchModel, history []string, listHeight int) {
	indices := filteredHistoryIndices(history, m.Filter)
	if len(indices) == 0 {
		m.Scroll = 0
		return
	}
	if m.Index < 0 {
		m.Index = 0
	}
	if m.Index >= len(indices) {
		m.Index = len(indices) - 1
	}
	if m.Index < m.Scroll {
		m.Scroll = m.Index
	}
	if m.Index >= m.Scroll+listHeight {
		m.Scroll = m.Index - listHeight + 1
	}
	if m.Scroll < 0 {
		m.Scroll = 0
	}
}
