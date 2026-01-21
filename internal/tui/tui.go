package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mycli-go/internal/completion"
	"github.com/mvgrimes/mycli-go/internal/formatter"
	"github.com/mvgrimes/mycli-go/internal/parser"
	"github.com/mvgrimes/mycli-go/internal/special"
	"github.com/mvgrimes/mycli-go/internal/vim"
)

var (
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3C3C3C")).
			Padding(0, 1)

	statusFocusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#626262")).
				Padding(0, 1).
				Bold(true)

	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF00")).
			Padding(0, 1).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Italic(true)

	headerFocusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#444444")).
				Bold(true)
)

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
	queryAreaHeight := len(strings.Split(m.textarea.Value(), "\n"))
	if queryAreaHeight < 3 {
		queryAreaHeight = 3
	}
	if m.textarea.Value() == "" {
		queryAreaHeight = len(strings.Split(m.textarea.Placeholder, "\n"))
		if queryAreaHeight < 3 {
			queryAreaHeight = 3
		}
	}

	// 1 (qHeader) + queryAreaHeight + 1 (helpText) + 1 (statusLine)
	overhead := 3 + queryAreaHeight
	m.viewport.Height = m.height - overhead - m.headerViewport.Height
	if m.viewport.Height < 0 {
		m.viewport.Height = 0
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case *completion.DBCache:
		m.completer.UpdateCache(msg)
		return m, nil

	case tea.KeyMsg:
		if m.showMenu {
			if m.menuType == MenuSaveFavorite {
				switch msg.Type {
				case tea.KeyEnter:
					m.SaveFavorite(m.favoriteInput)
					m.showMenu = false
					m.favoriteInput = ""
					return m, nil
				case tea.KeyEsc:
					m.showMenu = false
					m.favoriteInput = ""
					return m, nil
				case tea.KeyBackspace:
					if len(m.favoriteInput) > 0 {
						m.favoriteInput = m.favoriteInput[:len(m.favoriteInput)-1]
					}
					return m, nil
				case tea.KeyRunes:
					m.favoriteInput += string(msg.Runes)
					return m, nil
				}
				// Also handle Ctrl+Space/At to close
				if msg.String() == "ctrl+ " || msg.String() == "ctrl+space" || msg.Type == tea.KeyCtrlAt {
					m.showMenu = false
					m.favoriteInput = ""
					return m, nil
				}
				return m, nil
			}

			switch msg.String() {
			case "up", "k":
				if m.menuIndex > 0 {
					m.menuIndex--
				} else {
					m.menuIndex = len(m.GetCommands()) - 1
				}
			case "down", "j":
				if m.menuIndex < len(m.GetCommands())-1 {
					m.menuIndex++
				} else {
					m.menuIndex = 0
				}
			case "enter":
				oldType := m.menuType
				cmd := m.GetCommands()[m.menuIndex].Action(&m)
				if m.menuType == oldType {
					m.showMenu = false
				}
				return m, cmd
			case "esc", "ctrl+ ", "ctrl+space":
				m.showMenu = false
			}
			if msg.Type == tea.KeyCtrlAt {
				m.showMenu = false
			}
			return m, nil
		}

		if m.showSuggestions {
			switch msg.String() {
			case "up", "k", "shift+tab":
				if m.suggestionIndex > 0 {
					m.suggestionIndex--
				} else {
					m.suggestionIndex = len(m.suggestions) - 1
				}
			case "down", "j", "tab":
				if m.suggestionIndex < len(m.suggestions)-1 {
					m.suggestionIndex++
				} else {
					m.suggestionIndex = 0
				}
			case "enter":
				m.applySuggestion()
				m.showSuggestions = false
			case "esc", "ctrl+k":
				m.showSuggestions = false
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlK:
			m.showSuggestions = true
			m.suggestionIndex = 0
			m.updateSuggestions()
			return m, nil
		case tea.KeyCtrlP:
			if m.focus == FocusQuery {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
					m.recalculateHeight()
				}
				return m, nil
			}
		case tea.KeyCtrlN:
			if m.focus == FocusQuery {
				if m.historyIndex < len(m.history)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
					m.recalculateHeight()
				} else if m.historyIndex == len(m.history)-1 {
					m.historyIndex++
					m.textarea.Reset()
					m.recalculateHeight()
				}
				return m, nil
			}
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlD:
			if m.focus == FocusQuery && m.textarea.Value() == "" {
				return m, tea.Quit
			}
		case tea.KeyTab:
			if m.focus == FocusQuery {
				m.focus = FocusResults
				m.textarea.Blur()
				return m, nil
			} else {
				m.focus = FocusQuery
				return m, m.textarea.Focus()
			}

		case tea.KeyUp:
			if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
				m.textarea.CursorUp()
				return m, nil
			}
			if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
					m.recalculateHeight()
				}
				return m, nil
			}

		case tea.KeyDown:
			if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
				m.textarea.CursorDown()
				return m, nil
			}
			if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
				if m.historyIndex < len(m.history)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
					m.recalculateHeight()
				} else if m.historyIndex == len(m.history)-1 {
					m.historyIndex++
					m.textarea.Reset()
					m.recalculateHeight()
				}
				return m, nil
			}
		default:
			// Many terminals send NUL for Ctrl+Space
			if msg.Type == tea.KeyCtrlAt || (msg.Type == tea.KeySpace && msg.Alt) || msg.String() == "ctrl+ " || msg.String() == "ctrl+space" {
				m.showMenu = true
				m.menuIndex = 0
				m.menuType = MenuMain
				return m, nil
			}
		}

		if m.focus == FocusResults {
			switch msg.String() {
			case "q", "esc":
				m.focus = FocusQuery
				return m, m.textarea.Focus()
			case "j", "down":
				m.viewport.LineDown(1)
				return m, nil
			case "k", "up":
				m.viewport.LineUp(1)
				return m, nil
			case "h", "left":
				m.viewport.ScrollLeft(5)
				m.headerViewport.ScrollLeft(5)
				return m, nil
			case "l", "right":
				m.viewport.ScrollRight(5)
				m.headerViewport.ScrollRight(5)
				return m, nil
			case "G":
				m.viewport.GotoBottom()
				return m, nil
			}
		}

		if m.focus == FocusQuery {
			if m.vimState.Mode == vim.NormalMode {
				keyStr := msg.String()
				switch keyStr {
				case "i":
					m.vimState.Mode = vim.InsertMode
					return m, m.textarea.Focus()
				case "a":
					m.vimState.Mode = vim.InsertMode
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
					return m, m.textarea.Focus()
				case "A":
					m.vimState.Mode = vim.InsertMode
					m.textarea.CursorEnd()
					return m, m.textarea.Focus()
				case "I":
					m.vimState.Mode = vim.InsertMode
					m.textarea.CursorStart()
					return m, m.textarea.Focus()
				case "h":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
				case "l":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
				case "j":
					m.textarea.CursorDown()
				case "k":
					m.textarea.CursorUp()
				case "w":
					if m.vimPendingKey == "d" {
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
						m.vimPendingKey = ""
					} else if m.vimPendingKey == "c" {
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}, Alt: true})
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					} else {
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
					}
				case "b":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
				case "0", "^":
					m.textarea.CursorStart()
				case "$":
					m.textarea.CursorEnd()
				case "o":
					m.vimState.Mode = vim.InsertMode
					m.textarea.CursorEnd()
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
					return m, m.textarea.Focus()
				case "O":
					m.vimState.Mode = vim.InsertMode
					m.textarea.CursorStart()
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
					m.textarea.CursorUp()
					return m, m.textarea.Focus()
				case "D":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
				case "C":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
					m.vimState.Mode = vim.InsertMode
					return m, m.textarea.Focus()
				case "d":
					if m.vimPendingKey == "d" {
						// Delete current line
						m.textarea.CursorStart()
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
						m.vimPendingKey = ""
					} else {
						m.vimPendingKey = "d"
					}
					return m, nil
				case "c":
					if m.vimPendingKey == "c" {
						// Change current line
						m.textarea.CursorStart()
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					} else {
						m.vimPendingKey = "c"
					}
					return m, nil
				case "x":
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
				default:
					m.vimPendingKey = ""
				}
				if keyStr != "d" && keyStr != "c" {
					m.vimPendingKey = ""
				}
				return m, nil
			} else {
				// Insert Mode
				if msg.Type == tea.KeyEsc {
					m.vimState.Mode = vim.NormalMode
					return m, nil
				}
				if msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlJ {
					if msg.Alt || msg.Type == tea.KeyCtrlJ {
						m.textarea, tiCmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
						m.recalculateHeight()
						return m, tiCmd
					}
					query := m.textarea.Value()
					if strings.TrimSpace(query) != "" {
						return m.executeQuery(query)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)
		m.headerViewport.Width = msg.Width
		m.viewport.Width = msg.Width
		m.recalculateHeight()
		return m, nil
	}

	if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
		oldVal := m.textarea.Value()
		m.textarea, tiCmd = m.textarea.Update(msg)
		if m.textarea.Value() != oldVal {
			m.lastError = parser.Validate(m.textarea.Value())
			m.recalculateHeight()
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

	_, isKey := msg.(tea.KeyMsg)
	if !isKey || m.focus == FocusResults {
		m.headerViewport, _ = m.headerViewport.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
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
		m.headerViewport.SetContent("")
		m.headerViewport.Height = 0
		m.viewport.SetContent(m.specialOutput.String())
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
		m.headerViewport.SetContent("")
		m.headerViewport.Height = 0
		wrappedError := lipgloss.NewStyle().Width(m.width - 2).Render(fmt.Sprintf("Error: %v", err))
		m.viewport.SetContent(wrappedError)
		m.recalculateHeight()
		return m, nil
	} else {
		fullResult := formatter.FormatResult(result, format, m.config)
		lines := strings.Split(fullResult, "\n")

		if (format == formatter.FormatTable || format == formatter.FormatUnicode) && len(lines) > 3 {
			m.headerViewport.SetContent(strings.Join(lines[:3], "\n"))
			m.headerViewport.Height = 3
			m.viewport.SetContent(strings.Join(lines[3:], "\n"))
		} else {
			m.headerViewport.SetContent("")
			m.headerViewport.Height = 0
			m.viewport.SetContent(fullResult)
		}

		m.recalculateHeight()
		m.textarea.Reset()
		m.focus = FocusResults
		m.textarea.Blur()

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

func (m Model) renderHighlightedText(text string) string {
	lexer := lexers.Get("sql")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		return text
	}

	var b strings.Builder
	for _, tok := range iterator.Tokens() {
		style := m.getStyleForToken(tok.Type)
		b.WriteString(style.Render(tok.Value))
	}
	return b.String()
}

func (m Model) getStyleForToken(t chroma.TokenType) lipgloss.Style {
	s := lipgloss.NewStyle()
	switch t {
	case chroma.Keyword, chroma.KeywordReserved, chroma.KeywordType:
		return s.Foreground(lipgloss.Color("#00AAFF")).Bold(true)
	case chroma.String, chroma.StringSingle, chroma.StringDouble:
		return s.Foreground(lipgloss.Color("#00FF88"))
	case chroma.Number, chroma.NumberInteger, chroma.NumberFloat:
		return s.Foreground(lipgloss.Color("#FF8800"))
	case chroma.Comment, chroma.CommentSingle, chroma.CommentMultiline:
		return s.Foreground(lipgloss.Color("#666666")).Italic(true)
	case chroma.NameLabel, chroma.NameVariable:
		return s.Foreground(lipgloss.Color("#CC88FF"))
	case chroma.Operator, chroma.Punctuation:
		return s.Foreground(lipgloss.Color("#AAAAAA"))
	default:
		return s.Foreground(lipgloss.Color("#FFFFFF"))
	}
}

func (m Model) renderQueryArea() string {
	val := m.textarea.Value()
	isPlaceholder := false
	if val == "" {
		val = m.textarea.Placeholder
		isPlaceholder = true
	}

	lines := strings.Split(val, "\n")
	curLineIdx := m.textarea.Line()
	curColIdx := m.textarea.LineInfo().ColumnOffset

	var b strings.Builder
	for i, line := range lines {
		displayLine := ""
		if isPlaceholder {
			placeholderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
			if i == 0 && m.textarea.Focused() {
				// Show cursor at start of placeholder
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
				if m.vimState.Mode == vim.NormalMode {
					cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
				}
				runes := []rune(line)
				if len(runes) > 0 {
					displayLine = cursorStyle.Render(string(runes[0])) + placeholderStyle.Render(string(runes[1:]))
				} else {
					displayLine = cursorStyle.Render(" ")
				}
			} else {
				displayLine = placeholderStyle.Render(line)
			}
		} else if i == curLineIdx && m.textarea.Focused() {
			runes := []rune(line)
			before := ""
			cursorChar := " "
			after := ""

			if curColIdx < len(runes) {
				before = string(runes[:curColIdx])
				cursorChar = string(runes[curColIdx])
				after = string(runes[curColIdx+1:])
			} else {
				before = line
			}

			hBefore := m.renderHighlightedText(before)
			hAfter := m.renderHighlightedText(after)

			cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
			if m.vimState.Mode == vim.NormalMode {
				cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("#CCCCCC")).Foreground(lipgloss.Color("#000000"))
			}

			displayLine = hBefore + cursorStyle.Render(cursorChar) + hAfter
		} else {
			displayLine = m.renderHighlightedText(line)
		}

		b.WriteString(fmt.Sprintf("%2d | %s\n", i+1, displayLine))
	}

	if m.lastError != nil && !isPlaceholder {
		padding := m.lastError.Col + 5
		b.WriteString(strings.Repeat(" ", padding) + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("^ "+m.lastError.Message) + "\n")
	}

	return b.String()
}

func (m Model) View() string {
	m.UpdateCursorStyle()

	user := m.conn.Config.User
	host := m.conn.Config.Host
	port := m.conn.Config.Port
	database := m.conn.GetCurrentDatabase()

	bg := lipgloss.Color("#3C3C3C")
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(bg)
	if user == "root" {
		userStyle = userStyle.Foreground(lipgloss.Color("#FF5555")).Bold(true)
	}

	hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(bg)
	isLocal := host == "localhost" || host == "127.0.0.1" || m.conn.Config.Socket != ""
	if !isLocal {
		hostStyle = hostStyle.Foreground(lipgloss.Color("#FFA500"))
	}

	atStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Background(bg)
	restStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(bg)

	statusLineStyle := lipgloss.NewStyle().
		Background(bg).
		Width(m.width)

	statusStr := lipgloss.JoinHorizontal(lipgloss.Left,
		" ",
		userStyle.Render(user),
		atStyle.Render("@"),
		hostStyle.Render(host),
		restStyle.Render(fmt.Sprintf(":%d/%s ", port, database)),
	)

	mode := " INSERT "
	if m.vimState.Mode == vim.NormalMode {
		mode = " NORMAL "
	}

	renderedStatus := statusStr // background will be handled by statusLineStyle
	renderedMode := modeStyle.Render(mode)

	// Calculate filler to push mode to the right
	// We need to account for the width of the status string (with its styles) and mode
	fillerWidth := m.width - lipgloss.Width(renderedStatus) - lipgloss.Width(renderedMode)
	if fillerWidth < 0 {
		fillerWidth = 0
	}

	statusLine := statusLineStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left,
		renderedStatus,
		strings.Repeat(" ", fillerWidth),
		renderedMode,
	))

	queryHeaderStr := " [QUERY] "
	if m.focus == FocusResults {
		queryHeaderStr = " [RESULT] "
	}

	qHeader := headerFocusStyle.Render(queryHeaderStr)

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Margin(0, 1)
	helpText := helpStyle.Render("Ctrl+K: autocomplete • Ctrl+Space: menu • Ctrl+P/N: history • Tab: switch focus")

	queryView := m.renderQueryArea()

	view := lipgloss.JoinVertical(lipgloss.Left,
		m.headerViewport.View(),
		m.viewport.View(),
		qHeader,
		queryView,
		helpText,
		statusLine,
	)

	if m.showSuggestions {
		overlay := m.renderSuggestions()
		h_s := lipgloss.Height(overlay)

		// Create a temporary shrunken viewport
		origHeight := m.viewport.Height
		m.viewport.Height -= h_s
		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}
		shrunkenViewport := m.viewport.View()
		m.viewport.Height = origHeight

		// Align overlay with cursor column
		curColIdx := m.textarea.LineInfo().ColumnOffset
		leftMargin := 5 + curColIdx
		w_s := lipgloss.Width(overlay)
		if leftMargin+w_s > m.width {
			leftMargin = m.width - w_s
		}
		if leftMargin < 0 {
			leftMargin = 0
		}
		alignedOverlay := lipgloss.NewStyle().PaddingLeft(leftMargin).Render(overlay)

		return lipgloss.JoinVertical(lipgloss.Left,
			m.headerViewport.View(),
			shrunkenViewport,
			alignedOverlay,
			qHeader,
			queryView,
			statusLine,
		)
	}

	if m.showMenu {
		overlay := m.renderMenu()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	return view
}

