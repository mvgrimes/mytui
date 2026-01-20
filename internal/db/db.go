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

func (c *Connection) GetSchemas() ([]string, error) {
	rows, err := c.db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

func (c *Connection) GetSchemaTables() (map[string][]string, error) {
	schemas, err := c.GetSchemas()
	if err != nil {
		return nil, err
	}

	res := make(map[string][]string)
	for _, s := range schemas {
		rows, err := c.db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", s))
		if err != nil {
			continue
		}
		var tables []string
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				continue
			}
			tables = append(tables, table)
		}
		rows.Close()
		res[s] = tables
	}
	return res, nil
}

func (c *Connection) DescribeDatabaseTableBySchema(schema string) ([]*ColumnDesc, error) {
	query := fmt.Sprintf(`
		SELECT 
			TABLE_SCHEMA, 
			TABLE_NAME, 
			COLUMN_NAME, 
			COLUMN_TYPE, 
			IS_NULLABLE, 
			COLUMN_KEY, 
			COLUMN_DEFAULT, 
			EXTRA 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = '%s'
	`, schema)

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []*ColumnDesc
	for rows.Next() {
		var desc ColumnDesc
		err := rows.Scan(
			&desc.Schema,
			&desc.Table,
			&desc.Name,
			&desc.Type,
			&desc.Null,
			&desc.Key,
			&desc.Default,
			&desc.Extra,
		)
		if err != nil {
			return nil, err
		}
		columns = append(columns, &desc)
	}
	return columns, nil
}

func (c *Connection) DescribeForeignKeysBySchema(schema string) ([]*ForeignKey, error) {
	query := fmt.Sprintf(`
		SELECT 
			CONSTRAINT_NAME, 
			TABLE_NAME, 
			COLUMN_NAME, 
			REFERENCED_TABLE_NAME, 
			REFERENCED_COLUMN_NAME 
		FROM information_schema.KEY_COLUMN_USAGE 
		WHERE TABLE_SCHEMA = '%s' 
			AND REFERENCED_TABLE_NAME IS NOT NULL
	`, schema)

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []*ForeignKey
	var lastConstraint string
	var currentFk *ForeignKey

	for rows.Next() {
		var constraint, table, column, refTable, refColumn string
		if err := rows.Scan(&constraint, &table, &column, &refTable, &refColumn); err != nil {
			return nil, err
		}

		if constraint != lastConstraint {
			if currentFk != nil {
				fks = append(fks, currentFk)
			}
			currentFk = &ForeignKey{}
			lastConstraint = constraint
		}

		*currentFk = append(*currentFk, [2]*ColumnBase{
			{Schema: schema, Table: table, Name: column},
			{Schema: schema, Table: refTable, Name: refColumn},
		})
	}
	if currentFk != nil {
		fks = append(fks, currentFk)
	}

	return fks, nil
}

func (c *Connection) GetTablesFromSchema(schema string) ([]string, error) {
	rows, err := c.db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", schema))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			continue
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
