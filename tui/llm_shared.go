package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ionut-t/perp/pkg/utils"
)

// addTablesSchemaToLLM processes the `/add` command to include table schemas in the LLM context.
func (m *model) addTablesSchemaToLLM() (string, error) {
	if !m.server.ShareDatabaseSchemaLLM {
		return "", fmt.Errorf("cannot add tables to LLM schema when this feature is disabled")
	}

	value := strings.TrimSpace(strings.TrimPrefix(m.editor.GetCurrentContent(), "/add"))
	if value == "" {
		return "", fmt.Errorf("no tables specified to add")
	}

	tables := utils.ParseTableNames(value)
	if len(tables) == 0 {
		return "", fmt.Errorf("no valid table names provided")
	}

	var newTables []string
	for _, tableName := range tables {
		if !slices.Contains(m.llmSharedTablesSchema, tableName) {
			newTables = append(newTables, tableName)
		}
	}

	finalTableList := append(m.llmSharedTablesSchema, newTables...)

	schema, err := m.db.GenerateSchemaForTables(finalTableList)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema: %w", err)
	}

	if strings.TrimSpace(schema) == "" {
		return "", fmt.Errorf("no schema found for the specified tables; please check they exist")
	}

	m.llmSharedTablesSchema = finalTableList
	m.llm.ResetInstructions()

	m.llm.AppendInstructions("Database Schema:\n\n" + schema)

	return schema, nil
}

func (m *model) removeTablesSchemaToLLM() (string, error) {
	if !m.server.ShareDatabaseSchemaLLM {
		return "", nil
	}

	value := m.editor.GetCurrentContent()
	value = strings.TrimPrefix(value, "/remove")
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("no tables specified to remove from LLM schema")
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

	if len(tables) == 0 {
		return "", fmt.Errorf("no valid tables specified to remove from LLM schema")
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

	schema, err := m.db.GenerateSchemaForTables(m.llmSharedTablesSchema)
	if err != nil {
		return "", fmt.Errorf("failed to generate schema for tables: %w", err)
	}

	m.llm.ResetInstructions()
	m.llm.AppendInstructions(schema)

	return schema, nil
}
