package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/tui/content"
)

func (m model) sendQueryCmd() tea.Cmd {
	prompt := strings.TrimSpace(m.editor.GetCurrentContent())
	if prompt == "" {
		return nil
	}

	// Try LLM commands first
	if cmd := m.tryLLMCommands(prompt); cmd != nil {
		return cmd
	}

	// Try schema management commands
	if cmd := m.trySchemaCommands(prompt); cmd != nil {
		return cmd
	}

	// Try psql commands
	if strings.HasPrefix(prompt, "\\") {
		return m.executePsqlCommand(prompt)
	}

	// Default to SQL query execution
	return m.executeQuery(prompt)
}

// tryLLMCommands checks if the prompt is an LLM command and returns the appropriate command
func (m model) tryLLMCommands(prompt string) tea.Cmd {
	if llm.IsAskCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Ask)
	}

	if llm.IsExplainCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Explain)
	}

	if llm.IsOptimiseCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Optimise)
	}

	if llm.IsFixCommand(prompt) {
		m.focused = focusedContent
		if error := m.content.GetError(); error != nil {
			prompt += "\nError: " + error.Error()
		}
		return m.ask(prompt, llm.Fix)
	}

	return nil
}

// trySchemaCommands checks if the prompt is a schema management command
func (m model) trySchemaCommands(prompt string) tea.Cmd {
	if strings.HasPrefix(prompt, "/add") {
		return m.handleAddTables()
	}

	if strings.HasPrefix(prompt, "/remove") {
		return m.handleRemoveTables()
	}

	return nil
}

// handleAddTables processes the /add command
func (m model) handleAddTables() tea.Cmd {
	schema, err := m.addTablesSchemaToLLM()
	if err != nil {
		return utils.Dispatch(notificationErrorMsg{err: err})
	}

	return func() tea.Msg {
		message := "Table added to LLM schema"
		if len(m.llmSharedTablesSchema) > 1 {
			message = "Tables added to LLM schema"
		}

		return llmSharedSchemaMsg{
			schema:  schema,
			message: message,
			tables:  m.llmSharedTablesSchema,
		}
	}
}

// handleRemoveTables processes the /remove command
func (m model) handleRemoveTables() tea.Cmd {
	schema, err := m.removeTablesSchemaToLLM()
	if err != nil {
		return utils.Dispatch(notificationErrorMsg{err: err})
	}

	return func() tea.Msg {
		var message string
		switch len(m.llmSharedTablesSchema) {
		case 0:
			message = "All tables removed from LLM instructions"
		case 1:
			message = "Table removed from LLM schema"
		default:
			message = "Tables removed from LLM schema"
		}

		return llmSharedSchemaMsg{
			schema:  schema,
			message: message,
			tables:  m.llmSharedTablesSchema,
		}
	}
}

func (m model) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), DatabaseQueryTimeout)
		defer cancel()

		result, err := m.db.Query(ctx, query)
		if err != nil {
			return queryFailureMsg{err: err}
		}

		var queryResult content.ParsedQueryResult

		rows, columns, err := db.ExtractResults(result.Rows())
		if err != nil {
			return queryFailureMsg{err: err}
		}

		queryResult.Type = result.Type()
		queryResult.Query = result.Query()
		result.Rows().Close()
		queryResult.AffectedRows = result.Rows().CommandTag().RowsAffected()
		queryResult.Columns = columns
		queryResult.Rows = rows
		queryResult.ExecutionTime = result.ExecutionTime()

		return executeQueryMsg(queryResult)
	}
}

func (m model) handleQueryResult(msg executeQueryMsg) (tea.Model, tea.Cmd) {
	resetCmd := m.resetEditor()
	m.finishQueryExecution()

	err := m.content.SetQueryResults(content.ParsedQueryResult(msg))
	if err != nil {
		return m, nil
	}

	message := m.formatQuerySuccessMessage(msg.AffectedRows, msg.ExecutionTime)

	var schemaCmd tea.Cmd
	if msg.Type == db.QueryCreate ||
		msg.Type == db.QueryDrop ||
		msg.Type == db.QueryAlter {
		schemaCmd = m.generateSchema()
	}

	return m, tea.Batch(
		resetCmd,
		m.successNotification(message),
		schemaCmd,
	)
}

// formatQuerySuccessMessage creates a success message for query execution
func (m *model) formatQuerySuccessMessage(affectedRows int64, executionTime time.Duration) string {
	message := fmt.Sprintf("Query executed successfully. Affected rows: %d", affectedRows)
	if m.server.TimingEnabled {
		message += fmt.Sprintf(". Execution time: %s", utils.Duration(executionTime))
	}
	return message
}
