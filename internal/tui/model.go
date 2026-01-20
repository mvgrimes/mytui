package tui

import (
	"bufio"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/config"
	"github.com/mvgrimes/mycli-go/internal/db"
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
