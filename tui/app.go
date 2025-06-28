package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/constants"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	llmFactory "github.com/ionut-t/perp/pkg/llm/llm_factory"
	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
	exportStore "github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/tui/command"
	"github.com/ionut-t/perp/tui/content"
	exportData "github.com/ionut-t/perp/tui/export_data"
	"github.com/ionut-t/perp/tui/servers"
	"github.com/ionut-t/perp/ui/help"
	"github.com/ionut-t/perp/ui/styles"
)

type schemaFetchedMsg string
type schemaFailureMsg struct {
	err error
}

type llmResponseMsg llm.Response
type llmFailureMsg struct {
	err error
}

type executeQueryMsg content.ParsedQueryResult
type queryFailureMsg struct {
	err error
}

type llmSharedSchemaMsg struct {
	schema  string
	message string
	tables  []string
}

type notificationErrorMsg struct {
	err error
}

type view int

const (
	viewServers view = iota
	viewMain
	viewExportData
	viewHelp
)

type focused int

const (
	focusedNone focused = iota
	focusedEditor
	focusedContent
	focusedCommand
)

type model struct {
	config          config.Config
	width, height   int
	view            view
	focused         focused
	serverSelection servers.Model
	server          server.Server
	db              db.Database
	error           error
	llm             llm.LLM
	llmError        error
	editor          editor.Model

	loading bool
	spinner spinner.Model

	queryResults          []map[string]any
	exportData            exportData.Model
	command               command.Model
	notification          string
	content               content.Model
	help                  help.Model
	llmSharedTablesSchema []string

	// styles
	sqlKeywords  map[string]lipgloss.Style
	llmKeywords  map[string]lipgloss.Style
	psqlCommands map[string]lipgloss.Style

	// commands
	expandedDisplay bool
	timingEnabled   bool

	// history management
	historyLogs           []history.HistoryLog
	currentHistoryIndex   int
	historyNavigating     bool
	originalEditorContent string
}

