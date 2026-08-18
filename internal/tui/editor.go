package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type queryEditorFinishedMsg struct {
	content string
	err     error
}

var (
	createEditorTempFile = os.CreateTemp
	readEditorFile       = os.ReadFile
	removeEditorFile     = os.Remove
	editorExecProcess    = tea.ExecProcess
	editorCommand        = func(editor, path string) *exec.Cmd {
		return exec.Command("sh", "-c", editor+" \"$1\"", "mytui-editor", path)
	}
)

func editorProgram() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	return "vi"
}

func (m *Model) openQueryInEditor() tea.Cmd {
	queryText := m.query.Textarea.Value()
	if queryText == "" {
		queryText = m.lastQuery
	}

	tmpFile, err := createEditorTempFile("", "mytui-query-*.sql")
	if err != nil {
		return func() tea.Msg { return queryEditorFinishedMsg{err: fmt.Errorf("creating editor temp file: %w", err)} }
	}

	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(queryText); err != nil {
		tmpFile.Close()
		removeEditorFile(tmpPath)
		return func() tea.Msg { return queryEditorFinishedMsg{err: fmt.Errorf("writing editor temp file: %w", err)} }
	}
	if err := tmpFile.Close(); err != nil {
		removeEditorFile(tmpPath)
		return func() tea.Msg { return queryEditorFinishedMsg{err: fmt.Errorf("closing editor temp file: %w", err)} }
	}

	cmd := editorCommand(editorProgram(), tmpPath)
	return editorExecProcess(cmd, func(err error) tea.Msg {
		defer removeEditorFile(tmpPath)
		if err != nil {
			return queryEditorFinishedMsg{err: fmt.Errorf("running editor: %w", err)}
		}

		content, err := readEditorFile(tmpPath)
		if err != nil {
			return queryEditorFinishedMsg{err: fmt.Errorf("reading editor temp file: %w", err)}
		}

		return queryEditorFinishedMsg{content: strings.TrimSpace(string(content))}
	})
}
