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
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/olekukonko/tablewriter"
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
func PrintResult(result *db.Result, out io.Writer, format Format) {
	var writer io.Writer = out
	var cmd *exec.Cmd

	// Use pager if results are large
	if len(result.Rows) > 20 {
		pager := os.Getenv("PAGER")
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
	}

	fmt.Fprintf(writer, "%s (%.2f sec)\n", result.Status, result.Duration.Seconds())
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
			if val == nil {
				rowStrings[i] = "NULL"
			} else if b, ok := val.([]byte); ok {
				rowStrings[i] = string(b)
			} else {
				rowStrings[i] = fmt.Sprintf("%v", val)
			}
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
			str := ""
			if val == nil {
				str = "NULL"
			} else if b, ok := val.([]byte); ok {
				str = string(b)
			} else {
				str = fmt.Sprintf("%v", val)
			}
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
	for i, row := range result.Rows {
		fmt.Fprintf(out, "*************************** %d. row ***************************\n", i+1)
		for j, header := range result.Headers {
			val := row[j]
			valStr := ""
			if val == nil {
				valStr = "NULL"
			} else if b, ok := val.([]byte); ok {
				valStr = string(b)
			} else {
				valStr = fmt.Sprintf("%v", val)
			}
			fmt.Fprintf(out, "%s: %s\n", header, valStr)
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
			} else if b, ok := val.([]byte); ok {
				rowStrings[i] = string(b)
			} else {
				rowStrings[i] = fmt.Sprintf("%v", val)
			}
		}
		fmt.Fprintln(out, strings.Join(rowStrings, delimiter))
	}
}

// HighlightSQL highlight the SQL string and prints it
func HighlightSQL(sql string) {
	fmt.Println(FormatSQL(sql))
}
