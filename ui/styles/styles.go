package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	ViewPadding  = lipgloss.NewStyle().Padding(1, 1)
	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary.GetForeground())
	InactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Overlay0.
				GetForeground())
)
