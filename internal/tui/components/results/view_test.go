package results

import (
	"strings"
	"testing"

	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/tui/styles"
)

func TestErrorResultStyling(t *testing.T) {
	for _, width := range []int{1, 30, 100} {
		m := Model{}
		text := "Error: query failed\nPlease try again"
		AddErrorResult(&m, text, "SELECT broken", width, 10)
		r := m.Results[0]
		want := styles.ErrorStyle.Width(max(width-2, 1)).Render(text)
		if r.Formatted != want || !strings.Contains(want, "\x1b[") {
			t.Fatalf("width %d: error was not styled: %q", width, r.Formatted)
		}
		for _, focus := range []core.Focus{core.FocusQuery, core.FocusResults} {
			panel, _ := RenderPanel(&m, focus, width, 100)
			if !strings.Contains(panel, "\x1b[38;2;255;85;85m") {
				t.Fatalf("missing styled error entry: %q", panel)
			}
		}
	}
}

func TestOrdinaryTextResultIsNotErrorStyled(t *testing.T) {
	m := Model{}
	AddResultFromText(&m, "Query OK", "UPDATE users", 80, 10)
	if m.Results[0].Formatted != "Query OK" {
		t.Fatal("ordinary result text should remain unstyled")
	}
}
