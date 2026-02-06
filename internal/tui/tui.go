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
	availableHeight := m.height - overhead
	if availableHeight < 0 {
		availableHeight = 0
	}

	if len(m.results) == 0 {
		return
	}

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
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
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
					m.menuFilter = ""
					return m, nil
				case tea.KeyEsc:
					m.showMenu = false
					m.favoriteInput = ""
					m.menuFilter = ""
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
					m.menuFilter = ""
					return m, nil
				}
				return m, nil
			}

			// Filterable menus (MenuMain, MenuRunFavorite)
			switch msg.String() {
			case "up", "k":
				cmds := m.filteredMenuCommands()
				if len(cmds) > 0 {
					if m.menuIndex > 0 {
						m.menuIndex--
					} else {
						m.menuIndex = len(cmds) - 1
					}
				}
			case "down", "j":
				cmds := m.filteredMenuCommands()
				if len(cmds) > 0 {
					if m.menuIndex < len(cmds)-1 {
						m.menuIndex++
					} else {
						m.menuIndex = 0
					}
				}
			case "enter":
				oldType := m.menuType
				cmds := m.filteredMenuCommands()
				if len(cmds) == 0 {
					return m, nil
				}
				if m.menuIndex < 0 || m.menuIndex >= len(cmds) {
					m.menuIndex = 0
				}
				cmd := cmds[m.menuIndex].Action(&m)
				if m.menuType == oldType {
					m.showMenu = false
					m.menuFilter = ""
				}
				return m, cmd
			case "esc", "ctrl+ ", "ctrl+space":
				m.showMenu = false
				m.menuFilter = ""
				return m, nil
			}
			if msg.Type == tea.KeyCtrlAt {
				m.showMenu = false
				m.menuFilter = ""
				return m, nil
			}
			// Live filtering input
			if msg.Type == tea.KeyBackspace {
				if len(m.menuFilter) > 0 {
					m.menuFilter = m.menuFilter[:len(m.menuFilter)-1]
				}
				m.menuIndex = 0
				return m, nil
			}
			if msg.Type == tea.KeyRunes {
				m.menuFilter += string(msg.Runes)
				m.menuIndex = 0
				return m, nil
			}
			return m, nil
		}

		if m.showSuggestions {
			consumed := false
			switch msg.String() {
			case "up", "shift+tab":
				if len(m.suggestions) > 0 {
					if m.suggestionIndex > 0 {
						m.suggestionIndex--
					} else {
						m.suggestionIndex = len(m.suggestions) - 1
					}
				}
				consumed = true
			case "down":
				if len(m.suggestions) > 0 {
					if m.suggestionIndex < len(m.suggestions)-1 {
						if m.suggestionIndex < 0 {
							m.suggestionIndex = 0
						} else {
							m.suggestionIndex++
						}
					} else {
						m.suggestionIndex = 0
					}
				}
				consumed = true
			case "tab":
				if len(m.suggestions) > 0 {
					if m.suggestionIndex < 0 {
						m.suggestionIndex = 0
					} else if m.suggestionIndex < len(m.suggestions)-1 {
						m.suggestionIndex++
					} else {
						m.suggestionIndex = 0
					}
				}
				consumed = true
			case "enter":
				if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
					m.applySuggestion()
					m.showSuggestions = false
					consumed = true
				}
			case " ":
				if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
					m.applySuggestion()
					m.showSuggestions = false
					// Insert a trailing space after applying the suggestion
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeySpace})
					consumed = true
				}
			case ";":
				if m.suggestionIndex >= 0 && m.suggestionIndex < len(m.suggestions) {
					m.applySuggestion()
					m.showSuggestions = false
					m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{';'}})
					consumed = true
				}
			case "esc":
				if m.suggestionIndex >= 0 {
					// First ESC: unselect
					m.suggestionIndex = -1
				} else {
					// Second ESC: close overlay
					m.showSuggestions = false
				}
				consumed = true
			default:
				// While suggestions are open, allow typing to edit the query.
				// Recompute suggestions on every edit (Insert mode only).
				if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
					oldVal := m.textarea.Value()
					var tiCmd tea.Cmd
					m.textarea, tiCmd = m.textarea.Update(msg)
					if m.textarea.Value() != oldVal {
						m.recalculateHeight()
						m.updateSuggestions()
						if m.shouldOpenSuggestionsOnEdit() && len(m.suggestions) > 0 {
							if m.suggestionIndex >= len(m.suggestions) {
								m.suggestionIndex = -1
							}
							m.showSuggestions = true
						} else {
							m.showSuggestions = false
						}
					}
					return m, tiCmd
				}
			}
			if consumed {
				return m, nil
			}
			// If not consumed, let other handlers process (e.g., Enter without selection)
		}

		switch msg.Type {
		case tea.KeyCtrlK:
			m.updateSuggestions()
			if len(m.suggestions) > 0 {
				m.showSuggestions = true
				m.suggestionIndex = -1
			} else {
				m.showSuggestions = false
			}
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
				if len(m.results) > 0 {
					m.focus = FocusResults
					m.focusedResult = len(m.results) - 1
					m.textarea.Blur()
				}
				return m, nil
			} else {
				if m.focusedResult > 0 {
					m.focusedResult--
				} else {
					m.focus = FocusQuery
					m.focusedResult = -1
					return m, m.textarea.Focus()
				}
				return m, nil
			}

		case tea.KeyShiftTab:
			if m.focus == FocusQuery {
				if len(m.results) > 0 {
					m.focus = FocusResults
					m.focusedResult = 0
					m.textarea.Blur()
				}
				return m, nil
			} else {
				if m.focusedResult < len(m.results)-1 {
					m.focusedResult++
				} else {
					m.focus = FocusQuery
					m.focusedResult = -1
					return m, m.textarea.Focus()
				}
				return m, nil
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
				m.menuFilter = ""
				return m, nil
			}
		}

		if m.focus == FocusResults && m.focusedResult >= 0 {
			res := m.results[m.focusedResult]
			switch msg.String() {
			case "q", "esc":
				m.focus = FocusQuery
				m.focusedResult = -1
				return m, m.textarea.Focus()
			case "j", "down":
				res.Viewport.LineDown(1)
				return m, nil
			case "k", "up":
				res.Viewport.LineUp(1)
				return m, nil
			case "h", "left":
				res.Viewport.ScrollLeft(5)
				res.XOffset -= 5
				if res.XOffset < 0 {
					res.XOffset = 0
				}
				return m, nil
			case "l", "right":
				res.Viewport.ScrollRight(5)
				res.XOffset += 5
				// Clamp XOffset to not exceed content width minus viewport width
				contentWidth := maxContentWidth(res.Formatted)
				maxOffset := contentWidth - m.width
				if maxOffset < 0 {
					maxOffset = 0
				}
				if res.XOffset > maxOffset {
					res.XOffset = maxOffset
				}
				return m, nil
			case "G":
				res.Viewport.GotoBottom()
				return m, nil
			case "e":
				res.Expanded = true
				m.recalculateHeight()
				return m, nil
			case "c":
				res.Expanded = false
				m.recalculateHeight()
				return m, nil
			case "+":
				res.DisplaySize += 2
				m.recalculateHeight()
				return m, nil
			case "-":
				if res.DisplaySize > 2 {
					res.DisplaySize -= 2
				}
				m.recalculateHeight()
				return m, nil
			case "d":
				m.results = append(m.results[:m.focusedResult], m.results[m.focusedResult+1:]...)
				if len(m.results) == 0 {
					m.focus = FocusQuery
					m.focusedResult = -1
					return m, m.textarea.Focus()
				}
				if m.focusedResult >= len(m.results) {
					m.focusedResult = len(m.results) - 1
				}
				m.recalculateHeight()
				return m, nil
			case "r":
				m.textarea.SetValue(res.Query)
				m.focus = FocusQuery
				m.focusedResult = -1
				return m, m.textarea.Focus()
			case "R":
				// Re-run the query and update the current result
				newResult, err := m.conn.ExecuteQuery(res.Query)
				if err == nil {
					res.DbResult = newResult
					res.Formatted = formatter.FormatResult(newResult, res.Format, m.config)
					res.FormattedHeader, res.FormattedData = splitTableHeaderAndData(res.Formatted, res.Format)
					res.Viewport.SetContent(res.FormattedData)
					res.XOffset = 0
					res.Viewport.GotoTop()
					res.Timestamp = time.Now()
					res.Duration = newResult.Duration
				}
				return m, nil
			}
		}

		if m.focus == FocusQuery {
			if m.vimState.Mode == vim.NormalMode {
				keyStr := msg.String()

				// Handle f/F pending - find character
				if m.vimPendingKey == "f" || m.vimPendingKey == "F" {
					if len(keyStr) == 1 {
						targetChar := rune(keyStr[0])
						m.findCharInLine(targetChar, m.vimPendingKey == "f")
					}
					m.vimPendingKey = ""
					return m, nil
				}

				// Handle ci/di pending - inner text object
				if m.vimPendingKey == "ci" || m.vimPendingKey == "di" {
					if keyStr == "w" {
						// ciw/diw - change/delete inner word
						m.deleteInnerWord()
						if m.vimPendingKey == "ci" {
							m.vimState.Mode = vim.InsertMode
							m.vimPendingKey = ""
							return m, m.textarea.Focus()
						}
					}
					m.vimPendingKey = ""
					return m, nil
				}

				// Handle g pending - SQL shortcuts
				if m.vimPendingKey == "g" {
					switch keyStr {
					case "i":
						m.insertSQLTemplate(sqlTemplateInsert, sqlOffsetInsert)
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					case "s":
						m.insertSQLTemplate(sqlTemplateSelect, sqlOffsetSelect)
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					case "d":
						m.insertSQLTemplate(sqlTemplateDelete, sqlOffsetDelete)
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					case "c":
						m.insertSQLTemplate(sqlTemplateCreate, sqlOffsetCreate)
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					case "f":
						m.jumpToFieldsPosition()
						m.vimPendingKey = ""
						return m, nil
					case "t":
						m.jumpToTablePosition()
						m.vimPendingKey = ""
						return m, nil
					case "w":
						m.jumpToWherePosition()
						m.vimPendingKey = ""
						return m, nil
					}
					m.vimPendingKey = ""
					return m, nil
				}

				switch keyStr {
				case "i":
					if m.vimPendingKey == "c" || m.vimPendingKey == "d" {
						m.vimPendingKey = m.vimPendingKey + "i"
						return m, nil
					}
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
					if m.vimPendingKey == "d" {
						// d0 - delete to start of line
						m.deleteToLineStart()
						m.vimPendingKey = ""
						return m, nil
					} else if m.vimPendingKey == "c" {
						// c0 - change to start of line
						m.deleteToLineStart()
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					}
					m.textarea.CursorStart()
				case "$":
					if m.vimPendingKey == "d" {
						// d$ - delete to end of line
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
						m.vimPendingKey = ""
						return m, nil
					} else if m.vimPendingKey == "c" {
						// c$ - change to end of line
						m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
						m.vimState.Mode = vim.InsertMode
						m.vimPendingKey = ""
						return m, m.textarea.Focus()
					}
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
				case "f":
					m.vimPendingKey = "f"
					return m, nil
				case "F":
					m.vimPendingKey = "F"
					return m, nil
				case "g":
					m.vimPendingKey = "g"
					return m, nil
				default:
					m.vimPendingKey = ""
				}
				if keyStr != "d" && keyStr != "c" && keyStr != "f" && keyStr != "F" && keyStr != "i" && keyStr != "g" {
					m.vimPendingKey = ""
				}
				return m, nil
			} else {
				// Insert Mode

				// Handle ctrl+x pending - SQL shortcuts in insert mode
				if m.vimPendingKey == "ctrl+x" {
					switch msg.String() {
					case "i":
						m.insertSQLTemplate(sqlTemplateInsert, sqlOffsetInsert)
					case "s":
						m.insertSQLTemplate(sqlTemplateSelect, sqlOffsetSelect)
					case "d":
						m.insertSQLTemplate(sqlTemplateDelete, sqlOffsetDelete)
					case "c":
						m.insertSQLTemplate(sqlTemplateCreate, sqlOffsetCreate)
					case "f":
						m.jumpToFieldsPosition()
					case "t":
						m.jumpToTablePosition()
					case "w":
						m.jumpToWherePosition()
					}
					m.vimPendingKey = ""
					return m, nil
				}

				// Set ctrl+x pending state
				if msg.Type == tea.KeyCtrlX {
					m.vimPendingKey = "ctrl+x"
					return m, nil
				}

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
			// Always-on suggestions: recompute and open/close as needed
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
	helpTextStr := "e:expand c:collapse +/-:size d:delete r:rerun R:refresh • Tab: focus"
	if m.focus == FocusQuery {
		helpTextStr = "Ctrl+K: autocomplete • Ctrl+Space: menu • Ctrl+P/N: history • Tab: focus"
	}
	helpText := helpStyle.Render(helpTextStr)

	queryView := m.renderQueryArea()

	var resultsView []string
	resultsLines := 0
	for i, r := range m.results {
		focused := (m.focus == FocusResults && m.focusedResult == i)
		resultsView = append(resultsView, m.renderResultHeader(r, focused))
		resultsLines += 1
		if r.Expanded {
			// Render pinned header if available (for table formats)
			if r.FormattedHeader != "" {
				header := applyHorizontalOffset(r.FormattedHeader, r.XOffset, m.width)
				resultsView = append(resultsView, header)
				resultsLines += strings.Count(r.FormattedHeader, "\n") + 1
			}
			resultsView = append(resultsView, r.Viewport.View())
			resultsLines += r.Viewport.Height
		}
	}

	view := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, resultsView...),
		qHeader,
		queryView,
		helpText,
	)

	if m.showSuggestions {
		// Overlay suggestions near the text cursor
		bg := view
		fg := m.renderSuggestions()
		fgWidth, fgHeight := lipgloss.Size(fg)
		xOff, yOff := m.computeSuggestionOffsets(resultsLines, fgHeight)

		// Ensure background is tall enough for suggestions placed below cursor
		bgWidth, bgHeight := lipgloss.Size(bg)
		requiredHeight := yOff + fgHeight
		if requiredHeight > bgHeight {
			padding := strings.Repeat("\n", requiredHeight-bgHeight)
			bg = bg + padding
		}

		// Ensure background is wide enough for suggestions
		requiredWidth := xOff + fgWidth
		if requiredWidth > bgWidth {
			// Pad each line to required width
			lines := strings.Split(bg, "\n")
			for i, line := range lines {
				lineWidth := lipgloss.Width(line)
				if lineWidth < requiredWidth {
					lines[i] = line + strings.Repeat(" ", requiredWidth-lineWidth)
				}
			}
			bg = strings.Join(lines, "\n")
		}

		return overlay.Composite(fg, bg, overlay.Left, overlay.Top, xOff, yOff)
	}

	if m.showMenu {
		// Center the command menu as a modal over the existing view
		bg := view
		fg := m.renderMenu()
		return overlay.Composite(fg, bg, overlay.Center, overlay.Center, 0, 0)
	}

	return view
}
