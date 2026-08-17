package query

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	handled, cmd := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyEnter}, UpdateDeps{
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

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyUp}, deps)
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

	handled, _ = UpdateKey(&m, tea.KeyMsg{Type: tea.KeyDown}, deps)
	if !handled {
		t.Fatal("Down was not handled")
	}
	if got := m.Textarea.Line(); got != 2 {
		t.Fatalf("line after Down = %d, want 2", got)
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

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyCtrlP}, deps)
	if !handled {
		t.Fatal("Ctrl-P was not handled")
	}
	if got := m.Textarea.Value(); got != "select 2" {
		t.Fatalf("query after Ctrl-P = %q, want %q", got, "select 2")
	}

	handled, _ = UpdateKey(&m, tea.KeyMsg{Type: tea.KeyCtrlN}, deps)
	if !handled {
		t.Fatal("Ctrl-N was not handled")
	}
	if got := m.Textarea.Value(); got != "" {
		t.Fatalf("query after Ctrl-N = %q, want empty query", got)
	}
}

func TestInsertModeAltEnterInsertsNewline(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.Textarea.SetValue("select 1")
	m.Textarea.CursorEnd()
	vimState := vim.NewVimState()

	handled, _ := UpdateKey(&m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, UpdateDeps{
		Focus:             core.FocusQuery,
		VimState:          vimState,
		Suggestions:       &suggestions.Model{Show: true},
		RecalculateHeight: func() {},
	})
	if !handled {
		t.Fatal("Alt-Enter was not handled")
	}
	if got := m.Textarea.Value(); got != "select 1\n" {
		t.Fatalf("query after Alt-Enter = %q, want %q", got, "select 1\n")
	}
}
