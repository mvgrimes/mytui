package core

import (
	"time"

	"charm.land/bubbles/v2/viewport"
	"github.com/mvgrimes/mytui/internal/db"
	"github.com/mvgrimes/mytui/internal/formatter"
)

type Focus int

const (
	FocusQuery Focus = iota
	FocusResults
)

// SQL templates for shortcuts
const (
	SQLTemplateInsert = "INSERT INTO  () VALUES ()"
	SQLTemplateSelect = "SELECT * FROM "
	SQLTemplateDelete = "DELETE FROM "
	SQLTemplateCreate = "CREATE TABLE  (\n)"
)

// Cursor offsets within templates (position after insertion)
const (
	SQLOffsetInsert = 12 // After "INSERT INTO "
	SQLOffsetSelect = 14 // After "SELECT * FROM "
	SQLOffsetDelete = 12 // After "DELETE FROM "
	SQLOffsetCreate = 13 // After "CREATE TABLE "
)

type Result struct {
	Query           string
	Timestamp       time.Time
	DisplaySize     int
	Expanded        bool
	DbResult        *db.Result
	Duration        time.Duration
	Formatted       string
	FormattedHeader string // Pinned header (first 3 lines of table)
	FormattedData   string // Scrollable data (remaining lines)
	Viewport        viewport.Model
	Format          formatter.Format
	XOffset         int    // Track horizontal scroll offset for pinned header
	SelectedRow     int    // -1 = no selection, 0 = first data row
	SearchActive    bool   // search prompt is open
	SearchQuery     string // last accepted search term (for n/N)
	SearchInput     string // text being typed in prompt
	PreSearchRow    int    // SelectedRow before search started (for Esc restore)
}

type MenuType int

const (
	MenuMain MenuType = iota
	MenuSaveFavorite
	MenuRunFavorite
)

// CopyFormat represents the format for copying a row
type CopyFormat int

const (
	CopyFormatCSV CopyFormat = iota
	CopyFormatTSV
	CopyFormatJSON
	CopyFormatVertical
	CopyFormatASCIITable
	CopyFormatUnicodeTable
)
