package completion

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mvgrimes/mytui/internal/db"
)

func TestCacheIdentifierResolution(t *testing.T) {
	for _, mode := range []int{-1, 0, 1, 2} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			cache := NewDBCache()
			cache.LowerCaseTableNames = mode
			cache.DefaultSchema = "App"
			cache.Schemas["App"] = "App"
			cache.SchemaTables["App"] = []string{"Users"}
			cache.ColumnsWithParent[columnDatabaseKey("App", "Users")] = []*db.ColumnDesc{
				{ColumnBase: db.ColumnBase{Name: "UserID"}},
			}
			if got, ok := cache.Database("App"); !ok || got != "App" {
				t.Fatalf("exact schema: %q, %v", got, ok)
			}
			if _, ok := cache.Database("app"); ok != (mode != 0) {
				t.Fatalf("folded schema: %v", ok)
			}
			if got, ok := cache.SortedTablesByDBName("APP"); ok != (mode != 0) || (ok && !reflect.DeepEqual(got, []string{"Users"})) {
				t.Fatalf("folded schema tables: %v, %v", got, ok)
			}
			for _, names := range [][2]string{{"App", "users"}, {"app", "Users"}, {"APP", "USERS"}} {
				if _, ok := cache.ColumnDatabase(names[0], names[1]); ok != (mode != 0) {
					t.Errorf("columns %v: %v", names, ok)
				}
			}
			if col, ok := cache.Column("Users", "userid"); !ok || col.Name != "UserID" {
				t.Fatalf("column names must always ignore case: %v, %v", col, ok)
			}
		})
	}
}

func TestCacheCaseCollisions(t *testing.T) {
	for _, mode := range []int{-1, 0, 1, 2} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			cache := NewDBCache()
			cache.LowerCaseTableNames = mode
			for _, schema := range []string{"App", "app"} {
				cache.Schemas[schema] = schema
				cache.SchemaTables[schema] = []string{"Users", "users"}
				for _, table := range []string{"Users", "users"} {
					cache.ColumnsWithParent[columnDatabaseKey(schema, table)] = []*db.ColumnDesc{
						{ColumnBase: db.ColumnBase{Name: schema + "_" + table}},
					}
				}
			}
			if got := cache.SortedSchemas(); !reflect.DeepEqual(got, []string{"App", "app"}) {
				t.Fatalf("schemas were conflated: %v", got)
			}
			for _, schema := range []string{"App", "app"} {
				for _, table := range []string{"Users", "users"} {
					cols, ok := cache.ColumnDatabase(schema, table)
					if !ok || len(cols) != 1 || cols[0].Name != schema+"_"+table {
						t.Fatalf("wrong exact columns for %s.%s: %v", schema, table, cols)
					}
				}
				if _, ok := cache.ColumnDatabase(schema, "USERS"); ok {
					t.Fatal("ambiguous table should not resolve")
				}
			}
			if _, ok := cache.Database("APP"); ok {
				t.Fatal("ambiguous schema should not resolve")
			}
			if _, ok := cache.ColumnDatabase("APP", "Users"); ok {
				t.Fatal("ambiguous column schema should not resolve")
			}
		})
	}
}

func TestCacheDoesNotCrossExactSchema(t *testing.T) {
	cache := NewDBCache()
	cache.Schemas["App"] = "App"
	cache.Schemas["app"] = "app"
	cache.SchemaTables["app"] = []string{"users"}
	cache.ColumnsWithParent[columnDatabaseKey("app", "users")] = []*db.ColumnDesc{
		{ColumnBase: db.ColumnBase{Name: "id"}},
	}
	if _, ok := cache.SortedTablesByDBName("App"); ok {
		t.Fatal("must not use tables from differently cased schema")
	}
	if _, ok := cache.ColumnDatabase("App", "users"); ok {
		t.Fatal("must not use columns from differently cased schema")
	}
}
