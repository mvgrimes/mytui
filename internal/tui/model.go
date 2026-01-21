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
	"github.com/spf13/viper"
)

type Focus int

const (
	FocusQuery Focus = iota
	FocusResults
)

type MenuType int

const (
	MenuMain MenuType = iota
	MenuSaveFavorite
	MenuRunFavorite
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
	menuType  MenuType

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
					m.headerViewport.SetContent("")
					m.headerViewport.Height = 0
					m.viewport.SetContent(m.specialOutput.String())
					m.recalculateHeight()
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
	m.headerViewport.SetContent("")
	m.headerViewport.Height = 0
	m.viewport.SetContent(m.specialOutput.String())
	m.recalculateHeight()
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
	m.headerViewport.SetContent("")
	m.headerViewport.Height = 0
	m.viewport.SetContent(m.specialOutput.String())
	m.recalculateHeight()
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
