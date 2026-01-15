package completion

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

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

var specialCommands = []prompt.Suggest{
	{Text: "\\f", Description: "Change output format"},
	{Text: "\\q", Description: "Quit"},
}

var formatTypes = []prompt.Suggest{
	{Text: "table", Description: "Standard table format"},
	{Text: "vertical", Description: "Vertical format (like \\G)"},
	{Text: "csv", Description: "Comma-separated values"},
	{Text: "tsv", Description: "Tab-separated values"},
	{Text: "unicode", Description: "Fancy unicode table format"},
}

type Completer struct {
	keywords []prompt.Suggest
	tables   []prompt.Suggest
	columns  []prompt.Suggest
	metadata map[string][]string // table name -> columns
}

func NewCompleter() *Completer {
	suggestions := make([]prompt.Suggest, len(sqlKeywords))
	for i, k := range sqlKeywords {
		suggestions[i] = prompt.Suggest{Text: k}
	}

	return &Completer{
		keywords: suggestions,
		metadata: make(map[string][]string),
	}
}

func (c *Completer) Complete(d prompt.Document) []prompt.Suggest {
	// Manual word extraction using only whitespace as separators
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
		if parts[0] == "\\F" || parts[0] == "\\f" {
			// If we are exactly at "\f ", suggest formats
			if strings.HasSuffix(lineBefore, " ") {
				return formatTypes
			}
			// If we are mid-word after "\f ", filter formats
			if len(parts) > 1 {
				return prompt.FilterHasPrefix(formatTypes, parts[len(parts)-1], true)
			}
			return specialCommands
		}
		return prompt.FilterHasPrefix(specialCommands, word, true)
	}

	var suggestions []prompt.Suggest

	// 1. Handle aliased/table qualified columns: "SELECT t.^" or "WHERE alias.^"
	if strings.Contains(word, ".") {
		parts := strings.Split(word, ".")
		prefix := parts[0]

		// Resolve prefix to a table name using FULL text of the document
		tableName := c.resolveTable(fullText, prefix)
		if cols, ok := c.metadata[tableName]; ok {
			var dotSuggestions []prompt.Suggest
			for _, col := range cols {
				dotSuggestions = append(dotSuggestions, prompt.Suggest{
					Text:        prefix + "." + col,
					Description: "column of " + tableName,
				})
			}
			// If we're at "alias.", we only want to show that table's columns
			return prompt.FilterHasPrefix(dotSuggestions, word, true)
		}
	}

	if len(wordsBefore) > 0 {
		lastWord := wordsBefore[len(wordsBefore)-1]
		// 2. Prioritize tables after certain keywords
		if (word == "" || word == lastWord) && (lastWord == "FROM" || lastWord == "JOIN" || lastWord == "UPDATE" || lastWord == "INTO" || lastWord == "TABLE") {
			suggestions = append(suggestions, c.tables...)
			// If we are mid-word, FilterHasPrefix will be applied later
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
					suggestions = append(suggestions, prompt.Suggest{
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

	return prompt.FilterHasPrefix(suggestions, word, true)
}

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
	c.tables = make([]prompt.Suggest, 0, len(metadata))

	columnSet := make(map[string]struct{})
	for table, columns := range metadata {
		c.tables = append(c.tables, prompt.Suggest{Text: table, Description: "table"})
		for _, col := range columns {
			columnSet[col] = struct{}{}
		}
	}

	c.columns = make([]prompt.Suggest, 0, len(columnSet))
	for col := range columnSet {
		c.columns = append(c.columns, prompt.Suggest{Text: col, Description: "column"})
	}
}
