package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	table "github.com/ionut-t/gotable"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/pkg/clipboard"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/llm/gemini"
	"github.com/ionut-t/perp/pkg/server"
	exportStore "github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/tui/command"
	exportData "github.com/ionut-t/perp/tui/export_data"
	"github.com/ionut-t/perp/tui/servers"
	"github.com/ionut-t/perp/ui/list"
	"github.com/ionut-t/perp/ui/styles"
)

const padding = 2

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

type clearYankMsg struct{}

type clearNotificationMsg struct{}

type view int

const (
	viewServers view = iota
	viewMain
	viewLLMLogs
	viewDBSchema
	viewExportData
)

type mode int

const (
	modeNormal mode = iota
	modeInsert
)

type focusedView int

const (
	focusedViewEditor focusedView = iota
	focusedViewTable
	focusedViewLLMLogs
	focusedViewCommand
	focusedViewNone
)

type model struct {
	config              config.Config
	width, height       int
	serverSelection     servers.Model
	server              server.Server
	db                  db.Database
	dbError             error
	error               error
	loading             bool
	viewport            viewport.Model
	view                view
	mode                mode
	llm                 llm.LLM
	llmLogs             list.Model
	logs                []chatLog
	editor              editor.Model
	sqlKeywords         map[string]lipgloss.Style
	llmKeywords         map[string]lipgloss.Style
	table               table.Model
	dbSchema            string
	message             string
	queryResults        []map[string]any
	historyLogs         []history.HistoryLog
	currentHistoryIndex int
	exportData          exportData.Model
	command             command.Model
	focusedView         focusedView
	notification        string
}

