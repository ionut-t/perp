package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/ui/styles"
)

func (m *model) renderDBError(width, height int) string {
	return lipgloss.NewStyle().
		Padding(0, 1).
		Height(height).
		Width(width).
		Border(lipgloss.RoundedBorder()).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.Error.Render(m.error.Error()),
				"\n",
				styles.Subtext0.Render("Press 'q' to go back to server selection"),
			),
		)
}

func (m *model) renderServers() string {
	return styles.ViewPadding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Height(m.height-lipgloss.Height(m.editor.View())-4).Render(
			m.serverSelection.View(),
		),
	))
}

func (m *model) renderMain(width, height int) string {
	var commandLine string

	if m.focused == focusedCommand {
		commandLine = m.command.View()
	} else {
		commandLine = m.renderStatusBar()
	}

	if m.notification != "" {
		commandLine = m.notification
	}

	editorBorder := styles.InactiveBorder
	if m.focused == focusedEditor {
		editorBorder = styles.ActiveBorder
	}

	contentBorder := styles.InactiveBorder
	if m.focused == focusedContent {
		contentBorder = styles.ActiveBorder
	}

	primaryView := lipgloss.JoinVertical(
		lipgloss.Left,
		editorBorder.Render(
			m.editor.View(),
		),
		commandLine,
	)

	contentHeight := height - lipgloss.Height(m.editor.View()) - lipgloss.Height(m.command.View()) - styles.ViewPadding.GetVerticalBorderSize()*2 - 2

	padding := lipgloss.NewStyle().Padding(1, 1, 0)

	if m.loading {
		return padding.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			contentBorder.Width(width).
				Height(contentHeight+2).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center).
				Render(
					m.spinner.View(),
				),
			primaryView))
	}

	return padding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		contentBorder.Width(width).
			Height(contentHeight).
			Render(m.content.View()),
		primaryView))
}

func (m *model) renderStatusBar() string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(m.server.Name)

	database := styles.Accent.Background(bg).Render(m.server.Database)

	llm := lipgloss.NewStyle().Background(bg).Render(m.renderLLMModel())

	left := serverName + separator + database + separator + llm

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := m.width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func (m *model) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	statusBarHeight := 1

	availableHeight := m.height - v - statusBarHeight - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h - styles.ActiveBorder.GetBorderLeftSize()

	return availableWidth, availableHeight
}

func (m *model) renderLLMModel() string {
	llmModel, _ := m.config.GetLLMModel()

	if llmModel == "" {
		return styles.Subtext0.Render("No LLM model set")
	}

	if m.server.ShareDatabaseSchemaLLM {
		return styles.Accent.Render(llmModel)
	}

	return styles.Accent.Render(llmModel)
}
