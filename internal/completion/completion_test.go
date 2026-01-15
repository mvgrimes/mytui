package completion

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/c-bata/go-prompt"
)

// Helper to create Document with cursor position since fields are unexported
func NewDocument(text string, cursorPosition int) prompt.Document {
	d := prompt.Document{Text: text}

	// Use reflection and unsafe to set unexported cursorPosition
	v := reflect.ValueOf(&d).Elem()
	f := v.FieldByName("cursorPosition")

	// This is a bit hacky but necessary for testing since the field is unexported
	// and no constructor allows setting it.
	ptr := unsafe.Pointer(f.UnsafeAddr())
	*(*int)(ptr) = cursorPosition

	return d
}

func TestCompleter_resolveTable(t *testing.T) {
	c := NewCompleter()
	metadata := map[string][]string{
		"users":    {"id", "name", "email"},
		"projects": {"id", "user_id", "title"},
	}
	c.UpdateSchema(metadata)

	tests := []struct {
		name     string
		fullText string
		prefix   string
		want     string
	}{
		{
			name:     "direct table name",
			fullText: "SELECT * FROM users",
			prefix:   "users",
			want:     "users",
		},
		{
			name:     "simple alias",
			fullText: "SELECT u.id FROM users u",
			prefix:   "u",
			want:     "users",
		},
		{
			name:     "alias with AS",
			fullText: "SELECT u.id FROM users AS u",
			prefix:   "u",
			want:     "users",
		},
		{
			name:     "alias in JOIN",
			fullText: "SELECT p.title FROM users u JOIN projects p ON u.id = p.user_id",
			prefix:   "p",
			want:     "projects",
		},
		{
			name:     "alias before definition (SELECT clause)",
			fullText: "SELECT u. FROM users u",
			prefix:   "u",
			want:     "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.resolveTable(tt.fullText, tt.prefix); got != tt.want {
				t.Errorf("Completer.resolveTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompleter_Complete(t *testing.T) {
	c := NewCompleter()
	metadata := map[string][]string{
		"users": {"id", "name"},
	}
	c.UpdateSchema(metadata)

	tests := []struct {
		name          string
		textBefore    string
		fullText      string
		wantFirst     string
		wantHasPrefix string
	}{
		{
			name:       "keyword completion",
			textBefore: "SEL",
			fullText:   "SEL",
			wantFirst:  "SELECT",
		},
		{
			name:       "table completion after FROM",
			textBefore: "SELECT * FROM ",
			fullText:   "SELECT * FROM ",
			wantFirst:  "users",
		},
		{
			name:       "dot notation completion",
			textBefore: "SELECT users.",
			fullText:   "SELECT users.",
			wantFirst:  "users.id",
		},
		{
			name:       "alias dot notation completion",
			textBefore: "SELECT u.",
			fullText:   "SELECT u. FROM users u",
			wantFirst:  "u.id",
		},
		{
			name:       "alias dot notation completion",
			textBefore: "SELECT u.",
			fullText:   "SELECT u. FROM users u;",
			wantFirst:  "u.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDocument(tt.fullText, len(tt.textBefore))

			got := c.Complete(d)
			if len(got) == 0 {
				t.Errorf("%s: Completer.Complete() returned no suggestions", tt.name)
				return
			}

			if tt.wantFirst != "" && got[0].Text != tt.wantFirst {
				t.Errorf("%s: got first suggestion %v, want %v. All suggestions: %v", tt.name, got[0].Text, tt.wantFirst, got)
			}
		})
	}
}