func New(config config.Config) model {
	apiKey, _ := config.GetGeminiApiKey()
	list := list.New([]list.Item{}, 0, 0)

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

	editor.SetInsertMode()
	editor.Focus()
	editor.DisableCommandMode(true)

	t := table.New()
	t.SetSize(80, 20)
	t.SetSelectionMode(table.SelectionCell | table.SelectionRow)

	historyLogs, err := history.Get()
	if err != nil {
		historyLogs = []history.HistoryLog{}
	}

	return model{
		config:          config,
		viewport:        viewport.New(0, 0),
		llm:             gemini.New(apiKey),
		llmLogs:         list,
		editor:          editor,
		sqlKeywords:     sqlKeywordsMap,
		llmKeywords:     llmKeywordsMap,
		table:           t,
		command:         command.New(),
		mode:            modeInsert,
		serverSelection: servers.New(config.Storage()),
		historyLogs:     historyLogs,
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
		m.editor.SetSize(msg.Width-padding*2, 10)
		m.viewport.Width = msg.Width - padding*2
		m.viewport.Height = msg.Height - padding*2

		height := max(m.height-lipgloss.Height(m.editor.View())-lipgloss.Height(m.command.View())-padding*2, 1)

		listWidth := max(m.width-padding*2, 1)
		m.llmLogs.SetSize(listWidth, height)

		m.table.SetSize(m.width-padding*2, height)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.closeDbConnection()
			return m, tea.Quit
		}

		if m.focusedView == focusedViewCommand || m.view == viewServers || m.view == viewExportData {
			break
		}

		switch msg.String() {
		case "q":
			if m.dbError != nil {
				m.serverSelection = servers.New(m.config.Storage())
				_, cmd := m.serverSelection.Update(nil)

				m.view = viewServers
				m.dbError = nil
				return m, cmd
			}

			if m.view == viewDBSchema {
				m.view = viewMain
				break
			}

			if m.editor.IsNormalMode() {
				m.closeDbConnection()
				return m, tea.Quit
			}

		case ":":
			if m.editor.IsNormalMode() {
				m.focusedView = focusedViewCommand
				m.editor.Blur()
				return m, cursor.Blink
			}

		case "S":
			if m.editor.IsNormalMode() {
				m.message = ""
				m.error = nil
				m.view = viewDBSchema
			}

		case "|":
			if m.editor.IsNormalMode() {
				m.message = ""
				m.serverSelection = servers.New(m.config.Storage())
				_, cmd := m.serverSelection.Update(nil)

				m.view = viewServers
				m.dbError = nil
				return m, cmd
			}

		case "i":
			if m.mode == modeNormal && m.view != viewServers {
				m.mode = modeInsert
				m.editor.Focus()
				m.editor.SetInsertMode()

				return m, m.editor.CursorBlink()
			}

		case "alt+enter":
			m.error = nil
			m.message = ""
			if logs, err := history.Add(m.editor.GetCurrentContent()); err == nil {
				m.historyLogs = logs
				m.currentHistoryIndex = 0
			}

			return m, m.sendQueryCmd()

		case "esc":
			m.error = nil

			if m.view == viewMain && m.mode == modeInsert {
				if m.editor.IsNormalMode() {
					if m.editor.IsFocused() {
						m.mode = modeNormal
						m.editor.Blur()
					}
				}
			}

		case "enter":
			if m.view == viewLLMLogs {
				selected, ok := m.llmLogs.GetSelectedItem()

				if !ok {
					return m, nil
				}

				m.editor.SetContent(strings.TrimSpace(selected.Description))
				m.editor.Focus()
				m.editor.SetInsertMode()
				m.editor.SetCursorPositionEnd()
				m.view = viewMain
			}

		case "y":
			if m.mode == modeNormal {
				if cell, ok := m.table.GetSelectedCell(); ok {
					clipboard.Write(cell)

					defaultTheme := table.DefaultTheme()
					theme := table.Theme{
						Header:      defaultTheme.Header,
						Border:      defaultTheme.Border,
						Cell:        defaultTheme.Cell,
						SelectedRow: defaultTheme.SelectedRow,
						SelectedCell: defaultTheme.SelectedCell.
							Background(defaultTheme.SelectedCell.GetForeground()).
							Foreground(defaultTheme.SelectedCell.GetBackground()),
					}

					m.table.SetTheme(theme)

					return m, m.dispatchClearYankMsg()
				}
			}

		case "shift+up":
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

		case "shift+down":
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

		case "g":
			if m.mode == modeNormal {
				m.message = ""
				m.view = viewExportData
				exportStore := exportStore.New(m.config.Storage(), m.config.Editor())
				m.exportData = exportData.New(exportStore, m.width, m.height)
			}
		}

	case servers.SelectedServerMsg:
		m.closeDbConnection()
		m.view = viewMain
		m.loading = true
		m.server = msg.Server
		m.db, m.dbError = db.New(m.server.ConnectionString())
		if m.dbError == nil {
			m.displayConnectionInfo()
			return m, m.generateSchema()
		}
		m.loading = false

	case clearYankMsg:
		m.table.SetTheme(table.DefaultTheme())

	case clearNotificationMsg:
		m.notification = ""

	case schemaFetchedMsg:
		m.dbSchema = string(msg)
		m.loading = false
		m.viewport.SetContent(m.dbSchema)
		m.llm.AppendInstructions(m.dbSchema)

	case schemaFailureMsg:
		m.loading = false
		m.error = msg.err

	case executeQueryMsg:
		m.message = ""
		m.loading = false
		m.editor.SetContent("")
		m.queryResults = nil

		if msg.Type() != db.QuerySelect {
			rows := msg.Rows()
			rows.Close()
			affected := rows.CommandTag().RowsAffected()

			m.message = lipgloss.JoinVertical(
				lipgloss.Left,
				msg.Query(),
				"\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(
					fmt.Sprintf("Query executed successfully. Affected rows: %d", affected),
				),
			)

			return m, nil
		}

		results, headers, err := m.db.FetchQueryResults(msg.Rows())
		m.queryResults = results

		if err != nil {
			m.error = err
			return m, nil
		}

		if len(results) == 0 {
			// TODO: create a method in table to clear it
			m.table.SetHeaders([]string{})
			m.table.SetRows([][]string{})
			m.table.SetSelectedCell(0, 0)
			m.message = "No results found."
			return m, nil
		}

		rows, headers := m.buildDataTable(headers, results)

		m.table.SetHeaders(headers)
		m.table.SetRows(rows)
		m.table.SetSelectedCell(0, 0)

		m.mode = modeNormal
		m.focusedView = focusedViewTable
		m.editor.Blur()
		m.editor.SetNormalMode()

	case queryFailureMsg:
		m.loading = false
		m.error = msg.err

	case llmResponseMsg:
		m.loading = false

		newLog := chatLog{
			Prompt:   strings.Trim(m.editor.GetCurrentContent(), "/ask"),
			Response: msg.Response,
			Time:     msg.Time,
		}

		m.logs = append([]chatLog{newLog}, m.logs...)
		m.view = viewLLMLogs
		m.llmLogs.SetItems(processLogs(m.logs))
		m.editor.SetContent("")

	case llmFailureMsg:
		m.loading = false
		newLog := chatLog{
			Prompt: strings.Trim(m.editor.GetCurrentContent(), "/ask"),
			Error:  msg.err,
			Time:   time.Now(),
		}

		m.logs = append([]chatLog{newLog}, m.logs...)
		m.llmLogs.SetItems(processLogs(m.logs))

	case exportData.ClosedMsg:
		m.view = viewMain
		m.editor.SetContent("")
		m.editor.Focus()

		if m.queryResults == nil {
			m.displayConnectionInfo()
		}

	case command.QuitMsg:
		m.closeDbConnection()
		return m, tea.Quit

	case command.CancelMsg:
		m.focusedView = focusedViewEditor
		m.editor.Focus()

	case command.ExportMsg:
		if m.queryResults != nil {
			rows := msg.Rows
			all := msg.All
			fileName := msg.FileName

			var data any

			if len(rows) > 1 {
				data = make([]map[string]any, 0)

				for _, rowIdx := range rows {
					idx := rowIdx - 1
					if idx >= 0 && idx < len(m.queryResults) {
						data = append(data.([]map[string]any), m.queryResults[idx])
					}
				}
			} else if len(rows) == 1 {
				idx := rows[0] - 1
				if idx >= 0 && idx < len(m.queryResults) {
					data = m.queryResults[idx]
				}
			}

			if all {
				data = make([]map[string]any, 0)
				data = append(data.([]map[string]any), m.queryResults...)
			}

			fileName, err := export.AsJson(m.config.Storage(), data, fileName)

			if err != nil {
				return m, m.errorNotification(err)
			}

			m.focusedView = focusedViewEditor
			m.editor.Focus()
			m.command.Reset()

			return m, m.successNotification(
				fmt.Sprintf("Data exported successfully to %s.json", fileName),
			)
		} else {
		}

		m.focusedView = focusedViewEditor
		m.editor.Focus()
		return m, m.errorNotification(fmt.Errorf("no query results to export"))

	case command.ErrorMsg:
		return m, m.errorNotification(msg.Err)
	}

	var cmds []tea.Cmd

	if m.view == viewMain && m.mode == modeNormal {
		vp, cmd := m.viewport.Update(msg)
		m.viewport = vp
		cmds = append(cmds, cmd)
	}

	m.editor.SetHighlightedWords(m.setHighlightedKeywords())

	if m.focusedView == focusedViewTable && m.view == viewMain {
		t, cmd := m.table.Update(msg)
		m.table = t
		cmds = append(cmds, cmd)
	}

	if m.view == viewLLMLogs {
		l, cmd := m.llmLogs.Update(msg)
		m.llmLogs = l
		cmds = append(cmds, cmd)
	}

	if m.view == viewDBSchema {
		vp, cmd := m.viewport.Update(msg)
		m.viewport = vp
		cmds = append(cmds, cmd)
	}

	if m.view == viewMain || m.view == viewLLMLogs {
		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel.(editor.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewServers {
		s, cmd := m.serverSelection.Update(msg)
		m.serverSelection = s.(servers.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewExportData {
		exportDataModel, cmd := m.exportData.Update(msg)
		m.exportData = exportDataModel.(exportData.Model)
		cmds = append(cmds, cmd)
		m.editor.SetContent(fmt.Sprintf("%v", m.view))
	}

	if m.focusedView == focusedViewCommand {
		cmdModel, cmd := m.command.Update(msg)
		m.command = cmdModel.(command.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.loading {
		return "Loading...\n"
	}

	var commandLine string

	if m.focusedView == focusedViewCommand {
		commandLine = m.command.View()
	}

	if m.notification != "" {
		commandLine = m.notification
	}

	availableHeight := max(m.height-lipgloss.Height(m.editor.View())-lipgloss.Height(m.command.View())-padding*2-2, 1)
	width := max(m.width-padding*2, 1)

	primaryView := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(
			m.editor.View(),
		),
		commandLine,
	)

	if m.dbError != nil {
		return m.renderDBError(width, m.height-padding)
	}

	if m.error != nil {
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderError(width, availableHeight),
			primaryView,
		))
	}

	if m.message != "" {
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Padding(0, 1).
				Height(availableHeight).
				Width(width).
				Border(lipgloss.RoundedBorder()).
				Render(m.message),
			primaryView,
		))
	}

	switch m.view {
	case viewServers:
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Height(m.height-lipgloss.Height(m.editor.View())-4).Render(
				m.serverSelection.View(),
			),
		))

	case viewMain:
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Height(availableHeight).Render(m.table.View()),
			primaryView))

	case viewLLMLogs:
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(m.llmLogs.View()),
			primaryView,
		))

	case viewDBSchema:
		return lipgloss.NewStyle().Padding(padding).Render(m.viewport.View())

	case viewExportData:
		return m.exportData.View()
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
		m.view = viewLLMLogs
		m.mode = modeNormal

		return m.ask(strings.Trim(prompt, "/ask "))
	}

	return m.executeQuery(prompt)
}

