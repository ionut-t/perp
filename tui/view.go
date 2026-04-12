package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/ionut-t/coffee/styles"
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
				m.styles.Error.Render(m.error.Error()),
				"\n",
				m.styles.Subtext0.Render("Press 'q' to go back to server selection"),
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

	h, _ := styles.ViewPadding.GetFrameSize()
	workWidth := m.width - h

	if m.focused == focusedCommand {
		commandLine = m.command.View()
	} else {
		commandLine = m.renderStatusBar(workWidth)
	}

	if m.notification != "" {
		commandLine = m.notification
	}

	editorBorder := m.styles.InactiveBorder
	if m.focused == focusedEditor {
		editorBorder = m.styles.ActiveBorder
	}

	contentBorder := m.styles.InactiveBorder
	if m.focused == focusedContent {
		contentBorder = m.styles.ActiveBorder
	}

	paneWidth := width + m.styles.ActiveBorder.GetHorizontalFrameSize()

	primaryView := lipgloss.JoinVertical(
		lipgloss.Left,
		editorBorder.Width(paneWidth).Render(
			m.editor.View(),
		),
		commandLine,
	)

	padding := lipgloss.NewStyle().Padding(0, 1)
	commandLineHeight := lipgloss.Height(commandLine)

	if m.fullScreen {
		if m.focused == focusedEditor {
			return padding.Render(primaryView)
		}

		fullScreenContentHeight := height - commandLineHeight

		fullScreenContentView := lipgloss.JoinVertical(
			lipgloss.Left,
			contentBorder.Width(paneWidth).
				Height(fullScreenContentHeight+m.styles.ActiveBorder.GetVerticalFrameSize()).
				Render(m.content.View()),
			commandLine,
		)
		return padding.Render(fullScreenContentView)
	}

	editorHeight := lipgloss.Height(m.editor.View())
	contentHeight := height - editorHeight - commandLineHeight

	if m.loading {
		return padding.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			contentBorder.Width(paneWidth).
				Height(contentHeight+m.styles.ActiveBorder.GetVerticalFrameSize()).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center).
				Render(
					m.spinner.View(),
				),
			primaryView))
	}

	return padding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		contentBorder.Width(paneWidth).
			Height(contentHeight+m.styles.ActiveBorder.GetVerticalFrameSize()).
			Render(m.content.View()),
		primaryView))
}

func (m *model) renderStatusBar(width int) string {
	bg := m.styles.Surface0.GetBackground()

	separator := m.styles.Surface0.Render(" | ")

	serverName := m.styles.Primary.Background(bg).Render(m.server.Name)

	database := m.styles.Accent.Background(bg).Render(m.server.Database)

	llm := lipgloss.NewStyle().Background(bg).Render(m.renderLLMModel())

	left := serverName + separator + database + separator + llm

	leftInfo := m.styles.Surface0.Padding(0, 1).Render(left)

	helpText := m.styles.Info.Background(bg).PaddingRight(1).Render("<leader>? Help")

	displayedInfoWidth := width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := m.styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return m.styles.Surface0.Width(width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func (m *model) getAvailableSizes() (int, int) {
	h, _ := styles.ViewPadding.GetFrameSize()
	borderV := m.styles.ActiveBorder.GetVerticalFrameSize()

	availableHeight := m.height - (2 * borderV)
	availableWidth := m.width - h - m.styles.ActiveBorder.GetHorizontalFrameSize()

	return availableWidth, availableHeight
}

func (m *model) renderLLMModel() string {
	llmModel, _ := m.config.GetLLMModel()

	if llmModel == "" {
		return m.styles.Subtext0.Render("No LLM model set")
	}

	return m.styles.Accent.Render(llmModel)
}
