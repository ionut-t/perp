package statusbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/ui/styles"
)

func StatusBarView(server server.Server, width int) string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(server.Name)

	database := styles.Accent.Background(bg).Render(server.Database)

	serverInfo := styles.Surface0.Padding(0, 1).Render(
		serverName + separator + database,
	)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := width -
		lipgloss.Width(serverInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			serverInfo,
			spaces,
			helpText,
		),
	)
}
