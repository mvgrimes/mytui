package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mvgrimes/mytui/internal/tui/components/modals"
	"github.com/mvgrimes/mytui/internal/tui/styles"
)

// DatabaseLogger captures driver diagnostics without writing over the terminal.
// The bounded queue must never block database operations, including queries run
// synchronously on the Bubble Tea update goroutine.
type DatabaseLogger struct {
	messages chan string
}

func NewDatabaseLogger() *DatabaseLogger {
	return &DatabaseLogger{messages: make(chan string, 32)}
}

func (l *DatabaseLogger) Print(v ...any) {
	select {
	case l.messages <- strings.TrimSpace(fmt.Sprint(v...)):
	default:
	}
}

type databaseAlertTickMsg time.Time

const databaseAlertDuration = 8 * time.Second

func (m Model) databaseAlertTickCmd() tea.Cmd {
	if m.databaseLogger == nil {
		return nil
	}
	return tea.Tick(200*time.Millisecond, func(now time.Time) tea.Msg {
		return databaseAlertTickMsg(now)
	})
}

func (m *Model) updateDatabaseAlert(now time.Time) {
	if !now.Before(m.alertExpires) {
		m.alertText = ""
	}
	if m.databaseLogger == nil {
		return
	}
	// Bound work per tick even if diagnostics are being produced continuously.
	for range cap(m.databaseLogger.messages) {
		select {
		case text := <-m.databaseLogger.messages:
			if text != "" {
				m.alertText = text
				m.alertExpires = now.Add(databaseAlertDuration)
			}
		default:
			return
		}
	}
}

func (m Model) renderDatabaseAlert(background string) string {
	if m.alertText == "" || m.width <= 0 || m.height <= 0 {
		return background
	}
	style := styles.ErrorStyle.Width(max(min(m.width, 70)-4, 1))
	if m.width >= 6 && m.height >= 3 {
		style = style.Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#FF5555")).Padding(0, 1)
	}
	foreground := style.MaxWidth(m.width).MaxHeight(m.height).Render("Database connection\n" + m.alertText + "\nEsc: dismiss")
	background = modals.EnsureBackgroundSize(background, foreground, m.width, m.height)
	w, _ := lipgloss.Size(foreground)
	return compositeOverlay(foreground, background, max(m.width-w, 0), 0)
}