func (m Model) renderSuggestions() string {
	if len(m.suggestions) == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1)

	activeStyle := style.Copy().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00AAFF")).
		Bold(true)

	var b strings.Builder
	start := 0
	if m.suggestionIndex > 5 {
		start = m.suggestionIndex - 5
	}

	for i := start; i < len(m.suggestions) && i < start+10; i++ {
		s := m.suggestions[i]
		line := fmt.Sprintf("%-20s %s", s.Text, s.Description)
		if i == m.suggestionIndex {
			b.WriteString(activeStyle.Render(line) + "\n")
		} else {
			b.WriteString(style.Render(line) + "\n")
		}
	}
	if len(m.suggestions) > start+10 {
		b.WriteString(style.Render("...") + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m Model) renderMenu() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1A1A1A")).
		Padding(0, 1)

	activeStyle := style.Copy().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00AAFF")).
		Bold(true)

	var b strings.Builder

	title := " COMMANDS "
	switch m.menuType {
	case MenuSaveFavorite:
		title = " SAVE FAVORITE "
	case MenuRunFavorite:
		title = " RUN FAVORITE "
	}

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Render(title))
	b.WriteString("\n\n")

	if m.menuType == MenuSaveFavorite {
		b.WriteString("Enter name for favorite:\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Width(30).
			Render(m.favoriteInput + "_"))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Enter: Save • Esc: Cancel"))
	} else {
		commands := m.GetCommands()
		for i, cmd := range commands {
			line := cmd.Label
			if i == m.menuIndex {
				b.WriteString(activeStyle.Render(line) + "\n")
			} else {
				b.WriteString(style.Render(line) + "\n")
			}
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(1, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(b.String())
}
