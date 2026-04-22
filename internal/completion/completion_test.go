package completion

import (
	"github.com/mvgrimes/mytui/internal/db"
	"testing"
)

func TestCompleter_resolveTable(t *testing.T) {
	c := NewCompleter()
	cache := NewDBCache()
	cache.DefaultSchema = "DB"
	cache.SchemaTables["DB"] = []string{"users", "projects"}
	c.UpdateCache(cache)

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
	cache := NewDBCache()
	cache.DefaultSchema = "DB"
	cache.SchemaTables["DB"] = []string{"users"}
	cache.ColumnsWithParent["DB\tUSERS"] = []*db.ColumnDesc{
		{ColumnBase: db.ColumnBase{Name: "id"}},
		{ColumnBase: db.ColumnBase{Name: "name"}},
	}
	c.UpdateCache(cache)

	tests := []struct {
		name       string
		textBefore string
		fullText   string
		wantFirst  string
		wantNot    []string
	}{
		{
			name:       "keyword completion",
			textBefore: "SEL",
			fullText:   "SEL",
			wantFirst:  "SELECT",
		},
		{
			name:       "select keyword context excludes table names",
			textBefore: "SELECT ",
			fullText:   "SELECT ",
			wantFirst:  "DISTINCT",
			wantNot:    []string{"users"},
		},
		{
			name:       "select star suggests from",
			textBefore: "SELECT * ",
			fullText:   "SELECT * ",
			wantFirst:  "FROM",
			wantNot:    []string{"SELECT", "INSERT", "UPDATE", "users"},
		},
		{
			name:       "table completion after FROM",
			textBefore: "SELECT * FROM ",
			fullText:   "SELECT * FROM ",
			wantFirst:  "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Document{
				Text:           tt.fullText,
				CursorPosition: len(tt.textBefore),
			}

			got := c.Complete(d)
			if len(got) == 0 {
				t.Errorf("%s: Completer.Complete() returned no suggestions", tt.name)
				return
			}

			found := false
			for _, s := range got {
				if s.Text == tt.wantFirst {
					found = true
					break
				}
			}
			if tt.wantFirst != "" && !found {
				t.Errorf("%s: could not find suggestion %v in %v", tt.name, tt.wantFirst, got)
			}

			for _, forbidden := range tt.wantNot {
				for _, s := range got {
					if s.Text == forbidden {
						t.Errorf("%s: unexpected suggestion %v in %v", tt.name, forbidden, got)
					}
				}
			}
		})
	}
}
