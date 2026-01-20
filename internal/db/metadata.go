package db

import "database/sql"

type ColumnBase struct {
	Schema string
	Table  string
	Name   string
}

type ColumnDesc struct {
	ColumnBase
	Type    string
	Null    string
	Key     string
	Default sql.NullString
	Extra   string
}

type ForeignKey [][2]*ColumnBase

type TableInfo struct {
	Schema string
	Name   string
}
