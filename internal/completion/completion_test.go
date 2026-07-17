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
	cache.SchemaTables["DB"] = []string{"users", "organizations", "billings"}
	cache.ColumnsWithParent["DB\tUSERS"] = []*db.ColumnDesc{
		{ColumnBase: db.ColumnBase{Name: "id"}},
		{ColumnBase: db.ColumnBase{Name: "name"}},
	}
	cache.ColumnsWithParent["DB\tBILLINGS"] = []*db.ColumnDesc{
		{ColumnBase: db.ColumnBase{Name: "id"}},
		{ColumnBase: db.ColumnBase{Name: "amount"}},
	}
	cache.ColumnsWithParent["DB\tORGANIZATIONS"] = []*db.ColumnDesc{
		{ColumnBase: db.ColumnBase{Name: "id"}},
		{ColumnBase: db.ColumnBase{Name: "display_name"}},
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
		{
			name:       "alias column completion in where clause",
			textBefore: "select * from organizations o where o.",
			fullText:   "select * from organizations o where o.",
			wantFirst:  "o.id",
		},
		{
			name:       "alias column completion in select list",
			textBefore: "select b.",
			fullText:   "select b. from billings b;",
			wantFirst:  "b.id",
			wantNot:    []string{"DISTINCT", "*"},
		},
		{
			name:       "column completion in select list before from",
			textBefore: "select ",
			fullText:   "select  from billings;",
			wantFirst:  "id",
			wantNot:    []string{"DISTINCT", "*"},
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
