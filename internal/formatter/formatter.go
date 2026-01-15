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
	"github.com/mvgrimes/mycli-go/internal/db"
	"github.com/olekukonko/tablewriter"
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

// PrintResult prints the database result in a tabular format
func PrintResult(result *db.Result, out io.Writer) {
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
		table := tablewriter.NewWriter(writer)
		table.SetHeader(result.Headers)
		table.SetAutoFormatHeaders(false)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetCenterSeparator("|")
		table.SetColumnSeparator("|")
		table.SetRowSeparator("-")
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

	fmt.Fprintf(out, "%s (%.2f sec)\n", result.Status, result.Duration.Seconds())
}

// PrintVerticalResult prints the database result in a vertical format (like \G)
func PrintVerticalResult(result *db.Result, out io.Writer) {
	if len(result.Headers) > 0 {
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
	fmt.Fprintf(out, "%s (%.2f sec)\n", result.Status, result.Duration.Seconds())
}

// HighlightSQL highlight the SQL string and prints it
func HighlightSQL(sql string) {
	fmt.Println(FormatSQL(sql))
}
