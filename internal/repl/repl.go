package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mycli-go/internal/db"
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

		// Execute query
		result, err := conn.ExecuteQuery(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}

		// Display results
		if len(result.Headers) > 0 {
			// Print headers
			fmt.Println(strings.Join(result.Headers, "\t"))
			fmt.Println(strings.Repeat("-", len(strings.Join(result.Headers, "\t"))+8))

			// Print rows
			for _, row := range result.Rows {
				var rowStrings []string
				for _, val := range row {
					if val == nil {
						rowStrings = append(rowStrings, "NULL")
					} else if b, ok := val.([]byte); ok {
						rowStrings = append(rowStrings, string(b))
					} else {
						rowStrings = append(rowStrings, fmt.Sprintf("%v", val))
					}
				}
				fmt.Println(strings.Join(rowStrings, "\t"))
			}
		}

		fmt.Printf("%s (%.2f sec)\n", result.Status, result.Duration.Seconds())
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{}
}
