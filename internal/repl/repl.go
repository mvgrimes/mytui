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
)

func RunREPL(conn *db.Connection) {
	c := completion.NewCompleter()

	// Initial schema fetch
	metadata, err := getMetadata(conn)
	if err == nil {
		c.UpdateSchema(metadata)
	}

	// Background refresh every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			metadata, err := getMetadata(conn)
			if err == nil {
				c.UpdateSchema(metadata)
			}
		}
	}()

	p := prompt.New(
		executor(conn, c),
		c.Complete,
		prompt.OptionPrefix("mysql> "),
		prompt.OptionTitle("mycli"),
		prompt.OptionHistory([]string{}),
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

func executor(conn *db.Connection, c *completion.Completer) func(string) {
	return func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}

		// Handle exit commands
		lowerLine := strings.ToLower(line)
		if lowerLine == "exit" || lowerLine == "quit" || lowerLine == "\\q" {
			fmt.Println("Goodbye!")
			os.Exit(0)
		}

		// Check for vertical output (\G)
		isVertical := false
		if strings.HasSuffix(line, "\\G") {
			isVertical = true
			line = strings.TrimSuffix(line, "\\G")
			line = strings.TrimSpace(line)
		}

		// Execute query
		result, err := conn.ExecuteQuery(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}

		// Display results
		if isVertical {
			formatter.PrintVerticalResult(result, os.Stdout)
		} else {
			formatter.PrintResult(result, os.Stdout)
		}

		// Trigger schema refresh for DDL/USE commands
		upperLine := strings.ToUpper(line)
		if strings.HasPrefix(upperLine, "USE") ||
			strings.HasPrefix(upperLine, "CREATE") ||
			strings.HasPrefix(upperLine, "DROP") ||
			strings.HasPrefix(upperLine, "ALTER") {
			go func() {
				metadata, err := getMetadata(conn)
				if err == nil {
					c.UpdateSchema(metadata)
				}
			}()
		}
	}
}
