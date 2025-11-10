package export_data

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/ui/help"
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
	)
}

func (m Model) renderUsefulHelp() string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("<leader>c"),
			key.WithHelp("leader>c", "go back to the main view"),
		),
		keymap.ForceQuit,
		changeFocused,
		keymap.Editor,
	}

	title := styles.Text.Bold(true).Render("Useful Shortcuts")

	return title + help.RenderHelpView(m.width, bindings)
}

func (m Model) renderEditorHelp() string {
	commands := []struct {
		Command     string
		Description string
	}{
		{"esc", "back to normal mode"},
		{"i", "insert mode"},
		{"v", "visual mode (select text)"},
		{"V", "visual line mode (select text)"},
		{"y", "yank selected text (copy to clipboard)"},
		{"p", "paste (normal mode)"},
		{"u", "undo (normal mode)"},
		{"U", "redo (normal mode)"},
		{"d", "delete selected text"},
		{"dd/D", "delete row"},
		{":w", "saves the current file"},
		{":rename <file>", "renames the current file to <file>"},
		{":delete", "deletes the current file"},
		{"esc", "back to normal mode"},
	}

	title := styles.Text.Bold(true).Render("Editor")

	description := styles.Subtext1.Render(
		"These shortcuts are available when the editor is focused.",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderCmdHelp(m.width, commands),
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
