# mycli-go

A command-line client for MySQL and MariaDB with auto-completion and syntax highlighting, reimagined in Go.

## Motivation

`mycli-go` is a port of the popular [mycli](https://github.com/dbcli/mycli) tool from Python to Go. While the original `mycli` is an excellent tool, the Go implementation offers several key advantages:

*   **Zero Dependencies**: Distributes as a single static binary. No more Python environment issues or dependency conflicts.
*   **Performance**: Improved startup time and faster rendering of large result sets.
*   **Stability**: Leverages Go's strong typing and concurrency primitives for a more robust experience.
*   **Maintenance**: Simplifies the codebase and makes it easier to contribute to and deploy across different platforms.

## Features

*   **Auto-completion**: Suggestions for SQL keywords, tables, views, and columns as you type.
*   **Smart-completion**: Context-sensitive suggestions (e.g., suggesting only columns after a `WHERE` clause).
*   **Syntax Highlighting**: Beautiful SQL syntax coloring powered by Chroma.
*   **Pretty Printing**: High-quality tabular output with support for multiple formats:
    *   `table` (standard)
    *   `unicode` (fancy box-drawing characters)
    *   `vertical` (equivalent to `\G` in the standard client)
    *   `csv` and `tsv`
*   **Auto-Vertical Mode**: Automatically switches to vertical output if the table is too wide for your terminal.
*   **Favorite Queries**: Save queries with `\fs` and execute them later with `\f`. Supports positional parameters.
*   **Vim Mode**: Full Vim-style keybindings in the REPL (Normal/Insert modes).
*   **External Editor**: Open your current query in your favorite editor with `\e`.
*   **Clipboard Integration**: Quickly copy query results to your system clipboard with `\clip`.
*   **Configuration**: Highly customizable via `~/.config/mycli-go/config.toml`.
*   **Pager Support**: Automatic paging for large result sets using your system's `PAGER` (defaulting to `less`).
*   **SSL Support**: Secure connections to your database instances.

## Installation

### From Source

Ensure you have Go installed (1.24 or later), then run:

```bash
go install github.com/mvgrimes/mycli-go/cmd/mycli-go@latest
```

Alternatively, clone the repository and build it manually:

```bash
git clone https://github.com/mvgrimes/mycli-go.git
cd mycli-go
go build -o mycli-go ./cmd/mycli-go
```

## Usage

Connect to a database:

```bash
mycli-go -h localhost -u root -p mypassword -D mydatabase
```

Or using the positional argument for the database:

```bash
mycli-go -u root mydatabase
```

Execute a query and quit:

```bash
mycli-go -e "SELECT * FROM users LIMIT 10" mydatabase
```

### Special Commands

`mycli-go` supports several backslash commands:

*   `\T [format]`: Change the output format (table, vertical, csv, tsv, unicode).
*   `\f [name] [args...]`: Execute a favorite query.
*   `\fs [name] [query]`: Save a new favorite query.
*   `\fd [name]`: Delete a favorite query.
*   `\e`: Open the last query in an external editor.
*   `\clip`: Copy the last result to the clipboard.
*   `\once [format]`: Use a specific format for the next query only.
*   `\| [command]`: Pipe the next result to a shell command.
*   `\q`: Quit the application.

## Configuration

On first launch, a default configuration file is created at `~/.config/mycli-go/config.toml`. You can customize your prompt, default table format, syntax highlighting theme, and more.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
