package splitview

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/ui/help"
)

// RenderCommonEditorHelp returns the standard vim-like editor help section
func RenderCommonEditorHelp(width int) string {
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
		help.RenderCmdHelp(width, commands),
	)
}

// RenderCommonListHelp returns the standard list navigation help section
func RenderCommonListHelp(width int, listModel list.Model) string {
	bindings := []key.Binding{
		listModel.KeyMap.CursorDown,
		listModel.KeyMap.CursorUp,
		listModel.KeyMap.Filter,
	}

	title := styles.Text.Bold(true).Render("List Navigation")

	description := styles.Subtext1.Render(
		"These shortcuts are available when the list is focused.",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(width, bindings),
	)
}

// RenderCommonUsefulHelp returns a standard "Useful Shortcuts" section
func RenderCommonUsefulHelp(width int, additionalBindings []key.Binding) string {
	title := styles.Text.Bold(true).Render("Useful Shortcuts")
	return title + help.RenderHelpView(width, additionalBindings)
}
