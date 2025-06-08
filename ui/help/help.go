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

		totalIndentation := 2 + lipgloss.Width(renderedKey) + maxKeyWidth - currentWidth + 2

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
