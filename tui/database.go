package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/lsp"
	"github.com/ionut-t/perp/tui/servers"
)

// closeDbConnection safely closes the database connection and LSP client.
func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}

	if m.lspClient != nil {
		m.lspClient.Close()
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
	m.lspClient = nil
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

		return m, tea.Batch(m.generateSchema(), m.startLSP())
	}

	m.loading = false

	return m, m.spinner.Tick
}

// startLSP starts the postgres-language-server subprocess asynchronously.
func (m *model) startLSP() tea.Cmd {
	return func() tea.Msg {
		binPath, ok := lsp.FindBinary()
		if !ok {
			return nil
		}

		client, err := lsp.New(binPath, m.server)
		if err != nil {
			return lspFailedMsg{err: err}
		}

		if err := client.DidOpen(""); err != nil {
			client.Close()
			return lspFailedMsg{err: err}
		}

		return lspConnectedMsg{client: client}
	}
}
