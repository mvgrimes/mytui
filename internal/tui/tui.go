package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/parser"
	"github.com/mvgrimes/mytui/internal/special"
	"github.com/mvgrimes/mytui/internal/vim"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// Styles moved to styles.go

func (m Model) Init() tea.Cmd {
	return m.refreshCacheCmd()
}

func (m *Model) saveToHistory(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	// Update in-memory
	m.history = append(m.history, line)
	m.historyIndex = len(m.history)

	// Update file
	now := time.Now().Format("2006-01-02 15:04:05")
	f, err := os.OpenFile(m.config.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# %s\n", now)
	lines := strings.Split(line, "\n")
	for _, l := range lines {
		fmt.Fprintf(f, "+%s\n", l)
	}
}

func (m *Model) recalculateHeight() {
	// Give each result its needed space, up to its DisplaySize
	for _, r := range m.results {
		r.Viewport.Width = m.width
		if r.Expanded {
			// Use FormattedData if available (pinned header case), otherwise use full Formatted
			content := r.FormattedData
			if content == "" {
				content = r.Formatted
			}
			lines := strings.Count(content, "\n") + 1
			height := lines
			if height > r.DisplaySize {
				height = r.DisplaySize
			}
			r.Viewport.Height = height
		} else {
			r.Viewport.Height = 0
		}
	}
	m.clampResultsScrollOffset()
}

// ensureFocusedResultVisible adjusts resultsScrollOffset so the focused result is visible.
func (m *Model) ensureFocusedResultVisible() {
	if m.focusedResult < 0 || m.focusedResult >= len(m.results) {
		return
	}

	// Compute the top y-position of the focused result in the full results rendering.
	top := 0
	for i, r := range m.results {
		if i == m.focusedResult {
			break
		}
		top++ // result header line
		if r.Expanded {
			if r.FormattedHeader != "" {
				top += strings.Count(r.FormattedHeader, "\n") + 1
			}
			top += r.Viewport.Height
		}
	}

	r := m.results[m.focusedResult]
	height := 1 // result header line
	if r.Expanded {
		if r.FormattedHeader != "" {
			height += strings.Count(r.FormattedHeader, "\n") + 1
		}
		height += r.Viewport.Height
	}
	bottom := top + height

	available := m.computeAvailableHeight()

	// Scroll up if result is above the visible area.
	if top < m.resultsScrollOffset {
		m.resultsScrollOffset = top
	}
	// Scroll down if result extends below the visible area.
	if bottom > m.resultsScrollOffset+available {
		m.resultsScrollOffset = bottom - available
	}
	if m.resultsScrollOffset < 0 {
		m.resultsScrollOffset = 0
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var tiCmd tea.Cmd

	switch msg := msg.(type) {
	case *completion.DBCache:
		m.completer.UpdateCache(msg)
		return m, nil
	case tea.KeyMsg:
		if updated, cmd, handled := m.updateModalKey(msg); handled {
			return updated, cmd
		}
		if updated, cmd, handled := m.updateSuggestionsKey(msg); handled {
			return updated, cmd
		}
		if updated, cmd, handled := m.updateGlobalKey(msg); handled {
			return updated, cmd
		}
		if updated, cmd, handled := m.updateResultsKey(msg); handled {
			return updated, cmd
		}
		if updated, cmd, handled := m.updateQueryKey(msg); handled {
			return updated, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)
		for _, r := range m.results {
			r.Viewport.Width = msg.Width
		}
		m.recalculateHeight()
		return m, nil
	}

	if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
		oldVal := m.textarea.Value()
		m.textarea, tiCmd = m.textarea.Update(msg)
		if m.textarea.Value() != oldVal {
			m.lastError = parser.Validate(m.textarea.Value())
			m.recalculateHeight()
			m.updateSuggestions()
			if m.shouldOpenSuggestionsOnEdit() && len(m.suggestions) > 0 {
				if !m.showSuggestions {
					m.suggestionIndex = -1
				} else if m.suggestionIndex >= len(m.suggestions) {
					m.suggestionIndex = -1
				}
				m.showSuggestions = true
			} else {
				m.showSuggestions = false
			}
		}
	} else if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
		oldVal := m.textarea.Value()
		if _, ok := msg.(tea.KeyMsg); !ok {
			m.textarea, tiCmd = m.textarea.Update(msg)
		}
		if m.textarea.Value() != oldVal {
			m.recalculateHeight()
		}
	}

	var resCmds []tea.Cmd
	_, isKey := msg.(tea.KeyMsg)
	if !isKey {
		for _, r := range m.results {
			var cmd tea.Cmd
			r.Viewport, cmd = r.Viewport.Update(msg)
			resCmds = append(resCmds, cmd)
		}
	} else if m.focus == FocusResults && m.focusedResult >= 0 && m.focusedResult < len(m.results) {
		var cmd tea.Cmd
		m.results[m.focusedResult].Viewport, cmd = m.results[m.focusedResult].Viewport.Update(msg)
		resCmds = append(resCmds, cmd)
	}

	return m, tea.Batch(append(resCmds, tiCmd)...)
}

