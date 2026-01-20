package completion

import (
	"strings"
)

type Suggestion struct {
	Text        string
	Description string
}

type Document struct {
	Text           string
	CursorPosition int
}

func (d Document) TextBeforeCursor() string {
	if d.CursorPosition < 0 {
		return ""
	}
	if d.CursorPosition > len(d.Text) {
		return d.Text
	}
	return d.Text[:d.CursorPosition]
}

var sqlKeywords = []string{
	"SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "WHERE", "AND", "OR", "LIMIT",
	"ORDER BY", "GROUP BY", "HAVING", "JOIN", "LEFT JOIN", "RIGHT JOIN", "INNER JOIN",
	"ON", "AS", "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX", "IN", "BETWEEN",
	"LIKE", "IS NULL", "IS NOT NULL", "UNION", "ALL", "EXISTS", "CASE", "WHEN",
	"THEN", "ELSE", "END", "CREATE", "TABLE", "DROP", "ALTER", "ADD", "COLUMN",
	"INDEX", "PRIMARY KEY", "FOREIGN KEY", "REFERENCES", "SHOW", "DATABASES",
	"TABLES", "COLUMNS", "DESCRIBE", "EXPLAIN", "USE", "SET", "GRANT", "REVOKE",
	"COMMIT", "ROLLBACK", "TRANSACTION", "VALUES", "INTO", "TRUNCATE",
}

var specialCommands = []Suggestion{
	{Text: "\\T", Description: "Change output format"},
	{Text: "\\f", Description: "Execute favorite query"},
	{Text: "\\fs", Description: "Save favorite query"},
	{Text: "\\fd", Description: "Delete favorite query"},
	{Text: "\\clip", Description: "Copy last result to clipboard"},
	{Text: "\\once", Description: "Use format for next query only"},
	{Text: "\\|", Description: "Pipe next result to shell command"},
	{Text: "\\s", Description: "Get status information from the server"},
	{Text: "\\v", Description: "Show client configuration"},
	{Text: "\\q", Description: "Quit"},
	{Text: "\\e", Description: "Open external editor"},
}

var formatTypes = []Suggestion{
	{Text: "table", Description: "Standard table format"},
	{Text: "vertical", Description: "Vertical format (like \\G)"},
	{Text: "csv", Description: "Comma-separated values"},
	{Text: "tsv", Description: "Tab-separated values"},
	{Text: "unicode", Description: "Fancy unicode table format"},
}

type Completer struct {
	keywords        []Suggestion
	tables          []Suggestion
	columns         []Suggestion
	metadata        map[string][]string // table name -> columns
	SmartCompletion bool
}

func NewCompleter() *Completer {
	suggestions := make([]Suggestion, len(sqlKeywords))
	for i, k := range sqlKeywords {
		suggestions[i] = Suggestion{Text: k}
	}

	return &Completer{
		keywords:        suggestions,
		metadata:        make(map[string][]string),
		SmartCompletion: true,
	}
}

