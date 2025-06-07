package styles

import (
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
)

func EditorTheme() editor.Theme {
	normalModeBG := Primary.GetForeground()
	insertModeBG := Success.GetForeground()
	visualModeBG := Accent.GetForeground()
	commandModeBG := Warning.GetForeground()

	return editor.Theme{
		NormalModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(normalModeBG),

		InsertModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(insertModeBG),

		VisualModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(visualModeBG),

		CommandModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(commandModeBG),

		CommandLineStyle: Surface0,

		StatusLineStyle: Surface1.
			Foreground(Subtext0.GetForeground()),

		MessageStyle: Info,
		ErrorStyle:   Error,

		LineNumberStyle: Subtext0.
			Width(4).
			Align(lipgloss.Right),

		CurrentLineNumberStyle: Text.
			Width(4).
			Align(lipgloss.Right),

		SelectionStyle: Surface1,

		HighlighYankStyle: Highlight.
			Bold(true),

		PlaceholderStyle: Overlay0,
	}
}
