package completion

import (
	"regexp"
	"strings"

	"github.com/mvgrimes/mytui/internal/parser"
	"github.com/mvgrimes/mytui/internal/parser/ast"
	"github.com/mvgrimes/mytui/internal/parser/astutil"
	"github.com/mvgrimes/mytui/internal/parser/parseutil"
	"github.com/mvgrimes/mytui/internal/parser/token"
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
	cache           *DBCache
	SmartCompletion bool
}

type completionType int

const (
	_ completionType = iota
	CompletionTypeKeyword
	CompletionTypeFunction
	CompletionTypeColumn
	CompletionTypeTable
	CompletionTypeReferencedTable
	CompletionTypeSchema
	CompletionTypeSubQuery
	CompletionTypeSubQueryColumn
	CompletionTypeJoin
	CompletionTypeJoinOn
)

type completionParent struct {
	Type ParentType
	Name string
}

type ParentType int

const (
	_ ParentType = iota
	ParentTypeNone
	ParentTypeSchema
	ParentTypeTable
	ParentTypeSubQuery
)

var noneParent = &completionParent{Type: ParentTypeNone}

type CompletionContext struct {
	types  []completionType
	parent *completionParent
}

func NewCompleter() *Completer {
	suggestions := make([]Suggestion, len(sqlKeywords))
	for i, k := range sqlKeywords {
		suggestions[i] = Suggestion{Text: k}
	}

	return &Completer{
		keywords:        suggestions,
		cache:           NewDBCache(),
		SmartCompletion: true,
	}
}

func (c *Completer) Complete(d Document) []Suggestion {
	if !c.SmartCompletion {
		word := getLastWord(d.Text, d.CursorPosition)
		return filterHasPrefix(c.keywords, word, true)
	}

	parsed, err := parser.Parse(d.Text)
	if err != nil {
		// Fallback to simple completion if parse fails
		word := getLastWord(d.Text, d.CursorPosition)
		return filterHasPrefix(c.keywords, word, true)
	}

	// Calculate cursor position in terms of Line and Col
	pos := calculatePos(d.Text, d.CursorPosition)

	nodeWalker := parseutil.NewNodeWalker(parsed, pos)
	ctx := getCompletionTypes(nodeWalker)

	definedTables, _ := parseutil.ExtractTable(parsed, pos)
	// definedSubQueries, _ := parseutil.ExtractSubQueryViews(parsed, pos)

	lastWord := getLastWord(d.Text, d.CursorPosition)
	// withBackQuote := strings.HasPrefix(lastWord, "`")

	var suggestions []Suggestion

	if completionTypeIs(ctx.types, CompletionTypeColumn) {
		suggestions = append(suggestions, c.columnCandidates(definedTables, ctx.parent)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeTable) {
		suggestions = append(suggestions, c.tableCandidates(ctx.parent, definedTables)...)
	}
	if completionTypeIs(ctx.types, CompletionTypeFunction) {
		// Just keywords for now as placeholders for functions
		suggestions = append(suggestions, c.keywords...)
	}
	if completionTypeIs(ctx.types, CompletionTypeKeyword) {
		suggestions = append(suggestions, c.keywords...)
	}

	return filterHasPrefix(suggestions, lastWord, true)
}

func calculatePos(text string, cursor int) token.Pos {
	line := 0
	col := 0
	for i := 0; i < cursor && i < len(text); i++ {
		if text[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return token.Pos{Line: line, Col: col}
}

func completionTypeIs(types []completionType, expect completionType) bool {
	for _, t := range types {
		if t == expect {
			return true
		}
	}
	return false
}

func getCompletionTypes(nw *parseutil.NodeWalker) *CompletionContext {
	memberIdentifierMatcher := astutil.NodeMatcher{
		NodeTypes: []ast.NodeType{ast.TypeMemberIdentifier},
	}

	syntaxPos := parseutil.CheckSyntaxPosition(nw)
	var t []completionType
	p := noneParent
	switch {
	case syntaxPos == parseutil.ColName:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{CompletionTypeColumn, CompletionTypeSubQueryColumn}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.Parent.String(),
			}
		} else {
			t = []completionType{CompletionTypeColumn, CompletionTypeTable, CompletionTypeFunction}
			p = noneParent
		}
	case syntaxPos == parseutil.SelectExpr || syntaxPos == parseutil.CaseValue:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{CompletionTypeColumn}
			p = &completionParent{
				Type: ParentTypeTable,
				Name: mi.ParentTok.NoQuoteString(),
			}
		} else {
			t = []completionType{CompletionTypeColumn, CompletionTypeTable, CompletionTypeFunction}
		}
	case syntaxPos == parseutil.TableReference:
		if nw.CurNodeIs(memberIdentifierMatcher) {
			mi := nw.CurNodeTopMatched(memberIdentifierMatcher).(*ast.MemberIdentifier)
			t = []completionType{CompletionTypeTable}
			p = &completionParent{
				Type: ParentTypeSchema,
				Name: mi.ParentTok.NoQuoteString(),
			}
		} else {
			t = []completionType{CompletionTypeTable, CompletionTypeSchema}
		}
	case syntaxPos == parseutil.WhereCondition:
		t = []completionType{CompletionTypeColumn, CompletionTypeFunction}
	default:
		t = []completionType{CompletionTypeKeyword}
	}
	return &CompletionContext{
		types:  t,
		parent: p,
	}
}

