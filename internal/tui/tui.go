package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mvgrimes/mycli-go/internal/formatter"
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
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showMenu {
			switch msg.String() {
			case "up", "k":
				if m.menuIndex > 0 {
					m.menuIndex--
				}
			case "down", "j":
				if m.menuIndex < len(m.GetCommands())-1 {
					m.menuIndex++
				}
			case "enter":
				cmd := m.GetCommands()[m.menuIndex].Action(&m)
				m.showMenu = false
				return m, cmd
			case "esc", "ctrl+k":
				m.showMenu = false
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlK:
			m.showMenu = true
			m.menuIndex = 0
			return m, nil
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

		case tea.KeyCtrlP:
			if m.focus == FocusQuery {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
				}
				return m, nil
			}

		case tea.KeyCtrlN:
			if m.focus == FocusQuery {
				if m.historyIndex < len(m.history)-1 {
					m.historyIndex++
					m.textarea.SetValue(m.history[m.historyIndex])
					m.textarea.CursorEnd()
				} else if m.historyIndex == len(m.history)-1 {
					m.historyIndex++
					m.textarea.Reset()
				}
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
						// If there's a line below, delete the newline character too
						// But textarea' Ctrl+K might not delete the line itself if it's empty?
						// Actually, if we are at start of line, Ctrl+K deletes the whole line content.
						// We might need to delete the line break if we want true 'dd'.
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
					// Don't blur, just stay focused so cursor shows.
					// We'll handle keys above.
					return m, nil
				}
				if msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlJ {
					if msg.Alt || msg.Type == tea.KeyCtrlJ {
						m.textarea, tiCmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
		m.viewport.Height = m.height - 6 - m.headerViewport.Height
		return m, nil
	}

	if m.focus == FocusQuery && m.vimState.Mode == vim.InsertMode {
		m.textarea, tiCmd = m.textarea.Update(msg)
	} else if m.focus == FocusQuery && m.vimState.Mode == vim.NormalMode {
		// Update textarea with non-key messages even in normal mode to keep cursor blinking
		if _, ok := msg.(tea.KeyMsg); !ok {
			m.textarea, tiCmd = m.textarea.Update(msg)
		}
	}

	// Always update viewports for non-key messages or if focused
	_, isKey := msg.(tea.KeyMsg)
	if !isKey || m.focus == FocusResults {
		m.headerViewport, _ = m.headerViewport.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) executeQuery(query string) (Model, tea.Cmd) {
	trimmedQuery := strings.TrimSpace(query)
	lowerQuery := strings.ToLower(trimmedQuery)
	if lowerQuery == "exit" || lowerQuery == "quit" {
		return m, tea.Quit
	}

	// Handle special commands
	m.specialOutput.Reset()
	if special.Handle(trimmedQuery, &m) {
		m.headerViewport.SetContent("")
		m.headerViewport.Height = 0
		m.viewport.SetContent(m.specialOutput.String())
		m.viewport.Height = m.height - 6 - m.headerViewport.Height
		m.textarea.Reset()
		// Leave focus in query window for special commands
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

	// Save to history
	m.saveToHistory(query)
	m.lastQuery = query

	result, err := m.conn.ExecuteQuery(query)
	if err != nil {
		m.headerViewport.SetContent("")
		m.headerViewport.Height = 0
		// Wrap error message
		wrappedError := lipgloss.NewStyle().Width(m.width - 2).Render(fmt.Sprintf("Error: %v", err))
		m.viewport.SetContent(wrappedError)
		m.viewport.Height = m.height - 6 - m.headerViewport.Height
		// Keep focus in query and stay in insert mode
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

		m.viewport.Height = m.height - 6 - m.headerViewport.Height
		m.textarea.Reset()
		m.focus = FocusResults
		m.textarea.Blur()
	}

	return m, nil
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

func (m Model) View() string {
	m.UpdateCursorStyle()
	status := fmt.Sprintf(" %s@%s:%d/%s ",
		m.conn.Config.User, m.conn.Config.Host, m.conn.Config.Port, m.conn.GetCurrentDatabase())

	mode := " INSERT "
	if m.vimState.Mode == vim.NormalMode {
		mode = " NORMAL "
	}

	statusBarStyle := statusStyle
	if m.focus == FocusResults {
		statusBarStyle = statusFocusStyle
	}

	tableHeaderStr := " [TABLE] "
	if m.focus == FocusResults {
		tableHeaderStr = " [TABLE] "
	} else {
		tableHeaderStr = " table "
	}

	queryHeaderStr := " [QUERY] "
	if m.focus == FocusQuery {
		queryHeaderStr = " [QUERY] "
	} else {
		queryHeaderStr = " query "
	}

	tHeader := headerStyle.Render(tableHeaderStr)
	if m.focus == FocusResults {
		tHeader = headerFocusStyle.Render(tableHeaderStr)
	}

	qHeader := headerStyle.Render(queryHeaderStr)
	if m.focus == FocusQuery {
		qHeader = headerFocusStyle.Render(queryHeaderStr)
	}

	statusLine := lipgloss.JoinHorizontal(lipgloss.Bottom,
		statusBarStyle.Render(status),
		lipgloss.NewStyle().Width(m.width-lipgloss.Width(status)-lipgloss.Width(mode)).Render(""),
		modeStyle.Render(mode),
	)

	view := lipgloss.JoinVertical(lipgloss.Left,
		tHeader,
		m.headerViewport.View(),
		m.viewport.View(),
		qHeader,
		m.textarea.View(),
		statusLine,
	)

	if m.showMenu {
		overlay := m.renderMenu()
		// Overlay the menu on top of the view
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	return view
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
	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00AAFF")).
		Padding(0, 1).
		Render(" COMMANDS "))
	b.WriteString("\n\n")

	commands := m.GetCommands()
	for i, cmd := range commands {
		line := cmd.Label
		if i == m.menuIndex {
			b.WriteString(activeStyle.Render(line) + "\n")
		} else {
			b.WriteString(style.Render(line) + "\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00AAFF")).
		Padding(1, 1).
		Background(lipgloss.Color("#1A1A1A")).
		Render(b.String())
}
