package results

import "github.com/mvgrimes/mytui/internal/tui/core"

type Model struct {
	Results            []*core.Result
	FocusedResultIndex int
	ScrollOffset       int
	VimPendingKey      string
}
