package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/pkg/utils"
)

// addTablesSchemaToLLM processes the `/add` command to include table schemas in the LLM context.
func (m *model) addTablesSchemaToLLM() (string, error) {
	if err := m.validateLLMSchemaSharing(); err != nil {
		return "", err
	}

	tables, err := m.parseAddTablesCommand()
	if err != nil {
		return "", err
	}

	newTables := m.filterNewTables(tables)
	finalTableList := append(m.llmSharedTablesSchema, newTables...)

	schema, err := m.updateLLMWithSchema(finalTableList)
	if err != nil {
		return "", err
	}

	m.llmSharedTablesSchema = finalTableList
	return schema, nil
}

// validateLLMSchemaSharing checks if LLM is ready and schema sharing is enabled
func (m *model) validateLLMSchemaSharing() error {
	if err := m.requireLLM(); err != nil {
		return err
	}

	if !m.server.ShareDatabaseSchemaLLM {
		return fmt.Errorf("cannot add tables to LLM schema when this feature is disabled")
	}

	return nil
}

// parseAddTablesCommand extracts and validates table names from the /add command
func (m *model) parseAddTablesCommand() ([]string, error) {
	value := strings.TrimSpace(strings.TrimPrefix(m.editor.GetCurrentContent(), "/add"))
	if value == "" {
		return nil, fmt.Errorf("no tables specified to add")
	}

	tables := utils.ParseTableNames(value)
	if len(tables) == 0 {
		return nil, fmt.Errorf("no valid table names provided")
	}

	return tables, nil
}

// filterNewTables returns only tables not already in the shared schema
func (m *model) filterNewTables(tables []string) []string {
	var newTables []string
	for _, tableName := range tables {
		if !slices.Contains(m.llmSharedTablesSchema, tableName) {
			newTables = append(newTables, tableName)
		}
	}
	return newTables
}

// updateLLMWithSchema generates schema for tables and updates LLM instructions
func (m *model) updateLLMWithSchema(tables []string) (string, error) {
	schema, err := m.generateSchemaForTables(tables)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema: %w", err)
	}

	if strings.TrimSpace(schema) == "" {
		return "", fmt.Errorf("no schema found for the specified tables; please check they exist")
	}

	m.llm.ResetInstructions()
	m.llm.AppendInstructions("Database Schema:\n\n" + schema)

	return schema, nil
}

func (m *model) removeTablesSchemaToLLM() (string, error) {
	if err := m.requireLLM(); err != nil {
		return "", err
	}

	value := m.editor.GetCurrentContent()
	value = strings.TrimPrefix(value, "/remove")
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("no tables specified to remove from LLM schema")
	}

	if !m.server.ShareDatabaseSchemaLLM {
		return "", nil
	}

	if value == "*" {
		m.llmSharedTablesSchema = []string{}
		m.llm.ResetInstructions()

		return "", nil
	}

	tables := utils.ParseTableNames(value)
	if len(tables) == 0 {
		return "", fmt.Errorf("no valid table names provided")
	}

	for _, tableName := range tables {
		idx := slices.Index(m.llmSharedTablesSchema, tableName)

		if idx > -1 {
			m.llmSharedTablesSchema = slices.Delete(m.llmSharedTablesSchema, idx, idx+1)
		}
	}

	if len(m.llmSharedTablesSchema) == 0 {
		m.llm.ResetInstructions()
		return "", nil
	}

	schema, err := m.generateSchemaForTables(m.llmSharedTablesSchema)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema for tables: %w", err)
	}

	m.llm.ResetInstructions()
	m.llm.AppendInstructions(schema)

	return schema, nil
}

// generateSchemaForTables uses the psql executor to describe tables.
// It returns schema information for the LLM context.
func (m *model) generateSchemaForTables(tables []string) (string, error) {
	// No tables specified means no schema to generate, which is not an error here.
	if len(tables) == 0 {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabaseQueryTimeout)
	defer cancel()

	executor := psql.New(m.db)
	var sb strings.Builder

	for i, tableName := range tables {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Validate table name to prevent SQL injection
		// Uses the canonical validator from pkg/psql
		if _, err := psql.SanitiseIdentifier(tableName); err != nil {
			return "", fmt.Errorf("tui: invalid table name %q: %w", tableName, err)
		}

		// Parse the \d command for this table
		cmd, err := psql.Parse(fmt.Sprintf("\\d %s", tableName))
		if err != nil {
			return "", fmt.Errorf("tui: failed to parse describe command for table %q: %w", tableName, err)
		}

		// Execute the command
		result, err := executor.Execute(ctx, cmd)
		if err != nil {
			return "", fmt.Errorf("tui: failed to execute describe command for table %q: %w", tableName, err)
		}

		// Format the result as text
		sb.WriteString(fmt.Sprintf("Table: %s\n", tableName))

		if len(result.Rows) == 0 {
			sb.WriteString("  (no columns found)\n")
			continue
		}

		// Write columns section
		sb.WriteString("Columns:\n")
		for _, row := range result.Rows {
			// Check type assertions with ok boolean for robustness
			column, ok := row["Column"].(string)
			if !ok {
				return "", fmt.Errorf("tui: failed to get 'Column' (string) from psql result for table %q", tableName)
			}
			colType, ok := row["Type"].(string)
			if !ok {
				return "", fmt.Errorf("tui: failed to get 'Type' (string) from psql result for table %q", tableName)
			}
			// Modifiers might legitimately be empty
			modifiers, _ := row["Modifiers"].(string)

			sb.WriteString(fmt.Sprintf("  %s %s", column, colType))
			if modifiers != "" {
				sb.WriteString(fmt.Sprintf(" %s", modifiers))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
