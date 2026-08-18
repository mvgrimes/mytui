package suggestions

import (
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
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

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyDown}, UpdateDeps{})
	if !handled {
		t.Fatal("Down was not consumed by visible suggestions")
	}
	if m.Index != 0 {
		t.Fatalf("suggestion index after Down = %d, want 0", m.Index)
	}

	handled, _ = UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyUp}, UpdateDeps{})
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

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt}, UpdateDeps{
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

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyEsc}, UpdateDeps{
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

func TestSpaceAppliesSuggestionAndInsertsSpace(t *testing.T) {
	m := Model{Show: true, Index: 0, Items: []completion.Suggestion{{Text: "users"}}}
	ta := textarea.New()
	ta.Focus()
	vimState := vim.NewVimState()
	applied := false

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: ' ', Text: " "}, UpdateDeps{
		FocusQuery:      true,
		VimState:        vimState,
		Textarea:        &ta,
		ApplySuggestion: func() { applied = true },
		UpdateSuggestions: func() {
		},
	})

	if !handled {
		t.Fatal("space was not handled")
	}
	if !applied {
		t.Fatal("suggestion was not applied")
	}
	if got := ta.Value(); got != " " {
		t.Fatalf("textarea value = %q, want a space", got)
	}
}
