package psql

import (
	"fmt"
	"strings"
)

// CommandType represents the type of psql command
type CommandType int

const (
	CmdUnknown CommandType = iota
	CmdDescribe
	CmdDescribeTable
	CmdListTables
	CmdListViews
	CmdListIndexes
	CmdListFunctions
	CmdListSchemas
	CmdListDatabases
	CmdConnect
	CmdToggleExpanded
	CmdToggleTiming
	CmdHelp
	CmdListUsers
	CmdListSequences
	CmdListForeignTables
	CmdQuit
)

// Command represents a parsed psql command
type Command struct {
	Type      CommandType
	Arguments []string
	Raw       string
}

// PostgreSQL command string constants
const (
	PSQL_Describe              = "\\d"
	PSQL_ListTables            = "\\dt"
	PSQL_ListTablesPlus        = "\\dt+"
	PSQL_ListViews             = "\\dv"
	PSQL_ListViewsPlus         = "\\dv+"
	PSQL_ListIndexes           = "\\di"
	PSQL_ListIndexesPlus       = "\\di+"
	PSQL_ListFunctions         = "\\df"
	PSQL_ListFunctionsPlus     = "\\df+"
	PSQL_ListSchemas           = "\\dn"
	PSQL_ListSchemasPlus       = "\\dn+"
	PSQL_ListSequences         = "\\ds"
	PSQL_ListSequencesPlus     = "\\ds+"
	PSQL_ListForeignTables     = "\\dE"
	PSQL_ListForeignTablesPlus = "\\dE+"
	PSQL_ListUsers             = "\\du"
	PSQL_ListUsersPlus         = "\\du+"
	PSQL_ListDatabases         = "\\l"
	PSQL_ListDatabasesPlus     = "\\l+"
	PSQL_ListDatabasesAlt      = "\\list"
	PSQL_Connect               = "\\c"
	PSQL_ConnectAlt            = "\\connect"
	PSQL_ToggleExpanded        = "\\x"
	PSQL_ToggleTiming          = "\\timing"
	PSQL_Help                  = "\\h"
	PSQL_HelpAlt               = "\\help"
	PSQL_HelpPsql              = "\\?"
	PSQL_ExecuteFile           = "\\i"
	PSQL_Quit                  = "\\q"
)

// PostgreSQL command mappings
var PSQL_COMMANDS = map[string]CommandType{
	PSQL_Describe:              CmdDescribe,
	PSQL_ListTables:            CmdListTables,
	PSQL_ListTablesPlus:        CmdListTables,
	PSQL_ListViews:             CmdListViews,
	PSQL_ListViewsPlus:         CmdListViews,
	PSQL_ListIndexes:           CmdListIndexes,
	PSQL_ListIndexesPlus:       CmdListIndexes,
	PSQL_ListFunctions:         CmdListFunctions,
	PSQL_ListFunctionsPlus:     CmdListFunctions,
	PSQL_ListSchemas:           CmdListSchemas,
	PSQL_ListSchemasPlus:       CmdListSchemas,
	PSQL_ListSequences:         CmdListSequences,
	PSQL_ListSequencesPlus:     CmdListSequences,
	PSQL_ListForeignTables:     CmdListForeignTables,
	PSQL_ListForeignTablesPlus: CmdListForeignTables,
	PSQL_ListUsers:             CmdListUsers,
	PSQL_ListUsersPlus:         CmdListUsers,

	// Database listing
	PSQL_ListDatabases:     CmdListDatabases,
	PSQL_ListDatabasesPlus: CmdListDatabases,
	PSQL_ListDatabasesAlt:  CmdListDatabases,

	// Connection
	PSQL_Connect:    CmdConnect,
	PSQL_ConnectAlt: CmdConnect,

	// Toggle commands
	// PSQL_ToggleExpanded: CmdToggleExpanded,
	PSQL_ToggleTiming: CmdToggleTiming,

	// Help commands
	PSQL_Help:     CmdHelp,
	PSQL_HelpAlt:  CmdHelp,
	PSQL_HelpPsql: CmdHelp,

	PSQL_Quit: CmdQuit,
}

// Commands that require arguments
var commandsRequiringArgs = map[CommandType]bool{
	CmdConnect: true,
}

