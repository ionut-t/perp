package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/goeditor/adapter-bubbletea/editor"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/constants"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/gemini"
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
	config                config.Config
	width, height         int
	view                  view
	focused               focused
	serverSelection       servers.Model
	server                server.Server
	db                    db.Database
	error                 error
	loading               bool
	llm                   llm.LLM
	editor                editor.Model
	sqlKeywords           map[string]lipgloss.Style
	llmKeywords           map[string]lipgloss.Style
	queryResults          []map[string]any
	exportData            exportData.Model
	command               command.Model
	notification          string
	content               content.Model
	help                  help.Model
	llmSharedTablesSchema []string

	// history management
	historyLogs           []history.HistoryLog
	currentHistoryIndex   int
	historyNavigating     bool
	originalEditorContent string
}

func New(config config.Config) model {
	llmApiKey, _ := config.GetLLMApiKey()
	llmModel, _ := config.GetLLMModel()

	editor := editor.New(80, 10)
	editor.ShowMessages(false)
	editor.SetCursorBlinkMode(true)

	sqlKeywordsMap := make(map[string]lipgloss.Style, len(SQL_KEYWORDS)*2)

	for _, keyword := range SQL_KEYWORDS {
		highlighted := styles.Accent.Bold(true)
		sqlKeywordsMap[strings.ToUpper(keyword)] = highlighted
		sqlKeywordsMap[strings.ToLower(keyword)] = highlighted
	}

	editor.SetHighlightedWords(sqlKeywordsMap)

	llmKeywordsMap := map[string]lipgloss.Style{
		"/ask":    styles.Info.Bold(true),
		"/add":    styles.Info.Bold(true),
		"/remove": styles.Info.Bold(true),
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

	return model{
		config:          config,
		llm:             gemini.New(llmApiKey, llmModel, instructions),
		editor:          editor,
		sqlKeywords:     sqlKeywordsMap,
		llmKeywords:     llmKeywordsMap,
		command:         command.New(),
		serverSelection: servers.New(config.Storage()),
		historyLogs:     historyLogs,
		content:         content.New(0, 0),
		help:            help.New(),
	}
}

func (m model) Init() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		width, height := m.getAvailableSizes()

		m.editor.SetSize(width, max(height/2-5, 10))

		contentHeight := height - lipgloss.Height(m.editor.View()) - lipgloss.Height(m.command.View())

		m.content.SetSize(width, contentHeight)
		m.help.SetSize(msg.Width, msg.Height)
		m.help.SetContent(m.renderHelp())

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

				m.resetHistory()

				isLLMCommand := strings.HasPrefix(content, "/ask") ||
					strings.HasPrefix(content, "/add") ||
					strings.HasPrefix(content, "/remove")

				if !isLLMCommand && strings.HasSuffix(content, ";") && len(content) > 5 {
					m.addToHistory()
					return m, m.sendQueryCmd()
				}
			}

		case key.Matches(msg, executeQuery):
			m.resetHistory()
			m.addToHistory()

			return m, m.sendQueryCmd()

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

	case utils.ClearNotificationMsg:
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

		return m, tea.Batch(
			cmd,
			m.successNotification(
				fmt.Sprintf("Rows affected: %d", msg.AffectedRows),
			),
		)

	case queryFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case llmResponseMsg:
		m.loading = false
		query := strings.TrimPrefix(m.editor.GetCurrentContent(), "/ask")
		query = strings.TrimSpace(query)
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
		query := strings.Trim(m.editor.GetCurrentContent(), "/ask")
		m.content.SetLLMLogsError(msg.err, query)

	case llmSharedSchemaMsg:
		m.editor.SetContent("")
		m.content.SetLLMSharedSchema(msg.schema)
		m.llmSharedTablesSchema = msg.tables

		return m, m.successNotification(msg.message)

	case notificationErrorMsg:
		return m, m.errorNotification(msg.err)

	case content.LLMResponseSelectedMsg:
		m.editor.SetContent(msg.Response)
		m.editor.Focus()
		m.editor.SetInsertMode()
		m.editor.SetCursorPositionEnd()
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

	case command.QuitMsg:
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
		focusEditor := func() {
			m.focused = focusedEditor
			m.editor.Focus()
		}

		if m.server.ShareDatabaseSchemaLLM == msg.Enabled {
			focusEditor()
			return m, m.successNotification("No change in LLM database schema usage")
		}

		if err := m.server.EnableDatabaseSchemaLLM(msg.Enabled, m.config.Storage()); err != nil {
			return m, m.errorNotification(err)
		}

		if msg.Enabled {
			focusEditor()
			return m, m.successNotification("LLM will now use the database schema")
		}

		focusEditor()
		m.llm.ResetInstructions()
		return m, m.successNotification("LLM will no longer use the database schema")

	case command.LLMModelChangedMsg:
		existingModel, _ := m.config.GetLLMModel()
		if existingModel == msg.Model {
			return m, m.successNotification("LLM model is already set to " + msg.Model)
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
	if m.loading {
		return "Loading...\n"
	}

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

		return executeQueryMsg(queryResult)
	}
}

func (m model) ask(prompt string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.llm.Ask(prompt)
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

		return m.ask(strings.Trim(prompt, "/ask "))
	}

	if strings.HasPrefix(prompt, "/add") {
		return func() tea.Msg {
			schema, err := m.addTablesSchemaToLLM()

			if err != nil {
				m.editor.SetContent("")

				return func() tea.Msg {
					return notificationErrorMsg{err: err}
				}()
			}

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
		return func() tea.Msg {
			schema, err := m.removeTablesSchemaToLLM()

			if err != nil {
				return m.errorNotification(err)
			}

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

	return utils.ClearNotification()
}

func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = styles.Error.Render(err.Error())

	return utils.ClearNotification()
}

func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}
}
