# Plan to Reimplement mycli in Go

## Overview

This plan outlines the reimplementation of mycli, a Python-based command-line interface for MySQL/MariaDB, in Go. The goal is to create a more stable, performant, and maintainable version that avoids dependency conflicts common in the Python ecosystem.

## Current mycli Features

Based on the existing codebase, mycli provides:

- **Auto-completion**: SQL keywords, tables, views, columns, and smart context-aware suggestions
- **Syntax highlighting**: Using Pygments for SQL syntax coloring
- **Multiline queries**: Support for multi-line SQL input
- **Favorite queries**: Save and execute parameterized queries
- **Timing**: Execution time display for queries
- **Logging**: Query and result logging to files
- **Pretty printing**: Colorized tabular output with various formats (table, CSV, TSV, vertical)
- **SSL connections**: Secure database connections
- **Shell redirects**: Support for `$>`, `$>>`, `$|` operators
- **Special commands**: `\u` (use database), `\r` (reconnect), `\fs` (save favorite), etc.
- **Configuration**: Extensive config file support (~/.myclirc)
- **SSH tunneling**: Connect through SSH tunnels
- **Key bindings**: Customizable keyboard shortcuts
- **LLM integration**: AI-powered query assistance
- **Pager support**: Automatic paging for large result sets

## Go Ecosystem Analysis

### Key Libraries and Frameworks

- **CLI Framework**: Cobra (popular, extensive features) or urfave/cli (simpler)
- **Database Driver**: `go-sql-driver/mysql` for MySQL/MariaDB support
- **Interactive Prompt**: `c-bata/go-prompt` or `charmbracelet/bubbletea` for REPL interface
- **Syntax Highlighting**: `alecthomas/chroma` for syntax highlighting
- **Configuration**: `spf13/viper` for config file handling
- **SQL Parsing**: `pingcap/tidb/parser` or `vitessio/vitess/go/vt/sqlparser`
- **Tabular Output**: `olekukonko/tablewriter` or custom implementation
- **SSH**: `golang.org/x/crypto/ssh` for tunneling
- **Logging**: Standard `log` package or `sirupsen/logrus`

### Advantages of Go Implementation

- Single binary deployment (no dependency management issues)
- Better performance and memory usage
- Strong typing reduces runtime errors
- Excellent concurrency support
- Cross-platform compilation
- Mature standard library

## Architecture Overview

### Core Components

1. **Main Application**: Entry point, CLI argument parsing, configuration loading
2. **REPL Engine**: Interactive prompt, command processing, history management
3. **SQL Executor**: Database connection management, query execution, result handling
4. **Completion Engine**: Auto-completion logic, schema introspection
5. **Output Formatter**: Result formatting, paging, colorization
6. **Special Commands**: Non-SQL commands (\u, \r, \fs, etc.)
7. **Configuration Manager**: Config file parsing, defaults handling
8. **Key Bindings**: Customizable keyboard shortcuts
9. **SSH Tunneling**: Secure connection establishment

### Data Flow

```
CLI Args/Config → Main App → REPL Engine ↔ SQL Executor
                                      ↘ Completion Engine
                                      ↘ Output Formatter
                                      ↘ Special Commands
```

## Detailed Implementation Plan

### Phase 1: Foundation and Core Infrastructure

1. **Project Setup**
   - [x] Initialize Go module
   - [x] Set up basic directory structure
   - [ ] Configure build system and CI/CD

2. **CLI Framework Integration**
   - [x] Implement command-line argument parsing
   - [x] Add basic connection options (host, port, user, password, database)
   - [ ] Support for DSN and config files

3. **Database Connection Layer**
   - [x] Implement basic MySQL/MariaDB connection
   - [x] Handle SSL/TLS configuration
   - [ ] Add SSH tunneling support
   - [ ] Implement connection pooling and reconnection logic

4. **Basic REPL**
   - [x] Create interactive prompt interface
   - [x] Implement basic query execution and result display
   - [x] Add history support

### Phase 2: Core Features