// CommandDescriptions holds all command descriptions in their defined order.
var CommandDescriptions = []struct {
	Command     string
	Description string
}{
	// Describe and List commands
	{PSQL_Describe, "List tables, views, and sequences"},
	{PSQL_ListTables, "List tables"},
	{PSQL_ListTablesPlus, "List tables with additional information"},
	{PSQL_ListViews, "List views"},
	{PSQL_ListViewsPlus, "List views with additional information"},
	{PSQL_ListIndexes, "List indexes"},
	{PSQL_ListIndexesPlus, "List indexes with additional information"},
	{PSQL_ListFunctions, "List functions"},
	{PSQL_ListFunctionsPlus, "List functions with additional information"},
	{PSQL_ListSchemas, "List schemas"},
	{PSQL_ListSchemasPlus, "List schemas with additional information"},
	{PSQL_ListSequences, "List sequences"},
	{PSQL_ListSequencesPlus, "List sequences with additional information"},
	{PSQL_ListForeignTables, "List foreign tables"},
	{PSQL_ListForeignTablesPlus, "List foreign tables with additional information"},
	{PSQL_ListUsers, "List users and roles"},
	{PSQL_ListUsersPlus, "List users and roles with additional information"},
	{PSQL_ListDatabases, "List databases"},
	{PSQL_ListDatabasesPlus, "List databases with additional information"},
	{PSQL_ListDatabasesAlt, "List databases (alternative syntax)"},

	// Connection commands
	{PSQL_Connect, "Connect to database"},
	{PSQL_ConnectAlt, "Connect to a database (alternative syntax)"},

	// Toggle commands
	// {PSQL_ToggleExpanded, "Toggle expanded output"},
	{PSQL_ToggleTiming, "Toggle timing of commands"},

	// Help commands
	{PSQL_Help, "Show help"},
	{PSQL_HelpAlt, "Show help (alternative syntax)"},
	{PSQL_HelpPsql, "Show psql help"},

	// File execution
	// {PSQL_ExecuteFile, "Execute commands from a file"},

	// Quit command
	{PSQL_Quit, "Quit"},
}

// String returns a string representation of the command type
func (c CommandType) String() string {
	switch c {
	case CmdDescribe:
		return "describe"
	case CmdDescribeTable:
		return "describe-table"
	case CmdListTables:
		return "list-tables"
	case CmdListViews:
		return "list-views"
	case CmdListIndexes:
		return "list-indexes"
	case CmdListFunctions:
		return "list-functions"
	case CmdListSchemas:
		return "list-schemas"
	case CmdListDatabases:
		return "list-databases"
	case CmdConnect:
		return "connect"
	case CmdToggleExpanded:
		return "toggle-expanded"
	case CmdToggleTiming:
		return "toggle-timing"
	case CmdHelp:
		return "help"
	case CmdListUsers:
		return "list-users"
	case CmdListSequences:
		return "list-sequences"
	case CmdListForeignTables:
		return "list-foreign-tables"
	case CmdQuit:
		return "quit"
	default:
		return "unknown"
	}
}

// IsExtended returns true if the command includes the + modifier
func (c *Command) IsExtended() bool {
	cmd := strings.Trim(c.Raw, ";")
	cmd = strings.TrimSpace(cmd)

	return strings.HasSuffix(cmd, "+")
}

// Parse parses a psql command string
func Parse(input string) (*Command, error) {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, "\\") {
		return nil, fmt.Errorf("not a psql command: %s", input)
	}

	parts := strings.Fields(strings.TrimSuffix(input, ";"))

	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := &Command{
		Raw:       input,
		Arguments: parts[1:],
	}

	// Special handling for \d command with arguments
	if parts[0] == PSQL_Describe {
		if len(parts) > 1 {
			cmd.Type = CmdDescribeTable
		} else {
			cmd.Type = CmdDescribe
		}
		return cmd, nil
	}

	// Look up command type in the map
	cmdType, exists := PSQL_COMMANDS[parts[0]]
	if !exists {
		cmd.Type = CmdUnknown
		return nil, fmt.Errorf("unknown command: %s", parts[0])
	}

	cmd.Type = cmdType

	// Validate required arguments
	if requiresArgs, exists := commandsRequiringArgs[cmdType]; exists && requiresArgs {
		if len(cmd.Arguments) == 0 {
			return nil, fmt.Errorf("%s requires a database name", parts[0])
		}
	}

	return cmd, nil
}
