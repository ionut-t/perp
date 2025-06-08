package export_data

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/ui/help"
	"github.com/ionut-t/perp/ui/styles"
)

var changeFocused = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "change focus between editor and list"),
)

func (m Model) renderHelp() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderUsefulHelp(),
		m.renderListHelp(),
		m.renderEditorHelp(),
		m.renderCommandHelp(),
	)
}

func (m Model) renderUsefulHelp() string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "go back to the main view (available when the editor is not focused)"),
		),
		keymap.ForceQuit,
		changeFocused,
		keymap.Help,
		keymap.Insert,
		keymap.Editor,
	}

	title := styles.Text.Bold(true).Render("Useful Shortcuts")

	return title + help.RenderHelpView(m.width, bindings)
}

func (m Model) renderEditorHelp() string {
	bindings := []key.Binding{
		keymap.Insert,
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "new line (insert mode)"),
		),
		key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to normal mode"),
		),
		key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual mode (select text)"),
		),
		key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "visual line mode (select text)"),
		),
		key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "paste (normal mode)"),
		),
		key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "undo (normal mode)"),
		),
		key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "redo (normal mode)"),
		),
		key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete selected text"),
		),
		key.NewBinding(
			key.WithKeys("dd"),
			key.WithHelp("dd/D", "delete row"),
		),
	}

	title := styles.Text.Bold(true).Render("Editor")

	description := styles.Subtext1.Render(
		"These shortcuts are available when the editor is focused.",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(m.width, bindings),
	)
}

func (m Model) renderListHelp() string {
	bindings := []key.Binding{
		m.list.KeyMap.CursorDown,
		m.list.KeyMap.CursorUp,
		m.list.KeyMap.Filter,
	}

	title := styles.Text.Bold(true).Render("List Navigation")

	description := styles.Subtext1.Render(
		"These shortcuts are available when the list is focused.",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(m.width, bindings),
	)
}

func (m Model) renderCommandHelp() string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("rename <file>", "renames the current file to <file>"),
		),
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("delete", "deletes the current file"),
		),
	}

	title := styles.Text.Bold(true).Render("Command Mode")

	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			"These commands are available when the editor is focused.",
		),
		styles.Subtext1.Render(
			"You can execute commands by pressing ",
		)+styles.Accent.Render(":")+
			styles.Subtext1.Render("."),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(m.width, bindings),
	)
}