func (m Model) refreshCacheCmd() tea.Cmd {
	return func() tea.Msg {
		cache, err := m.fetchCache()
		if err != nil {
			return nil
		}
		return cache
	}
}

func (m Model) executeQuery(query string) (Model, tea.Cmd) {
	trimmedQuery := strings.TrimSpace(query)
	lowerQuery := strings.ToLower(trimmedQuery)
	if lowerQuery == "exit" || lowerQuery == "quit" {
		return m, tea.Quit
	}

	m.specialOutput.Reset()
	if special.Handle(trimmedQuery, &m) {
		m.addResultFromText(m.specialOutput.String(), trimmedQuery)
		m.recalculateHeight()
		m.textarea.Reset()
		m.saveToHistory(query)
		return m, nil
	}

	format := m.currentFormat
	if m.onceFormat != "" {
		format = m.onceFormat
		m.onceFormat = ""
	}

	if strings.HasSuffix(trimmedQuery, "\\G") {
		format = formatter.FormatVertical
		query = strings.TrimSuffix(trimmedQuery, "\\G")
	}

	m.saveToHistory(query)
	m.lastQuery = query

	result, err := m.conn.ExecuteQuery(query)
	if err != nil {
		wrappedError := lipgloss.NewStyle().Width(m.width - 2).Render(fmt.Sprintf("Error: %v", err))
		m.addResultFromText(wrappedError, query)
		m.recalculateHeight()
		return m, nil
	} else {
		m.addResult(result, query, format)
		m.recalculateHeight()
		m.textarea.Reset()
		if len(result.Headers) > 0 {
			m.focus = FocusResults
			m.textarea.Blur()
		}

		upperQuery := strings.ToUpper(trimmedQuery)
		if strings.HasPrefix(upperQuery, "USE") ||
			strings.HasPrefix(upperQuery, "CREATE") ||
			strings.HasPrefix(upperQuery, "DROP") ||
			strings.HasPrefix(upperQuery, "ALTER") {
			return m, m.refreshCacheCmd()
		}
	}

	return m, nil
}

// View helpers moved into separate files for clarity

func (m Model) View() string {
	m.UpdateCursorStyle()

	qHeader := m.renderQueryHeader(m.focus == FocusQuery)

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Margin(0, 1)
	helpTextStr := "j/k:select · /:search · n/N:next/prev · Enter:detail · y:copy · v:edit · d:delete · R:rerun · +/-:size · Tab:focus"
	if m.focus == FocusQuery {
		if m.vimState.Mode == vim.NormalMode {
			helpTextStr = "gi:INSERT · gs:SELECT · gd:DELETE · gc:CREATE · gf:fields · gt:table · gw:where · Tab:focus"
		} else {
			helpTextStr = "Ctrl+K:autocomplete · Ctrl+Space:menu · Ctrl+R:history search · Ctrl+P/N:history · Ctrl+X s/i/d/c:SQL · Ctrl+X f/t/w:jump · Tab:focus"
		}
	}
	helpText := helpStyle.Render(helpTextStr)

	queryView := m.renderQueryArea()
	visibleResultsStr, visibleResultLines := m.renderResultsPanel()

	view := lipgloss.JoinVertical(lipgloss.Left,
		visibleResultsStr,
		qHeader,
		queryView,
		helpText,
	)

	// Modal overlays take priority (checked first).
	// The view is now bounded to m.height lines, so m.height/2 correctly
	// centres the modal on the visible screen.
	if m.showRowDetail {
		fg := m.renderRowDetailModal()
		bg := ensureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if m.showCopyMenu {
		fg := m.renderCopyMenu()
		bg := ensureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if m.showHistorySearch {
		fg := m.renderHistorySearch()
		bg := ensureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if m.showMenu {
		fg := m.renderMenu()
		bg := ensureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if m.showSuggestions {
		fg := m.renderSuggestions()
		bg := ensureBackgroundSize(view, fg, m.width, m.height)
		_, fgHeight := lipgloss.Size(fg)
		xOff, yOff := m.computeSuggestionOffsets(visibleResultLines, fgHeight)
		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, xOff, yOff)
	}

	return view
}
