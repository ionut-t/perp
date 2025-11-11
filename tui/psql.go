package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/tui/servers"
)

// Message types moved to messages.go

// executePsqlCommand parses and initiates execution of a psql command
func (m model) executePsqlCommand(input string) tea.Cmd {
	return func() tea.Msg {
		cmd, err := psql.Parse(input)
		if err != nil {
			return psqlErrorMsg{err: err}
		}

		switch cmd.Type {
		case psql.CmdToggleExpanded:
			return toggleExpandedMsg{}
		case psql.CmdToggleTiming:
			return toggleTimingMsg{}
		case psql.CmdHelp:
			return showPsqlHelpMsg{}
		case psql.CmdQuit:
			return psqlQuitMsg{}
		case psql.CmdConnect:
			if len(cmd.Arguments) > 0 {
				return m.connectToDatabase(cmd.Arguments[0])
			}
			return psqlErrorMsg{err: fmt.Errorf("\\c requires a database name")}
		}

		return psqlCommandMsg{command: cmd}
	}
}

// runPsqlCommand executes a psql command against the database
func (m model) runPsqlCommand(cmd *psql.Command) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), DatabaseQueryTimeout)
		defer cancel()

		executor := psql.New(m.db)
		result, err := executor.Execute(ctx, cmd)
		if err != nil {
			return psqlErrorMsg{err: err}
		}

		return psqlResultMsg{result: result}
	}
}

// connectToDatabase handles the \c command
func (m model) connectToDatabase(database string) tea.Msg {
	oldDatabase := m.server.Database

	if oldDatabase == database {
		return m.errorNotification(fmt.Errorf("already connected to database '%s'", database))
	}

	m.server.Database = database

	m.closeDbConnection()

	newDb, err := db.New(m.server.String())
	if err != nil {
		m.server.Database = oldDatabase
		return psqlErrorMsg{err: fmt.Errorf("failed to connect to database '%s': %w", database, err)}
	}

	m.db = newDb
	return servers.SelectedServerMsg{
		Server: m.server,
	}
}

func (m model) handlePsqlResult(msg psqlResultMsg) (tea.Model, tea.Cmd) {
	resetCmd := m.resetEditor()
	m.finishQueryExecution()

	var timingCmd tea.Cmd
	if m.server.TimingEnabled {
		timingCmd = m.successNotification(fmt.Sprintf("Execution time: %s", utils.Duration(msg.result.ExecutionTime)))
	}

	m.content.SetPsqlResult(msg.result)

	return m, tea.Batch(
		resetCmd,
		timingCmd,
	)
}

func (m model) toggleExpandedDisplay() (tea.Model, tea.Cmd) {
	m.loading = false
	m.expandedDisplay = !m.expandedDisplay
	m.content.SetExpandedDisplay(m.expandedDisplay)

	resetCmd := m.resetEditor()

	return m, tea.Batch(
		resetCmd,
		m.successNotification(fmt.Sprintf("Expanded display is %s", toggleStatus(m.expandedDisplay))),
	)
}

func (m model) toggleQueryTiming() (tea.Model, tea.Cmd) {
	m.loading = false
	oldValue := m.server.TimingEnabled

	if err := m.server.ToggleTiming(m.config.Storage()); err != nil {
		// Restore old value on error
		m.server.TimingEnabled = oldValue
		return m, m.errorNotification(err)
	}

	resetCmd := m.resetEditor()

	return m, tea.Batch(
		resetCmd,
		m.successNotification(fmt.Sprintf("Timing is %s", toggleStatus(m.server.TimingEnabled))),
	)
}
