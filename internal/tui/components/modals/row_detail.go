package modals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/tui/core"
	"github.com/mvgrimes/mytui/internal/tui/styles"
)

// EnsureBackgroundSize pads the background to ensure it's large enough for the overlay.
func EnsureBackgroundSize(bg, fg string, minWidth, minHeight int) string {
	fgWidth, fgHeight := lipgloss.Size(fg)
	bgWidth, bgHeight := lipgloss.Size(bg)

	// Use terminal dimensions as minimum if provided
	requiredWidth := max(fgWidth, minWidth)
	requiredHeight := max(fgHeight, minHeight)

	// Ensure background is tall enough
	if requiredHeight > bgHeight {
		padding := strings.Repeat("\n", requiredHeight-bgHeight)
		bg = bg + padding
	}

	// Ensure background is wide enough
	if requiredWidth > bgWidth {
		lines := strings.Split(bg, "\n")
		for i, line := range lines {
			lineWidth := lipgloss.Width(line)
			if lineWidth < requiredWidth {
				lines[i] = line + strings.Repeat(" ", requiredWidth-lineWidth)
			}
		}
		bg = strings.Join(lines, "\n")
	}

	return bg
}

// OpenRowDetail initializes and shows the row detail modal.
func OpenRowDetail(m *RowDetailModel, res *core.Result, width, height int) {
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return
	}

	row := res.DbResult.Rows[res.SelectedRow]
	content := formatRowVertical(res.DbResult.Headers, core.InterfaceSliceToStrings(row))

	// Create viewport for scrolling
	modalWidth := min(width-10, 100)
	modalHeight := min(height-10, 40)

	m.Viewport = viewport.New(modalWidth, modalHeight)
	m.Viewport.SetContent(content)
	m.Show = true
}

// RenderRowDetail renders the row detail modal overlay.
func RenderRowDetail(m *RowDetailModel) string {
	title := styles.RowDetailTitleStyle.Render("Row Details")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("j/k:scroll  PgUp/PgDn:page  h/l:scroll x  0/$:home/end x  q/Esc:close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.Viewport.View(),
		"",
		help,
	)

	return styles.RowDetailBorderStyle.Render(content)
}

// CopyRowToClipboard copies the selected row to clipboard in the specified format.
func CopyRowToClipboard(res *core.Result, format core.CopyFormat, cfg *config.Config) string {
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return ""
	}

	row := core.InterfaceSliceToStrings(res.DbResult.Rows[res.SelectedRow])
	headers := res.DbResult.Headers

	var content string
	switch format {
	case core.CopyFormatCSV:
		content = formatRowCSV(headers, row)
	case core.CopyFormatTSV:
		content = formatRowTSV(headers, row)
	case core.CopyFormatJSON:
		content = formatRowJSON(headers, row)
	case core.CopyFormatVertical:
		content = formatRowVerticalText(headers, row)
	case core.CopyFormatASCIITable:
		var buf bytes.Buffer
		formatter.RenderResult(res.DbResult, &buf, formatter.FormatTable, cfg)
		content = buf.String()
	case core.CopyFormatUnicodeTable:
		var buf bytes.Buffer
		formatter.RenderResult(res.DbResult, &buf, formatter.FormatUnicode, cfg)
		content = buf.String()
	}

	clipboard.WriteAll(content)
	return formatName(format)
}

// RenderCopyMenu renders the copy format selection menu.
func RenderCopyMenu(m *CopyMenuModel) string {
	title := styles.CopyMenuTitleStyle.Render("Copy Row As")

	options := []string{
		"CSV          - Comma-separated values",
		"TSV          - Tab-separated values",
		"JSON         - JSON object",
		"Vertical     - Column: value format",
		"ASCII Table  - Full result as ASCII table",
		"Unicode Table - Full result as unicode table",
	}

	var items []string
	for i, opt := range options {
		if i == m.Index {
			items = append(items, styles.CopyMenuSelectedStyle.Render("> "+opt))
		} else {
			items = append(items, styles.CopyMenuItemStyle.Render("  "+opt))
		}
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("j/k: select  Enter: copy  Esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(items, "\n"),
		"",
		help,
	)

	return styles.CopyMenuBorderStyle.Render(content)
}

// OpenRowInEditor opens the selected row in an external editor.
func OpenRowInEditor(res *core.Result) tea.Cmd {
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return nil
	}

	row := core.InterfaceSliceToStrings(res.DbResult.Rows[res.SelectedRow])
	content := formatRowVerticalText(res.DbResult.Headers, row)

	tmpFile, err := os.CreateTemp("", "mytui-row-*.txt")
	if err != nil {
		return nil
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tmpPath := tmpFile.Name()
	c := exec.Command(editor, tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		os.Remove(tmpPath)
		return editorFinishedMsg{err}
	})
}