5. **SQL Execution Engine**
   - [x] Implement query parsing and execution
   - [ ] Add support for multi-statement queries
   - [x] Handle different result types (SELECT, DDL, DML)
   - [x] Implement timing and logging

6. **Output Formatting**
   - [x] Implement tabular output formats (table, vertical)
   - [x] Add colorization and syntax highlighting
   - [x] Implement pager support for large results
   - [ ] Add auto-vertical output based on terminal width

7. **Configuration System**
   - Port myclirc configuration format
   - Default config files should live in ~/.config/sqlcli/config (with appropriate suffix)
   - Support for system and user config files
   - Implement all configuration options from Python version

### Phase 3: Advanced Features

8. **Auto-Completion System**
   - [x] Implement SQL keyword completion
   - [x] Add schema introspection (tables, columns, views)
   - [x] Implement smart completion based on context
   - [x] Add background completion refresh
   - [x] Syntax highlight the query

9. **Special Commands**
   - Implement all special commands (\u, \r, \fs, \f, etc.)
   - Add favorite queries with parameters
   - Implement destructive query warnings
   - Add shell redirects ($>, $>>, $|)

10. **Key Bindings and Editor Integration**
    - Port key binding system
    - Add external editor support
    - Implement prettify/unprettify commands

### Phase 4: Polish and Testing

11. **Comprehensive Testing**
    - Unit tests for all components
    - Integration tests with real MySQL instances
    - Feature tests mirroring Python version
    - Cross-platform testing

12. **Documentation and Packaging**
    - Create comprehensive README and documentation
    - Implement installation scripts/packages
    - Add man pages and help system

### Phase x: Not planned

13. **LLM Integration**
    - Port LLM command support
    - Implement AI-powered query assistance

## Directory Structure

```
mycli-go/
├── cmd/
│   └── mycli/
│       └── main.go
├── internal/
│   ├── cli/
│   ├── config/
│   ├── db/
│   ├── repl/
│   ├── completion/
│   ├── formatter/
│   ├── special/
│   └── ssh/
├── pkg/
│   ├── lexer/
│   ├── parser/
│   └── utils/
├── test/
│   ├── integration/
│   └── fixtures/
├── docs/
├── scripts/
└── go.mod
```

## Dependencies

### Core Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/c-bata/go-prompt` - Interactive prompt
- `github.com/alecthomas/chroma` - Syntax highlighting
- `github.com/spf13/viper` - Configuration
- `github.com/olekukonko/tablewriter` - Table formatting

### Additional Dependencies
- `golang.org/x/crypto/ssh` - SSH tunneling
- `github.com/mattn/go-runewidth` - Terminal width handling
- `github.com/fatih/color` - Color output
- `github.com/pingcap/tidb/parser` - SQL parsing
- `github.com/sirupsen/logrus` - Logging

## Testing Strategy

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test with real MySQL/MariaDB instances using Docker
3. **Feature Tests**: Behavioral tests matching Python version features
4. **Performance Tests**: Compare execution speed and memory usage
5. **Cross-Platform Tests**: Ensure compatibility on Linux, macOS, Windows

## Migration Considerations

1. **Feature Parity**: Ensure all Python features are implemented
2. **Configuration Compatibility**: Maintain myclirc format compatibility
3. **Command Line Compatibility**: Preserve CLI interface and options
4. **Performance**: Aim for better performance than Python version
5. **Stability**: Eliminate dependency-related breakages

## Risk Assessment

1. **SQL Parsing Complexity**: Implementing full SQL parsing may be challenging
2. **Completion Engine**: Schema introspection and smart completion require careful design
3. **Cross-Platform Compatibility**: Ensure consistent behavior across platforms
4. **SSH Tunneling**: Complex networking requirements
5. **LLM Integration**: API dependencies and rate limiting

## Success Criteria

- All major features from Python version implemented
- Configuration files fully compatible
- CLI interface matches Python version
- Performance improvements over Python version
- Comprehensive test coverage
- Easy installation and deployment
- Active maintenance and community support
