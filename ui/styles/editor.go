package styles

import (
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
)

func EditorTheme() editor.Theme {
	normalModeBG := Info.GetForeground()
	insertModeBG := Primary.GetForeground()
	visualModeBG := Accent.GetForeground()
	commandModeBG := Warning.GetForeground()

	return editor.Theme{
		NormalModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(normalModeBG).
			Bold(true),

		InsertModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(insertModeBG).
			Bold(true),

		VisualModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(visualModeBG).
			Bold(true),

		CommandModeStyle: lipgloss.NewStyle().
			Foreground(Base.GetForeground()).
			Background(commandModeBG).
			Bold(true),

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

		HighlightYankStyle: Highlight.
			Bold(true),

		PlaceholderStyle: Overlay0,
	}
}
