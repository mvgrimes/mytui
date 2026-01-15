package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
)

func RunREPL(conn *db.Connection) {
	p := prompt.New(
		executor(conn),
		completer,
		prompt.OptionPrefix("mysql> "),
		prompt.OptionTitle("mycli"),
		prompt.OptionHistory([]string{}),
	)
	p.Run()
}

func executor(conn *db.Connection) func(string) {
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
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{}
}
