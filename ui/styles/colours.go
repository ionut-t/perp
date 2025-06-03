package styles

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
)

var (
	Base = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Base().Hex, Dark: catppuccin.Mocha.Base().Hex})

	Text = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Text().Hex, Dark: catppuccin.Mocha.Text().Hex})

	Primary = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Sapphire().Hex, Dark: catppuccin.Mocha.Sapphire().Hex})

	Accent = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Teal().Hex, Dark: catppuccin.Mocha.Teal().Hex})

	Success = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Green().Hex, Dark: catppuccin.Mocha.Green().Hex})

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Red().Hex, Dark: catppuccin.Mocha.Red().Hex})

	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Yellow().Hex, Dark: catppuccin.Mocha.Yellow().Hex})

	Info = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Blue().Hex, Dark: catppuccin.Mocha.Blue().Hex})

	Highlight = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Base().Hex, Dark: catppuccin.Latte.Base().Hex}).
			Background(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Surface0().Hex, Dark: catppuccin.Mocha.Surface0().Hex})

	Subtext0 = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Subtext0().Hex, Dark: catppuccin.Mocha.Subtext0().Hex})

	Subtext1 = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Subtext1().Hex, Dark: catppuccin.Mocha.Subtext1().Hex})

	Overlay0 = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Overlay0().Hex, Dark: catppuccin.Mocha.Overlay0().Hex})

	Overlay1 = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Overlay1().Hex, Dark: catppuccin.Mocha.Overlay1().Hex})

	Surface0 = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Surface0().Hex, Dark: catppuccin.Mocha.Surface0().Hex})

	Surface1 = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Surface1().Hex, Dark: catppuccin.Mocha.Surface1().Hex})

	Crust = lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: catppuccin.Latte.Crust().Hex, Dark: catppuccin.Mocha.Crust().Hex})
)
