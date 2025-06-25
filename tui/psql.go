package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/tui/servers"
)

type psqlCommandMsg struct {
	command *psql.Command
}

type psqlResultMsg struct {
	result *psql.Result
}

type psqlErrorMsg struct {
	err error
}

type toggleExpandedMsg struct{}
type toggleTimingMsg struct{}
type showPsqlHelpMsg struct{}
type psqlQuitMsg struct{}

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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	newDb, err := db.New(m.server.ConnectionString())
	if err != nil {
		m.server.Database = oldDatabase
		return psqlErrorMsg{err: fmt.Errorf("failed to connect to database '%s': %w", database, err)}
	}

	m.db = newDb
	return servers.SelectedServerMsg{
		Server: m.server,
	}
}
