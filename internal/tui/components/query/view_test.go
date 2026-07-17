package query

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/vim"
)

func TestWrapLineSplitsLongLine(t *testing.T) {
	segments := wrapLine("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if strings.Join(segments, ",") != strings.Join(want, ",") {
		t.Fatalf("wrapLine() = %#v, want %#v", segments, want)
	}
}

func TestRenderAreaWrapsLongQuery(t *testing.T) {
	m := NewModel(&config.Config{HistoryFile: t.TempDir() + "/history"})
	m.Textarea.SetWidth(15)
	m.Textarea.Blur()
	m.Textarea.SetValue("select abcdefghij")

	view := ansi.Strip(m.RenderArea(vim.NewVimState()))
	if !strings.Contains(view, " 1 | select abc") {
		t.Fatalf("rendered view missing first wrapped segment:\n%s", view)
	}
	if !strings.Contains(view, "   | defghij") {
		t.Fatalf("rendered view missing continuation segment:\n%s", view)
	}
}
