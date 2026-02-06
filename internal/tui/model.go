package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/parser"
	"github.com/mvgrimes/mytui/internal/special"
	"github.com/mvgrimes/mytui/internal/vim"
	"github.com/spf13/viper"
)

type Focus int

const (
	FocusQuery Focus = iota
	FocusResults
)

type Result struct {
	Query           string
	Timestamp       time.Time
	DisplaySize     int
	Expanded        bool
	DbResult        *db.Result
	Duration        time.Duration
	Formatted       string
	FormattedHeader string // Pinned header (first 3 lines of table)
	FormattedData   string // Scrollable data (remaining lines)
	Viewport        viewport.Model
	Format          formatter.Format
	XOffset         int // Track horizontal scroll offset for pinned header
}

type MenuType int

const (
	MenuMain MenuType = iota
	MenuSaveFavorite
	MenuRunFavorite
)

type Model struct {
	textarea      textarea.Model
	results       []*Result
	focusedResult int
	conn          *db.Connection
	config        *config.Config
	completer     *completion.Completer
	vimState      *vim.VimState
	focus         Focus
	history       []string
	historyIndex  int
	vimPendingKey string
	err           error
	width         int
	height        int

	currentFormat formatter.Format
	onceFormat    formatter.Format
	pagerOverride string
	lastQuery     string
	specialOutput bytes.Buffer

	showMenu   bool
	menuIndex  int
	menuType   MenuType
	menuFilter string

	favoriteNames []string
	favoriteInput string

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
	switch m.menuType {
	case MenuSaveFavorite:
		return []MenuCommand{
			{
				Label: "Confirm Save (Prompt for name in Update)",
				Action: func(m *Model) tea.Cmd {
					return nil
				},
			},
		}
	case MenuRunFavorite:
		var cmds []MenuCommand
		m.favoriteNames = nil
		for name := range m.config.FavoriteQueries {
			m.favoriteNames = append(m.favoriteNames, name)
		}
		// Sort names for consistency
		for i := 0; i < len(m.favoriteNames); i++ {
			for j := i + 1; j < len(m.favoriteNames); j++ {
				if m.favoriteNames[i] > m.favoriteNames[j] {
					m.favoriteNames[i], m.favoriteNames[j] = m.favoriteNames[j], m.favoriteNames[i]
				}
			}
		}
		for _, name := range m.favoriteNames {
			n := name
			cmds = append(cmds, MenuCommand{
				Label: n,
				Action: func(m *Model) tea.Cmd {
					query := m.config.FavoriteQueries[n]
					query = strings.ReplaceAll(query, "\\n", "\n")
					m.textarea.SetValue(query)
					m.showMenu = false
					return nil
				},
			})
		}
		if len(cmds) == 0 {
			cmds = append(cmds, MenuCommand{
				Label: "No favorites saved",
				Action: func(m *Model) tea.Cmd {
					m.showMenu = false
					return nil
				},
			})
		}
		return cmds

	default:
		return []MenuCommand{
			{
				Label: "Status (\\s)",
				Action: func(m *Model) tea.Cmd {
					m.specialOutput.Reset()
					special.Handle("\\s", m)
					m.addResultFromText(m.specialOutput.String(), "\\s")
					m.focus = FocusResults
					m.textarea.Blur()
					return nil
				},
			},
			{
				Label: "Copy clipboard (unicode)",
				Action: func(m *Model) tea.Cmd {
					m.copyToClipboard(m.currentFormat)
					return nil
				},
			},
			{
				Label: "Copy to clipboard (ascii)",
				Action: func(m *Model) tea.Cmd {
					m.copyToClipboard(formatter.FormatTable)
					return nil
				},
			},
			{
				Label: "Copy clipboard (CSV)",
				Action: func(m *Model) tea.Cmd {
					m.copyToClipboard(formatter.FormatCSV)
					return nil
				},
			},
			{
				Label: "Save query as favorite",
				Action: func(m *Model) tea.Cmd {
					m.menuType = MenuSaveFavorite
					m.menuIndex = 0
					return nil
				},
			},
			{
				Label: "Run query as favorite",
				Action: func(m *Model) tea.Cmd {
					m.menuType = MenuRunFavorite
					m.menuIndex = 0
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
}

func (m *Model) copyToClipboard(format formatter.Format) {
	query := m.lastQuery
	if query == "" {
		query = m.textarea.Value()
	}
	if query == "" {
		return
	}

	result, err := m.conn.ExecuteQuery(query)
	if err != nil {
		return
	}

	var buf bytes.Buffer
	formatter.RenderResult(result, &buf, format, m.config)
	clipboard.WriteAll(buf.String())

	m.specialOutput.Reset()
	fmt.Fprintf(&m.specialOutput, "Result copied to clipboard as %s.\n", format)
	m.addResultFromText(m.specialOutput.String(), "Copy to Clipboard")
	m.focus = FocusResults
	m.textarea.Blur()
}

func (m *Model) SaveFavorite(name string) {
	query := m.textarea.Value()
	if query == "" {
		query = m.lastQuery
	}
	if query == "" || name == "" {
		return
	}

	if m.config.FavoriteQueries == nil {
		m.config.FavoriteQueries = make(map[string]string)
	}
	safeQuery := strings.ReplaceAll(query, "\n", "\\n")
	m.config.FavoriteQueries[name] = safeQuery

	viper.Set("favorite_queries", m.config.FavoriteQueries)
	viper.WriteConfig()

	m.specialOutput.Reset()
	fmt.Fprintf(&m.specialOutput, "Saved favorite query '%s'\n", name)
	m.addResultFromText(m.specialOutput.String(), "Save Favorite")
	m.focus = FocusResults
	m.textarea.Blur()
}

func NewModel(conn *db.Connection, cfg *config.Config) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Cursor.SetMode(cursor.CursorStatic)
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	history, _ := loadHistory(cfg.HistoryFile)

	m := Model{
		textarea:      ta,
		results:       []*Result{},
		focusedResult: -1,
		conn:          conn,
		config:        cfg,
		completer:     completion.NewCompleter(),
		vimState:      vim.NewVimState(),
		focus:         FocusQuery,
		history:       history,
		historyIndex:  len(history),
		currentFormat: formatter.Format(cfg.TableFormat),
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

func (m *Model) addResult(result *db.Result, query string, format formatter.Format) {
	fullResult := formatter.FormatResult(result, format, m.config)

	// Split into header (first 3 lines) and data for pinned header
	header, data := splitTableHeaderAndData(fullResult, format)

	r := &Result{
		Query:           query,
		Timestamp:       time.Now(),
		Duration:        result.Duration,
		DisplaySize:     10,
		Expanded:        true,
		DbResult:        result,
		Formatted:       fullResult,
		FormattedHeader: header,
		FormattedData:   data,
		Viewport:        viewport.New(m.width, 7), // Reduced height to account for pinned header
		Format:          format,
	}
	r.Viewport.SetContent(data)
	m.results = append(m.results, r)
	m.focusedResult = len(m.results) - 1
	if len(m.results) > m.config.MaxResults {
		m.results = m.results[1:]
		m.focusedResult--
	}
}

func (m *Model) addResultFromText(text string, query string) {
	r := &Result{
		Query:       query,
		Timestamp:   time.Now(),
		DisplaySize: 10,
		Expanded:    true,
		Formatted:   text,
		Viewport:    viewport.New(m.width, 10),
	}
	r.Viewport.SetContent(text)
	m.results = append(m.results, r)
	m.focusedResult = len(m.results) - 1
	if len(m.results) > m.config.MaxResults {
		m.results = m.results[1:]
		m.focusedResult--
	}
}

// splitTableHeaderAndData splits a formatted table into header (first 3 lines) and data portions.
// For table formats (unicode, table), the first 3 lines are: top border, column headers, separator.
// For other formats (csv, tsv, vertical), no split is performed.
func splitTableHeaderAndData(formatted string, format formatter.Format) (header, data string) {
	// Only split for table formats that have a clear header structure
	if format != formatter.FormatUnicode && format != formatter.FormatTable {
		return "", formatted
	}

	lines := strings.Split(formatted, "\n")
	if len(lines) <= 3 {
		// Not enough lines to split, return as-is
		return "", formatted
	}

	// Header is first 3 lines (top border, headers, separator)
	header = strings.Join(lines[:3], "\n")
	// Data is the rest (data rows + bottom border)
	data = strings.Join(lines[3:], "\n")
	return header, data
}

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

func (m *Model) fetchCache() (*completion.DBCache, error) {
	cache := completion.NewDBCache()
	cache.DefaultSchema = m.conn.GetCurrentDatabase()

	schemas, err := m.conn.GetSchemas()
	if err != nil {
		return nil, err
	}

	for _, s := range schemas {
		cache.Schemas[strings.ToUpper(s)] = s
		tables, err := m.conn.GetTablesFromSchema(s)
		if err != nil {
			continue
		}
		cache.SchemaTables[strings.ToUpper(s)] = tables

		cols, err := m.conn.DescribeDatabaseTableBySchema(s)
		if err == nil {
			for _, col := range cols {
				key := strings.ToUpper(s) + "\t" + strings.ToUpper(col.Table)
				cache.ColumnsWithParent[key] = append(cache.ColumnsWithParent[key], col)
			}
		}

		fks, err := m.conn.DescribeForeignKeysBySchema(s)
		if err == nil {
			for _, fk := range fks {
				if len(*fk) > 0 {
					table := (*fk)[0][0].Table
					refTable := (*fk)[0][1].Table

					if cache.ForeignKeys[table] == nil {
						cache.ForeignKeys[table] = make(map[string][]*db.ForeignKey)
					}
					cache.ForeignKeys[table][refTable] = append(cache.ForeignKeys[table][refTable], fk)

					if cache.ForeignKeys[refTable] == nil {
						cache.ForeignKeys[refTable] = make(map[string][]*db.ForeignKey)
					}
					cache.ForeignKeys[refTable][table] = append(cache.ForeignKeys[refTable][table], fk)
				}
			}
		}
	}

	return cache, nil
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
	// Try to set cursor after the inserted suggestion text.
	m.textarea.SetCursor(start + len(s.Text))
}

func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '`'
}

// deleteToLineStart deletes text from cursor to start of line (vim d0/c0)
func (m *Model) deleteToLineStart() {
	text := m.textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.textarea.Line()
	if currentRow >= len(lines) {
		return
	}
	lineInfo := m.textarea.LineInfo()
	col := lineInfo.ColumnOffset

	// Delete characters from start to current position
	for i := 0; i < col; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
}

// deleteInnerWord deletes the word under cursor (vim diw/ciw)
func (m *Model) deleteInnerWord() {
	text := m.textarea.Value()
	pos := m.cursorPosition()

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
	for m.cursorPosition() > start {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}

	// Delete the word
	for i := start; i < end; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	}
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// findCharInLine implements vim's f/F motion to find a character on the current line
func (m *Model) findCharInLine(target rune, forward bool) {
	text := m.textarea.Value()
	lines := strings.Split(text, "\n")
	currentRow := m.textarea.Line()
	if currentRow >= len(lines) {
		return
	}
	line := lines[currentRow]
	lineInfo := m.textarea.LineInfo()
	col := lineInfo.ColumnOffset

	if forward {
		// Search forward from current position
		for i := col + 1; i < len(line); i++ {
			if rune(line[i]) == target {
				// Move cursor right (i - col) times
				for j := col; j < i; j++ {
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
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
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
				}
				return
			}
		}
	}
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
