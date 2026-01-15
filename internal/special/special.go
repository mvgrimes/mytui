package special

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/mvgrimes/mycli-go/internal/config"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
	"github.com/spf13/viper"
)

type REPL interface {
	GetConn() *db.Connection
	GetConfig() *config.Config
	GetCurrentFormat() formatter.Format
	SetCurrentFormat(formatter.Format)
	GetLastQuery() string
	SetLastQuery(string)
	ExecuteQueryWithFormat(query string, format formatter.Format)
	SetOnceFormat(format formatter.Format)
	SetPagerOverride(command string)
}

func Handle(line string, r REPL) bool {
	if !strings.HasPrefix(line, "\\") && !strings.HasPrefix(line, "|") {
		return false
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	cmd := parts[0]

	switch cmd {
	case "\\q":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "\\T":
		handleFormat(parts, r)
		return true
	case "\\f":
		handleFavorite(parts, r)
		return true
	case "\\fs":
		handleFavoriteSave(line, parts, r)
		return true
	case "\\fd":
		handleFavoriteDelete(parts, r)
		return true
	case "\\clip":
		handleClip(r)
		return true
	case "\\once":
		handleOnce(line, parts, r)
		return true
	case "\\|":
		handlePipe(line, parts, r)
		return true
	case "\\e":
		// \e is handled specifically in REPL because it needs terminal control
		return false
	}

	// Handle the case where | is used without backslash if that's a thing,
	// but the plan says \|
	if cmd == "\\|" {
		handlePipe(line, parts, r)
		return true
	}

	return false
}

func handleFormat(parts []string, r REPL) {
	if len(parts) < 2 {
		fmt.Printf("Current format: %s\n", r.GetCurrentFormat())
		fmt.Println("Usage: \\T [table|vertical|csv|tsv|unicode]")
		return
	}
	newFormat := formatter.Format(parts[1])
	switch newFormat {
	case formatter.FormatTable, formatter.FormatVertical, formatter.FormatCSV, formatter.FormatTSV, formatter.FormatUnicode:
		r.SetCurrentFormat(newFormat)
		fmt.Printf("Format changed to: %s\n", r.GetCurrentFormat())
	default:
		fmt.Printf("Unknown format: %s\n", newFormat)
	}
}

func handleFavoriteSave(line string, parts []string, r REPL) {
	if len(parts) < 3 {
		fmt.Println("Usage: \\fs name query")
		return
	}
	name := parts[1]
	// Join the rest of the line as the query, but we need to find where the name ends
	// A simple way is to find the first occurrence of name after \fs
	idx := strings.Index(line, name)
	query := strings.TrimSpace(line[idx+len(name):])

	cfg := r.GetConfig()
	if cfg.FavoriteQueries == nil {
		cfg.FavoriteQueries = make(map[string]string)
	}
	cfg.FavoriteQueries[name] = query

	viper.Set("favorite_queries", cfg.FavoriteQueries)
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error saving favorite query: %v\n", err)
	} else {
		fmt.Printf("Saved favorite query '%s'\n", name)
	}
}

func handleFavorite(parts []string, r REPL) {
	if len(parts) < 2 {
		fmt.Println("Usage: \\f name [args...]")
		cfg := r.GetConfig()
		if len(cfg.FavoriteQueries) > 0 {
			fmt.Println("\nAvailable favorites:")
			for name, query := range cfg.FavoriteQueries {
				fmt.Printf("  %s: %s\n", name, query)
			}
		}
		return
	}
	name := parts[1]
	cfg := r.GetConfig()
	query, ok := cfg.FavoriteQueries[name]
	if !ok {
		fmt.Printf("Favorite query '%s' not found\n", name)
		return
	}

	// Handle parameters $1, $2, etc.
	for i := 2; i < len(parts); i++ {
		placeholder := fmt.Sprintf("$%d", i-1)
		query = strings.ReplaceAll(query, placeholder, parts[i])
	}

	fmt.Printf("Executing favorite query '%s': %s\n", name, query)
	r.ExecuteQueryWithFormat(query, r.GetCurrentFormat())
}

func handleFavoriteDelete(parts []string, r REPL) {
	if len(parts) < 2 {
		fmt.Println("Usage: \\fd name")
		return
	}
	name := parts[1]
	cfg := r.GetConfig()
	if _, ok := cfg.FavoriteQueries[name]; !ok {
		fmt.Printf("Favorite query '%s' not found\n", name)
		return
	}

	delete(cfg.FavoriteQueries, name)
	viper.Set("favorite_queries", cfg.FavoriteQueries)
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error deleting favorite query: %v\n", err)
	} else {
		fmt.Printf("Deleted favorite query '%s'\n", name)
	}
}

func handleClip(r REPL) {
	lastQuery := r.GetLastQuery()
	if lastQuery == "" {
		fmt.Println("No previous query to copy.")
		return
	}

	result, err := r.GetConn().ExecuteQuery(lastQuery)
	if err != nil {
		fmt.Printf("Error re-executing query for clipboard: %v\n", err)
		return
	}

	var buf strings.Builder
	formatter.PrintResult(result, &buf, r.GetCurrentFormat(), r.GetConfig(), "")

	err = clipboard.WriteAll(buf.String())
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	} else {
		fmt.Println("Last result copied to clipboard.")
	}
}

func handleOnce(line string, parts []string, r REPL) {
	if len(parts) < 2 {
		fmt.Println("Usage: \\once format [query]")
		return
	}
	format := formatter.Format(parts[1])

	// Check if query is provided on the same line
	query := ""
	if len(parts) > 2 {
		idx := strings.Index(line, parts[1])
		query = strings.TrimSpace(line[idx+len(parts[1]):])
	}

	if query != "" {
		r.ExecuteQueryWithFormat(query, format)
	} else {
		r.SetOnceFormat(format)
		fmt.Printf("Next query will use format: %s\n", format)
	}
}

func handlePipe(line string, parts []string, r REPL) {
	if len(parts) < 2 {
		fmt.Println("Usage: \\| command")
		return
	}

	// Extract command after \|
	idx := strings.Index(line, "\\|")
	if idx == -1 {
		idx = strings.Index(line, "|") // Handle case without backslash if needed, but plan says \|
	}
	command := strings.TrimSpace(line[idx+2:])

	r.SetPagerOverride(command)
	fmt.Printf("Next query result will be piped to: %s\n", command)
}
