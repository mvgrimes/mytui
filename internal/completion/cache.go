package completion

import (
	"sort"
	"strings"

	"github.com/mvgrimes/mytui/internal/db"
)

type DBCache struct {
	// LowerCaseTableNames is the server's identifier comparison mode (0, 1, 2).
	// -1 means unknown: allow only unambiguous case-insensitive fallback matches.
	LowerCaseTableNames int
	DefaultSchema       string
	Schemas             map[string]string
	SchemaTables        map[string][]string
	ColumnsWithParent   map[string][]*db.ColumnDesc
	ForeignKeys         map[string]map[string][]*db.ForeignKey
}

func NewDBCache() *DBCache {
	return &DBCache{
		LowerCaseTableNames: -1,
		Schemas:             make(map[string]string),
		SchemaTables:        make(map[string][]string),
		ColumnsWithParent:   make(map[string][]*db.ColumnDesc),
		ForeignKeys:         make(map[string]map[string][]*db.ForeignKey),
	}
}

func (dc *DBCache) Database(dbName string) (db string, ok bool) {
	db, ok = lookupIdentifier(dc.Schemas, dbName, dc.LowerCaseTableNames != 0)
	return
}

func (dc *DBCache) SortedSchemas() []string {
	dbs := []string{}
	for _, db := range dc.Schemas {
		dbs = append(dbs, db)
	}
	sort.Strings(dbs)
	return dbs
}

func (dc *DBCache) SortedTablesByDBName(dbName string) (tbls []string, ok bool) {
	schema, ok := dc.resolveSchema(dbName)
	if !ok {
		return nil, false
	}
	tbls, ok = dc.SchemaTables[schema]
	tbls = append([]string(nil), tbls...)
	sort.Strings(tbls)
	return
}

func (dc *DBCache) SortedTables() []string {
	tbls, _ := dc.SortedTablesByDBName(dc.DefaultSchema)
	return tbls
}

func (dc *DBCache) ColumnDescs(tableName string) (cols []*db.ColumnDesc, ok bool) {
	return dc.ColumnDatabase(dc.DefaultSchema, tableName)
}

func (dc *DBCache) ColumnDatabase(dbName, tableName string) (cols []*db.ColumnDesc, ok bool) {
	// Resolve the schema separately so a folded table match cannot cross into
	// another, differently cased schema when an exact schema exists.
	schema, ok := dc.resolveSchema(dbName)
	if !ok {
		return nil, false
	}
	if cols, ok := dc.ColumnsWithParent[columnDatabaseKey(schema, tableName)]; ok {
		return cols, true
	}
	if dc.LowerCaseTableNames == 0 {
		return nil, false
	}
	tables := make(map[string][]*db.ColumnDesc)
	for key, columns := range dc.ColumnsWithParent {
		name, table, _ := strings.Cut(key, "\t")
		if name == schema {
			tables[table] = columns
		}
	}
	// Include tables without loaded columns in ambiguity detection too.
	for _, table := range dc.SchemaTables[schema] {
		if _, exists := tables[table]; !exists {
			tables[table] = nil
		}
	}
	cols, ok = lookupIdentifier(tables, tableName, dc.LowerCaseTableNames != 0)
	return cols, ok && cols != nil
}

func (dc *DBCache) resolveSchema(dbName string) (string, bool) {
	if _, ok := dc.Schemas[dbName]; ok {
		return dbName, true
	}
	if _, ok := dc.SchemaTables[dbName]; ok {
		return dbName, true
	}
	schemas := make(map[string]string)
	for name := range dc.Schemas {
		schemas[name] = name
	}
	for name := range dc.SchemaTables {
		schemas[name] = name
	}
	for key := range dc.ColumnsWithParent {
		name, _, _ := strings.Cut(key, "\t")
		schemas[name] = name
	}
	return lookupIdentifier(schemas, dbName, dc.LowerCaseTableNames != 0)
}

func (dc *DBCache) Column(tableName, colName string) (*db.ColumnDesc, bool) {
	cols, ok := dc.ColumnDescs(tableName)
	if !ok {
		return nil, false
	}
	for _, col := range cols {
		if strings.EqualFold(col.Name, colName) {
			return col, true
		}
	}
	return nil, false
}

func columnDatabaseKey(dbName, tableName string) string {
	return dbName + "\t" + tableName
}

// lookupIdentifier prefers exact spelling and never chooses an arbitrary
// identifier when multiple names differ only in case.
func lookupIdentifier[T any](items map[string]T, name string, ignoreCase bool) (value T, ok bool) {
	if value, ok = items[name]; ok || !ignoreCase {
		return
	}
	for candidate, item := range items {
		if strings.EqualFold(candidate, name) {
			if ok {
				var zero T
				return zero, false
			}
			value, ok = item, true
		}
	}
	return
}
