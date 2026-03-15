package prompt

import (
	"path/filepath"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/tui/command"
)

type CancelMsg struct{}

type Action int

const (
	EditorAction Action = iota
	LLMModelAction
	ExportAllAsJSONAction
	ExportAllAsCSVAction
	ChangeLeaderKeyAction
	SaveSnippetAction
)

func (a Action) prompt() string {
	switch a {
	case EditorAction:
		return "Editor"
	case LLMModelAction:
		return "LLM Model"
	case ExportAllAsJSONAction, ExportAllAsCSVAction:
		return "Filename"
	case ChangeLeaderKeyAction:
		return "Leader key"
	case SaveSnippetAction:
		return "Snippet name"
	default:
		return "unknown"
	}
}

func (a Action) title() string {
	switch a {
	case EditorAction:
		return "Change external editor"
	case LLMModelAction:
		return "Change LLM model"
	case ExportAllAsJSONAction:
		return "Export all rows as JSON"
	case ExportAllAsCSVAction:
		return "Export all rows as CSV"
	case ChangeLeaderKeyAction:
		return "Change leader key"
	case SaveSnippetAction:
		return "Save current query as snippet"
	default:
		return "unknown"
	}
}

type Model struct {
	width, height int
	input         textinput.Model
	action        Action
	styles        styles.Styles
}

func New() Model {
	input := textinput.New()
	input.Prompt = "> "
	input.CharLimit = 256
	input.SetWidth(50)
	input.Focus()

	return Model{
		input: input,
	}
}

func (m Model) SetStyles(s styles.Styles) {
	m.input.Styles().Focused.Prompt.Foreground(s.Primary.GetForeground())
	m.input.Styles().Focused.Text.Foreground(s.Primary.GetForeground())
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetAction(action Action) {
	m.action = action
	m.input.Prompt = action.prompt() + ": "
}

func (m *Model) SetInitialValue(value string) {
	m.input.SetValue(value)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.input.SetValue("")
			return m, utils.Dispatch(CancelMsg{})
		case "enter":
			value := m.input.Value()

			if value == "" {
				return m, nil
			}

			m.input.SetValue("")
			return m, tea.Batch(
				m.handleAction(value),
				utils.Dispatch(CancelMsg{}),
			)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Primary.GetForeground()).
		Padding(1, 2)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.styles.Primary.Bold(true).MarginBottom(1).Render(m.action.title()),
		m.input.View(),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		border.Render(content),
	)
}

func (m Model) handleAction(value string) tea.Cmd {
	switch m.action {
	case EditorAction:
		return utils.Dispatch(command.EditorChangedMsg{Editor: value})

	case LLMModelAction:
		return utils.Dispatch(command.LLMModelChangedMsg{Model: value})

	case ExportAllAsJSONAction:
		extension := filepath.Ext(value)

		if extension != ".json" {
			value += ".json"
		}

		return utils.Dispatch(command.ExportMsg{
			All:      true,
			Filename: value,
		})

	case ExportAllAsCSVAction:
		extension := filepath.Ext(value)

		if extension != ".csv" {
			value += ".csv"
		}

		return utils.Dispatch(command.ExportMsg{
			All:      true,
			Filename: value,
		})

	case ChangeLeaderKeyAction:
		return utils.Dispatch(command.LeaderKeyChangedMsg{Key: value})

	case SaveSnippetAction:
		return utils.Dispatch(command.SaveSnippetMsg{Name: value})
	}

	return nil
}
