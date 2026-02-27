package suggestions

import "github.com/mvgrimes/mytui/internal/completion"

type Model struct {
	Show  bool
	Items []completion.Suggestion
	Index int
}
