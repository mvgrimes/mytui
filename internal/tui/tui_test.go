package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
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
