package styles

import (
	"github.com/charmbracelet/lipgloss"
	table "github.com/ionut-t/gotable"
)

func TableTheme() table.Theme {
	return table.Theme{
		Header:       AccentBackground.Bold(true),
		Cell:         Subtext0,
		Border:       Overlay0,
		SelectedRow:  Highlight.Bold(true),
		SelectedCell: lipgloss.NewStyle().Bold(true).Background(Text.GetForeground()).Foreground(AccentBackground.GetBackground()),
	}
}
