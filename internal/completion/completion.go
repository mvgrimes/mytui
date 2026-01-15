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

type Completer struct {
	keywords []prompt.Suggest
	tables   map[string][]string // table name -> columns
}

func NewCompleter() *Completer {
	suggestions := make([]prompt.Suggest, len(sqlKeywords))
	for i, k := range sqlKeywords {
		suggestions[i] = prompt.Suggest{Text: k}
	}

	return &Completer{
		keywords: suggestions,
		tables:   make(map[string][]string),
	}
}

func (c *Completer) Complete(d prompt.Document) []prompt.Suggest {
	word := d.GetWordBeforeCursor()
	if word == "" {
		return []prompt.Suggest{}
	}

	// Basic keyword completion
	return prompt.FilterHasPrefix(c.keywords, word, true)
}

func (c *Completer) UpdateSchema(tables map[string][]string) {
	c.tables = tables
	// TODO: Update suggestions with table and column names
}
