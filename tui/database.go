package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/tui/servers"
)

// closeDbConnection safely closes the database connection
func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}
}

// generateSchema fetches the database schema
func (m model) generateSchema() tea.Cmd {
	return func() tea.Msg {
		schema, err := m.db.GenerateSchema()
		if err != nil {
			return schemaFailureMsg{err: err}
		}

		return schemaFetchedMsg(schema)
	}
}

// handleServerConnection processes server selection and establishes database connection
func (m *model) handleServerConnection(msg servers.SelectedServerMsg) (tea.Model, tea.Cmd) {
	m.closeDbConnection()
	m.view = viewMain
	m.focused = focusedEditor
	m.loading = true
	m.server = msg.Server
	m.db, m.error = db.New(m.server.String())

	if m.error == nil {
		m.content.SetConnectionInfo(m.server)

		if m.server.ShareDatabaseSchemaLLM {
			m.editor.SetPlaceholder("Type your SQL query or /ask your question here...")
		} else {
			m.editor.SetPlaceholder("Type your SQL query")
		}

		return m, m.generateSchema()
	}

	m.loading = false
	return m, nil
}