func formatName(f core.CopyFormat) string {
	switch f {
	case core.CopyFormatCSV:
		return "CSV"
	case core.CopyFormatTSV:
		return "TSV"
	case core.CopyFormatJSON:
		return "JSON"
	case core.CopyFormatVertical:
		return "Vertical"
	case core.CopyFormatASCIITable:
		return "ASCII Table"
	case core.CopyFormatUnicodeTable:
		return "Unicode Table"
	}
	return "Unknown"
}

// formatRowVertical formats a row in vertical format (column: value per line)
func formatRowVertical(headers []string, row []string) string {
	var sb strings.Builder

	// Find the max header length for alignment
	maxLen := 0
	for _, h := range headers {
		if len(h) > maxLen {
			maxLen = len(h)
		}
	}

	for i, header := range headers {
		value := ""
		if i < len(row) {
			value = row[i]
		}

		label := styles.RowDetailLabelStyle.Render(fmt.Sprintf("%*s", maxLen, header))
		val := styles.RowDetailValueStyle.Render(value)
		sb.WriteString(fmt.Sprintf("%s: %s\n", label, val))
	}

	return sb.String()
}

// formatRowVerticalText formats a row as plain text for editor
func formatRowVerticalText(headers []string, row []string) string {
	var sb strings.Builder

	maxLen := 0
	for _, h := range headers {
		if len(h) > maxLen {
			maxLen = len(h)
		}
	}

	for i, header := range headers {
		value := ""
		if i < len(row) {
			value = row[i]
		}
		sb.WriteString(fmt.Sprintf("%*s: %s\n", maxLen, header, value))
	}

	return sb.String()
}

// formatRowCSV formats a row as CSV with header
func formatRowCSV(headers []string, row []string) string {
	escapeCSV := func(s string) string {
		if strings.ContainsAny(s, ",\"\n") {
			return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
		}
		return s
	}

	var headerParts, valueParts []string
	for _, h := range headers {
		headerParts = append(headerParts, escapeCSV(h))
	}
	for _, v := range row {
		valueParts = append(valueParts, escapeCSV(v))
	}

	return strings.Join(headerParts, ",") + "\n" + strings.Join(valueParts, ",")
}

// formatRowTSV formats a row as TSV with header
func formatRowTSV(headers []string, row []string) string {
	escapeTSV := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\t", "\\t"), "\n", "\\n")
	}

	var headerParts, valueParts []string
	for _, h := range headers {
		headerParts = append(headerParts, escapeTSV(h))
	}
	for _, v := range row {
		valueParts = append(valueParts, escapeTSV(v))
	}

	return strings.Join(headerParts, "\t") + "\n" + strings.Join(valueParts, "\t")
}

// formatRowJSON formats a row as a JSON object
func formatRowJSON(headers []string, row []string) string {
	obj := make(map[string]string)
	for i, h := range headers {
		if i < len(row) {
			obj[h] = row[i]
		} else {
			obj[h] = ""
		}
	}

	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// highlightSelectedRow applies a highlight to the selected row in the formatted data.
// It handles ANSI-colored content properly.
func highlightSelectedRow(data string, selectedRow int, width int) string {
	if selectedRow < 0 {
		return data
	}

	lines := strings.Split(data, "\n")
	if selectedRow >= len(lines) {
		return data
	}

	// Strip ANSI codes to get raw text, then re-apply with background
	line := lines[selectedRow]
	// Preserve the original line content but add background highlight
	highlighted := styles.SelectedRowStyle.Width(width).Render(ansi.Strip(line))
	lines[selectedRow] = highlighted

	return strings.Join(lines, "\n")
}

type editorFinishedMsg struct {
	err error
}
