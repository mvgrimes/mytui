package repl

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
	"github.com/mvgrimes/mycli-go/internal/vim"
)

type REPL struct {
	conn          *db.Connection
	completer     *completion.Completer
	vimState      *vim.VimState
	currentFormat formatter.Format
}

func NewREPL(conn *db.Connection) *REPL {
	return &REPL{
		conn:          conn,
		completer:     completion.NewCompleter(),
		vimState:      vim.NewVimState(),
		currentFormat: formatter.FormatTable,
	}
}

func RunREPL(conn *db.Connection) {
	r := NewREPL(conn)

	// Initial schema fetch
	metadata, err := getMetadata(conn)
	if err == nil {
		r.completer.UpdateSchema(metadata)
	}

	// Background refresh every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			metadata, err := getMetadata(conn)
			if err == nil {
				r.completer.UpdateSchema(metadata)
			}
		}
	}()

	p := prompt.New(
		r.executor,
		r.completer.Complete,
		prompt.OptionTitle("mycli"),
		prompt.OptionHistory([]string{}),
		prompt.OptionSwitchKeyBindMode(prompt.CommonKeyBind),
		prompt.OptionAddKeyBind(r.vimState.GetKeyBindings()...),
		prompt.OptionAddASCIICodeBind(r.vimState.GetASCIICodeBindings()...),
		prompt.OptionLivePrefix(r.vimState.GetLivePrefix()),
	)
	p.Run()
}

func getMetadata(conn *db.Connection) (map[string][]string, error) {
	tables, err := conn.GetTables()
	if err != nil {
		return nil, err
	}

	metadata := make(map[string][]string)
	for _, table := range tables {
		columns, err := conn.GetColumns(table)
		if err == nil {
			metadata[table] = columns
		}
	}
	return metadata, nil
}

func (r *REPL) executor(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Handle special commands
	if strings.HasPrefix(line, "\\") {
		r.handleSpecialCommand(line)
		return
	}

	// Handle exit commands
	lowerLine := strings.ToLower(line)
	if lowerLine == "exit" || lowerLine == "quit" {
		fmt.Println("Goodbye!")
		os.Exit(0)
	}

	// Check for vertical output (\G)
	format := r.currentFormat
	if strings.HasSuffix(line, "\\G") {
		format = formatter.FormatVertical
		line = strings.TrimSuffix(line, "\\G")
		line = strings.TrimSpace(line)
	}

	// Execute query
	result, err := r.conn.ExecuteQuery(line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	// Display results
	formatter.PrintResult(result, os.Stdout, format)

	// Trigger schema refresh for DDL/USE commands
	upperLine := strings.ToUpper(line)
	if strings.HasPrefix(upperLine, "USE") ||
		strings.HasPrefix(upperLine, "CREATE") ||
		strings.HasPrefix(upperLine, "DROP") ||
		strings.HasPrefix(upperLine, "ALTER") {
		go func() {
			metadata, err := getMetadata(r.conn)
			if err == nil {
				r.completer.UpdateSchema(metadata)
			}
		}()
	}
}

func (r *REPL) handleSpecialCommand(line string) {
	parts := strings.Fields(line)
	cmd := parts[0]

	switch cmd {
	case "\\q":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "\\f":
		if len(parts) < 2 {
			fmt.Printf("Current format: %s\n", r.currentFormat)
			fmt.Println("Usage: \\f [table|vertical|csv|tsv|unicode]")
			return
		}
		newFormat := formatter.Format(parts[1])
		switch newFormat {
		case formatter.FormatTable, formatter.FormatVertical, formatter.FormatCSV, formatter.FormatTSV, formatter.FormatUnicode:
			r.currentFormat = newFormat
			fmt.Printf("Format changed to: %s\n", r.currentFormat)
		default:
			fmt.Printf("Unknown format: %s\n", newFormat)
		}
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}
