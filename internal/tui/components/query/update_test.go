package query

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/vim"
)

func TestNormalModeEnterExecutesQuery(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.Textarea.SetValue("select 1")
	vimState := vim.NewVimState()
	vimState.Mode = vim.NormalMode
	suggestionModel := suggestions.Model{Show: true}

	var executed string
	cmdRan := false
	handled, cmd := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyEnter}, UpdateDeps{
		Focus:       core.FocusQuery,
		VimState:    vimState,
		Suggestions: &suggestionModel,
		Completer:   completion.NewCompleter(),
		ExecuteQuery: func(q string) tea.Cmd {
			executed = q
			return func() tea.Msg {
				cmdRan = true
				return nil
			}
		},
	})

	if !handled {
		t.Fatal("Enter was not handled")
	}
	if executed != "select 1" {
		t.Fatalf("executed query = %q, want %q", executed, "select 1")
	}
	if suggestionModel.Show {
		t.Fatal("suggestions should close before query execution")
	}
	if cmd == nil {
		t.Fatal("expected execution command")
	}
	cmd()
	if !cmdRan {
		t.Fatal("execution command did not run")
	}
}

func TestInsertModeArrowKeysNavigateMultilineQuery(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.History = []string{"select old"}
	m.HistoryIndex = len(m.History)
	m.Textarea.SetValue("select 1\nfrom dual\nwhere true")
	m.Textarea.CursorEnd()
	vimState := vim.NewVimState()

	deps := UpdateDeps{
		Focus:       core.FocusQuery,
		VimState:    vimState,
		Suggestions: &suggestions.Model{},
	}

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyUp}, deps)
	if !handled {
		t.Fatal("Up was not handled")
	}
	if got := m.Textarea.Line(); got != 1 {
		t.Fatalf("line after Up = %d, want 1", got)
	}
	if got := m.Textarea.Value(); got != "select 1\nfrom dual\nwhere true" {
		t.Fatalf("query after Up = %q, want multiline query unchanged", got)
	}
	if got := m.HistoryIndex; got != len(m.History) {
		t.Fatalf("history index after Up = %d, want %d", got, len(m.History))
	}

	handled, _ = UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyDown}, deps)
	if !handled {
		t.Fatal("Down was not handled")
	}
	if got := m.Textarea.Line(); got != 2 {
		t.Fatalf("line after Down = %d, want 2", got)
	}
}

func TestArrowKeysNavigateHistoryAtQueryBoundaries(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.History = []string{"select 1", "select 2"}
	m.HistoryIndex = len(m.History)
	m.Textarea.SetValue("select current\nfrom dual")
	m.Textarea.CursorUp()
	vimState := vim.NewVimState()

	recalculated := 0
	deps := UpdateDeps{
		Focus:             core.FocusQuery,
		VimState:          vimState,
		Suggestions:       &suggestions.Model{},
		RecalculateHeight: func() { recalculated++ },
	}

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyUp}, deps)
	if !handled {
		t.Fatal("Up was not handled")
	}
	if got := m.Textarea.Value(); got != "select 2" {
		t.Fatalf("query after Up on first line = %q, want %q", got, "select 2")
	}
	if got := m.HistoryIndex; got != 1 {
		t.Fatalf("history index after Up on first line = %d, want 1", got)
	}

	handled, _ = UpdateKey(&m, tea.KeyPressMsg{Code: tea.KeyDown}, deps)
	if !handled {
		t.Fatal("Down was not handled")
	}
	if got := m.Textarea.Value(); got != "" {
		t.Fatalf("query after Down on last line = %q, want empty query", got)
	}
	if got := m.HistoryIndex; got != len(m.History) {
		t.Fatalf("history index after Down on last line = %d, want %d", got, len(m.History))
	}
	if recalculated != 2 {
		t.Fatalf("height recalculations = %d, want 2", recalculated)
	}
}

func TestCtrlPNavigateHistory(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.History = []string{"select 1", "select 2"}
	m.HistoryIndex = len(m.History)
	vimState := vim.NewVimState()

	deps := UpdateDeps{
		Focus:             core.FocusQuery,
		VimState:          vimState,
		Suggestions:       &suggestions.Model{},
		RecalculateHeight: func() {},
	}

	handled, _ := UpdateKey(&m, tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}, deps)
	if !handled {
		t.Fatal("Ctrl-P was not handled")
	}
	if got := m.Textarea.Value(); got != "select 2" {
		t.Fatalf("query after Ctrl-P = %q, want %q", got, "select 2")
	}

	handled, _ = UpdateKey(&m, tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}, deps)
	if !handled {
		t.Fatal("Ctrl-N was not handled")
	}
	if got := m.Textarea.Value(); got != "" {
		t.Fatalf("query after Ctrl-N = %q, want empty query", got)
	}
}

func TestInsertModeModifiedEnterInsertsNewline(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{name: "alt enter", msg: tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt}},
		{name: "ctrl j", msg: tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
			m.Textarea.SetValue("select 1")
			vimState := vim.NewVimState()
			suggestionModel := suggestions.Model{}

			handled, _ := UpdateKey(&m, tt.msg, UpdateDeps{
				Focus:             core.FocusQuery,
				VimState:          vimState,
				Suggestions:       &suggestionModel,
				RecalculateHeight: func() {},
			})

			if !handled {
				t.Fatal("modified Enter was not handled")
			}
			if got, want := m.Textarea.Value(), "select 1\n"; got != want {
				t.Fatalf("textarea value = %q, want %q", got, want)
			}
		})
	}
}
