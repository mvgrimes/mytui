package repl

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/dbcli/mycli-go/internal/db"
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
		if line == "exit" || line == "quit" {
			fmt.Println("Goodbye!")
			os.Exit(0)
		}
		fmt.Printf("Executing: %s\n", line)
		// TODO: Actually execute the query
	}
}

func completer(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{}
}