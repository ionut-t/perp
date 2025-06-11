package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
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

type clearNotificationMsg struct{}

type llmSharedSchemaMsg struct {
	schema  string
	message string
	tables  []string
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
					if logs, err := history.Add(m.editor.GetCurrentContent(), m.config.Storage()); err == nil {
						m.historyLogs = logs
					}

					return m, m.sendQueryCmd()
				}
			}

		case key.Matches(msg, executeQuery):
			m.resetHistory()

			if logs, err := history.Add(m.editor.GetCurrentContent(), m.config.Storage()); err == nil {
				m.historyLogs = logs
				m.currentHistoryIndex = 0
			}

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
				// If we're not already navigating, save the current content
				if !m.historyNavigating {
					m.originalEditorContent = m.editor.GetCurrentContent()
					m.currentHistoryIndex = -1 // Start before the first item
					m.historyNavigating = true
				}

				// Move to older entry
				if m.currentHistoryIndex < len(m.historyLogs)-1 {
					m.currentHistoryIndex++
					m.editor.SetContent(m.historyLogs[m.currentHistoryIndex].Query)
					m.editor.SetCursorPositionEnd()
				}
			}
			// if m.editor.IsFocused() && len(m.historyLogs) > 0 {
			// 	lastQuery := m.historyLogs[m.currentHistoryIndex].Query

			// 	if m.currentHistoryIndex < len(m.historyLogs)-1 {
			// 		m.currentHistoryIndex++
			// 	} else {
			// 		m.currentHistoryIndex = 0
			// 	}

			// 	m.editor.SetContent(lastQuery)
			// 	m.editor.SetCursorPositionEnd()
			// }

		case key.Matches(msg, nextHistory):
			if m.editor.IsFocused() && m.historyNavigating {
				// Move to newer entry
				if m.currentHistoryIndex > 0 {
					m.currentHistoryIndex--
					m.editor.SetContent(m.historyLogs[m.currentHistoryIndex].Query)
					m.editor.SetCursorPositionEnd()
				} else if m.currentHistoryIndex == 0 {
					// Return to original content
					m.currentHistoryIndex = -1
					m.editor.SetContent(m.originalEditorContent)
					m.editor.SetCursorPositionEnd()
					m.historyNavigating = false
				}
			}

			// if m.editor.IsFocused() && len(m.historyLogs) > 0 {
			// 	lastQuery := m.historyLogs[m.currentHistoryIndex].Query

			// 	if m.currentHistoryIndex > 0 {
			// 		m.currentHistoryIndex--
			// 	} else {
			// 		m.currentHistoryIndex = len(m.historyLogs) - 1
			// 	}

			// 	m.editor.SetContent(lastQuery)
			// 	m.editor.SetCursorPositionEnd()
			// }

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

	case clearNotificationMsg:
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

	var commandLine string

	if m.focused == focusedCommand {
		commandLine = m.command.View()
	} else {
		commandLine = m.renderStatusBar()
	}

	if m.notification != "" {
		commandLine = m.notification
	}

	editorBorder := styles.InactiveBorder
	if m.focused == focusedEditor {
		editorBorder = styles.ActiveBorder
	}

	contentBorder := styles.InactiveBorder
	if m.focused == focusedContent {
		contentBorder = styles.ActiveBorder
	}

	primaryView := lipgloss.JoinVertical(
		lipgloss.Left,
		editorBorder.Render(
			m.editor.View(),
		),
		commandLine,
	)

	width, height := m.getAvailableSizes()

	if m.error != nil {
		return m.renderDBError(width, height)
	}

	switch m.view {
	case viewServers:
		return styles.ViewPadding.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Height(m.height-lipgloss.Height(m.editor.View())-4).Render(
				m.serverSelection.View(),
			),
		))

	case viewMain:
		return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			contentBorder.Width(width).
				Height(height-lipgloss.Height(m.editor.View())-lipgloss.Height(m.command.View())-styles.ViewPadding.GetVerticalBorderSize()*2-2).
				Render(m.content.View()),
			primaryView))

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

		result, err := m.db.ExecuteQuery(ctx, query)
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
				return m.errorNotification(err)
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
	if queryResults != nil {
		rows := msg.Rows
		all := msg.All
		fileName := msg.FileName

		var data any
		if len(rows) > 1 {
			data = make([]map[string]any, 0)

			for _, rowIdx := range rows {
				idx := rowIdx - 1
				if idx >= 0 && idx < len(queryResults) {
					data = append(data.([]map[string]any), queryResults[idx])
				}
			}
		} else if len(rows) == 1 {
			idx := rows[0] - 1
			if idx >= 0 && idx < len(queryResults) {
				data = queryResults[idx]
			}
		}

		if all {
			data = make([]map[string]any, 0)
			data = append(data.([]map[string]any), queryResults...)
		}

		storage := filepath.Join(m.config.Storage(), m.server.Name)
		fileName, err := export.AsJson(storage, data, fileName)

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

	m.focused = focusedEditor
	m.editor.Focus()
	return m, m.errorNotification(fmt.Errorf("no query results to export"))
}

