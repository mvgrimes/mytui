package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

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
	db     *sql.DB
	Config Config
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

	return &Connection{db: db, Config: config}, nil
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

func (c *Connection) GetCurrentDatabase() string {
	var dbName sql.NullString
	err := c.db.QueryRow("SELECT DATABASE()").Scan(&dbName)
	if err != nil || !dbName.Valid {
		return "(none)"
	}
	return dbName.String
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

type Result struct {
	Headers  []string
	Rows     [][]interface{}
	Status   string
	Duration time.Duration
}

func (c *Connection) ExecuteQuery(query string) (*Result, error) {
	start := time.Now()

	if isSelectQuery(query) {
		rows, err := c.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		var resultRows [][]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			pointers := make([]interface{}, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				return nil, err
			}
			resultRows = append(resultRows, values)
		}

		duration := time.Since(start)
		return &Result{
			Headers:  columns,
			Rows:     resultRows,
			Status:   fmt.Sprintf("%d rows in set", len(resultRows)),
			Duration: duration,
		}, nil
	} else {
		result, err := c.db.Exec(query)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		duration := time.Since(start)
		return &Result{
			Status:   fmt.Sprintf("Query OK, %d rows affected", affected),
			Duration: duration,
		}, nil
	}
}

func (c *Connection) GetTables() ([]string, error) {
	rows, err := c.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func (c *Connection) GetColumns(table string) ([]string, error) {
	rows, err := c.db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var (
			field, typ, null, key, def, extra sql.NullString
		)
		if err := rows.Scan(&field, &typ, &null, &key, &def, &extra); err != nil {
			return nil, err
		}
		if field.Valid {
			columns = append(columns, field.String)
		}
	}
	return columns, nil
}

func isSelectQuery(query string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(query))
	return strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "SHOW") || strings.HasPrefix(trimmed, "DESCRIBE")
}
