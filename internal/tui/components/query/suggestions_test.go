package query

import (
	"testing"

	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
)

func TestApplyCrossSchemaSuggestion(t *testing.T) {
	c := completion.NewCompleter()
	cache := completion.NewDBCache()
	cache.DefaultSchema = "app"
	cache.Schemas["REPORTING"] = "reporting"
	cache.SchemaTables["REPORTING"] = []string{"events"}
	c.UpdateCache(cache)
	for _, prefix := range []string{"reporting.", "reporting.ev", "`reporting`.`ev"} {
		m := NewModel(&config.Config{})
		m.Textarea.SetValue("SELECT * FROM " + prefix)
		m.Textarea.CursorEnd()
		sugg := suggestions.Model{}
		m.UpdateSuggestions(c, &sugg)
		if len(sugg.Items) != 1 {
			t.Fatalf("%s: expected one suggestion, got %v", prefix, sugg.Items)
		}
		m.ApplySuggestion(&sugg)
		want := "SELECT * FROM reporting.events"
		if prefix[0] == '`' {
			want = "SELECT * FROM `reporting`.`events`"
		}
		if got := m.Textarea.Value(); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}
