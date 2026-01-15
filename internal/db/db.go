package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Socket   string
	Charset  string
	SSLMode  string
	SSLCa    string
	SSLCert  string
	SSLKey   string
}

type Connection struct {
	db *sql.DB
}

func NewConnection(config Config) (*Connection, error) {
	dsn := buildDSN(config)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Connection{db: db}, nil
}

func buildDSN(config Config) string {
	// Build the DSN string for MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s",
		config.User, config.Password, config.Host, config.Port, config.Database, config.Charset)

	if config.Socket != "" {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?charset=%s",
			config.User, config.Password, config.Socket, config.Database, config.Charset)
	}

	// Add SSL parameters if needed
	if config.SSLMode != "off" {
		dsn += "&tls=" + config.SSLMode
		if config.SSLCa != "" {
			dsn += "&tlsCA=" + config.SSLCa
		}
		if config.SSLCert != "" {
			dsn += "&tlsCert=" + config.SSLCert
		}
		if config.SSLKey != "" {
			dsn += "&tlsKey=" + config.SSLKey
		}
	}

	return dsn
}

func (c *Connection) Close() error {
	return c.db.Close()
}

func (c *Connection) Execute(query string) (*sql.Rows, error) {
	return c.db.Query(query)
}

func (c *Connection) ExecuteNonQuery(query string) (sql.Result, error) {
	return c.db.Exec(query)
}

func (c *Connection) GetServerInfo() (string, error) {
	var version string
	err := c.db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", err
	}
	return version, nil
}