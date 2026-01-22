package repl

import (
	"github.com/c-bata/go-prompt"
	"github.com/mvgrimes/mytui/internal/completion"
)

func (r *REPL) completerBridge(d prompt.Document) []prompt.Suggest {
	lineBefore := d.TextBeforeCursor()
	doc := completion.Document{
		Text:           d.Text,
		CursorPosition: len(lineBefore),
	}
	suggestions := r.completer.Complete(doc)

	res := make([]prompt.Suggest, len(suggestions))
	for i, s := range suggestions {
		res[i] = prompt.Suggest{
			Text:        s.Text,
			Description: s.Description,
		}
	}
	return res
}
