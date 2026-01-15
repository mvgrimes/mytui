package main

import (
	"fmt"
	"os"

	"github.com/dbcli/mycli-go/internal/db"
	"github.com/dbcli/mycli-go/internal/repl"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mycli",
		Short: "A MySQL terminal client with auto-completion and syntax highlighting",
		Long: `mycli is a command line interface for MySQL and MariaDB that provides
auto-completion, syntax highlighting, and other useful features for database
administration and development.`,
		Version: version,
		RunE:    runREPL,
	}

	// Disable default help flag to avoid conflict with -h for host
	rootCmd.Flags().BoolP("help", "", false, "help for mycli")

	// Connection flags
	rootCmd.Flags().String("host", "localhost", "Host address of the database")
	rootCmd.Flags().IntP("port", "P", 3306, "Port number to use for connection")
	rootCmd.Flags().StringP("user", "u", "", "User name to connect to the database")
	rootCmd.Flags().StringP("password", "p", "", "Password to connect to the database")
	rootCmd.Flags().StringP("database", "D", "", "Database to use")
	rootCmd.Flags().StringP("socket", "S", "", "The socket file to use for connection")

	// SSL flags
	rootCmd.Flags().String("ssl-mode", "auto", "Set desired SSL behavior (auto, on, off)")
	rootCmd.Flags().String("ssl-ca", "", "CA file in PEM format")
	rootCmd.Flags().String("ssl-cert", "", "X509 cert in PEM format")
	rootCmd.Flags().String("ssl-key", "", "X509 key in PEM format")

	// Other flags
	rootCmd.Flags().String("charset", "utf8", "Character set for MySQL session")
	rootCmd.Flags().Bool("local-infile", false, "Enable/disable LOAD DATA LOCAL INFILE")
	rootCmd.Flags().String("init-command", "", "SQL statement to execute after connecting")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runREPL(cmd *cobra.Command, args []string) error {
	// For now, just print connection info and exit
	// This will be replaced with the actual REPL implementation

	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	database, _ := cmd.Flags().GetString("database")

	fmt.Printf("Connecting to MySQL at %s:%d as user %s", host, port, user)
	if database != "" {
		fmt.Printf(" (database: %s)", database)
	}
	fmt.Println()

	fmt.Println("REPL implementation coming soon...")
	return nil
}