func (m *model) renderDBError(width, height int) string {
	return lipgloss.NewStyle().
		Padding(0, 1).
		Height(height).
		Width(width).
		Border(lipgloss.RoundedBorder()).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.Error.Render(m.error.Error()),
				"\n",
				styles.Subtext0.Render("Press 'q' to go back to server selection"),
			),
		)
}

func (m *model) renderStatusBar() string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(m.server.Name)

	database := styles.Accent.Background(bg).Render(m.server.Database)

	llm := lipgloss.NewStyle().Background(bg).Render(m.renderLLMModel())

	left := serverName + separator + database + separator + llm

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := m.width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func (m *model) successNotification(msg string) tea.Cmd {
	m.notification = styles.Success.Render(msg)

	return m.clearNotification()
}

func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = styles.Error.Render(err.Error())

	return m.clearNotification()
}

func (m *model) clearNotification() tea.Cmd {
	return tea.Tick(
		time.Second*2,
		func(t time.Time) tea.Msg {
			return clearNotificationMsg{}
		},
	)
}

func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}
}

func (m *model) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	statusBarHeight := 1

	availableHeight := m.height - v - statusBarHeight - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h - styles.ActiveBorder.GetBorderLeftSize()

	return availableWidth, availableHeight
}

func (m *model) renderLLMModel() string {
	llmModel, _ := m.config.GetLLMModel()

	if llmModel == "" {
		return styles.Subtext0.Render("No LLM model set")
	}

	if m.server.ShareDatabaseSchemaLLM {
		return styles.Accent.Render(llmModel + " (DB Schema enabled)")
	}

	return styles.Accent.Render(llmModel)
}

// addTablesSchemaToLLM processes the `/add` command to include table schemas in the LLM context.
func (m *model) addTablesSchemaToLLM() (string, error) {
	if !m.server.ShareDatabaseSchemaLLM {
		return "", fmt.Errorf("cannot add tables to LLM schema when this feature is disabled")
	}

	value := strings.TrimSpace(strings.TrimPrefix(m.editor.GetCurrentContent(), "/add"))
	if value == "" {
		return "", fmt.Errorf("no tables specified to add")
	}

	tables := parseTableNames(value)
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

	tables := parseTableNames(value)
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

// parseTableNames is a helper function that extracts and deduplicates table names from a raw input string.
func parseTableNames(input string) []string {
	var tables []string
	seen := make(map[string]bool)

	// Split the input string by common delimiters like comma, space, tab, or newline.
	for _, table := range strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}) {
		trimmedTable := strings.TrimSpace(table)
		if trimmedTable != "" && !seen[trimmedTable] {
			tables = append(tables, trimmedTable)
			seen[trimmedTable] = true
		}
	}
	return tables
}

func (m *model) resetHistory() {
	m.historyNavigating = false
	m.currentHistoryIndex = -1
	m.originalEditorContent = ""
}
