package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/tui/command"
	"github.com/ionut-t/perp/tui/content"
)

// ask sends a query to the LLM
func (m model) ask(prompt string, cmd llm.Command) tea.Cmd {
	return func() tea.Msg {
		if m.llmError != nil {
			return llmFailureMsg{err: fmt.Errorf("LLM is not configured: %w", m.llmError)}
		}

		response, err := m.llm.Ask(prompt, cmd)
		if err != nil {
			return llmFailureMsg{err: err}
		}

		return llmResponseMsg(*response)
	}
}

// handleLLMResponse processes LLM responses
func (m model) handleLLMResponse(msg llmResponseMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	query := strings.TrimSpace(m.editor.GetCurrentContent())
	m.content.SetLLMLogs(llm.Response(msg), query)

	if msg.Command == llm.Optimise || msg.Command == llm.Fix {
		content := llm.ExtractQuery(string(msg.Response))
		m.editor.SetContent(content)
		m.editor.SetNormalMode()
		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)
		return m, cmd
	}

	m.focusContent()
	return m, m.resetEditor()
}

// updateSharedSchema updates the LLM shared schema state
func (m model) updateSharedSchema(msg llmSharedSchemaMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.content.SetLLMSharedSchema(msg.schema)
	m.llmSharedTablesSchema = msg.tables
	m.content.SetLLMSharedTables(m.llmSharedTablesSchema)

	resetCmd := m.resetEditor()

	return m, tea.Batch(
		resetCmd,
		m.successNotification(msg.message),
	)
}

// applyLLMResponse applies an LLM response to the editor
func (m model) applyLLMResponse(msg content.LLMResponseSelectedMsg) (tea.Model, tea.Cmd) {
	return m, m.applyQueryToEditor(msg.Response)
}

// updateLLMModel changes the LLM model
func (m model) updateLLMModel(msg command.LLMModelChangedMsg) (tea.Model, tea.Cmd) {
	if err := m.requireLLM(); err != nil {
		return m, m.errorNotification(err)
	}

	existingModel, _ := m.config.GetLLMModel()
	if existingModel == msg.Model {
		return m, m.successNotification("LLM model is already set to " + msg.Model)
	}

	if err := m.llm.SetModel(msg.Model); err != nil {
		return m, m.errorNotification(fmt.Errorf("invalid LLM model: %w", err))
	}

	if err := m.config.SetLLMModel(msg.Model); err != nil {
		return m, m.errorNotification(err)
	}

	m.focusEditor()
	return m, m.successNotification("LLM model changed to " + msg.Model)
}

// toggleDBSchemaSharing enables or disables database schema sharing with LLM
func (m model) toggleDBSchemaSharing(msg command.LLMUseDatabaseSchemaMsg) (tea.Model, tea.Cmd) {
	if err := m.requireLLM(); err != nil {
		return m, m.errorNotification(err)
	}

	done := func() {
		m.content.SetConnectionInfo(m.server)
		m.focusEditor()
	}

	if m.server.ShareDatabaseSchemaLLM == msg.Enabled {
		done()
		return m, m.successNotification("No change in LLM database schema usage")
	}

	if err := m.server.EnableDatabaseSchemaLLM(msg.Enabled, m.config.Storage()); err != nil {
		return m, m.errorNotification(err)
	}

	if msg.Enabled {
		done()
		return m, m.successNotification("LLM will now use the database schema")
	}

	done()
	m.llm.ResetInstructions()
	m.llmSharedTablesSchema = []string{}
	m.content.SetLLMSharedSchema("")
	m.content.SetLLMSharedTables(m.llmSharedTablesSchema)
	return m, m.successNotification("LLM will no longer use the database schema")
}
