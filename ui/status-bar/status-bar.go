package statusbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/ui/styles"
)

func StatusBarView(server server.Server, llmModel string, width int) string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(server.Name)

	database := styles.Accent.Background(bg).Render(server.Database)

	var llmModelDisplay string

	left := serverName + separator + database

	if llmModel != "" {
		llmModelDisplay = styles.Accent.Background(bg).Render(llmModel)
		left += separator + llmModelDisplay
	}

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}
