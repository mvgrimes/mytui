package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func TestCompositeOverlay(t *testing.T) {
	tests := []struct {
		name string
		fg   string
		bg   string
		x    int
		y    int
		want string
	}{
		{
			name: "positions foreground",
			fg:   "XY\nZZ",
			bg:   "abcdef\nghijkl\nmnopqr",
			x:    2,
			y:    1,
			want: "abcdef\nghXYkl\nmnZZqr",
		},
		{
			name: "clamps to bounds",
			fg:   "XX",
			bg:   "abc\ndef",
			x:    99,
			y:    99,
			want: "abc\ndXX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ansi.Strip(compositeOverlay(tt.fg, tt.bg, tt.x, tt.y)); got != tt.want {
				t.Fatalf("compositeOverlay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRecalculateHeightKeepsExpandedResultBottomVisible(t *testing.T) {
	m := NewModel(nil, &config.Config{
		HistoryFile: t.TempDir() + "/history",
		TableFormat: "table",
	})
	m.height = 12
	m.focus = core.FocusResults
	m.results.FocusedResultIndex = 1

	content := strings.Repeat("row\n", 9) + "row"
	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(4))
	vp.SetContent(content)
	m.results.Results = []*core.Result{
		{Expanded: false},
		{
			Expanded:      true,
			DisplaySize:   6,
			FormattedData: content,
			Viewport:      vp,
		},
	}

	m.recalculateHeight()

	if got := m.results.Results[1].Viewport.Height(); got != 6 {
		t.Fatalf("expanded viewport height = %d, want 6", got)
	}
	if got := m.results.ScrollOffset; got != 2 {
		t.Fatalf("results panel scroll offset = %d, want 2", got)
	}
}
