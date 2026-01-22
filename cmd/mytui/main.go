package main

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/mvgrimes/mytui/internal/config"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
	"github.com/mvgrimes/mytui/internal/repl"
	"github.com/mvgrimes/mytui/internal/tui"
)

var version = "0.1.1"

func main() {
	rootCmd := &cobra.Command{
		Use:   "mytui",
		Short: "A MySQL terminal client with auto-completion and syntax highlighting",
		Long: `mycli is a command line interface for MySQL and MariaDB that provides
auto-completion, syntax highlighting, and other useful features for database
administration and development.`,
		Version: version,
		RunE:    runREPL,
	}

	// Disable default help flag to avoid conflict with -h for host
	rootCmd.Flags().BoolP("help", "", false, "help for mytui")

	// Connection flags
	rootCmd.Flags().StringP("host", "h", "localhost", "Host address of the database")
	rootCmd.Flags().IntP("port", "P", 3306, "Port number to use for connection")
	rootCmd.Flags().StringP("user", "u", "", "User name to connect to the database")
	rootCmd.Flags().StringP("password", "p", "", "Password to connect to the database")
	rootCmd.Flags().StringP("database", "D", "", "Database to use")
	rootCmd.Flags().StringP("socket", "S", "", "The socket file to use for connection")

	// SSL flags
	rootCmd.Flags().String("ssl-mode", "preferred", "Set desired SSL behavior (disabled, preferred, required, verify-ca, verify-full)")
	rootCmd.Flags().String("ssl-ca", "", "CA file in PEM format")
	rootCmd.Flags().String("ssl-cert", "", "X509 cert in PEM format")
	rootCmd.Flags().String("ssl-key", "", "X509 key in PEM format")

	// Other flags
	rootCmd.Flags().String("charset", "utf8mb4", "Character set for MySQL session")
	rootCmd.Flags().Bool("local-infile", false, "Enable/disable LOAD DATA LOCAL INFILE")
	rootCmd.Flags().String("init-command", "", "SQL statement to execute after connecting")
	rootCmd.Flags().StringP("prompt", "R", "", "Prompt template")
	rootCmd.Flags().StringP("config", "c", "", "Path to config file")
	rootCmd.Flags().StringP("execute", "e", "", "Execute a command and quit")
	rootCmd.Flags().Bool("tui", false, "Use the new TUI interface")

	// Allow database as positional argument
	rootCmd.Args = cobra.MaximumNArgs(1)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runREPL(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	database, _ := cmd.Flags().GetString("database")
	socket, _ := cmd.Flags().GetString("socket")
	charset, _ := cmd.Flags().GetString("charset")
	sslMode, _ := cmd.Flags().GetString("ssl-mode")
	sslCa, _ := cmd.Flags().GetString("ssl-ca")
	sslCert, _ := cmd.Flags().GetString("ssl-cert")
	sslKey, _ := cmd.Flags().GetString("ssl-key")
	promptFlag, _ := cmd.Flags().GetString("prompt")
	configPath, _ := cmd.Flags().GetString("config")
	executeFlag, _ := cmd.Flags().GetString("execute")
	tuiFlag, _ := cmd.Flags().GetBool("tui")

	// Use positional argument for database if flag is not set
	if database == "" && len(args) > 0 {
		database = args[0]
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = &config.Config{
			TableFormat: "table",
			SyntaxStyle: "monokai",
			KeyBindings: "vim",
			HistoryFile: "/tmp/mytui_history",
		}
	}

	// Override prompt if flag is provided
	if promptFlag != "" {
		cfg.Prompt = promptFlag
	}

	dbConfig := db.Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
		Socket:   socket,
		Charset:  charset,
		SSLMode:  sslMode,
		SSLCa:    sslCa,
		SSLCert:  sslCert,
		SSLKey:   sslKey,
	}

	// Check for piped input first
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		query, err := io.ReadAll(os.Stdin)
		if err == nil && len(query) > 0 {
			conn, err := db.NewConnection(dbConfig)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %v", err)
			}
			defer conn.Close()

			result, err := conn.ExecuteQuery(string(query))
			if err != nil {
				return fmt.Errorf("failed to execute query: %v", err)
			}
			formatter.PrintResult(result, os.Stdout, formatter.Format(cfg.TableFormat), cfg, "")
			return nil
		}
	}

	fmt.Printf("Connecting to MySQL at %s:%d as user %s...\n", host, port, user)
	conn, err := db.NewConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	if executeFlag != "" {
		result, err := conn.ExecuteQuery(executeFlag)
		if err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
		formatter.PrintResult(result, os.Stdout, formatter.Format(cfg.TableFormat), cfg, "")
		return nil
	}

	serverVersion, err := conn.GetServerInfo()
	if err == nil {
		fmt.Printf("Connected! Server version: %s\n", serverVersion)
	}

	if tuiFlag {
		p := tea.NewProgram(tui.NewModel(conn, cfg), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %v", err)
		}
		return nil
	}

	repl.RunREPL(conn, cfg)
	return nil
}
