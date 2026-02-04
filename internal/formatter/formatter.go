package formatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/fatih/color"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

type Format string

const (
	FormatTable    Format = "table"
	FormatVertical Format = "vertical"
	FormatCSV      Format = "csv"
	FormatTSV      Format = "tsv"
	FormatUnicode  Format = "unicode"
)

// FormatSQL returns a colorized version of the SQL string
func FormatSQL(sql string) string {
	lexer := lexers.Get("sql")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, sql)
	if err != nil {
		return sql
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return sql
	}

	return buf.String()
}

// PrintResult prints the database result in the specified format
func PrintResult(result *db.Result, out io.Writer, format Format, cfg *config.Config, pagerOverride string) {
	var writer io.Writer = out
	var cmd *exec.Cmd

	// Handle auto-vertical output
	if cfg.AutoVerticalOutput && format == FormatTable {
		if calculateTotalWidth(result) > getTerminalWidth() {
			format = FormatVertical
		}
	}

	// Use pager if results are large or override is set
	if len(result.Rows) > 20 || pagerOverride != "" {
		pager := pagerOverride
		if pager == "" {
			pager = cfg.Pager
		}
		if pager == "" {
			pager = os.Getenv("PAGER")
		}
		if pager == "" {
			pager = "less -SRXF"
		}
		args := strings.Fields(pager)
		cmd = exec.Command(args[0], args[1:]...)
		pagerWriter, err := cmd.StdinPipe()
		if err == nil {
			cmd.Stdout = out
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err == nil {
				writer = pagerWriter
				defer func() {
					pagerWriter.Close()
					cmd.Wait()
				}()
			}
		}
	}

	RenderResult(result, writer, format, cfg)
}

// RenderResult draws the result to the provided writer
func RenderResult(result *db.Result, writer io.Writer, format Format, cfg *config.Config) {
	if len(result.Headers) > 0 {
		switch format {
		case FormatVertical:
			printVertical(result, writer)
		case FormatCSV:
			printCSV(result, writer, ",")
		case FormatTSV:
			printCSV(result, writer, "\t")
		case FormatUnicode:
			printUnicode(result, writer)
		default:
			printTable(result, writer)
		}
	} else if result.Status != "" {
		if cfg.Timing {
			fmt.Fprintf(writer, "%s (%.2f sec)\n", result.Status, result.Duration.Seconds())
		} else {
			fmt.Fprintf(writer, "%s\n", result.Status)
		}
	}
}

// FormatResult returns the formatted result as a string
func FormatResult(result *db.Result, format Format, cfg *config.Config) string {
	var buf bytes.Buffer
	RenderResult(result, &buf, format, cfg)
	return buf.String()
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default
	}
	return width
}

func calculateTotalWidth(result *db.Result) int {
	if len(result.Headers) == 0 {
		return 0
	}

	widths := make([]int, len(result.Headers))
	for i, h := range result.Headers {
		widths[i] = runewidth.StringWidth(h)
	}

	for _, row := range result.Rows {
		for j, val := range row {
			str := formatValue(val)
			w := runewidth.StringWidth(str)
			if w > widths[j] {
				widths[j] = w
			}
		}
	}

	total := 1 // Initial |
	for _, w := range widths {
		total += w + 3 // " " + content + " " + |
	}
	return total
}

func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	} else if b, ok := val.([]byte); ok {
		return string(b)
	} else {
		return fmt.Sprintf("%v", val)
	}
}

func printTable(result *db.Result, out io.Writer) {
	table := tablewriter.NewWriter(out)
	table.SetHeader(result.Headers)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoWrapText(false)

	// Set header colors
	headerColors := make([]tablewriter.Colors, len(result.Headers))
	for i := range headerColors {
		headerColors[i] = tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor}
	}
	table.SetHeaderColor(headerColors...)

	for _, row := range result.Rows {
		rowStrings := make([]string, len(row))
		for i, val := range row {
			rowStrings[i] = formatValue(val)
		}
		table.Append(rowStrings)
	}
	table.Render()
}

func printUnicode(result *db.Result, out io.Writer) {
	// Calculate column widths
	widths := make([]int, len(result.Headers))
	for i, h := range result.Headers {
		widths[i] = runewidth.StringWidth(h)
	}

	// Convert all rows to strings and update widths
	rows := make([][]string, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = make([]string, len(row))
		for j, val := range row {
			str := formatValue(val)
			rows[i][j] = str
			w := runewidth.StringWidth(str)
			if w > widths[j] {
				widths[j] = w
			}
		}
	}

	// Helper to draw a line
	drawLine := func(left, middle, right, horizontal string) {
		fmt.Fprint(out, left)
		for i, w := range widths {
			fmt.Fprint(out, strings.Repeat(horizontal, w+2))
			if i < len(widths)-1 {
				fmt.Fprint(out, middle)
			}
		}
		fmt.Fprintln(out, right)
	}

	// Top border
	drawLine("┌", "┬", "┐", "─")

	// Headers
	headerColor := color.New(color.Bold, color.FgCyan)
	fmt.Fprint(out, "│")
	for i, h := range result.Headers {
		w := runewidth.StringWidth(h)
		padding := strings.Repeat(" ", widths[i]-w)
		headerColor.Fprintf(out, " %s%s ", h, padding)
		fmt.Fprint(out, "│")
	}
	fmt.Fprintln(out)

	// Middle border
	drawLine("├", "┼", "┤", "─")

	// Rows
	for _, row := range rows {
		fmt.Fprint(out, "│")
		for i, val := range row {
			w := runewidth.StringWidth(val)
			padding := strings.Repeat(" ", widths[i]-w)
			fmt.Fprintf(out, " %s%s ", val, padding)
			fmt.Fprint(out, "│")
		}
		fmt.Fprintln(out)
	}

	// Bottom border
	drawLine("└", "┴", "┘", "─")
}

func printVertical(result *db.Result, out io.Writer) {
	// Find maximum header width for alignment
	maxHeaderWidth := 0
	for _, header := range result.Headers {
		w := runewidth.StringWidth(header)
		if w > maxHeaderWidth {
			maxHeaderWidth = w
		}
	}

	headerColor := color.New(color.Bold, color.FgCyan)
	separator := color.New(color.FgHiBlack)

	for i, row := range result.Rows {
		separator.Fprintf(out, "*************************** %d. row ***************************\n", i+1)
		for j, header := range result.Headers {
			valStr := formatValue(row[j])
			// Right-align the header
			headerWidth := runewidth.StringWidth(header)
			padding := strings.Repeat(" ", maxHeaderWidth-headerWidth)
			fmt.Fprint(out, padding)
			headerColor.Fprint(out, header)
			fmt.Fprintf(out, ": %s\n", valStr)
		}
	}
}

func printCSV(result *db.Result, out io.Writer, delimiter string) {
	fmt.Fprintln(out, strings.Join(result.Headers, delimiter))
	for _, row := range result.Rows {
		rowStrings := make([]string, len(row))
		for i, val := range row {
			if val == nil {
				rowStrings[i] = ""
			} else {
				rowStrings[i] = formatValue(val)
			}
		}
		fmt.Fprintln(out, strings.Join(rowStrings, delimiter))
	}
}

// HighlightSQL highlight the SQL string and prints it
func HighlightSQL(sql string) {
	fmt.Println(FormatSQL(sql))
}
