# AGENTS.md - Developer's Guide for AI Agents

This file provides context and instructions for AI agents working on the `mytui` project.

## Project Overview

`mytui` is a reimplementation of the Python-based `mycli` in Go. It provides a terminal-based REPL for MySQL/MariaDB with auto-completion and syntax highlighting.

## Architecture

The project is structured into several internal packages:

-   `internal/db`: Handles database connections (MySQL/MariaDB) and query execution. It uses `go-sql-driver/mysql`.
-   `internal/repl`: Manages the main REPL loop using `go-prompt`. Handles input, history, and integration with other packages.
-   `internal/completion`: Implements context-aware SQL auto-completion. It handles keywords, tables, and columns, including alias resolution (e.g., `u.id` -> `users.id`).
-   `internal/formatter`: Responsible for rendering query results in various formats (`table`, `unicode`, `vertical`, `csv`, `tsv`). It uses `tablewriter` for tabular output.
-   `internal/special`: Handles backslash commands (e.g., `\T`, `\f`, `\clip`, `\e`).
-   `internal/config`: Manages user configuration using `viper`. Config is stored in `~/.config/mytui/config.toml`.
-   `internal/vim`: Implements the Vim-style keybinding state machine.

## Technical Stack

-   **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
-   **REPL**: [go-prompt](https://github.com/c-bata/go-prompt)
-   **Syntax Highlighting**: [Chroma](https://github.com/alecthomas/chroma)
-   **Configuration**: [Viper](https://github.com/spf13/viper)
-   **Database Driver**: [MySQL](https://github.com/go-sql-driver/mysql)
-   **Table Formatting**: [Tablewriter (v0.0.5)](https://github.com/olekukonko/tablewriter)
-   **Clipboard**: [Clipboard](https://github.com/atotto/clipboard)

## Key Conventions for Agents

### 1. Special Commands
New backslash commands should be implemented in the `internal/special` package. The `REPL` interface in that package should be updated if the command needs new capabilities from the REPL engine.

### 2. Formatter State
The `formatter` package is mostly functional. It receives a `db.Result` and prints it. If adding a new format, add it to the `Format` type and update `PrintResult`.

### 3. Vim Mode
Vim mode is implemented by switching the prompt mode. Keybindings and ASCII code bindings are defined in `internal/vim`.

### 4. Configuration
Always use `viper` for configuration. New configuration options should be added to the `Config` struct in `internal/config/config.go` and have a default value set in `LoadConfig`.

### 5. Dependency Management
Stick to the existing libraries. Note that we are using `tablewriter v0.0.5` specifically for its stable method-based API; do not upgrade it to `v1.x` without a full refactor of the `formatter` package.

## Common Tasks for Agents

-   **Adding a special command**:
    1. Update `internal/special/special.go` to handle the new command.
    2. Update `internal/completion/completion.go` to add suggestions for the new command.
-   **Improving completion**:
    1. Logic resides in `internal/completion/completion.go`.
    2. Use the `Completer` struct to track schema metadata.
-   **Modifying the UI**:
    1. Prompt logic is in `internal/repl/repl.go` (see `formatPrompt`).
    2. Syntax highlighting logic is in `internal/formatter/formatter.go`.
