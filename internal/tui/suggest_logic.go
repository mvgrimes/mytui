package tui

import "strings"

// shouldOpenSuggestionsOnEdit determines if suggestions should be visible based on
// current text and cursor position.
func (m *Model) shouldOpenSuggestionsOnEdit() bool {
	if m.focus != FocusQuery {
		return false
	}
	text := m.textarea.Value()
	if strings.TrimSpace(text) == "" {
		return false
	}
	if isAtQueryBoundary(text, m.cursorPosition()) {
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
