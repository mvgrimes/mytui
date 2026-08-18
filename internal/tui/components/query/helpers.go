package query

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) CursorPosition() int {
	text := m.Textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.Textarea.Line()
	info := m.Textarea.LineInfo()

	pos := 0
	for i := 0; i < currentRow && i < len(lines); i++ {
		pos += len(lines[i]) + 1 // +1 for \n
	}

	// ColumnOffset is the offset in the soft-wrapped line.
	// This is still tricky if there's wrapping.
	// For now, let's assume it's approximately correct or we use ColumnOffset.
	pos += info.ColumnOffset
	return pos
}

func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '`'
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// deleteToLineStart deletes text from cursor to start of line (vim d0/c0)
func (m *Model) deleteToLineStart() {
	text := m.Textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.Textarea.Line()
	if currentRow >= len(lines) {
		return
	}
	lineInfo := m.Textarea.LineInfo()
	col := lineInfo.ColumnOffset

	// Delete characters from start to current position
	for i := 0; i < col; i++ {
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
}

// deleteInnerWord deletes the word under cursor (vim diw/ciw)
func (m *Model) deleteInnerWord() {
	text := m.Textarea.Value()
	pos := m.CursorPosition()

	if pos >= len(text) {
		return
	}

	// Find word boundaries
	start := pos
	end := pos

	// Move start backward to find word beginning
	for start > 0 && isWordChar(rune(text[start-1])) {
		start--
	}

	// Move end forward to find word end
	for end < len(text) && isWordChar(rune(text[end])) {
		end++
	}

	if start == end {
		return
	}

	// Move cursor to start of word
	for m.CursorPosition() > start {
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	}

	// Delete the word
	for i := start; i < end; i++ {
		m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	}
}

// insertSQLTemplate inserts an SQL template at the current cursor position
func (m *Model) insertSQLTemplate(template string, cursorOffset int) {
	text := m.Textarea.Value()
	pos := m.CursorPosition()
	newText := text[:pos] + template + text[pos:]
	m.Textarea.SetValue(newText)
	m.Textarea.SetCursorColumn(pos + cursorOffset)
}

// jumpToFieldsPosition moves cursor to the fields position in SELECT or INSERT
func (m *Model) jumpToFieldsPosition() {
	text := m.Textarea.Value()
	upperText := strings.ToUpper(text)

	// For SELECT: position after "SELECT " before "FROM"
	if idx := strings.Index(upperText, "SELECT "); idx != -1 {
		fromIdx := strings.Index(upperText[idx:], " FROM")
		if fromIdx != -1 {
			m.Textarea.SetCursorColumn(idx + fromIdx)
		} else {
			m.Textarea.SetCursorColumn(idx + 7) // After "SELECT "
		}
		return
	}

	// For INSERT: position inside first ()
	if idx := strings.Index(upperText, "INSERT INTO "); idx != -1 {
		if parenIdx := strings.Index(text[idx:], "("); parenIdx != -1 {
			m.Textarea.SetCursorColumn(idx + parenIdx + 1)
		}
	}
}

// jumpToTablePosition moves cursor to the table name position
func (m *Model) jumpToTablePosition() {
	text := m.Textarea.Value()
	upperText := strings.ToUpper(text)

	// Keywords followed by table name, check in order
	patterns := []struct {
		keyword string
		offset  int
	}{
		{"INSERT INTO ", 12},
		{"DELETE FROM ", 12},
		{"UPDATE ", 7},
		{"CREATE TABLE ", 13},
		{"FROM ", 5},
		{"INTO ", 5},
		{"TABLE ", 6},
	}

	for _, p := range patterns {
		if idx := strings.Index(upperText, p.keyword); idx != -1 {
			m.Textarea.SetCursorColumn(idx + p.offset)
			return
		}
	}
}

// jumpToWherePosition moves cursor to WHERE clause, inserting it if missing
func (m *Model) jumpToWherePosition() {
	text := m.Textarea.Value()
	upperText := strings.ToUpper(text)

	// If WHERE exists, position after it
	if idx := strings.Index(upperText, "WHERE "); idx != -1 {
		m.Textarea.SetCursorColumn(idx + 6)
		return
	}

	// Find insertion point (before ORDER BY, GROUP BY, LIMIT, or end)
	insertPos := len(text)
	for _, kw := range []string{" ORDER BY", " GROUP BY", " LIMIT", ";"} {
		if idx := strings.Index(upperText, kw); idx != -1 && idx < insertPos {
			insertPos = idx
		}
	}

	// Insert " WHERE " and position cursor
	newText := text[:insertPos] + " WHERE " + text[insertPos:]
	m.Textarea.SetValue(newText)
	m.Textarea.SetCursorColumn(insertPos + 7)
}

// findCharInLine implements vim's f/F motion to find a character on the current line
func (m *Model) findCharInLine(target rune, forward bool) {
	text := m.Textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.Textarea.Line()
	if currentRow >= len(lines) {
		return
	}
	line := lines[currentRow]
	lineInfo := m.Textarea.LineInfo()
	col := lineInfo.ColumnOffset

	if forward {
		// Search forward from current position
		for i := col + 1; i < len(line); i++ {
			if rune(line[i]) == target {
				// Move cursor right (i - col) times
				for j := col; j < i; j++ {
					m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyRight})
				}
				return
			}
		}
	} else {
		// Search backward from current position
		for i := col - 1; i >= 0; i-- {
			if rune(line[i]) == target {
				// Move cursor left (col - i) times
				for j := col; j > i; j-- {
					m.Textarea, _ = m.Textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
				}
				return
			}
		}
	}
}
