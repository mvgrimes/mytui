package query

import (
	"bufio"
	"os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/parser"
)

type Model struct {
	Textarea          textarea.Model
	History           []string
	HistoryTimestamps []string
	HistoryIndex      int
	VimPendingKey     string
	LastError         *parser.ParseError
	LastFindChar      rune
	LastFindForward   bool
}

func NewModel(cfg *config.Config) Model {
	ta := textarea.New()
	ta.SetVirtualCursor(false)
	ta.Placeholder = "Enter SQL query..."
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(3)

	history, timestamps := loadHistory(cfg.HistoryFile)

	return Model{
		Textarea:          ta,
		History:           history,
		HistoryTimestamps: timestamps,
		HistoryIndex:      len(history),
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
