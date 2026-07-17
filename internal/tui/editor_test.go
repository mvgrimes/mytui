package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/config"
)

func TestEditorProgramPrefersEditor(t *testing.T) {
	t.Setenv("EDITOR", "ed")
	t.Setenv("VISUAL", "vi")

	if got := editorProgram(); got != "ed" {
		t.Fatalf("editorProgram() = %q, want %q", got, "ed")
	}
}

func TestEditorProgramFallsBackToVisual(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "nano")

	if got := editorProgram(); got != "nano" {
		t.Fatalf("editorProgram() = %q, want %q", got, "nano")
	}
}

func TestOpenQueryInEditorSeedsCurrentQuery(t *testing.T) {
	t.Setenv("EDITOR", "true")
	tmpDir := t.TempDir()
	var editorPath string

	origCreate := createEditorTempFile
	origCommand := editorCommand
	origRemove := removeEditorFile
	origExec := editorExecProcess
	t.Cleanup(func() {
		createEditorTempFile = origCreate
		editorCommand = origCommand
		removeEditorFile = origRemove
		editorExecProcess = origExec
	})

	createEditorTempFile = func(dir, pattern string) (*os.File, error) {
		return os.CreateTemp(tmpDir, pattern)
	}
	editorCommand = func(editor, path string) *exec.Cmd {
		editorPath = path
		return exec.Command("sh", "-c", "printf '\nwhere id = 1' >> \"$1\"", "test-editor", path)
	}
	editorExecProcess = func(cmd *exec.Cmd, fn tea.ExecCallback) tea.Cmd {
		return func() tea.Msg { return fn(cmd.Run()) }
	}
	removeEditorFile = func(path string) error { return nil }

	m := NewModel(nil, &config.Config{TableFormat: "table", HistoryFile: filepath.Join(tmpDir, "history")})
	m.query.Textarea.SetValue("select * from users")

	cmd := m.openQueryInEditor()
	msg := cmd().(queryEditorFinishedMsg)
	if msg.err != nil {
		t.Fatalf("openQueryInEditor() error = %v", msg.err)
	}
	if msg.content != "select * from users\nwhere id = 1" {
		t.Fatalf("edited content = %q", msg.content)
	}
	seeded, err := os.ReadFile(editorPath)
	if err != nil {
		t.Fatalf("read seeded file: %v", err)
	}
	if string(seeded) != "select * from users\nwhere id = 1" {
		t.Fatalf("seeded file = %q", string(seeded))
	}
}
