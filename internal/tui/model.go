package tui

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/config"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
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
}

func NewModel(conn *db.Connection, cfg *config.Config) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to mycli-go!")

	hvp := viewport.New(80, 3)

	history, _ := loadHistory(cfg.HistoryFile)

	return Model{
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
