package menu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mvgrimes/mytui/internal/tui/core"
)

type Model struct {
	Show          bool
	Index         int
	Type          core.MenuType
	Filter        string
	FavoriteNames []string
	FavoriteInput string
}

type Command struct {
	Label  string
	Action func() tea.Cmd
}
