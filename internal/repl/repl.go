package repl

import (
	"bufio"
	"fmt"
	"io"
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
	conn              *db.Connection
	completer         *completion.Completer
	vimState          *vim.VimState
	currentFormat     formatter.Format
	onceFormat        formatter.Format
	pagerOverride     string
	lastQuery         string
	config            *config.Config
	history           []string
	historyTimestamps []string
}

func NewREPL(conn *db.Connection, cfg *config.Config) *REPL {
	history, timestamps := loadHistory(cfg.HistoryFile)
	r := &REPL{
		conn:              conn,
		completer:         completion.NewCompleter(),
		vimState:          vim.NewVimState(),
		currentFormat:     formatter.Format(cfg.TableFormat),
		config:            cfg,
		history:           history,
		historyTimestamps: timestamps,
	}
	r.completer.SmartCompletion = cfg.SmartCompletion
	return r
}

func RunREPL(conn *db.Connection, cfg *config.Config) {
	r := NewREPL(conn, cfg)

	// Initial schema fetch
	cache, err := r.fetchCache()
	if err == nil {
		r.completer.UpdateCache(cache)
	}

	// Background refresh every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			cache, err := r.fetchCache()
			if err == nil {
				r.completer.UpdateCache(cache)
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
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlR,
			Fn: func(buf *prompt.Buffer) {
				r.handleHistorySearch(buf)
			},
		}),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlN,
			Fn: func(buf *prompt.Buffer) {
				// History mapping
			},
		}),
	}

	if r.config.KeyBindings == "vim" {
		options = append(options,
			prompt.OptionSwitchKeyBindMode(prompt.CommonKeyBind),
			prompt.OptionAddKeyBind(r.getVimKeyBindings()...),
			prompt.OptionAddASCIICodeBind(r.getVimASCIICodeBindings()...),
		)
	}

	p := prompt.New(
		r.executor,
		r.completerBridge,
		options...,
	)
	p.Run()
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
		} else {
			if len(currentEntry) > 0 {
				history = append(history, strings.Join(currentEntry, "\n"))
				timestamps = append(timestamps, currentTimestamp)
				currentEntry = nil
			}
		}
	}
	if len(currentEntry) > 0 {
		history = append(history, strings.Join(currentEntry, "\n"))
		timestamps = append(timestamps, currentTimestamp)
	}
	return history, timestamps
}

func (r *REPL) saveToHistory(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	// Update in-memory history
	r.history = append(r.history, line)
	r.historyTimestamps = append(r.historyTimestamps, now)

	f, err := os.OpenFile(r.config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("# %s\n", now)); err != nil {
		return
	}
	lines := strings.Split(line, "\n")
	for _, l := range lines {
		if _, err := f.WriteString(fmt.Sprintf("+%s\n", l)); err != nil {
			return
		}
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

	// Strip ANSI codes to avoid width calculation bugs in go-prompt
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(p, "")
}

func (r *REPL) fetchCache() (*completion.DBCache, error) {
	cache := completion.NewDBCache()
	cache.DefaultSchema = r.conn.GetCurrentDatabase()

	schemas, err := r.conn.GetSchemas()
	if err != nil {
		return nil, err
	}

	for _, s := range schemas {
		cache.Schemas[strings.ToUpper(s)] = s
		tables, err := r.conn.GetTablesFromSchema(s)
		if err != nil {
			continue
		}
		cache.SchemaTables[strings.ToUpper(s)] = tables

		cols, err := r.conn.DescribeDatabaseTableBySchema(s)
		if err == nil {
			for _, col := range cols {
				key := strings.ToUpper(s) + "\t" + strings.ToUpper(col.Table)
				cache.ColumnsWithParent[key] = append(cache.ColumnsWithParent[key], col)
			}
		}

		fks, err := r.conn.DescribeForeignKeysBySchema(s)
		if err == nil {
			for _, fk := range fks {
				// sqls structure is map[table][referencedTable][]*ForeignKey
				if len(*fk) > 0 {
					table := (*fk)[0][0].Table
					refTable := (*fk)[0][1].Table

					if cache.ForeignKeys[table] == nil {
						cache.ForeignKeys[table] = make(map[string][]*db.ForeignKey)
					}
					cache.ForeignKeys[table][refTable] = append(cache.ForeignKeys[table][refTable], fk)

					// Also add reverse mapping for joining either way
					if cache.ForeignKeys[refTable] == nil {
						cache.ForeignKeys[refTable] = make(map[string][]*db.ForeignKey)
					}
					cache.ForeignKeys[refTable][table] = append(cache.ForeignKeys[refTable][table], fk)
				}
			}
		}
	}

	return cache, nil
}

func (r *REPL) GetConn() *db.Connection             { return r.conn }
func (r *REPL) GetConfig() *config.Config           { return r.config }
func (r *REPL) GetCurrentFormat() formatter.Format  { return r.currentFormat }
func (r *REPL) SetCurrentFormat(f formatter.Format) { r.currentFormat = f }
func (r *REPL) GetLastQuery() string                { return r.lastQuery }
func (r *REPL) SetLastQuery(q string)               { r.lastQuery = q }
func (r *REPL) SetOnceFormat(f formatter.Format)    { r.onceFormat = f }
func (r *REPL) SetPagerOverride(p string)           { r.pagerOverride = p }
func (r *REPL) GetWriter() io.Writer                { return os.Stdout }

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

func (r *REPL) handleHistorySearch(buf *prompt.Buffer) {
	// Simple fzf integration for history search
	tempFile, err := os.CreateTemp("", "mycli-history")
	if err != nil {
		return
	}
	defer os.Remove(tempFile.Name())

	// Use null bytes to separate entries for multiline support in fzf
	for i := len(r.history) - 1; i >= 0; i-- {
		ts := "Unknown Date"
		if i < len(r.historyTimestamps) {
			ts = r.historyTimestamps[i]
		}
		// Format: [Date] Query
		entry := fmt.Sprintf("[%s] %s", ts, r.history[i])
		tempFile.Write([]byte(entry))
		tempFile.Write([]byte{0})
	}
	tempFile.Close()

	// --read0 to read null-terminated items, --print0 to print selected item null-terminated
	fzfCmd := "fzf --read0 --print0 --scheme=history --tiebreak=index --bind ctrl-r:up,alt-r:up --preview-window=down:wrap --preview=\"printf '%s' {}\""
	cmd := exec.Command("sh", "-c", fzfCmd+" < "+tempFile.Name())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return
	}

	selected := strings.TrimRight(string(out), "\x00")
	if selected != "" {
		// Strip the [Date] prefix
		// Format is [YYYY-MM-DD HH:MM:SS] (21 chars) + " " (1 char) = 22 chars
		// But let's be more robust and find the first "] "
		if idx := strings.Index(selected, "] "); idx != -1 {
			selected = selected[idx+2:]
		}

		buf.DeleteBeforeCursor(len([]rune(buf.Document().TextBeforeCursor())))
		buf.Delete(len([]rune(buf.Document().TextAfterCursor())))
		buf.InsertText(selected, false, true)
	}
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
			cache, err := r.fetchCache()
			if err == nil {
				r.completer.UpdateCache(cache)
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
