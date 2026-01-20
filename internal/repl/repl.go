package repl

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/config"
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/mvgrimes/mycli-go/internal/formatter"
	"github.com/mvgrimes/mycli-go/internal/special"
	"github.com/mvgrimes/mycli-go/internal/vim"
)

type REPL struct {
	conn          *db.Connection
	completer     *completion.Completer
	vimState      *vim.VimState
	currentFormat formatter.Format
	onceFormat    formatter.Format
	pagerOverride string
	lastQuery     string
	config        *config.Config
	history       []string
}

func NewREPL(conn *db.Connection, cfg *config.Config) *REPL {
	r := &REPL{
		conn:          conn,
		completer:     completion.NewCompleter(),
		vimState:      vim.NewVimState(),
		currentFormat: formatter.Format(cfg.TableFormat),
		config:        cfg,
		history:       loadHistory(cfg.HistoryFile),
	}
	r.completer.SmartCompletion = cfg.SmartCompletion
	return r
}

func RunREPL(conn *db.Connection, cfg *config.Config) {
	r := NewREPL(conn, cfg)

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

	livePrefix := func() (string, bool) {
		base := r.formatPrompt()
		if r.config.KeyBindings == "vim" {
			if r.vimState.Mode == vim.NormalMode {
				return "(normal) " + base, true
			}
		}
		return base, true
	}

	options := []prompt.Option{
		prompt.OptionTitle("mycli-go"),
		prompt.OptionHistory(r.history),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionPrefixTextColor(prompt.Cyan),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlW,
			Fn: func(buf *prompt.Buffer) {
				buf.DeleteBeforeCursor(len([]rune(buf.Document().TextBeforeCursor())) - buf.Document().FindStartOfPreviousWordWithSpace())
			},
		}),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlU,
			Fn: func(buf *prompt.Buffer) {
				buf.DeleteBeforeCursor(len([]rune(buf.Document().TextBeforeCursor())))
			},
		}),
	}

	if r.config.KeyBindings == "vim" {
		options = append(options,
			prompt.OptionSwitchKeyBindMode(prompt.CommonKeyBind),
			prompt.OptionAddKeyBind(r.vimState.GetKeyBindings()...),
			prompt.OptionAddASCIICodeBind(r.vimState.GetASCIICodeBindings()...),
		)
	}

	p := prompt.New(
		r.executor,
		r.completer.Complete,
		options...,
	)
	p.Run()
}

func loadHistory(filename string) []string {
	var history []string
	file, err := os.Open(filename)
	if err != nil {
		return history
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			history = append(history, line)
		}
	}
	return history
}

func (r *REPL) saveToHistory(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	f, err := os.OpenFile(r.config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		return
	}
}

func (r *REPL) formatPrompt() string {
	p := r.config.Prompt
	now := time.Now()

	// Replace tokens
	p = strings.ReplaceAll(p, "\\D", now.Format("Mon Jan 2 15:04:05 2006"))
	p = strings.ReplaceAll(p, "\\d", r.conn.GetCurrentDatabase())
	p = strings.ReplaceAll(p, "\\h", r.conn.Config.Host)
	p = strings.ReplaceAll(p, "\\m", now.Format("04"))
	p = strings.ReplaceAll(p, "\\n", "\n")
	p = strings.ReplaceAll(p, "\\P", now.Format("PM"))
	p = strings.ReplaceAll(p, "\\p", fmt.Sprintf("%d", r.conn.Config.Port))
	p = strings.ReplaceAll(p, "\\R", now.Format("15"))
	p = strings.ReplaceAll(p, "\\r", now.Format("03"))
	p = strings.ReplaceAll(p, "\\s", now.Format("05"))
	p = strings.ReplaceAll(p, "\\u", r.conn.Config.User)

	// Handle product type (\t) - for now just MySQL
	p = strings.ReplaceAll(p, "\\t", "mysql")

	// Handle ANSI escape sequences
	p = strings.ReplaceAll(p, "\\x1b", "\x1b")
	p = strings.ReplaceAll(p, "\\033", "\x1b")
	p = strings.ReplaceAll(p, "\\e", "\x1b")

	return p
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

func (r *REPL) GetConn() *db.Connection             { return r.conn }
func (r *REPL) GetConfig() *config.Config           { return r.config }
func (r *REPL) GetCurrentFormat() formatter.Format  { return r.currentFormat }
func (r *REPL) SetCurrentFormat(f formatter.Format) { r.currentFormat = f }
func (r *REPL) GetLastQuery() string                { return r.lastQuery }
func (r *REPL) SetLastQuery(q string)               { r.lastQuery = q }
func (r *REPL) SetOnceFormat(f formatter.Format)    { r.onceFormat = f }
func (r *REPL) SetPagerOverride(p string)           { r.pagerOverride = p }

func (r *REPL) ExecuteQueryWithFormat(query string, format formatter.Format) {
	r.lastQuery = query
	result, err := r.conn.ExecuteQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	formatter.PrintResult(result, os.Stdout, format, r.config, r.pagerOverride)
	r.pagerOverride = "" // Reset after use
}

func (r *REPL) executor(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Save to history file
	r.saveToHistory(line)

	// Handle special commands
	if special.Handle(line, r) {
		return
	}

	// Handle exit commands
	lowerLine := strings.ToLower(line)
	if lowerLine == "exit" || lowerLine == "quit" {
		fmt.Println("Goodbye!")
		os.Exit(0)
	}

	// Check for onceFormat
	format := r.currentFormat
	if r.onceFormat != "" {
		format = r.onceFormat
		r.onceFormat = ""
	}

	// Check for vertical output (\G)
	if strings.HasSuffix(line, "\\G") {
		format = formatter.FormatVertical
		line = strings.TrimSuffix(line, "\\G")
		line = strings.TrimSpace(line)
	}

	// Execute query
	r.ExecuteQueryWithFormat(line, format)

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
	case "\\e":
		r.openExternalEditor()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}

func (r *REPL) openExternalEditor() {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi" // Default to vi
	}

	tempFile, err := os.CreateTemp("", "mycli-go-*.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		return
	}
	defer os.Remove(tempFile.Name())

	if r.lastQuery != "" {
		if _, err := tempFile.WriteString(r.lastQuery); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to temp file: %v\n", err)
		}
	}
	tempFile.Close()

	cmd := exec.Command("sh", "-c", editor+" "+tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
		return
	}

	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading temp file: %v\n", err)
		return
	}

	query := strings.TrimSpace(string(content))
	if query == "" {
		return
	}

	fmt.Printf("Executing: %s\n", query)
	result, err := r.conn.ExecuteQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	formatter.PrintResult(result, os.Stdout, r.currentFormat, r.config, r.pagerOverride)
}
