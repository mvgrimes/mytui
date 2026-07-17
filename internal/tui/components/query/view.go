package query

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m *Model) RenderHighlightedText(text string) string {
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
		style := getStyleForToken(tok.Type)
		b.WriteString(style.Render(tok.Value))
	}
	return b.String()
}

func getStyleForToken(t chroma.TokenType) lipgloss.Style {
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

func (m *Model) RenderArea(vimState *vim.VimState) string {
	val := m.Textarea.Value()
	isPlaceholder := false
	if val == "" {
		val = m.Textarea.Placeholder
		isPlaceholder = true
	}

	lines := strings.Split(val, "\n")
	curLineIdx := m.Textarea.Line()
	curColIdx := m.Textarea.LineInfo().ColumnOffset
	wrapWidth := m.Textarea.Width() - 5
	if wrapWidth < 1 {
		wrapWidth = 1
	}

	var b strings.Builder
	for i, line := range lines {
		segments := wrapLine(line, wrapWidth)
		for segmentIdx, segment := range segments {
			displayLine := ""
			segmentStart := segmentIdx * wrapWidth
			segmentEnd := segmentStart + len([]rune(segment))

			if isPlaceholder {
				placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
				if i == 0 && segmentIdx == 0 && m.Textarea.Focused() {
					cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
					if vimState.Mode == vim.NormalMode {
						cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
					}
					runes := []rune(segment)
					if len(runes) > 0 {
						displayLine = cursorStyle.Render(string(runes[0])) + placeholderStyle.Render(string(runes[1:]))
					} else {
						displayLine = cursorStyle.Render(" ")
					}
				} else {
					displayLine = placeholderStyle.Render(segment)
				}
			} else if i == curLineIdx && m.Textarea.Focused() && curColIdx >= segmentStart && curColIdx <= segmentEnd {
				runes := []rune(segment)
				relCol := curColIdx - segmentStart
				before := ""
				cursorChar := " "
				after := ""

				if relCol < len(runes) {
					before = string(runes[:relCol])
					cursorChar = string(runes[relCol])
					after = string(runes[relCol+1:])
				} else {
					before = segment
				}

				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
				if vimState.Mode == vim.NormalMode {
					cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
				}

				displayLine = m.RenderHighlightedText(before) + cursorStyle.Render(cursorChar) + m.RenderHighlightedText(after)
			} else {
				displayLine = m.RenderHighlightedText(segment)
			}

			linePrefix := fmt.Sprintf("%2d | ", i+1)
			if segmentIdx > 0 {
				linePrefix = "   | "
			}
			b.WriteString(linePrefix + displayLine + "\n")
		}
	}

	if m.LastError != nil && !isPlaceholder {
		padding := m.LastError.Col + 5
		b.WriteString(strings.Repeat(" ", padding) + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("^ "+m.LastError.Message) + "\n")
	}

	return b.String()
}

func wrapLine(line string, width int) []string {
	if width < 1 {
		width = 1
	}
	runes := []rune(line)
	if len(runes) == 0 {
		return []string{""}
	}

	segments := make([]string, 0, (len(runes)+width-1)/width)
	for len(runes) > width {
		segments = append(segments, string(runes[:width]))
		runes = runes[width:]
	}
	segments = append(segments, string(runes))
	return segments
}

func RenderHeader(focused bool, width int, connUser, connHost string, connPort int, database string, vimState *vim.VimState, socket string) string {
	// Determine background color based on focus
	bg := lipgloss.Color("#222222")
	defaultFg := lipgloss.Color("#AAAAAA")
	if focused {
		bg = lipgloss.Color("#FAB283")
		defaultFg = lipgloss.Color("#0A0A0A")
	}

	// Base style with background for all elements
	baseStyle := lipgloss.NewStyle().Background(bg)

	// Build QUERY label
	icon := " ≡ "
	labelStyle := baseStyle.Foreground(defaultFg)
	if focused {
		labelStyle = labelStyle.Bold(true)
	}
	label := labelStyle.Render(fmt.Sprintf("%sQUERY", icon))

	// Separator between QUERY and connection string
	sepStyle := baseStyle.Foreground(lipgloss.Color("#222222"))
	separator := sepStyle.Render("  •  ")

	// Build connection string with conditional coloring
	userStyle := baseStyle.Foreground(defaultFg)
	if connUser == "root" {
		userStyle = userStyle.Foreground(lipgloss.Color("#FF5555")).Bold(true)
	}

	hostStyle := baseStyle.Foreground(defaultFg)
	isLocal := connHost == "localhost" || connHost == "127.0.0.1" || socket != ""
	if !isLocal {
		hostStyle = hostStyle.Foreground(lipgloss.Color("#AA4400"))
	}

	atStyle := baseStyle.Foreground(lipgloss.Color("#888888"))
	restStyle := baseStyle.Foreground(defaultFg)

	connStr := lipgloss.JoinHorizontal(lipgloss.Left,
		userStyle.Render(connUser),
		atStyle.Render("@"),
		hostStyle.Render(connHost),
		restStyle.Render(fmt.Sprintf(":%d/%s", connPort, database)),
	)

	// Build mode indicator with square brackets
	var modeStr string
	modeStyle := baseStyle.Bold(true)
	if vimState.Mode == vim.NormalMode {
		modeStyle = modeStyle.Foreground(lipgloss.Color("#DCFEAF"))
		modeStr = modeStyle.Render("[NORMAL]")
	} else {
		modeStyle = modeStyle.Foreground(lipgloss.Color("#222222"))
		modeStr = modeStyle.Render("[INSERT]")
	}

	// Calculate filler width
	leftPart := label + separator + connStr
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(modeStr)
	fillerWidth := width - leftWidth - rightWidth - 1 // -1 for trailing space
	if fillerWidth < 1 {
		fillerWidth = 1
	}

	fillerStyle := lipgloss.NewStyle().Background(bg)
	filler := fillerStyle.Render(strings.Repeat(" ", fillerWidth))

	// Combine all parts
	headerStyle := lipgloss.NewStyle().Background(bg).Width(width)
	return headerStyle.Render(leftPart + filler + modeStr)
}

func UpdateCursorStyle(m *Model, focus core.Focus, vimState *vim.VimState) {
	if focus != core.FocusQuery || vimState.Mode == vim.NormalMode {
		m.Textarea.Cursor.Style = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC"))
	} else {
		// Try to make it look like a bar using a left border
		m.Textarea.Cursor.Style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AAFF")).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#00AAFF"))
	}
}
