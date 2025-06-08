package tui

import (
	"fmt"
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
	statusbar "github.com/ionut-t/perp/ui/status-bar"
	"github.com/ionut-t/perp/ui/styles"
)

var (
	padding      = lipgloss.NewStyle().Padding(1, 1)
	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary.GetForeground())
	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Overlay0.
				GetForeground())
)

type schemaFetchedMsg string
type schemaFailureMsg struct {
	err error
}

type llmResponseMsg llm.Response
type llmFailureMsg struct {
	err error
}

type executeQueryMsg db.QueryResult
type queryFailureMsg struct {
	err error
}

type clearNotificationMsg struct{}

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
	config              config.Config
	width, height       int
	view                view
	focused             focused
	serverSelection     servers.Model
	server              server.Server
	db                  db.Database
	error               error
	loading             bool
	llm                 llm.LLM
	editor              editor.Model
	sqlKeywords         map[string]lipgloss.Style
	llmKeywords         map[string]lipgloss.Style
	queryResults        []map[string]any
	historyLogs         []history.HistoryLog
	currentHistoryIndex int
	exportData          exportData.Model
	command             command.Model
	notification        string
	content             content.Model
	help                help.Model
}

func New(config config.Config) model {
	apiKey, _ := config.GetGeminiApiKey()

	editor := editor.New(80, 10)
	editor.ShowMessages(false)
	editor.SetCursorBlinkMode(true)

	sqlKeywordsMap := make(map[string]lipgloss.Style, len(SQL_KEYWORDS)*2)

	for _, keyword := range SQL_KEYWORDS {
		highlighted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5733"))
		sqlKeywordsMap[strings.ToUpper(keyword)] = highlighted
		sqlKeywordsMap[strings.ToLower(keyword)] = highlighted
	}

	editor.SetHighlightedWords(sqlKeywordsMap)

	llmKeywordsMap := map[string]lipgloss.Style{
		"/ask": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FF00")),
	}

	editor.SetPlaceholder("Type your SQL query or /ask your question here...")

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
		llm:             gemini.New(apiKey, instructions),
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

				isAskCommand := strings.HasPrefix(content, "/ask")

				if !isAskCommand && strings.HasSuffix(content, ";") && len(content) > 5 ||
					isAskCommand && len(content) > 5 && strings.HasSuffix(content, "?") {
					if logs, err := history.Add(m.editor.GetCurrentContent(), m.config.Storage()); err == nil {
						m.historyLogs = logs
						m.currentHistoryIndex = 0
					}

					return m, m.sendQueryCmd()
				}
			}

		case key.Matches(msg, executeQuery):
			if logs, err := history.Add(m.editor.GetCurrentContent(), m.config.Storage()); err == nil {
				m.historyLogs = logs
				m.currentHistoryIndex = 0
			}

			return m, m.sendQueryCmd()

		case key.Matches(msg, keymap.Cancel):
			if m.view == viewMain && m.focused == focusedEditor {
				if m.editor.IsNormalMode() {
					if m.editor.IsFocused() {
						m.focused = focusedContent
						m.editor.Blur()
					}
				}
			}

		case key.Matches(msg, previousHistory):
			if m.editor.IsFocused() && len(m.historyLogs) > 0 {
				lastQuery := m.historyLogs[m.currentHistoryIndex].Query

				if m.currentHistoryIndex < len(m.historyLogs)-1 {
					m.currentHistoryIndex++
				} else {
					m.currentHistoryIndex = 0
				}

				m.editor.SetContent(lastQuery)
				m.editor.SetCursorPositionEnd()
			}

		case key.Matches(msg, nextHistory):
			if m.editor.IsFocused() && len(m.historyLogs) > 0 {
				lastQuery := m.historyLogs[m.currentHistoryIndex].Query

				if m.currentHistoryIndex > 0 {
					m.currentHistoryIndex--
				} else {
					m.currentHistoryIndex = len(m.historyLogs) - 1
				}

				m.editor.SetContent(lastQuery)
				m.editor.SetCursorPositionEnd()
			}

		case key.Matches(msg, accessExportedData):
			if m.focused == focusedContent {
				m.view = viewExportData
				exportStore := exportStore.New(m.config.Storage(), m.config.Editor())
				m.exportData = exportData.New(exportStore, m.server, m.width, m.height)
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
			return m, m.generateSchema()
		}
		m.loading = false

	case clearNotificationMsg:
		m.notification = ""

	case schemaFetchedMsg:
		schema := string(msg)
		m.loading = false

		if m.config.ShouldUseDatabaseSchema() {
			m.llm.AppendInstructions(schema)
		}

		m.content.SetSchema(schema)

	case schemaFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case executeQueryMsg:
		m.loading = false
		m.editor.SetContent("")

		err := m.content.SetQueryResults(msg)

		if err != nil {
			return m, nil
		}

		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, cmd

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

			fileName, err := export.AsJson(m.config.Storage(), data, fileName)

			if err != nil {
				return m, m.errorNotification(err)
			}

			m.focused = focusedEditor
			m.editor.Focus()
			m.command.Reset()

			return m, m.successNotification(
				fmt.Sprintf("Data exported successfully to %s.json", fileName),
			)
		} else {
		}

		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.errorNotification(fmt.Errorf("no query results to export"))

	case command.EditorChangedMsg:
		err := m.config.SetEditor(msg.Editor)
		if err != nil {
			return m, m.errorNotification(err)
		}

		return m, m.successNotification(
			fmt.Sprintf("Editor changed to %s", msg.Editor),
		)

	case command.LLMUseDatabaseSchemaMsg:
		if err := m.config.EnableLLMDatabaseSchema(msg.Enabled); err != nil {
			return m, m.errorNotification(err)
		}

		if msg.Enabled {
			m.llm.AppendInstructions(m.content.GetDatabaseSchema())
			return m, m.successNotification("LLM will now use the database schema")
		}

		m.llm.ResetInstructions()
		return m, m.successNotification("LLM will no longer use the database schema")

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
		commandLine = statusbar.StatusBarView(m.server, m.width)
	}

	if m.notification != "" {
		commandLine = m.notification
	}

	editorBorder := inactiveBorder
	if m.focused == focusedEditor {
		editorBorder = activeBorder
	}

	contentBorder := inactiveBorder
	if m.focused == focusedContent {
		contentBorder = activeBorder
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
		return padding.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Height(m.height-lipgloss.Height(m.editor.View())-4).Render(
				m.serverSelection.View(),
			),
		))

	case viewMain:
		return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			contentBorder.Width(width).
				Height(height-lipgloss.Height(m.editor.View())-lipgloss.Height(m.command.View())-padding.GetVerticalBorderSize()*2-2).
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
		result, err := m.db.ExecuteQuery(query)
		if err != nil {
			return queryFailureMsg{err: err}
		}

		return executeQueryMsg(result)
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
	if strings.HasPrefix(m.editor.GetCurrentContent(), "/ask") {
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

	if strings.HasPrefix(prompt, "/ask ") {
		m.focused = focusedContent

		return m.ask(strings.Trim(prompt, "/ask "))
	}

	return m.executeQuery(prompt)
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
	h, v := padding.GetFrameSize()

	statusBarHeight := 1

	availableHeight := m.height - v - statusBarHeight - activeBorder.GetBorderBottomSize()
	availableWidth := m.width - h - activeBorder.GetBorderLeftSize()

	return availableWidth, availableHeight
}
