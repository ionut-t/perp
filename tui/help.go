package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	table "github.com/ionut-t/gotable"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/psql"
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
	key.WithHelp("alt+enter/ctrl+s", "execute query (no matter the editor mode)"),
)

var accessExportedData = key.NewBinding(
	key.WithKeys("g"),
	key.WithHelp("g", "manage exported data (available when the editor is not focused)"),
)

var accessDBSchema = key.NewBinding(
	key.WithKeys("S"),
	key.WithHelp("S", "view database schema (available when the editor is not focused)"),
)

var accessLLMSharedSchema = key.NewBinding(
	key.WithKeys("s"),
	key.WithHelp("s", "view LLM shared schema (available when the editor is not focused)"),
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

var viewLLMLogs = key.NewBinding(
	key.WithKeys("}"),
	key.WithHelp("}", "view LLM logs (available when the editor is not focused)"),
)

var viewHistoryEntries = key.NewBinding(
	key.WithKeys("ctrl+r"),
	key.WithHelp("ctrl+r", "view history logs"),
)

func (m model) renderHelp() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderUsefulHelp(),
		m.renderLLMHelp(),
		m.renderEditorHelp(),
		m.renderPsqlHelp(),
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
		accessLLMSharedSchema,
		accessServers,
		enterCommand,
		viewLLMLogs,
		viewHistoryEntries,
	}

	title := styles.Text.Bold(true).Render("Useful Shortcuts")

	return title + help.RenderHelpView(m.width, bindings)
}

func (m model) renderLLMHelp() string {
	commands := []struct {
		Command     string
		Description string
	}{
		{"/ask", `sends a query to the LLM
				 Example:
				 /ask join all users with their orders and return the user name, email and order total
				 `},
		{"-- EXPLAIN", `explains the provided query
				 Example:
				 -- EXPLAIN	
				 SELECT * FROM users;
				 `},
		{"-- OPTIMISE", `optimises the provided query
				 Example:
				 -- OPTIMISE
				 SELECT * FROM users WHERE id IN (SELECT user_id FROM orders WHERE created_at > '2024-01-01');
				 `},
		{"-- FIX", `fixes the provided query
				 Example:
				 -- FIX
				 SELECT name, COUNT(*) FROM users JOIN orders ON users.id = orders.user_id;
				 `},
		{"/add", `adds tables to the LLM instructions
				 Example:
				 /add users, orders
				 `},
		{"/remove", `removes tables from the LLM instructions
				 Example:
				 /remove users, orders
				 /remove * -> removes all tables from the LLM instructions
				 `},
	}

	title := styles.Text.Bold(true).Render("LLM Commands")

	description := styles.Subtext1.Render(
		styles.Wrap(m.width-1, "These commands are available when the editor is in INSERT mode."),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderCmdHelp(m.width, commands),
	)
}

func (m model) renderEditorHelp() string {
	commands := []struct {
		Command     string
		Description string
	}{
		{"i", "insert mode"},
		{"v", "visual mode (select text)"},
		{"V", "visual line mode (select text)"},
		{"y", "yank selected text (copy to clipboard)"},
		{"p", "paste (normal mode)"},
		{"u", "undo (normal mode)"},
		{"U", "redo (normal mode)"},
		{"d", "delete selected text"},
		{"dd/D", "delete row"},
		{"enter", "new line (insert mode) / execute query (normal mode)"},
		{"esc", "back to normal mode"},
		{"alt+enter/ctrl+s", "execute query (no matter the editor mode)"},
	}

	title := styles.Text.Bold(true).Render("Editor")

	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			styles.Wrap(m.width-1, "These shortcuts are available when the editor is focused."),
		),
		styles.Subtext1.Render(
			styles.Wrap(m.width-1, "If the editor is in NORMAL mode, the query will be executed automatically when enter is pressed."),
		),
		styles.Wrap(m.width-1, styles.Subtext1.Render(
			"If query starts with ",
		)+styles.Accent.Render("/ask")+
			styles.Subtext1.Render(", it will send a request to the LLM when submitted.")),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderCmdHelp(m.width, commands),
	)
}

// Helper to render psql help
func (m *model) renderPsqlHelp() string {
	title := styles.Text.Bold(true).Render("PSQL Commands (experimental)")

	description := styles.Subtext1.Render(
		styles.Wrap(m.width-1, "These commands are available when the editor is focused."),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderCmdHelp(m.width, psql.CommandDescriptions),
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
			styles.Wrap(m.width-1, "It is accessible when a query that returns data is executed."),
		),
		styles.Subtext1.Render(
			styles.Wrap(m.width-1, "These shortcuts are available when the table is focused."),
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
	commands := []struct {
		Command     string
		Description string
	}{
		{"export * <file>", `export all data returned by the query as JSON/CSV to a file
						 Example:
						 export * data.json
						 it exports to data.json
						 if the file already exists, it will create a new file with unique name derived from the	 input name
						 `},
		{"export <rows> <file>", `export specific rows to a file
						 Example:
						 export 1,2,3 data.json
						 it exports rows 1,2,3 to data.json;
						 if the file already exists, it will create a new file with unique name derived from the	 input name
						 `},
		{"set-editor <editor>", `sets the external editor to use for editing configuration or exported data
						 Example:
						 set-editor vim
						 `},
		{"llm-db-schema-enable", `enables the usage of database schema in LLM queries for the current server
						 Example:
						 llm-db-schema-enable
						 `},
		{"llm-db-schema-disable", `disables the usage of database schema in LLM queries for the current server
						 Example:
						 llm-db-schema-disable
						 `},
		{"llm-model <model>", `sets the LLM model to use for queries
						Example:
						llm-model gemini-1.5-flash
						`},
	}

	title := styles.Text.Bold(true).Render("Command Palette")

	description := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.Subtext1.Render(
			styles.Wrap(m.width-1, "These commands are available when the editor is not focused."),
		),
		styles.Wrap(m.width-1, styles.Subtext1.Render(
			"You can access the command palette by pressing ",
		)+styles.Accent.Render(":")+
			styles.Subtext1.Render(".")),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		description,
		help.RenderCmdHelp(m.width, commands),
	)
}
