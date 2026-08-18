package modals

import "charm.land/bubbles/v2/viewport"

type Model struct {
	RowDetail     RowDetailModel
	CopyMenu      CopyMenuModel
	HistorySearch HistorySearchModel
}

type RowDetailModel struct {
	Show     bool
	Viewport viewport.Model
}

type CopyMenuModel struct {
	Show  bool
	Index int
}

type HistorySearchModel struct {
	Show   bool
	Filter string
	Index  int
	Scroll int
}
