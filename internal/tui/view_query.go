package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m Model) renderHighlightedText(text string) string {
	lexer := lexers.Get("sql")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}

	var b strings.Builder
	for _, tok := range iterator.Tokens() {
		style := m.getStyleForToken(tok.Type)
		b.WriteString(style.Render(tok.Value))
	}
	return b.String()
}

func (m Model) getStyleForToken(t chroma.TokenType) lipgloss.Style {
	s := lipgloss.NewStyle()
	switch t {
	case chroma.Keyword, chroma.KeywordReserved, chroma.KeywordType:
		return s.Foreground(lipgloss.Color("#00AAFF")).Bold(true)
	case chroma.String, chroma.StringSingle, chroma.StringDouble:
		return s.Foreground(lipgloss.Color("#00FF88"))
	case chroma.Number, chroma.NumberInteger, chroma.NumberFloat:
		return s.Foreground(lipgloss.Color("#FF8800"))
	case chroma.Comment, chroma.CommentSingle, chroma.CommentMultiline:
		return s.Foreground(lipgloss.Color("#666666")).Italic(true)
	case chroma.NameLabel, chroma.NameVariable:
		return s.Foreground(lipgloss.Color("#CC88FF"))
	case chroma.Operator, chroma.Punctuation:
		return s.Foreground(lipgloss.Color("#AAAAAA"))
	default:
		return s.Foreground(lipgloss.Color("#FFFFFF"))
	}
}

func (m Model) renderQueryArea() string {
	val := m.textarea.Value()
	isPlaceholder := false
	if val == "" {
		val = m.textarea.Placeholder
		isPlaceholder = true
	}

	lines := strings.Split(val, "\n")
	curLineIdx := m.textarea.Line()
	curColIdx := m.textarea.LineInfo().ColumnOffset

	var b strings.Builder
	for i, line := range lines {
		displayLine := ""
		if isPlaceholder {
			placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
			if i == 0 && m.textarea.Focused() {
				// Show cursor at start of placeholder
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
				// Normal mode uses a dimmer cursor
				if m.vimState.Mode == vim.NormalMode {
					cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
				}
				runes := []rune(line)
				if len(runes) > 0 {
					displayLine = cursorStyle.Render(string(runes[0])) + placeholderStyle.Render(string(runes[1:]))
				} else {
					displayLine = cursorStyle.Render(" ")
				}
			} else {
				displayLine = placeholderStyle.Render(line)
			}
		} else if i == curLineIdx && m.textarea.Focused() {
			runes := []rune(line)
			before := ""
			cursorChar := " "
			after := ""

			if curColIdx < len(runes) {
				before = string(runes[:curColIdx])
				cursorChar = string(runes[curColIdx])
				after = string(runes[curColIdx+1:])
			} else {
				before = line
			}

			hBefore := m.renderHighlightedText(before)
			hAfter := m.renderHighlightedText(after)

			cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
			if m.vimState.Mode == vim.NormalMode {
				cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
			}

			displayLine = hBefore + cursorStyle.Render(cursorChar) + hAfter
		} else {
			displayLine = m.renderHighlightedText(line)
		}

		b.WriteString(fmt.Sprintf("%2d | %s\n", i+1, displayLine))
	}

	if m.lastError != nil && !isPlaceholder {
		padding := m.lastError.Col + 5
		b.WriteString(strings.Repeat(" ", padding) + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("^ "+m.lastError.Message) + "\n")
	}

	return b.String()
}

func (m Model) renderQueryHeader(focused bool) string {
	icon := " 🔍 "
	header := fmt.Sprintf("%s QUERY", icon)

	style := headerStyle
	if focused {
		style = headerFocusStyle
	}

	return style.Width(m.width).Render(header)
}