func New(config config.Config) model {
	editor := editor.New(80, 10)
	editor.ShowMessages(false)
	editor.SetCursorBlinkMode(true)

	sqlKeywordsMap := make(map[string]lipgloss.Style, len(constants.SQL_KEYWORDS)*2)

	for _, keyword := range constants.SQL_KEYWORDS {
		highlighted := styles.Primary.Bold(true)
		sqlKeywordsMap[strings.ToUpper(keyword)] = highlighted
		sqlKeywordsMap[strings.ToLower(keyword)] = highlighted
	}

	editor.SetHighlightedWords(sqlKeywordsMap)

	llmKeywordsMap := make(map[string]lipgloss.Style, len(llm.LLMKeywords))
	for _, keyword := range llm.LLMKeywords {
		llmKeywordsMap[keyword] = styles.Accent.Bold(true)
	}

	psqlCommands := make(map[string]lipgloss.Style, len(psql.PSQL_COMMANDS))

	for cmd := range psql.PSQL_COMMANDS {
		psqlCommands[cmd] = styles.Primary.Bold(true)
	}

	editor.SetPlaceholder("Type your SQL query here...")

	editor.Focus()
	editor.DisableCommandMode(true)
	editor.WithTheme(styles.EditorTheme())

	historyLogs, err := history.Get(config.Storage())
	if err != nil {
		historyLogs = []history.HistoryLog{}
	}

	instructions, err := config.GetLLMInstructions()

	if err != nil || instructions == "" {
		instructions = constants.LLMDefaultInstructions
	}

	llm, err := llmFactory.New(config, instructions)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Primary

	return model{
		config:          config,
		llm:             llm,
		editor:          editor,
		sqlKeywords:     sqlKeywordsMap,
		llmKeywords:     llmKeywordsMap,
		psqlCommands:    psqlCommands,
		command:         command.New(),
		serverSelection: servers.New(config.Storage()),
		historyLogs:     historyLogs,
		content:         content.New(0, 0),
		help:            help.New(),
		llmError:        err,
		spinner:         sp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("perp"),
		m.spinner.Tick,
		m.editor.CursorBlink(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		width, height := m.getAvailableSizes()

		m.editor.SetSize(width, max(height/2-4, 10))

		contentHeight := height - lipgloss.Height(m.editor.View()) - lipgloss.Height(m.command.View())

		m.content.SetSize(width, contentHeight)
		m.help.SetSize(msg.Width, msg.Height)
		m.help.SetContent(m.renderHelp())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.historyNavigating && m.editor.IsFocused() && m.focused == focusedEditor {
			// Check if it's a character input (not a special key)
			if len(msg.String()) == 1 || msg.Type == tea.KeySpace {
				// User is typing, exit history navigation
				m.resetHistory()
			}
		}

		if msg.Type == tea.KeyCtrlC {
			m.closeDbConnection()
			return m, tea.Quit
		}

		if m.focused == focusedCommand || m.view == viewServers || m.view == viewExportData {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit):
			if m.error != nil {
				m.serverSelection = servers.New(m.config.Storage())
				_, cmd := m.serverSelection.Update(nil)

				m.view = viewServers
				m.error = nil
				return m, cmd
			}

			if m.focused == focusedContent && m.content.IsViewChangeRequired() {
				break
			}

			if m.view == viewHelp {
				m.view = viewMain
				m.focused = focusedEditor
				m.editor.Focus()
				break
			}

			if m.editor.IsNormalMode() {
				m.closeDbConnection()
				return m, tea.Quit
			}

		case key.Matches(msg, changeFocused):
			if m.view == viewMain && !m.editor.IsInsertMode() {
				switch m.focused {
				case focusedEditor:
					m.focused = focusedContent
					m.editor.Blur()
				case focusedContent:
					m.focused = focusedEditor
					m.editor.Focus()
				}
				_, cmd := m.content.Update(nil)

				return m, tea.Batch(
					cmd,
					m.editor.CursorBlink(),
				)
			}

		case key.Matches(msg, enterCommand):
			if m.view == viewMain && m.editor.IsNormalMode() {
				m.focused = focusedCommand
				m.editor.Blur()

				ed, cmd := m.editor.Update(nil)
				m.editor = ed.(editor.Model)

				return m, tea.Batch(
					m.command.Focus(),
					cmd,
				)
			}

		case key.Matches(msg, accessDBSchema):
			if m.editor.IsNormalMode() {
				m.focused = focusedContent
				m.editor.Blur()
				m.content.ShowDBSchema()
				c, cmd := m.content.Update(nil)
				m.content = c.(content.Model)
				return m, cmd
			}

		case key.Matches(msg, accessLLMSharedSchema):
			if m.editor.IsNormalMode() {
				m.focused = focusedContent
				m.editor.Blur()
				m.content.ShowLLMSharedSchema()
				c, cmd := m.content.Update(nil)
				m.content = c.(content.Model)
				return m, cmd
			}

		case key.Matches(msg, accessServers):
			if m.editor.IsNormalMode() {
				m.serverSelection = servers.New(m.config.Storage())
				_, cmd := m.serverSelection.Update(nil)

				m.view = viewServers
				m.error = nil
				return m, cmd
			}

		case key.Matches(msg, keymap.Insert):
			if m.view == viewMain && m.focused == focusedContent {
				m.focused = focusedEditor
				m.editor.Focus()
				m.editor.SetInsertMode()

				_, cmd := m.editor.Update(nil)

				return m, tea.Batch(
					cmd,
					m.editor.CursorBlink(),
				)
			}

		case key.Matches(msg, keymap.Submit):
			if m.editor.IsInsertMode() || m.editor.IsNormalMode() {
				content := m.editor.GetCurrentContent()

				if content == "" {
					break
				}

				trimmedContent := strings.TrimSpace(content)
				isLLMCommand := false
				for _, prefix := range llm.LLMKeywords {
					if strings.HasPrefix(trimmedContent, prefix) {
						isLLMCommand = true
						break
					}
				}

				if !isLLMCommand && strings.HasSuffix(content, ";") {
					if !m.loading {
						m.loading = true
						m.resetHistory()
						m.addToHistory()
						return m, m.sendQueryCmd()
					}
				}
			}

		case key.Matches(msg, executeQuery):
			if !m.loading {
				m.loading = true
				m.resetHistory()
				m.addToHistory()

				return m, m.sendQueryCmd()
			}

		case key.Matches(msg, keymap.Cancel):
			if m.view == viewMain && m.focused == focusedEditor {
				m.resetHistory()

				if m.editor.IsNormalMode() {
					if m.editor.IsFocused() {
						m.focused = focusedContent
						m.editor.Blur()
					}
				}
			}

		case key.Matches(msg, previousHistory):
			if m.editor.IsFocused() && len(m.historyLogs) > 0 {
				m.previousHistory()
			}

		case key.Matches(msg, nextHistory):
			if m.editor.IsFocused() && m.historyNavigating {
				m.nextHistory()
			}

		case key.Matches(msg, accessExportedData):
			if m.focused == focusedContent {
				m.view = viewExportData
				storage := filepath.Join(m.config.Storage(), m.server.Name)
				exportStore := exportStore.New(storage, m.config.Editor())
				m.exportData = exportData.New(exportStore, m.server, m.width, m.height)
			}

		case key.Matches(msg, viewLLMLogs):
			if m.focused == focusedContent {
				m.content.ShowLLMLogs()
			}

		case key.Matches(msg, keymap.Help):
			if m.editor.IsInsertMode() {
				break
			}

			switch m.view {
			case viewMain:
				m.view = viewHelp
				m.editor.Blur()
			case viewHelp:
				m.view = viewMain
				m.focused = focusedEditor
				m.editor.Focus()
			}
		}

	case servers.SelectedServerMsg:
		m.closeDbConnection()
		m.view = viewMain
		m.focused = focusedEditor
		m.loading = true
		m.server = msg.Server
		m.db, m.error = db.New(m.server.ConnectionString())
		if m.error == nil {
			m.content.SetConnectionInfo(m.server)

			if m.server.ShareDatabaseSchemaLLM {
				m.editor.SetPlaceholder("Type your SQL query or /ask your question here...")
			} else {
				m.editor.SetPlaceholder("Type your SQL query")
			}

			return m, m.generateSchema()
		}
		m.loading = false

	case utils.ClearMsg:
		m.notification = ""

	case schemaFetchedMsg:
		schema := string(msg)
		m.loading = false

		m.content.SetSchema(schema)

	case schemaFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case executeQueryMsg:
		m.loading = false
		m.editor.SetContent("")

		err := m.content.SetQueryResults(content.ParsedQueryResult(msg))

		if err != nil {
			return m, nil
		}

		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		message := fmt.Sprintf("Query executed successfully. Affected rows: %d", msg.AffectedRows)

		if m.timingEnabled {
			message += fmt.Sprintf(". Execution time: %s", utils.Duration(msg.ExecutionTime))
		}

		return m, tea.Batch(
			cmd,
			m.successNotification(message),
		)

	case queryFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case psqlCommandMsg:
		m.loading = true
		return m, m.runPsqlCommand(msg.command)

	case psqlResultMsg:
		m.loading = false
		m.editor.SetContent("")

		var timingCmd tea.Cmd
		if m.timingEnabled {
			timingCmd = m.successNotification(fmt.Sprintf("Execution time: %s", utils.Duration(msg.result.ExecutionTime)))
		}

		queryResult := content.ParsedQueryResult{
			Type:    db.QuerySelect,
			Query:   msg.result.Message,
			Columns: msg.result.Columns,
			Rows:    msg.result.Rows,
		}

		err := m.content.SetQueryResults(queryResult)
		if err != nil {
			return m, nil
		}

		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			timingCmd,
		)

	case psqlErrorMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case toggleExpandedMsg:
		m.loading = false
		m.expandedDisplay = !m.expandedDisplay
		// TODO: Implement expanded display in content
		// m.content.SetExpandedDisplay(m.expandedDisplay)
		status := "OFF"
		if m.expandedDisplay {
			status = "ON"
		}
		return m, m.successNotification(fmt.Sprintf("Expanded display is %s", status))

	case toggleTimingMsg:
		m.loading = false
		m.timingEnabled = !m.timingEnabled
		status := "OFF"
		if m.timingEnabled {
			status = "ON"
		}

		m.editor.SetContent("")

		return m, m.successNotification(fmt.Sprintf("Timing is %s", status))

	case showPsqlHelpMsg:
		m.content.ShowPsqlHelp()
		m.editor.SetContent("")
		return m, nil

	case llmResponseMsg:
		m.loading = false
		query := strings.TrimSpace(m.editor.GetCurrentContent())
		m.content.SetLLMLogs(llm.Response(msg), query)
		m.editor.SetContent("")
		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)
		return m, cmd

	case llmFailureMsg:
		m.loading = false
		query := m.editor.GetCurrentContent()

		for _, keyword := range llm.LLMKeywords {
			query = strings.TrimPrefix(query, keyword)
		}

		query = strings.TrimSpace(query)
		m.content.SetLLMLogsError(msg.err, query)

	case llmSharedSchemaMsg:
		m.loading = false
		m.editor.SetContent("")
		m.content.SetLLMSharedSchema(msg.schema)
		m.llmSharedTablesSchema = msg.tables
		m.content.SetLLMSharedTables(m.llmSharedTablesSchema)

		return m, m.successNotification(msg.message)

	case notificationErrorMsg:
		m.loading = false
		return m, m.errorNotification(msg.err)

	case content.LLMResponseSelectedMsg:
		m.editor.SetContent(msg.Response)
		m.editor.Focus()
		m.editor.SetInsertMode()
		_ = m.editor.SetCursorPositionEnd()
		m.view = viewMain
		m.focused = focusedEditor
		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			m.editor.CursorBlink(),
		)

	case exportData.ClosedMsg:
		m.view = viewMain
		m.editor.SetContent("")
		m.editor.Focus()

		if m.queryResults == nil {
			m.content.SetConnectionInfo(m.server)
		}

	case command.QuitMsg, psqlQuitMsg:
		m.closeDbConnection()
		return m, tea.Quit

	case command.CancelMsg:
		m.focused = focusedEditor
		m.editor.Focus()

	case command.ExportMsg:
		return m.handleDataExport(msg)

	case command.EditorChangedMsg:
		err := m.config.SetEditor(msg.Editor)
		if err != nil {
			return m, m.errorNotification(err)
		}

		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.successNotification(
			fmt.Sprintf("Editor changed to %s", msg.Editor),
		)

	case command.LLMUseDatabaseSchemaMsg:
		if m.llmError != nil {
			return m, m.errorNotification(fmt.Errorf("LLM is not configured: %w", m.llmError))
		}

		done := func() {
			m.content.SetConnectionInfo(m.server)
			m.focused = focusedEditor
			m.editor.Focus()
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

	case command.LLMModelChangedMsg:
		if m.llmError != nil {
			return m, m.errorNotification(fmt.Errorf("LLM is not configured: %w", m.llmError))
		}

		existingModel, _ := m.config.GetLLMModel()
		if existingModel == msg.Model {
			return m, m.successNotification("LLM model is already set to " + msg.Model)
		}

		m.llm.SetModel(msg.Model)
		if _, err := m.llm.Ask("Test LLM model", llm.Ask); err != nil {
			m.llm.SetModel(existingModel)
			return m, m.errorNotification(fmt.Errorf("invalid LLM model: %v", msg.Model))
		}

		if err := m.config.SetLLMModel(msg.Model); err != nil {
			return m, m.errorNotification(err)
		}

		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.successNotification("LLM model changed to " + msg.Model)

	case command.ErrorMsg:
		return m, m.errorNotification(msg.Err)
	}

	var cmds []tea.Cmd

	m.editor.SetHighlightedWords(m.setHighlightedKeywords())

	if m.view == viewMain && m.focused == focusedEditor {
		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel.(editor.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewServers {
		s, cmd := m.serverSelection.Update(msg)
		m.serverSelection = s.(servers.Model)
		cmds = append(cmds, cmd)
	}

	if m.focused == focusedContent {
		contentModel, cmd := m.content.Update(msg)
		m.content = contentModel.(content.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewExportData {
		exportDataModel, cmd := m.exportData.Update(msg)
		m.exportData = exportDataModel.(exportData.Model)
		cmds = append(cmds, cmd)
	}

	if m.focused == focusedCommand {
		cmdModel, cmd := m.command.Update(msg)
		m.command = cmdModel.(command.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewHelp {
		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel.(help.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	width, height := m.getAvailableSizes()

	if m.error != nil {
		return m.renderDBError(width, height)
	}

	switch m.view {
	case viewServers:
		return m.renderServers()

	case viewMain:
		return m.renderMain(width, height)

	case viewExportData:
		return m.exportData.View()

	case viewHelp:
		return m.help.View()
	}

	return ""
}

func (m model) generateSchema() tea.Cmd {
	return func() tea.Msg {
		schema, err := m.db.GenerateSchema()
		if err != nil {
			return schemaFailureMsg{err: err}
		}

		return schemaFetchedMsg(schema)
	}
}

func (m model) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := m.db.Query(ctx, query)
		if err != nil {
			return queryFailureMsg{err: err}
		}

		var queryResult content.ParsedQueryResult

		rows, columns, err := db.ExtractResultsFromRows(result.Rows())

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

func (m model) setHighlightedKeywords() map[string]lipgloss.Style {
	if strings.HasPrefix(m.editor.GetCurrentContent(), "/") {
		return m.llmKeywords
	}

	if strings.HasPrefix(m.editor.GetCurrentContent(), "\\") {
		return m.psqlCommands
	}

	return m.sqlKeywords
}

func (m model) sendQueryCmd() tea.Cmd {
	prompt := m.editor.GetCurrentContent()

	if prompt == "" {
		return nil
	}

	prompt = strings.TrimSpace(prompt)

	if strings.HasPrefix(prompt, "/ask") {
		m.focused = focusedContent

		return m.ask(prompt, llm.Ask)
	}

	if strings.HasPrefix(prompt, "/explain") {
		m.focused = focusedContent

		return m.ask(prompt, llm.Explain)
	}

	if strings.HasPrefix(prompt, "/optimise") {
		m.focused = focusedContent

		return m.ask(prompt, llm.Optimise)
	}

	if strings.HasPrefix(prompt, "/fix") {
		m.focused = focusedContent

		if m.error != nil {
			prompt += "\nError: " + m.error.Error()
		}

		return m.ask(prompt, llm.Fix)
	}

	if strings.HasPrefix(prompt, "/add") {
		schema, err := m.addTablesSchemaToLLM()

		if err != nil {
			m.editor.SetContent("")

			return utils.Dispatch(notificationErrorMsg{err: err})
		}

		return func() tea.Msg {
			var message string
			if len(m.llmSharedTablesSchema) == 1 {
				message = "Table added to LLM schema"
			} else {
				message = "Tables added to LLM schema"
			}

			return llmSharedSchemaMsg{schema: schema, message: message, tables: m.llmSharedTablesSchema}
		}
	}

	if strings.HasPrefix(prompt, "/remove") {
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

			return llmSharedSchemaMsg{schema: schema, message: message, tables: m.llmSharedTablesSchema}
		}
	}

	if strings.HasPrefix(prompt, "\\") {
		return m.executePsqlCommand(prompt)
	}

	return m.executeQuery(prompt)
}

func (m model) handleDataExport(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	queryResults := m.content.GetQueryResults()

	data, err := utils.HandleDataExport(queryResults, msg.Rows, msg.All)

	if err != nil {
		m.focused = focusedEditor
		m.editor.Focus()

		return m, m.errorNotification(err)
	}

	storage := filepath.Join(m.config.Storage(), m.server.Name)
	fileName, err := export.AsJson(storage, data, msg.Filename)

	if err != nil {
		return m, m.errorNotification(err)
	}

	m.focused = focusedEditor
	m.editor.Focus()
	m.command.Reset()

	return m, m.successNotification(
		fmt.Sprintf("Data exported successfully to %s.json", fileName),
	)
}

func (m *model) successNotification(msg string) tea.Cmd {
	m.notification = styles.Success.Render(msg)

	return utils.ClearAfter(time.Second * 2)
}

func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = styles.Error.Render(err.Error())

	return utils.ClearAfter(time.Second * 2)
}

func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}
}
