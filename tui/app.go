package tui

import (
	"fmt"
	"strings"
	"time"

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
	"github.com/ionut-t/perp/tui/servers"
	"github.com/ionut-t/perp/ui/list"
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

type view int

const (
	viewServers view = iota
	viewMain
	viewLLMLogs
	viewDBSchema
)

type mode int

const (
	modeNormal mode = iota
	modeInsert
)

type model struct {
	width, height       int
	serverSelection     servers.Model
	server              server.Server
	db                  *db.Database
	err                 error
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
}

func New() model {
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

	t := table.New()
	t.SetSize(80, 20)
	t.SetSelectionMode(table.SelectionCell | table.SelectionRow)

	historyLogs, err := history.Get()
	if err != nil {
		historyLogs = []history.HistoryLog{}
	}

	return model{
		viewport:        viewport.New(0, 0),
		llm:             gemini.New(apiKey),
		llmLogs:         list,
		editor:          editor,
		sqlKeywords:     sqlKeywordsMap,
		llmKeywords:     llmKeywordsMap,
		table:           t,
		mode:            modeInsert,
		serverSelection: servers.New(),
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

		listHeight := max(m.height-lipgloss.Height(m.editor.View())-padding*2, 1)
		listWidth := max(m.width-padding*2, 1)
		m.llmLogs.SetSize(listWidth, listHeight)

		tableHeight := max(m.height-lipgloss.Height(m.editor.View())-padding*2, 1)

		m.table.SetSize(m.width-padding*2, tableHeight)

	case tea.KeyMsg:
		if m.view == viewServers {
			break
		}

		switch msg.String() {
		case "ctrl+c":
			db.Close(m.db)
			return m, tea.Quit

		case "q":
			if m.view == viewDBSchema {
				m.view = viewMain
				break
			}

			if m.editor.IsNormalMode() {
				db.Close(m.db)
				return m, tea.Quit
			}

		case "S":
			if m.editor.IsNormalMode() {
				m.err = nil
				m.view = viewDBSchema
			}

		case "i":
			if m.mode == modeNormal && m.view != viewServers {
				m.mode = modeInsert
				m.editor.Focus()
				m.editor.SetInsertMode()

				return m, m.editor.CursorBlink()
			}

		case "alt+enter":
			m.err = nil
			m.message = ""
			if logs, err := history.Add(m.editor.GetCurrentContent()); err == nil {
				m.historyLogs = logs
				m.currentHistoryIndex = 0
			}

			return m, m.sendQueryCmd()

		case "esc":
			m.err = nil

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

		case "e":
			if m.mode == modeNormal {
				if m.queryResults != nil {
					idx := m.table.GetSelectedRow()

					if idx >= 0 && idx < len(m.queryResults) {
						row := m.queryResults[idx]

						exportCmd, err := export.AsJson(row)

						if err != nil {
							m.err = err
							return m, nil
						}

						return m, exportCmd
					}
				}
			}

		case "E":
			if m.mode == modeNormal {
				if m.queryResults != nil {
					exportCmd, err := export.AsJson(m.queryResults)

					if err != nil {
						m.err = err
						return m, nil
					}

					return m, exportCmd
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
		}

	case servers.SelectedServerMsg:
		m.view = viewMain
		m.loading = true
		m.server = msg.Server
		m.db, m.err = db.Connect(m.server.ConnectionString())
		if m.err == nil {
			m.message = lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().Render(fmt.Sprintf("Connected to server: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Name))),
				lipgloss.NewStyle().Render(fmt.Sprintf("Database: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Database))),
				lipgloss.NewStyle().Render(fmt.Sprintf("Host: %s", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s:%d", m.server.Address, m.server.Port)))),
			)
			return m, m.generateSchema()
		}
		m.loading = false

	case clearYankMsg:
		m.table.SetTheme(table.DefaultTheme())

	case editor.SaveMsg:
		return m, m.sendQueryCmd()

	case editor.QuitMsg:
		return m, tea.Quit

	case schemaFetchedMsg:
		m.dbSchema = string(msg)
		m.loading = false
		m.viewport.SetContent(m.dbSchema)
		m.llm.AppendInstructions(m.dbSchema)

	case schemaFailureMsg:
		m.loading = false
		m.err = msg.err

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
			m.err = err
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

		m.table.SetHeaders(headers)
		m.table.SetRows(rows)
		m.table.SetSelectedCell(0, 0)

		m.mode = modeNormal
		m.editor.Blur()
		m.editor.SetNormalMode()

	case queryFailureMsg:
		m.loading = false
		m.err = msg.err

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
	}

	var cmds []tea.Cmd

	if m.view == viewMain && m.mode == modeNormal {
		vp, cmd := m.viewport.Update(msg)
		m.viewport = vp
		cmds = append(cmds, cmd)
	}

	m.editor.SetHighlightedWords(m.setHighlightedKeywords())
	if m.mode == modeNormal {
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

	if m.view == viewServers {
		s, cmd := m.serverSelection.Update(msg)
		m.serverSelection = s.(servers.Model)
		cmds = append(cmds, cmd)
	} else {
		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel.(editor.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.loading {
		return "Loading...\n"
	}

	height := max(m.height-lipgloss.Height(m.editor.View())-padding*2-2, 1)
	width := max(m.width-padding*2, 1)
	if m.err != nil {
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("9")).
				Height(height).
				Width(width).
				Border(lipgloss.RoundedBorder()).
				Render(m.err.Error()),
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(
				m.editor.View(),
			),
		))
	}

	if m.message != "" {
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().
				Padding(0, 1).
				Height(height).
				Width(width).
				Border(lipgloss.RoundedBorder()).
				Render(m.message),
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(
				m.editor.View(),
			),
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
			lipgloss.NewStyle().Height(m.height-lipgloss.Height(m.editor.View())-4).Render(
				m.table.View(),
			),
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(m.editor.View()),
		))

	case viewLLMLogs:
		return lipgloss.NewStyle().Padding(padding).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(m.llmLogs.View()),
			lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Render(
				m.editor.View(),
			),
		))

	case viewDBSchema:
		return lipgloss.NewStyle().Padding(padding).Render(m.viewport.View())
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