func (c *Completer) Complete(d Document) []Suggestion {
	lineBefore := d.TextBeforeCursor()
	word := ""
	if lineBefore != "" {
		lastSpace := strings.LastIndexAny(lineBefore, " \t\n\r")
		if lastSpace == -1 {
			word = lineBefore
		} else {
			word = lineBefore[lastSpace+1:]
		}
	}

	fullText := d.Text
	upperLineBefore := strings.ToUpper(lineBefore)
	wordsBefore := strings.Fields(upperLineBefore)

	// 0. Handle special commands: \f
	if strings.HasPrefix(lineBefore, "\\") {
		parts := strings.Fields(lineBefore)
		if len(parts) == 0 {
			return specialCommands
		}
		if parts[0] == "\\T" || parts[0] == "\\t" || parts[0] == "\\once" {
			// If we are exactly at "\T " or "\once ", suggest formats
			if strings.HasSuffix(lineBefore, " ") && len(parts) == 1 {
				return formatTypes
			}
			// If we are mid-word after command, filter formats
			if len(parts) == 2 {
				return filterHasPrefix(formatTypes, parts[1], true)
			}
			return specialCommands
		}
		return filterHasPrefix(specialCommands, word, true)
	}

	var suggestions []Suggestion

	if !c.SmartCompletion {
		suggestions = append(suggestions, c.keywords...)
		return filterHasPrefix(suggestions, word, true)
	}

	// 1. Handle aliased/table qualified columns: "SELECT t.^" or "WHERE alias.^"
	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		prefix := parts[0]

		// Resolve prefix to a table name using FULL text of the document
		tableName := c.resolveTable(fullText, prefix)
		if cols, ok := c.metadata[tableName]; ok {
			var dotSuggestions []Suggestion
			for _, col := range cols {
				dotSuggestions = append(dotSuggestions, Suggestion{
					Text:        prefix + "." + col,
					Description: "column of " + tableName,
				})
			}
			// If we're at "alias.", we only want to show that table's columns
			return filterHasPrefix(dotSuggestions, word, true)
		}
	}

	if len(wordsBefore) > 0 {
		lastWord := wordsBefore[len(wordsBefore)-1]
		// 2. Prioritize tables after certain keywords
		if (word == "" || word == lastWord) && (lastWord == "FROM" || lastWord == "JOIN" || lastWord == "UPDATE" || lastWord == "INTO" || lastWord == "TABLE") {
			suggestions = append(suggestions, c.tables...)
			// If we are word-less, just return table suggestions
			if word == "" {
				return suggestions
			}
		}
	}

	// 3. Prioritize columns from tables already mentioned in the query
	mentionedTables := c.extractMentionedTables(fullText)
	if len(mentionedTables) > 0 {
		for _, tableName := range mentionedTables {
			if cols, ok := c.metadata[tableName]; ok {
				for _, col := range cols {
					suggestions = append(suggestions, Suggestion{
						Text:        col,
						Description: "column of " + tableName,
					})
				}
			}
		}
	}

	// 4. Default: keywords, tables, then all columns
	suggestions = append(suggestions, c.keywords...)
	suggestions = append(suggestions, c.tables...)
	suggestions = append(suggestions, c.columns...)

	return filterHasPrefix(suggestions, word, true)
}

func filterHasPrefix(suggestions []Suggestion, sub string, ignoreCase bool) []Suggestion {
	if sub == "" {
		return suggestions
	}
	if ignoreCase {
		sub = strings.ToUpper(sub)
	}
	res := make([]Suggestion, 0, len(suggestions))
	for _, s := range suggestions {
		text := s.Text
		if ignoreCase {
			text = strings.ToUpper(text)
		}
		if strings.HasPrefix(text, sub) {
			res = append(res, s)
		}
	}
	return res
}

// ... (remaining methods)

// resolveTable attempts to map an alias or table name to an actual table in metadata
func (c *Completer) resolveTable(fullText, prefix string) string {
	upperText := strings.ToUpper(fullText)
	upperPrefix := strings.ToUpper(prefix)

	// Check if prefix is an actual table name
	for tableName := range c.metadata {
		if strings.ToUpper(tableName) == upperPrefix {
			return tableName
		}
	}

	// Simple alias detection: "FROM TableName Alias" or "JOIN TableName AS Alias"
	// Replace common delimiters with spaces for easier splitting
	cleanText := strings.NewReplacer(",", " ", ";", " ", "\n", " ", "\t", " ").Replace(upperText)
	words := strings.Fields(cleanText)

	for i := 0; i < len(words)-1; i++ {
		// Look for table name
		for tableName := range c.metadata {
			if strings.ToUpper(tableName) == words[i] {
				// Potential match: "TableName Alias" or "TableName AS Alias"
				// We check if the next word (or the one after AS) matches our prefix
				if words[i+1] == upperPrefix {
					return tableName
				}
				if i < len(words)-2 && words[i+1] == "AS" && words[i+2] == upperPrefix {
					return tableName
				}
			}
		}
	}

	return prefix
}

// extractMentionedTables finds all tables from the metadata that appear in the query
func (c *Completer) extractMentionedTables(fullText string) []string {
	upperText := strings.ToUpper(fullText)
	cleanText := " " + strings.NewReplacer(",", " ", "\n", " ", "\t", " ", ".", " ").Replace(upperText) + " "

	var mentioned []string
	for tableName := range c.metadata {
		upperTableName := " " + strings.ToUpper(tableName) + " "
		if strings.Contains(cleanText, upperTableName) {
			mentioned = append(mentioned, tableName)
		}
	}
	return mentioned
}

func (c *Completer) UpdateSchema(metadata map[string][]string) {
	c.metadata = metadata
	c.tables = make([]Suggestion, 0, len(metadata))

	columnSet := make(map[string]struct{})
	for table, columns := range metadata {
		c.tables = append(c.tables, Suggestion{Text: table, Description: "table"})
		for _, col := range columns {
			columnSet[col] = struct{}{}
		}
	}

	c.columns = make([]Suggestion, 0, len(columnSet))
	for col := range columnSet {
		c.columns = append(c.columns, Suggestion{Text: col, Description: "column"})
	}
}
