package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/config"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
	"github.com/mvgrimes/mycli-go/internal/parser"
	"github.com/mvgrimes/mycli-go/internal/special"
	"github.com/mvgrimes/mycli-go/internal/vim"
)

type Focus int

const (
	FocusQuery Focus = iota
	FocusResults
)

type Model struct {
	textarea       textarea.Model
	viewport       viewport.Model
	headerViewport viewport.Model
	conn           *db.Connection
	config         *config.Config
	completer      *completion.Completer
	vimState       *vim.VimState
	focus          Focus
	history        []string
	historyIndex   int
	vimPendingKey  string
	err            error
	width          int
	height         int

	currentFormat formatter.Format
	onceFormat    formatter.Format
	pagerOverride string
	lastQuery     string
	specialOutput bytes.Buffer

	showMenu  bool
	menuIndex int

	showSuggestions bool
	suggestions     []completion.Suggestion
	suggestionIndex int

	lastError *parser.ParseError
}

type MenuCommand struct {
	Label  string
	Action func(*Model) tea.Cmd
}

func (m *Model) GetCommands() []MenuCommand {
	return []MenuCommand{
		{
			Label: "Status (\\s)",
			Action: func(m *Model) tea.Cmd {
				m.specialOutput.Reset()
				special.Handle("\\s", m)
				m.headerViewport.SetContent("")
				m.headerViewport.Height = 0
				m.viewport.SetContent(m.specialOutput.String())
				m.viewport.Height = m.height - 6 - m.headerViewport.Height
				m.focus = FocusResults
				m.textarea.Blur()
				return nil
			},
		},
		{
			Label: "Copy last output to clipboard",
			Action: func(m *Model) tea.Cmd {
				m.specialOutput.Reset()
				special.Handle("\\clip", m)
				m.viewport.SetContent(m.specialOutput.String())
				m.viewport.Height = m.height - 6 - m.headerViewport.Height
				m.focus = FocusResults
				m.textarea.Blur()
				return nil
			},
		},
		{
			Label: "Copy output as CSV",
			Action: func(m *Model) tea.Cmd {
				if m.lastQuery == "" {
					return nil
				}
				result, err := m.conn.ExecuteQuery(m.lastQuery)
				if err != nil {
					return nil
				}
				var buf bytes.Buffer
				// Use RenderResult with FormatCSV
				formatter.RenderResult(result, &buf, formatter.FormatCSV, m.config)
				clipboard.WriteAll(buf.String())
				m.specialOutput.Reset()
				fmt.Fprintln(&m.specialOutput, "Last result copied to clipboard as CSV.")
				m.viewport.SetContent(m.specialOutput.String())
				m.viewport.Height = m.height - 6 - m.headerViewport.Height
				m.focus = FocusResults
				m.textarea.Blur()
				return nil
			},
		},
		{
			Label: "Exit",
			Action: func(m *Model) tea.Cmd {
				return tea.Quit
			},
		},
	}
}

func NewModel(conn *db.Connection, cfg *config.Config) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Cursor.SetMode(cursor.CursorStatic)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to mycli-go!")

	hvp := viewport.New(80, 3)

	history, _ := loadHistory(cfg.HistoryFile)

	m := Model{
		textarea:       ta,
		viewport:       vp,
		headerViewport: hvp,
		conn:           conn,
		config:         cfg,
		completer:      completion.NewCompleter(),
		vimState:       vim.NewVimState(),
		focus:          FocusQuery,
		history:        history,
		historyIndex:   len(history),
		currentFormat:  formatter.Format(cfg.TableFormat),
	}
	m.UpdateCursorStyle()
	return m
}

func (m *Model) GetConn() *db.Connection             { return m.conn }
func (m *Model) GetConfig() *config.Config           { return m.config }
func (m *Model) GetCurrentFormat() formatter.Format  { return m.currentFormat }
func (m *Model) SetCurrentFormat(f formatter.Format) { m.currentFormat = f }
func (m *Model) GetLastQuery() string                { return m.lastQuery }
func (m *Model) SetLastQuery(q string)               { m.lastQuery = q }
func (m *Model) SetOnceFormat(f formatter.Format)    { m.onceFormat = f }
func (m *Model) SetPagerOverride(p string)           { m.pagerOverride = p }
func (m *Model) GetWriter() io.Writer                { return &m.specialOutput }

func (m *Model) ExecuteQueryWithFormat(query string, format formatter.Format) {
	m.lastQuery = query
	result, err := m.conn.ExecuteQuery(query)
	if err != nil {
		fmt.Fprintf(&m.specialOutput, "Error: %v\n", err)
		return
	}
	// For TUI, we capture the formatted result in specialOutput
	// We use RenderResult instead of PrintResult to avoid pager in TUI
	formatter.RenderResult(result, &m.specialOutput, format, m.config)
	m.pagerOverride = ""
}

func (m *Model) cursorPosition() int {
	text := m.textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.textarea.Line()
	info := m.textarea.LineInfo()

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

func (m *Model) updateSuggestions() {
	doc := completion.Document{
		Text:           m.textarea.Value(),
		CursorPosition: m.cursorPosition(),
	}
	m.suggestions = m.completer.Complete(doc)
	if len(m.suggestions) == 0 {
		m.showSuggestions = false
	}
}

func (m *Model) applySuggestion() {
	if m.suggestionIndex < 0 || m.suggestionIndex >= len(m.suggestions) {
		return
	}
	s := m.suggestions[m.suggestionIndex]
	text := m.textarea.Value()
	pos := m.cursorPosition()

	// Find the word being completed
	start := pos
	for start > 0 && isIdentifierChar(rune(text[start-1])) {
		start--
	}

	newText := text[:start] + s.Text + text[pos:]
	m.textarea.SetValue(newText)
	// We need to set the cursor back. This is also tricky with SetCursor.
	// m.textarea.SetCursor(start + len(s.Text))
}

func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '`'
}

func (m *Model) UpdateCursorStyle() {
	if m.focus != FocusQuery || m.vimState.Mode == vim.NormalMode {
		m.textarea.Cursor.Style = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC"))
	} else {
		// Try to make it look like a bar using a left border
		m.textarea.Cursor.Style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AAFF")).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#00AAFF"))
	}
}

func loadHistory(filename string) ([]string, []string) {
	var history []string
	var timestamps []string
	file, err := os.Open(filename)
	if err != nil {
		return history, timestamps
	}
	defer file.Close()

	var currentEntry []string
	var currentTimestamp string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			if len(currentEntry) > 0 {
				history = append(history, strings.Join(currentEntry, "\n"))
				timestamps = append(timestamps, currentTimestamp)
				currentEntry = nil
			}
			currentTimestamp = strings.TrimSpace(line[1:])
		} else if strings.HasPrefix(line, "+") {
			currentEntry = append(currentEntry, line[1:])
		}
	}
	if len(currentEntry) > 0 {
		history = append(history, strings.Join(currentEntry, "\n"))
		timestamps = append(timestamps, currentTimestamp)
	}
	return history, timestamps
}
