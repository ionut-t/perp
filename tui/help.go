package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	table "github.com/ionut-t/gotable"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/ui/help"
	"github.com/ionut-t/perp/ui/styles"
)

var yankCell = key.NewBinding(
	key.WithKeys("y"),
	key.WithHelp("y", "yank selected cell"),
)

var yankRow = key.NewBinding(
	key.WithKeys("Y"),
	key.WithHelp("Y", "yank selected row (copies selected row as JSON to clipboard)"),
)

var previousCell = key.NewBinding(
	key.WithKeys("left", "h"),
	key.WithHelp("← / h", "previous cell"),
)

var nextCell = key.NewBinding(
	key.WithKeys("right", "l"),
	key.WithHelp("→ / l", "next cell"),
)

var changeFocused = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "change focus between editor and main content"),
)

var executeQuery = key.NewBinding(
	key.WithKeys("alt+enter", "ctrl+s"),
	key.WithHelp("alt+enter/ctrl+s", "execute query"),
)

var accessExportedData = key.NewBinding(
	key.WithKeys("g"),
	key.WithHelp("g", "manage exported data (available when the editor is not focused)"),
)

var accessDBSchema = key.NewBinding(
	key.WithKeys("S"),
	key.WithHelp("S", "view database schema (available when the editor is not focused)"),
)

var accessServers = key.NewBinding(
	key.WithKeys("|"),
	key.WithHelp("|", "view servers (available when the editor is not focused)"),
)

var previousHistory = key.NewBinding(
	key.WithKeys("shift+up"),
	key.WithHelp("shift+↑", "previous history log"),
)

var nextHistory = key.NewBinding(
	key.WithKeys("shift+down"),
	key.WithHelp("shift+↓", "next history log"),
)

var enterCommand = key.NewBinding(
	key.WithKeys(":"),
	key.WithHelp(":", "enter command mode (available when the editor is not focused)"),
)

func (m model) renderHelp() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderUsefulHelp(),
		m.renderEditorHelp(),
		m.renderTableHelp(),
		m.renderCommandHelp(),
	)
}

func (m model) renderUsefulHelp() string {
	bindings := []key.Binding{
		keymap.Quit,
		keymap.ForceQuit,
		changeFocused,
		keymap.Help,
		keymap.Insert,
		accessExportedData,
		accessDBSchema,
		accessServers,
		enterCommand,
	}

	title := styles.Text.Bold(true).Render("Useful Shortcuts")

	return title + help.RenderHelpView(m.width, bindings)
}

func (m model) renderEditorHelp() string {
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
		previousHistory,
		nextHistory,
		executeQuery,
	}

	title := styles.Text.Bold(true).Render("Editor")

	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			"These shortcuts are available when the editor is focused.",
		),
		styles.Subtext1.Render(
			"If query ends with a semicolon, it will be executed automatically when enter is pressed.",
		),
		styles.Subtext1.Render(
			"If query starts with ",
		)+styles.Accent.Render("/ask")+
			styles.Subtext1.Render(", it will send a request to the LLM when submitted."),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(m.width, bindings),
	)
}

func (m model) renderTableHelp() string {
	tableKeyMap := table.DefaultKeyMap()
	bindings := []key.Binding{
		tableKeyMap.Down,
		tableKeyMap.Up,
		tableKeyMap.Left,
		tableKeyMap.Right,
		previousCell,
		nextCell,
		key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("← / h", "previous cell"),
		),
		tableKeyMap.PageDown,
		tableKeyMap.PageUp,
		tableKeyMap.Home,
		tableKeyMap.End,
		yankCell,
		yankRow,
	}

	title := styles.Text.Bold(true).Render("Table")
	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			"It is accessible when a query that returns data is executed.",
		),
		styles.Subtext1.Render(
			"These shortcuts are available when the table is focused.",
		),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderHelpView(m.width, bindings),
	)
}

func (m model) renderCommandHelp() string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("export all <file>", `export all data returned by the query as JSON to a file
						 Example:
						 export all data
						 it exports to data.json (extension is not required);
						 if the file already exists, it will create a new file with unique name derived from the input name.
						 `,
			),
		),
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("export <rows> <file>", `export specific rows to a file
						 Example:
						 export 1,2,3 data.json
						 it exports rows 1,2,3 to data.json;
						 if the file already exists, it will create a new file with unique name derived from the input name.
						 `,
			),
		),
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("set-editor <editor>", "sets the external editor to use for editing configuration or exported data"),
		),
		key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("llm-schema <bool>", `enables or disables the usage of database schema in LLM queries;
						if the value is not provided, it will be set as true.
						 `),
		),
	}

	title := styles.Text.Bold(true).Render("Command Palette")

	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			"These commands are available when the editor is not focused.",
		),
		styles.Subtext1.Render(
			"You can access the command palette by pressing ",
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
