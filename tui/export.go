package tui

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/tui/command"
)

// exportQueryData handles export requests for query results
func (m model) exportQueryData(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	if filepath.Ext(msg.Filename) != ".json" && filepath.Ext(msg.Filename) != ".csv" {
		return m, m.errorNotification(
			fmt.Errorf("invalid file extension: %s. Supported extensions are .json and .csv", msg.Filename),
		)
	}

	if filepath.Ext(msg.Filename) == ".csv" {
		return m.exportAsCSV(msg)
	}

	return m.exportAsJSON(msg)
}

// exportAsJSON exports query results as JSON
func (m model) exportAsJSON(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	queryResults := m.content.GetQueryResults()

	data, err := export.PrepareJSON(queryResults, msg.Rows, msg.All)
	if err != nil {
		m.focusEditor()
		return m, m.errorNotification(err)
	}

	storage := filepath.Join(m.config.Storage(), m.server.Name)
	fileName, err := export.AsJson(storage, data, msg.Filename)
	if err != nil {
		return m, m.errorNotification(err)
	}

	m.focusEditor()
	m.command.Reset()

	return m, m.successNotification(
		fmt.Sprintf("Data exported successfully as JSON to %s", fileName),
	)
}

// exportAsCSV exports query results as CSV
func (m model) exportAsCSV(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	queryResults := m.content.GetQueryResults()

	data, err := export.PrepareCSV(queryResults, msg.Rows, msg.All)
	if err != nil {
		m.focusEditor()
		return m, m.errorNotification(err)
	}

	storage := filepath.Join(m.config.Storage(), m.server.Name)
	fileName, err := export.AsCsv(storage, data, msg.Filename)
	if err != nil {
		return m, m.errorNotification(err)
	}

	m.focusEditor()
	m.command.Reset()

	return m, m.successNotification(
		fmt.Sprintf("Data exported successfully as CSV to %s", fileName),
	)
}
