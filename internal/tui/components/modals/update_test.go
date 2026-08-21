package modals

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

func TestUpdateRowDetailPageDownAndPageUp(t *testing.T) {
	m := RowDetailModel{
		Show:     true,
		Viewport: viewport.New(viewport.WithWidth(20), viewport.WithHeight(4)),
	}

	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}
	m.Viewport.SetContent(strings.Join(lines, "\n"))

	handled, _ := UpdateRowDetail(&m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	if !handled {
		t.Fatal("PageDown was not handled")
	}
	if m.Viewport.YOffset() != 4 {
		t.Fatalf("YOffset after PageDown = %d, want 4", m.Viewport.YOffset())
	}

	handled, _ = UpdateRowDetail(&m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	if !handled {
		t.Fatal("PageUp was not handled")
	}
	if m.Viewport.YOffset() != 0 {
		t.Fatalf("YOffset after PageUp = %d, want 0", m.Viewport.YOffset())
	}
}

func TestUpdateRowDetailLineNavigation(t *testing.T) {
	tests := []struct {
		name string
		down tea.KeyPressMsg
		up   tea.KeyPressMsg
	}{
		{
			name: "vim keys",
			down: tea.KeyPressMsg{Code: 'j', Text: "j"},
			up:   tea.KeyPressMsg{Code: 'k', Text: "k"},
		},
		{
			name: "arrow keys",
			down: tea.KeyPressMsg{Code: tea.KeyDown},
			up:   tea.KeyPressMsg{Code: tea.KeyUp},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := RowDetailModel{
				Show:     true,
				Viewport: viewport.New(viewport.WithWidth(20), viewport.WithHeight(4)),
			}
			m.Viewport.SetContent(strings.Repeat("line\n", 10))

			handled, _ := UpdateRowDetail(&m, tt.down)
			if !handled {
				t.Fatal("down key was not handled")
			}
			if got := m.Viewport.YOffset(); got != 1 {
				t.Fatalf("YOffset after down = %d, want 1", got)
			}

			handled, _ = UpdateRowDetail(&m, tt.up)
			if !handled {
				t.Fatal("up key was not handled")
			}
			if got := m.Viewport.YOffset(); got != 0 {
				t.Fatalf("YOffset after up = %d, want 0", got)
			}
		})
	}
}
