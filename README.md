# mytui

A command-line client for MySQL and MariaDB with auto-completion and syntax highlighting, reimagined in Go.

## Motivation

`mytui` is a port of the popular [mycli](https://github.com/dbcli/mycli) tool from Python to Go. While the original `mycli` is an excellent tool, the Go implementation offers several key advantages:

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
*   **Configuration**: Highly customizable via `~/.config/mytui/config.toml`.
*   **Pager Support**: Automatic paging for large result sets using your system's `PAGER` (defaulting to `less`).
*   **SSL Support**: Secure connections to your database instances.

## Installation

### From Source

Ensure you have Go installed (1.24 or later), then run:

```bash
go install github.com/mvgrimes/mytui/cmd/mytui@latest
```

Alternatively, clone the repository and build it manually:

```bash
git clone https://github.com/mvgrimes/mytui.git
cd mytui
go build -o mytui ./cmd/mytui
```

## Usage

Connect to a database:

```bash
mytui -h localhost -u root -p mypassword -D mydatabase
```

Or using the positional argument for the database:

```bash
mytui -u root mydatabase
```

Execute a query and quit:

```bash
mytui -e "SELECT * FROM users LIMIT 10" mydatabase
```

### Special Commands

`mytui` supports several backslash commands:

*   `\T [format]`: Change the output format (table, vertical, csv, tsv, unicode).
*   `\f [name] [args...]`: Execute a favorite query.
*   `\fs [name] [query]`: Save a new favorite query.
*   `\fd [name]`: Delete a favorite query.
*   `\e`: Open the last query in an external editor.
*   `\clip`: Copy the last result to the clipboard.
*   `\once [format]`: Use a specific format for the next query only.
*   `\| [command]`: Pipe the next result to a shell command.
*   `\q`: Quit the application.

## Vim Keybindings

`mytui` supports Vim-style keybindings in the query editor. Press `Esc` to enter Normal mode, and `i` to return to Insert mode.

### Normal Mode

#### Movement
| Key | Action |
|-----|--------|
| `h` | Move cursor left |
| `l` | Move cursor right |
| `j` | Move cursor down |
| `k` | Move cursor up |
| `w` | Move to next word |
| `b` | Move to previous word |
| `0` or `^` | Move to start of line |
| `$` | Move to end of line |
| `f{char}` | Jump to next occurrence of {char} on current line |
| `F{char}` | Jump to previous occurrence of {char} on current line |

#### Enter Insert Mode
| Key | Action |
|-----|--------|
| `i` | Insert before cursor |
| `a` | Insert after cursor |
| `I` | Insert at start of line |
| `A` | Insert at end of line |
| `o` | Open new line below |
| `O` | Open new line above |

#### Delete
| Key | Action |
|-----|--------|
| `x` | Delete character under cursor |
| `dd` | Delete entire line |
| `dw` | Delete to next word |
| `d$` | Delete to end of line |
| `d0` | Delete to start of line |
| `diw` | Delete inner word |
| `D` | Delete to end of line (same as `d$`) |

#### Change (delete and enter Insert mode)
| Key | Action |
|-----|--------|
| `cc` | Change entire line |
| `cw` | Change to next word |
| `c$` | Change to end of line |
| `c0` | Change to start of line |
| `ciw` | Change inner word |
| `C` | Change to end of line (same as `c$`) |

#### SQL Shortcuts
| Key | Action |
|-----|--------|
| `gi` | Insert `INSERT INTO () VALUES ()` template |
| `gs` | Insert `SELECT * FROM` template |
| `gd` | Insert `DELETE FROM` template |
| `gc` | Insert `CREATE TABLE` template |
| `gf` | Jump to fields position (after SELECT or inside INSERT parentheses) |
| `gt` | Jump to table name position |
| `gw` | Jump to WHERE clause (inserts if missing) |

### Insert Mode

| Key | Action |
|-----|--------|
| `Esc` | Return to Normal mode |
| `Enter` | Execute query |
| `Ctrl+J` or `Alt+Enter` | Insert newline |
| `Ctrl+K` | Open autocomplete |
| `Ctrl+P` | Previous history |
| `Ctrl+N` | Next history |

#### SQL Shortcuts (Ctrl+X prefix)
| Key | Action |
|-----|--------|
| `Ctrl+X i` | Insert `INSERT INTO () VALUES ()` template |
| `Ctrl+X s` | Insert `SELECT * FROM` template |
| `Ctrl+X d` | Insert `DELETE FROM` template |
| `Ctrl+X c` | Insert `CREATE TABLE` template |
| `Ctrl+X f` | Jump to fields position |
| `Ctrl+X t` | Jump to table name position |
| `Ctrl+X w` | Jump to WHERE clause (inserts if missing) |

## Configuration

On first launch, a default configuration file is created at `~/.config/mytui/config.toml`. You can customize your prompt, default table format, syntax highlighting theme, and more.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
