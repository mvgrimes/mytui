package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mvgrimes/mytui/internal/completion"
	"github.com/mvgrimes/mytui/internal/parser"
	"github.com/mvgrimes/mytui/internal/tui/components/menu"
	"github.com/mvgrimes/mytui/internal/tui/components/modals"
	"github.com/mvgrimes/mytui/internal/tui/components/query"
	"github.com/mvgrimes/mytui/internal/tui/components/results"
	"github.com/mvgrimes/mytui/internal/tui/components/suggestions"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/vim"
)

func (m Model) Init() tea.Cmd {
	return m.refreshCacheCmd()
}

func (m *Model) saveToHistory(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	// Update in-memory
	now := time.Now().Format("2006-01-02 15:04:05")
	m.query.History = append(m.query.History, line)
	m.query.HistoryIndex = len(m.query.History)
	m.query.HistoryTimestamps = append(m.query.HistoryTimestamps, now)

	// Update file
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
	for _, r := range m.results.Results {
		r.Viewport.SetWidth(m.width)
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
			r.Viewport.SetHeight(height)
		} else {
			r.Viewport.SetHeight(0)
		}
	}
	available := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
	results.ClampScrollOffset(&m.results, available)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var tiCmd tea.Cmd

	switch msg := msg.(type) {
	case *completion.DBCache:
		m.completer.UpdateCache(msg)
		return m, nil
	case queryEditorFinishedMsg:
		if msg.err != nil {
			m.specialOutput.Reset()
			fmt.Fprintf(&m.specialOutput, "Error: %v\n", msg.err)
			m.addResultFromText(m.specialOutput.String(), "Edit Query")
			m.recalculateHeight()
			return m, nil
		}
		m.query.Textarea.SetValue(msg.content)
		m.query.Textarea.CursorEnd()
		m.query.LastError = parser.Validate(msg.content)
		m.suggestions.Show = false
		m.focus = core.FocusQuery
		m.recalculateHeight()
		return m, m.query.Textarea.Focus()
	case tea.KeyPressMsg:
		if handled, cmd := modals.UpdateRowDetail(&m.modals.RowDetail, msg); handled {
			return m, cmd
		}
		if handled, cmd := modals.UpdateCopyMenu(&m.modals.CopyMenu, msg, modals.CopyMenuDeps{
			OnCopy: func(formatIndex int) tea.Cmd {
				if m.results.FocusedResultIndex < 0 || m.results.FocusedResultIndex >= len(m.results.Results) {
					return nil
				}
				res := m.results.Results[m.results.FocusedResultIndex]
				name := modals.CopyRowToClipboard(res, core.CopyFormat(formatIndex), m.config)
				m.specialOutput.Reset()
				fmt.Fprintf(&m.specialOutput, "Row copied to clipboard as %s.\n", name)
				m.addResultFromText(m.specialOutput.String(), "Copy Row")
				m.focus = core.FocusQuery
				return m.query.Textarea.Focus()
			},
		}); handled {
			return m, cmd
		}
		if handled, cmd := modals.UpdateHistorySearch(&m.modals.HistorySearch, msg, modals.HistoryDeps{
			History:            m.query.History,
			Timestamps:         m.query.HistoryTimestamps,
			ListHeight:         m.historyListHeight(),
			SetQueryText:       func(q string) { m.query.Textarea.SetValue(q); m.query.Textarea.CursorEnd() },
			TextareaFocus:      func() tea.Cmd { return m.query.Textarea.Focus() },
			SetShowSuggestions: func(show bool) { m.suggestions.Show = show },
		}); handled {
			return m, cmd
		}
		if handled, cmd := menu.Update(&m.menu, msg, m.buildMenuCommands(), menu.UpdateDeps{
			OnSaveFavorite: func(name string) tea.Cmd {
				m.SaveFavorite(name)
				return nil
			},
		}); handled {
			return m, cmd
		}
		if handled, cmd := suggestions.UpdateKey(&m.suggestions, msg, suggestions.UpdateDeps{
			FocusQuery:        m.focus == core.FocusQuery,
			VimState:          m.vimState,
			Textarea:          &m.query.Textarea,
			RecalculateHeight: m.recalculateHeight,
			UpdateSuggestions: func() { m.query.UpdateSuggestions(m.completer, &m.suggestions) },
			ApplySuggestion:   func() { m.query.ApplySuggestion(&m.suggestions) },
			ShouldOpenOnEdit:  func() bool { return m.query.ShouldOpenSuggestionsOnEdit(m.focus) },
		}); handled {
			return m, cmd
		}
		if handled, cmd := m.updateGlobalKey(msg); handled {
			return m, cmd
		}
		if handled, cmd := results.UpdateKey(&m.results, msg, results.UpdateDeps{
			Focus:              m.focus,
			FocusedResultIndex: m.results.FocusedResultIndex,
			Width:              m.width,
			Config:             m.config,
			ConnExecute:        m.conn.ExecuteQuery,
			OpenRowDetail: func(res *core.Result) {
				modals.OpenRowDetail(&m.modals.RowDetail, res, m.width, m.height)
			},
			OpenCopyMenu: func() {
				m.modals.CopyMenu.Show = true
				m.modals.CopyMenu.Index = 0
			},
			OpenRowInEditor: func(res *core.Result) tea.Cmd {
				return modals.OpenRowInEditor(res)
			},
			SetFocus:          func(f core.Focus) { m.focus = f },
			SetFocusedResult:  func(i int) { m.results.FocusedResultIndex = i },
			SetQueryText:      func(q string) { m.query.Textarea.SetValue(q) },
			TextareaFocus:     func() tea.Cmd { return m.query.Textarea.Focus() },
			RecalculateHeight: m.recalculateHeight,
		}); handled {
			return m, cmd
		}
		if handled, cmd := query.UpdateKey(&m.query, msg, query.UpdateDeps{
			Focus:             m.focus,
			VimState:          m.vimState,
			Suggestions:       &m.suggestions,
			Completer:         m.completer,
			OpenHistorySearch: func() { modals.OpenHistorySearch(&m.modals.HistorySearch) },
			RecalculateHeight: m.recalculateHeight,
			ExecuteQuery:      func(q string) tea.Cmd { return m.executeQuery(q) },
		}); handled {
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.query.Textarea.SetWidth(msg.Width)
		for _, r := range m.results.Results {
			r.Viewport.SetWidth(msg.Width)
		}
		m.recalculateHeight()
		return m, nil
	}

	if m.focus == core.FocusQuery && m.vimState.Mode == vim.InsertMode {
		oldVal := m.query.Textarea.Value()
		m.query.Textarea, tiCmd = m.query.Textarea.Update(msg)
		if m.query.Textarea.Value() != oldVal {
			m.query.LastError = parser.Validate(m.query.Textarea.Value())
			m.recalculateHeight()
			m.query.UpdateSuggestions(m.completer, &m.suggestions)
			if m.query.ShouldOpenSuggestionsOnEdit(m.focus) && len(m.suggestions.Items) > 0 {
				if !m.suggestions.Show {
					m.suggestions.Index = -1
				} else if m.suggestions.Index >= len(m.suggestions.Items) {
					m.suggestions.Index = -1
				}
				m.suggestions.Show = true
			} else {
				m.suggestions.Show = false
			}
		}
	} else if m.focus == core.FocusQuery && m.vimState.Mode == vim.NormalMode {
		oldVal := m.query.Textarea.Value()
		if _, ok := msg.(tea.KeyMsg); !ok {
			m.query.Textarea, tiCmd = m.query.Textarea.Update(msg)
		}
		if m.query.Textarea.Value() != oldVal {
			m.recalculateHeight()
		}
	}

	var resCmds []tea.Cmd
	_, isKey := msg.(tea.KeyMsg)
	if !isKey {
		for _, r := range m.results.Results {
			var cmd tea.Cmd
			r.Viewport, cmd = r.Viewport.Update(msg)
			resCmds = append(resCmds, cmd)
		}
	} else if m.focus == core.FocusResults && m.results.FocusedResultIndex >= 0 && m.results.FocusedResultIndex < len(m.results.Results) {
		var cmd tea.Cmd
		m.results.Results[m.results.FocusedResultIndex].Viewport, cmd = m.results.Results[m.results.FocusedResultIndex].Viewport.Update(msg)
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

func (m *Model) historyListHeight() int {
	modalHeight := m.height - 4
	if modalHeight < 10 {
		modalHeight = 10
	}
	chromeLines := 12
	listHeight := modalHeight - chromeLines
	if listHeight < 3 {
		listHeight = 3
	}
	return listHeight
}

func (m Model) View() tea.View {
	qHeader := query.RenderHeader(m.focus == core.FocusQuery, m.width, m.conn.Config.User, m.conn.Config.Host, m.conn.Config.Port, m.conn.GetCurrentDatabase(), m.vimState, m.conn.Config.Socket)

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Margin(0, 1)
	helpTextStr := "j/k:select · /:search · n/N:next/prev · Enter:detail · y:copy · v:edit · d:delete · R:rerun · +/-:size · Tab:focus"
	if m.focus == core.FocusQuery {
		if m.vimState.Mode == vim.NormalMode {
			helpTextStr = "gi:INSERT · gs:SELECT · gd:DELETE · gc:CREATE · gf:fields · gt:table · gw:where · Tab:focus"
		} else {
			helpTextStr = "Ctrl+K:autocomplete · Ctrl+L:clear query · Ctrl+Space:menu · Ctrl+R:history search · Ctrl+P/N:history · Ctrl+X s/i/d/c:SQL · Ctrl+X f/t/w:jump · Tab:focus"
		}
	}
	helpText := helpStyle.Render(helpTextStr)

	queryView := m.query.RenderArea(m.vimState)
	availableHeight := results.ComputeAvailableHeight(m.query.Textarea.Value(), m.query.Textarea.Placeholder, m.height)
	visibleResultsStr, visibleResultLines := results.RenderPanel(&m.results, m.focus, m.width, availableHeight)

	view := lipgloss.JoinVertical(lipgloss.Left,
		visibleResultsStr,
		qHeader,
		queryView,
		helpText,
	)

	if m.modals.RowDetail.Show {
		fg := modals.RenderRowDetail(&m.modals.RowDetail)
		bg := modals.EnsureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		view = compositeOverlay(fg, bg, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if !m.modals.RowDetail.Show && m.modals.CopyMenu.Show {
		fg := modals.RenderCopyMenu(&m.modals.CopyMenu)
		bg := modals.EnsureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		view = compositeOverlay(fg, bg, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if !m.modals.RowDetail.Show && !m.modals.CopyMenu.Show && m.modals.HistorySearch.Show {
		fg := modals.RenderHistorySearch(&m.modals.HistorySearch, m.query.History, m.query.HistoryTimestamps, m.width, m.height)
		bg := modals.EnsureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		view = compositeOverlay(fg, bg, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if !m.modals.RowDetail.Show && !m.modals.CopyMenu.Show && !m.modals.HistorySearch.Show && m.menu.Show {
		fg := menu.Render(&m.menu, m.buildMenuCommands())
		bg := modals.EnsureBackgroundSize(view, fg, m.width, m.height)
		fgWidth, fgHeight := lipgloss.Size(fg)
		view = compositeOverlay(fg, bg, m.width/2-fgWidth/2, m.height/2-fgHeight/2)
	}

	if !m.modals.RowDetail.Show && !m.modals.CopyMenu.Show && !m.modals.HistorySearch.Show && !m.menu.Show && m.suggestions.Show {
		fg := suggestions.Render(&m.suggestions)
		bg := modals.EnsureBackgroundSize(view, fg, m.width, m.height)
		_, fgHeight := lipgloss.Size(fg)
		xOff, yOff := suggestions.ComputeOffsets(visibleResultLines, fgHeight, &m.query.Textarea)
		view = compositeOverlay(fg, bg, xOff, yOff)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func compositeOverlay(fg, bg string, x, y int) string {
	if fg == "" {
		return bg
	}
	if bg == "" {
		return fg
	}

	bgWidth, bgHeight := lipgloss.Size(bg)
	fgWidth, fgHeight := lipgloss.Size(fg)
	x = min(max(x, 0), max(bgWidth-fgWidth, 0))
	y = min(max(y, 0), max(bgHeight-fgHeight, 0))

	canvas := lipgloss.NewCanvas(bgWidth, bgHeight)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(bg),
		lipgloss.NewLayer(fg).X(x).Y(y).Z(1),
	))
	return canvas.Render()
}
