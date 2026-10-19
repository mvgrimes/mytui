package tui

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
)

func TestDatabaseAlertPreservesEditorAndResults(t *testing.T) {
	logger := NewDatabaseLogger()
	m := NewModel(&db.Connection{Config: db.Config{Logger: logger}}, &config.Config{MaxResults: 10})
	m.width, m.height = 80, 24
	m.query.Textarea.SetValue("SELECT draft")
	m.addResultFromText("previous result", "SELECT 1")
	result := m.results.Results[0]
	focus := m.focus
	now := time.Now()
	logger.Print("closing bad idle connection: ", "EOF")
	updated, cmd := m.Update(databaseAlertTickMsg(now))
	m = updated.(Model)
	if cmd == nil || m.alertText != "closing bad idle connection: EOF" {
		t.Fatalf("expected alert and next tick, got %q", m.alertText)
	}
	if m.query.Textarea.Value() != "SELECT draft" || len(m.results.Results) != 1 || m.results.Results[0] != result || m.focus != focus {
		t.Fatal("driver diagnostic changed editor, results, or focus")
	}
	// A connection with no underlying sql.DB proves rendering does not issue a
	// query (which could itself reconnect or print further diagnostics).
	if rendered := m.View().Content; !strings.Contains(ansi.Strip(rendered), "closing bad idle connection: EOF") {
		t.Fatal("diagnostic missing from full TUI rendering")
	}
	background := strings.Repeat(strings.Repeat(".", 80)+"\n", 23) + strings.Repeat(".", 80)
	if rendered := m.renderDatabaseAlert(background); !strings.Contains(ansi.Strip(rendered), "closing bad idle connection: EOF") {
		t.Fatal("diagnostic missing from floating alert")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(Model)
	if m.alertText != "" || m.renderDatabaseAlert(background) != background {
		t.Fatal("Escape did not restore the underlying content")
	}
}

func TestDatabaseAlertExpiryAndReplacement(t *testing.T) {
	m := Model{databaseLogger: NewDatabaseLogger()}
	now := time.Now()
	m.databaseLogger.Print("first")
	m.updateDatabaseAlert(now)
	m.updateDatabaseAlert(now.Add(databaseAlertDuration - time.Second))
	if m.alertText != "first" {
		t.Fatal("alert expired too soon")
	}
	m.databaseLogger.Print("second")
	m.updateDatabaseAlert(now.Add(databaseAlertDuration))
	if m.alertText != "second" {
		t.Fatal("new diagnostic should replace and extend the alert")
	}
	m.updateDatabaseAlert(now.Add(2 * databaseAlertDuration))
	if m.alertText != "" {
		t.Fatal("alert did not expire")
	}
}

func TestDatabaseLoggerDoesNotBlock(t *testing.T) {
	l := NewDatabaseLogger()
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 1000 {
					l.Print("reconnect diagnostic")
				}
			})
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("logging blocked with a full queue")
	}
}

func TestDatabaseAlertFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {5, 2}, {20, 5}, {80, 24}} {
		m := Model{width: size[0], height: size[1], alertText: strings.Repeat("long message ", 40)}
		rendered := m.renderDatabaseAlert("")
		w, h := lipgloss.Size(rendered)
		if w > size[0] || h > size[1] {
			t.Fatalf("alert %dx%d exceeds terminal %v", w, h, size)
		}
	}
	if cmd := (Model{}).databaseAlertTickCmd(); cmd != nil {
		t.Fatal("no polling needed without a TUI logger")
	}
}
