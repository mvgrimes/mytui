package completion

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mvgrimes/mytui/internal/parser"
	"github.com/mvgrimes/mytui/internal/parser/ast"
	"github.com/mvgrimes/mytui/internal/parser/dialect"
	"github.com/mvgrimes/mytui/internal/parser/token"
)

type Diagnostic struct {
	Message string
	From    token.Pos
	To      token.Pos
}

func (c *Completer) Lint(text string) []Diagnostic {
	var diagnostics []Diagnostic

	// 1. Lexical errors
	src := bytes.NewBuffer([]byte(text))
	tokenizer := token.NewTokenizer(src, &dialect.GenericSQLDialect{})
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		// Tokenizer.Tokenize usually returns early on first err,
		// but let's check for ILLEGAL tokens in the list.
	}

	for _, tok := range tokens {
		if tok.Kind == token.ILLEGAL {
			diagnostics = append(diagnostics, Diagnostic{
				Message: fmt.Sprintf("Illegal token: %v", tok.Value),
				From:    tok.From,
				To:      tok.To,
			})
		}
	}

	// 2. Syntax errors via Parser
	parsed, err := parser.Parse(text)
	if err != nil {
		// Parser doesn't currently return detailed error positions in a clean way
		// based on the ported code, but we can look for "Unmatched" keywords.
	}

	if parsed != nil {
		diagnostics = append(diagnostics, c.checkNodes(parsed.GetTokens())...)
	}

	return diagnostics
}

func (c *Completer) checkNodes(nodes []ast.Node) []Diagnostic {
	var diagnostics []Diagnostic
	for _, node := range nodes {
		// Check for unmatched keywords (e.g. "FORM" instead of "FROM")
		if item, ok := node.(*ast.Item); ok {
			tok := item.GetToken()
			if tok.Kind == token.SQLKeyword {
				if sqlWord, ok := tok.Value.(*token.SQLWord); ok {
					if sqlWord.Kind == dialect.Unmatched {
						// This might be an identifier, so check if it's in our metadata
						// If it looks like a keyword but isn't matched, and isn't a known table/column,
						// it might be a misspelling.
						// For now, let's just flag keywords that look like typos.
						word := strings.ToUpper(sqlWord.Keyword)
						if isCommonTypo(word) {
							diagnostics = append(diagnostics, Diagnostic{
								Message: fmt.Sprintf("Possible typo: %s", word),
								From:    tok.From,
								To:      tok.To,
							})
						}
					}
				}
			}
		}

		// Recursive check for TokenLists
		if list, ok := node.(ast.TokenList); ok {
			diagnostics = append(diagnostics, c.checkNodes(list.GetTokens())...)
		}
	}
	return diagnostics
}

func isCommonTypo(word string) bool {
	typos := map[string]string{
		"FORM":    "FROM",
		"SELECTE": "SELECT",
		"WHER":    "WHERE",
		"UPDAT":   "UPDATE",
		"INSER":   "INSERT",
	}
	_, ok := typos[word]
	return ok
}
