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
