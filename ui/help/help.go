package help

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/ui/styles"
)

type Model struct {
	viewport viewport.Model
}

func New() Model {
	vp := viewport.New(0, 0)

	return Model{
		viewport: vp,
	}
}

func (m *Model) SetSize(width, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m *Model) SetContent(helpText string) {
	m.viewport.SetContent(lipgloss.NewStyle().Padding(1, 1).Render(helpText))
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp

	return m, cmd
}

func (m Model) View() string {
	return m.viewport.View()
}

// RenderHelpView renders a help view for key bindings with their descriptions.
func RenderHelpView(width int, keys []key.Binding) string {
	var sb strings.Builder

	enabledBindings := make([]key.Binding, 0)
	maxKeyWidth := 0

	bg := lipgloss.NewStyle()
	bgColour := bg.GetBackground()

	for _, binding := range keys {
		if !binding.Enabled() {
			continue
		}

		enabledBindings = append(enabledBindings, binding)
		renderedWidth := lipgloss.Width(styles.Subtext1.Render(binding.Help().Key))
		maxKeyWidth = max(maxKeyWidth, renderedWidth)
	}

	for _, binding := range enabledBindings {
		keyText := binding.Help().Key
		renderedKey := styles.Info.Background(bgColour).Render(keyText)
		currentWidth := lipgloss.Width(renderedKey)
		padding := bg.Render(strings.Repeat(" ", maxKeyWidth-currentWidth+2))

		totalIndentation := 2 + lipgloss.Width(renderedKey) + max(0, maxKeyWidth-currentWidth+2)

		desc := strings.Split(binding.Help().Desc, "\n")

		var renderedDescription strings.Builder
		for i, line := range desc {
			renderedLine := styles.Text.Background(bgColour).Render(strings.TrimSpace(line))

			if i != 0 {
				indentPadding := bg.Render(strings.Repeat(" ", totalIndentation))
				renderedDescription.WriteString("\n" + indentPadding)
			}

			renderedDescription.WriteString(renderedLine)
		}

		sb.WriteString(fmt.Sprintf("• %s%s%s\n",
			renderedKey,
			padding,
			renderedDescription.String(),
		))
	}

	return bg.Width(width).Padding(1, 1).Render(strings.Trim(sb.String(), "\n"))
}

// RenderCmdHelp renders a help view for commands with their descriptions.
func RenderCmdHelp(width int, entries []struct {
	Command     string
	Description string
}) string {
	var sb strings.Builder
	maxKeyWidth := 0

	for _, entry := range entries {
		renderedWidth := lipgloss.Width(styles.Info.Render(entry.Command))
		maxKeyWidth = max(maxKeyWidth, renderedWidth)
	}

	for _, entry := range entries {
		renderedKey := styles.Info.Render(entry.Command)
		currentWidth := lipgloss.Width(renderedKey)
		padding := strings.Repeat(" ", max(0, maxKeyWidth-currentWidth+2))

		totalIndentation := 2 + lipgloss.Width(renderedKey) + maxKeyWidth - currentWidth + 2

		desc := strings.Split(entry.Description, "\n")

		var renderedDescription strings.Builder
		for i, line := range desc {
			renderedLine := styles.Text.Render(strings.TrimSpace(line))

			if i != 0 {
				indentPadding := strings.Repeat(" ", totalIndentation)
				renderedDescription.WriteString("\n" + indentPadding)
			}

			renderedDescription.WriteString(renderedLine)
		}

		sb.WriteString(fmt.Sprintf("• %s%s%s\n",
			renderedKey,
			padding,
			renderedDescription.String(),
		))
	}

	return lipgloss.NewStyle().Width(width).Padding(1, 1).Render(strings.Trim(sb.String(), "\n"))
}
