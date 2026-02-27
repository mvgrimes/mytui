package query

import (
	"strings"

	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

func (m *Model) UpdateSuggestions(completer *completion.Completer, sugg *suggestions.Model) {
	doc := completion.Document{
		Text:           m.Textarea.Value(),
		CursorPosition: m.CursorPosition(),
	}
	sugg.Items = completer.Complete(doc)
	if len(sugg.Items) == 0 {
		sugg.Show = false
	}
}

func (m *Model) ApplySuggestion(sugg *suggestions.Model) {
	if sugg.Index < 0 || sugg.Index >= len(sugg.Items) {
		return
	}
	s := sugg.Items[sugg.Index]
	text := m.Textarea.Value()
	pos := m.CursorPosition()

	// Find the word being completed
	start := pos
	for start > 0 && isIdentifierChar(rune(text[start-1])) {
		start--
	}

	newText := text[:start] + s.Text + text[pos:]
	m.Textarea.SetValue(newText)
	// Try to set cursor after the inserted suggestion text.
	m.Textarea.SetCursor(start + len(s.Text))
}

// ShouldOpenSuggestionsOnEdit determines if suggestions should be visible based on
// current text and cursor position.
func (m *Model) ShouldOpenSuggestionsOnEdit(focus core.Focus) bool {
	if focus != core.FocusQuery {
		return false
	}
	text := m.Textarea.Value()
	if strings.TrimSpace(text) == "" {
		return false
	}
	if isAtQueryBoundary(text, m.CursorPosition()) {
		return false
	}
	return true
}

// isAtQueryBoundary returns true if the caret is at or beyond a trailing query
// terminator (";" or "\\G"/"\\g").
func isAtQueryBoundary(text string, cursor int) bool {
	// Find last non-space character index
	i := len(text) - 1
	for i >= 0 {
		if text[i] != ' ' && text[i] != '\t' && text[i] != '\n' && text[i] != '\r' {
			break
		}
		i--
	}
	if i < 0 {
		return false
	}

	// Check for semicolon
	if text[i] == ';' {
		return cursor > i
	}

	// Check for \G or \g
	if i >= 1 && text[i-1] == '\\' && (text[i] == 'G' || text[i] == 'g') {
		return cursor > i
	}

	return false
}