func (m model) dispatchClearYankMsg() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return clearYankMsg{}
	})
}

func (model) buildDataTable(headers []string, results []map[string]any) ([][]string, []string) {
	rows := [][]string{}

	headers = append([]string{"#"}, headers...)

	for i, row := range results {
		rowData := make([]string, len(headers))
		for j, header := range headers {
			if val, ok := row[header]; ok {
				rowData[j] = fmt.Sprintf("%v", val)
			} else {
				if header == "#" {
					rowData[j] = fmt.Sprintf("%d", i+1)
				} else {
					rowData[j] = "NULL"
				}
			}
		}
		rows = append(rows, rowData)
	}
	return rows, headers
}

func (m *model) displayConnectionInfo() {
	m.message = lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Render(fmt.Sprintf("Connected to server: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Name))),
		lipgloss.NewStyle().Render(fmt.Sprintf("Database: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Database))),
		lipgloss.NewStyle().Render(fmt.Sprintf("Host: %s", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s:%d", m.server.Address, m.server.Port)))),
	)
}

func (m *model) renderError(width, height int) string {
	return styles.Error.
		Padding(0, 1).
		Height(height).
		Width(width).
		Border(lipgloss.RoundedBorder()).
		Render(m.error.Error())
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
				styles.Error.Render(m.dbError.Error()),
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
