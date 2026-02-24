package tui

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
	"github.com/mvgrimes/mytui/internal/formatter"
)

// ensureBackgroundSize pads the background to ensure it's large enough for the overlay
func ensureBackgroundSize(bg, fg string, minWidth, minHeight int) string {
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

// interfaceSliceToStrings converts a slice of interface{} to a slice of strings
func interfaceSliceToStrings(row []interface{}) []string {
	result := make([]string, len(row))
	for i, v := range row {
		if v == nil {
			result[i] = "NULL"
		} else if b, ok := v.([]byte); ok {
			result[i] = string(b)
		} else {
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// CopyFormat represents the format for copying a row
type CopyFormat int

const (
	CopyFormatCSV CopyFormat = iota
	CopyFormatTSV
	CopyFormatJSON
	CopyFormatVertical
	CopyFormatASCIITable
	CopyFormatUnicodeTable
)

// editorFinishedMsg is sent when the external editor closes
type editorFinishedMsg struct {
	err error
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
	highlighted := selectedRowStyle.Width(width).Render(ansi.Strip(line))
	lines[selectedRow] = highlighted

	return strings.Join(lines, "\n")
}

// ensureSelectionVisible scrolls the viewport to keep the selected row visible
func (m *Model) ensureSelectionVisible(res *Result) {
	if res.SelectedRow < 0 {
		return
	}

	viewportTop := res.Viewport.YOffset
	viewportBottom := viewportTop + res.Viewport.Height - 1

	if res.SelectedRow < viewportTop {
		res.Viewport.SetYOffset(res.SelectedRow)
	} else if res.SelectedRow > viewportBottom {
		res.Viewport.SetYOffset(res.SelectedRow - res.Viewport.Height + 1)
	}
}

// openRowDetailModal initializes and shows the row detail modal
func (m *Model) openRowDetailModal(res *Result) {
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return
	}

	row := res.DbResult.Rows[res.SelectedRow]
	content := formatRowVertical(res.DbResult.Headers, interfaceSliceToStrings(row))

	// Create viewport for scrolling
	modalWidth := min(m.width-10, 80)
	modalHeight := min(m.height-10, 20)

	m.rowDetailViewport = viewport.New(modalWidth, modalHeight)
	m.rowDetailViewport.SetContent(content)
	m.showRowDetail = true
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

		label := rowDetailLabelStyle.Render(fmt.Sprintf("%*s", maxLen, header))
		val := rowDetailValueStyle.Render(value)
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

// renderRowDetailModal renders the row detail modal overlay
func (m Model) renderRowDetailModal() string {
	title := rowDetailTitleStyle.Render("Row Details")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("j/k: scroll  q/Esc: close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.rowDetailViewport.View(),
		"",
		help,
	)

	return rowDetailBorderStyle.Render(content)
}

// renderCopyMenu renders the copy format selection menu
func (m Model) renderCopyMenu() string {
	title := copyMenuTitleStyle.Render("Copy Row As")

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
		if i == m.copyMenuIndex {
			items = append(items, copyMenuSelectedStyle.Render("> "+opt))
		} else {
			items = append(items, copyMenuItemStyle.Render("  "+opt))
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

	return copyMenuBorderStyle.Render(content)
}

// copyRowToClipboard copies the selected row to clipboard in the specified format
func (m *Model) copyRowToClipboard(format CopyFormat) {
	if m.focusedResult < 0 || m.focusedResult >= len(m.results) {
		return
	}

	res := m.results[m.focusedResult]
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return
	}

	row := interfaceSliceToStrings(res.DbResult.Rows[res.SelectedRow])
	headers := res.DbResult.Headers

	var content string
	switch format {
	case CopyFormatCSV:
		content = formatRowCSV(headers, row)
	case CopyFormatTSV:
		content = formatRowTSV(headers, row)
	case CopyFormatJSON:
		content = formatRowJSON(headers, row)
	case CopyFormatVertical:
		content = formatRowVerticalText(headers, row)
	case CopyFormatASCIITable:
		var buf bytes.Buffer
		formatter.RenderResult(res.DbResult, &buf, formatter.FormatTable, m.config)
		content = buf.String()
	case CopyFormatUnicodeTable:
		var buf bytes.Buffer
		formatter.RenderResult(res.DbResult, &buf, formatter.FormatUnicode, m.config)
		content = buf.String()
	}

	clipboard.WriteAll(content)
	m.showCopyMenu = false

	// Show confirmation
	m.specialOutput.Reset()
	fmt.Fprintf(&m.specialOutput, "Row copied to clipboard as %s.\n", formatName(format))
	m.addResultFromText(m.specialOutput.String(), "Copy Row")
}

func formatName(f CopyFormat) string {
	switch f {
	case CopyFormatCSV:
		return "CSV"
	case CopyFormatTSV:
		return "TSV"
	case CopyFormatJSON:
		return "JSON"
	case CopyFormatVertical:
		return "Vertical"
	case CopyFormatASCIITable:
		return "ASCII Table"
	case CopyFormatUnicodeTable:
		return "Unicode Table"
	}
	return "Unknown"
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

// openRowInEditor opens the selected row in an external editor
func (m *Model) openRowInEditor(res *Result) tea.Cmd {
	if res.DbResult == nil || res.SelectedRow < 0 || res.SelectedRow >= len(res.DbResult.Rows) {
		return nil
	}

	row := interfaceSliceToStrings(res.DbResult.Rows[res.SelectedRow])
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
