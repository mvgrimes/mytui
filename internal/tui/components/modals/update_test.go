package modals

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateRowDetailPageDownAndPageUp(t *testing.T) {
	m := RowDetailModel{
		Show:     true,
		Viewport: viewport.New(20, 4),
	}

	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}
	m.Viewport.SetContent(strings.Join(lines, "\n"))

	handled, _ := UpdateRowDetail(&m, tea.KeyMsg{Type: tea.KeyPgDown})
	if !handled {
		t.Fatal("PageDown was not handled")
	}
	if m.Viewport.YOffset != 4 {
		t.Fatalf("YOffset after PageDown = %d, want 4", m.Viewport.YOffset)
	}

	handled, _ = UpdateRowDetail(&m, tea.KeyMsg{Type: tea.KeyPgUp})
	if !handled {
		t.Fatal("PageUp was not handled")
	}
	if m.Viewport.YOffset != 0 {
		t.Fatalf("YOffset after PageUp = %d, want 0", m.Viewport.YOffset)
	}
}
