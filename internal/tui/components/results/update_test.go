package results

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func TestUpdateKeyScrollsBottomBorderIntoView(t *testing.T) {
	tests := []struct {
		name        string
		key         tea.KeyMsg
		selectedRow int
	}{
		{name: "j", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, selectedRow: 3},
		{name: "down", key: tea.KeyMsg{Type: tea.KeyDown}, selectedRow: 3},
		{name: "page down", key: tea.KeyMsg{Type: tea.KeyPgDown}, selectedRow: 1},
		{name: "ctrl+d", key: tea.KeyMsg{Type: tea.KeyCtrlD}, selectedRow: 3},
		{name: "G", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}, selectedRow: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tableResult(tt.selectedRow)
			m := &Model{Results: []*core.Result{res}}

			handled, _ := UpdateKey(m, tt.key, UpdateDeps{
				Focus:              core.FocusResults,
				FocusedResultIndex: 0,
			})

			if !handled {
				t.Fatal("expected key to be handled")
			}
			if res.SelectedRow != 3 {
				t.Fatalf("SelectedRow = %d, want 3", res.SelectedRow)
			}
			if res.Viewport.YOffset != 2 {
				t.Fatalf("YOffset = %d, want 2 so the bottom border is visible", res.Viewport.YOffset)
			}
		})
	}
}

func TestUpdateKeyDoesNotShowBottomBorderUntilScrollingPastLastRow(t *testing.T) {
	res := tableResult(2)
	m := &Model{Results: []*core.Result{res}}
	deps := UpdateDeps{Focus: core.FocusResults, FocusedResultIndex: 0}
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	UpdateKey(m, key, deps)

	if res.SelectedRow != 3 {
		t.Fatalf("SelectedRow = %d, want 3", res.SelectedRow)
	}
	if res.Viewport.YOffset != 1 {
		t.Fatalf("YOffset = %d, want 1 before scrolling past the last row", res.Viewport.YOffset)
	}

	UpdateKey(m, key, deps)

	if res.Viewport.YOffset != 2 {
		t.Fatalf("YOffset = %d, want 2 after scrolling past the last row", res.Viewport.YOffset)
	}

	UpdateKey(m, key, deps)

	if res.Viewport.YOffset != 2 {
		t.Fatalf("YOffset = %d, want repeated scrolling to keep the border at the bottom", res.Viewport.YOffset)
	}
}

func TestUpdateKeyDoesNotAddEndScrollWithoutTableBorder(t *testing.T) {
	res := tableResult(3)
	res.FormattedHeader = ""
	m := &Model{Results: []*core.Result{res}}

	UpdateKey(m, tea.KeyMsg{Type: tea.KeyDown}, UpdateDeps{
		Focus:              core.FocusResults,
		FocusedResultIndex: 0,
	})

	if res.Viewport.YOffset != 1 {
		t.Fatalf("YOffset = %d, want 1 for a format without a table border", res.Viewport.YOffset)
	}
}

func tableResult(selectedRow int) *core.Result {
	vp := viewport.New(20, 3)
	vp.SetContent("row 1\nrow 2\nrow 3\nrow 4\nbottom border\n")
	vp.SetYOffset(1)

	return &core.Result{
		DbResult: &db.Result{Rows: [][]interface{}{
			{1},
			{2},
			{3},
			{4},
		}},
		FormattedHeader: "table header",
		Viewport:        vp,
		SelectedRow:     selectedRow,
	}
}
