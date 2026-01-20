package completion

import (
	"sort"
	"strings"

	"github.com/mvgrimes/mycli-go/internal/db"
)

type DBCache struct {
	DefaultSchema     string
	Schemas           map[string]string
	SchemaTables      map[string][]string
	ColumnsWithParent map[string][]*db.ColumnDesc
	ForeignKeys       map[string]map[string][]*db.ForeignKey
}

func NewDBCache() *DBCache {
	return &DBCache{
		Schemas:           make(map[string]string),
		SchemaTables:      make(map[string][]string),
		ColumnsWithParent: make(map[string][]*db.ColumnDesc),
		ForeignKeys:       make(map[string]map[string][]*db.ForeignKey),
	}
}

func (dc *DBCache) Database(dbName string) (db string, ok bool) {
	db, ok = dc.Schemas[strings.ToUpper(dbName)]
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
	tbls, ok = dc.SchemaTables[strings.ToUpper(dbName)]
	sort.Strings(tbls)
	return
}

func (dc *DBCache) SortedTables() []string {
	tbls, _ := dc.SortedTablesByDBName(dc.DefaultSchema)
	return tbls
}

func (dc *DBCache) ColumnDescs(tableName string) (cols []*db.ColumnDesc, ok bool) {
	cols, ok = dc.ColumnsWithParent[columnDatabaseKey(dc.DefaultSchema, tableName)]
	return
}

func (dc *DBCache) ColumnDatabase(dbName, tableName string) (cols []*db.ColumnDesc, ok bool) {
	cols, ok = dc.ColumnsWithParent[columnDatabaseKey(dbName, tableName)]
	return
}

func (dc *DBCache) Column(tableName, colName string) (*db.ColumnDesc, bool) {
	cols, ok := dc.ColumnsWithParent[columnDatabaseKey(dc.DefaultSchema, tableName)]
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
	return strings.ToUpper(dbName) + "\t" + strings.ToUpper(tableName)
}
