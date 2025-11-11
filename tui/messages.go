package tui

import (
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/pkg/update"
	"github.com/ionut-t/perp/tui/content"
)

// Schema-related messages
type schemaFetchedMsg string

type schemaFailureMsg struct {
	err error
}

// Query execution messages
type executeQueryMsg content.ParsedQueryResult

type queryFailureMsg struct {
	err error
}

// LLM-related messages
type llmResponseMsg llm.Response

type llmFailureMsg struct {
	err error
}

type llmSharedSchemaMsg struct {
	schema  string
	message string
	tables  []string
}

// PSQL command messages
type psqlCommandMsg struct {
	command *psql.Command
}

type psqlResultMsg struct {
	result *psql.Result
}

type psqlErrorMsg struct {
	err error
}

// PSQL toggle and control messages
type (
	toggleExpandedMsg struct{}
	toggleTimingMsg   struct{}
	showPsqlHelpMsg   struct{}
	psqlQuitMsg       struct{}
)

// Notification messages
type notificationErrorMsg struct {
	err error
}

// Update check messages
type updateAvailableMsg struct {
	release *update.LatestReleaseInfo
}
