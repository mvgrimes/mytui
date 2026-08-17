package suggestions

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/vim"
)

func TestArrowKeysNavigateVisibleSuggestions(t *testing.T) {
	m := Model{
		Show: true,
		Items: []completion.Suggestion{
			{Text: "SELECT"},
			{Text: "SET"},
		},
		Index: -1,
	}

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyDown}, UpdateDeps{})
	if !handled {
		t.Fatal("Down was not consumed by visible suggestions")
	}
	if m.Index != 0 {
		t.Fatalf("suggestion index after Down = %d, want 0", m.Index)
	}

	handled, _ = UpdateKey(&m, tea.KeyMsg{Type: tea.KeyUp}, UpdateDeps{})
	if !handled {
		t.Fatal("Up was not consumed by visible suggestions")
	}
	if m.Index != 1 {
		t.Fatalf("suggestion index after Up = %d, want 1", m.Index)
	}
}

func TestAltEnterIsNotConsumedByVisibleSuggestions(t *testing.T) {
	m := Model{Show: true}
	ta := textarea.New()

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, UpdateDeps{
		FocusQuery: true,
		VimState:   vim.NewVimState(),
		Textarea:   &ta,
	})
	if handled {
		t.Fatal("Alt-Enter should be passed to the Query Editor")
	}
}

func TestEscapeClosesSuggestionsWithoutLeavingInsertMode(t *testing.T) {
	m := Model{Show: true, Index: 1}
	vimState := vim.NewVimState()
	vimState.Mode = vim.InsertMode

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyEsc}, UpdateDeps{
		FocusQuery: true,
		VimState:   vimState,
	})
	if !handled {
		t.Fatal("Escape was not consumed by visible suggestions")
	}
	if m.Show {
		t.Fatal("suggestions remained visible after Escape")
	}
	if m.Index != -1 {
		t.Fatalf("suggestion index after Escape = %d, want -1", m.Index)
	}
	if vimState.Mode != vim.InsertMode {
		t.Fatalf("Vim mode after Escape = %v, want insert mode", vimState.Mode)
	}
}