func (c *Completer) columnCandidates(targetTables []*parseutil.TableInfo, parent *completionParent) []Suggestion {
	var candidates []Suggestion

	switch parent.Type {
	case ParentTypeNone:
		for _, table := range targetTables {
			if cols, ok := c.cache.ColumnDescs(table.Name); ok {
				for _, col := range cols {
					candidates = append(candidates, Suggestion{
						Text:        col.Name,
						Description: "column of " + table.Name,
					})
				}
			}
		}
	case ParentTypeTable:
		for _, table := range targetTables {
			if table.Name != parent.Name && table.Alias != parent.Name {
				continue
			}
			if cols, ok := c.cache.ColumnDescs(table.Name); ok {
				for _, col := range cols {
					candidates = append(candidates, Suggestion{
						Text:        parent.Name + "." + col.Name,
						Description: "column of " + table.Name,
					})
				}
			}
		}
	}
	return candidates
}

func (c *Completer) tableCandidates(parent *completionParent, targetTables []*parseutil.TableInfo) []Suggestion {
	var candidates []Suggestion
	switch parent.Type {
	case ParentTypeNone:
		for _, table := range c.cache.SortedTables() {
			candidates = append(candidates, Suggestion{Text: table, Description: "table"})
		}
	case ParentTypeSchema:
		if tables, ok := c.cache.SortedTablesByDBName(parent.Name); ok {
			for _, table := range tables {
				candidates = append(candidates, Suggestion{Text: table, Description: "table"})
			}
		}
	}
	return candidates
}

func getLastWord(text string, cursor int) string {
	if cursor <= 0 {
		return ""
	}
	sub := text[:cursor]
	reg := regexp.MustCompile("[\\w\\.`]+$")
	match := reg.FindString(sub)
	return match
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
	for _, tableName := range c.cache.SortedTables() {
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
		for _, tableName := range c.cache.SortedTables() {
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
	for _, tableName := range c.cache.SortedTables() {
		upperTableName := " " + strings.ToUpper(tableName) + " "
		if strings.Contains(cleanText, upperTableName) {
			mentioned = append(mentioned, tableName)
		}
	}
	return mentioned
}

func (c *Completer) UpdateCache(cache *DBCache) {
	c.cache = cache
	c.tables = make([]Suggestion, 0)
	columnSet := make(map[string]struct{})

	for _, schema := range cache.SortedSchemas() {
		tables, _ := cache.SortedTablesByDBName(schema)
		for _, table := range tables {
			c.tables = append(c.tables, Suggestion{Text: table, Description: "table"})
			if cols, ok := cache.ColumnDatabase(schema, table); ok {
				for _, col := range cols {
					columnSet[col.Name] = struct{}{}
				}
			}
		}
	}

	c.columns = make([]Suggestion, 0, len(columnSet))
	for col := range columnSet {
		c.columns = append(c.columns, Suggestion{Text: col, Description: "column"})
	}
}
