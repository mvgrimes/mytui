package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/special"
	"github.com/mvgrimes/mytui/internal/tui/components/menu"
	"github.com/mvgrimes/mytui/internal/tui/components/modals"
	"github.com/mvgrimes/mytui/internal/tui/components/query"
	"github.com/mvgrimes/mytui/internal/tui/components/results"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/vim"
	"github.com/spf13/viper"
)

type Model struct {
	query       query.Model
	results     results.Model
	suggestions suggestions.Model
	menu        menu.Model
	modals      modals.Model

	conn      *db.Connection
	config    *config.Config
	completer *completion.Completer
	vimState  *vim.VimState
	focus     core.Focus
	err       error
	width     int
	height    int

	currentFormat formatter.Format
	onceFormat    formatter.Format
	pagerOverride string
	lastQuery     string
	specialOutput bytes.Buffer
}

func NewModel(conn *db.Connection, cfg *config.Config) Model {
	m := Model{
		query:         query.NewModel(cfg),
		results:       results.Model{FocusedResultIndex: -1},
		conn:          conn,
		config:        cfg,
		completer:     completion.NewCompleter(),
		vimState:      vim.NewVimState(),
		focus:         core.FocusQuery,
		currentFormat: formatter.Format(cfg.TableFormat),
	}
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

func (m *Model) copyToClipboard(format formatter.Format) {
	if m.results.FocusedResultIndex < 0 || m.results.FocusedResultIndex >= len(m.results.Results) {
		return
	}

	focused := m.results.Results[m.results.FocusedResultIndex]

	var buf bytes.Buffer
	if focused.DbResult != nil {
		formatter.RenderResult(focused.DbResult, &buf, format, m.config)
	} else {
		buf.WriteString(focused.Formatted)
	}

	if err := clipboard.WriteAll(ansi.Strip(buf.String())); err != nil {
		return
	}

	m.specialOutput.Reset()
	fmt.Fprintf(&m.specialOutput, "Result copied to clipboard as %s.\n", format)
	m.addResultFromText(m.specialOutput.String(), "Copy to Clipboard")
	m.focus = core.FocusResults
	m.query.Textarea.Blur()
}

func (m *Model) SaveFavorite(name string) {
	queryText := m.query.Textarea.Value()
	if queryText == "" {
		queryText = m.lastQuery
	}
	if queryText == "" || name == "" {
		return
	}

	if m.config.FavoriteQueries == nil {
		m.config.FavoriteQueries = make(map[string]string)
	}
	safeQuery := strings.ReplaceAll(queryText, "\n", "\\n")
	m.config.FavoriteQueries[name] = safeQuery

	viper.Set("favorite_queries", m.config.FavoriteQueries)
	viper.WriteConfig()

	m.specialOutput.Reset()
	fmt.Fprintf(&m.specialOutput, "Saved favorite query '%s'\n", name)
	m.addResultFromText(m.specialOutput.String(), "Save Favorite")
	m.focus = core.FocusResults
	m.query.Textarea.Blur()
}

func (m *Model) ExecuteQueryWithFormat(queryText string, format formatter.Format) {
	m.lastQuery = queryText
	result, err := m.conn.ExecuteQuery(queryText)
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

func (m *Model) addResult(result *db.Result, queryText string, format formatter.Format) {
	results.AddResult(&m.results, result, queryText, format, m.width, m.config.MaxResults, m.config)
	available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
	results.ScrollToBottom(&m.results, available)
}

func (m *Model) addResultFromText(text string, queryText string) {
	results.AddResultFromText(&m.results, text, queryText, m.width, m.config.MaxResults)
	available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
	results.ScrollToBottom(&m.results, available)
}

func (m *Model) executeQuery(queryText string) tea.Cmd {
	trimmedQuery := strings.TrimSpace(queryText)
	lowerQuery := strings.ToLower(trimmedQuery)
	if lowerQuery == "exit" || lowerQuery == "quit" {
		return tea.Quit
	}
	if trimmedQuery == "\\e" {
		return m.openQueryInEditor()
	}

	m.specialOutput.Reset()
	if special.Handle(trimmedQuery, m) {
		m.addResultFromText(m.specialOutput.String(), trimmedQuery)
		m.recalculateHeight()
		m.query.Textarea.Reset()
		m.saveToHistory(queryText)
		return nil
	}

	format := m.currentFormat
	if m.onceFormat != "" {
		format = m.onceFormat
		m.onceFormat = ""
	}

	if strings.HasSuffix(trimmedQuery, "\\G") {
		format = formatter.FormatVertical
		queryText = strings.TrimSuffix(trimmedQuery, "\\G")
	}

	m.saveToHistory(queryText)
	m.lastQuery = queryText

	result, err := m.conn.ExecuteQuery(queryText)
	if err != nil {
		wrappedError := lipgloss.NewStyle().Width(m.width - 2).Render(fmt.Sprintf("Error: %v", err))
		m.addResultFromText(wrappedError, queryText)
		m.recalculateHeight()
		return nil
	}

	m.addResult(result, queryText, format)
	m.recalculateHeight()
	m.query.Textarea.Reset()
	if len(result.Headers) > 0 {
		m.focus = core.FocusResults
		m.query.Textarea.Blur()
	}

	upperQuery := strings.ToUpper(trimmedQuery)
	if strings.HasPrefix(upperQuery, "USE") ||
		strings.HasPrefix(upperQuery, "CREATE") ||
		strings.HasPrefix(upperQuery, "DROP") ||
		strings.HasPrefix(upperQuery, "ALTER") {
		return m.refreshCacheCmd()
	}

	return nil
}
