package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	table "github.com/ionut-t/gotable"
	"github.com/ionut-t/perp/pkg/clipboard"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/ui/list"
	"github.com/ionut-t/perp/ui/styles"
)

var (
	padding = lipgloss.NewStyle().Padding(0, 1)
)

type LLMResponseSelectedMsg struct {
	Response string
}

type clearYankMsg struct{}

type view int

const (
	viewConnectionInfo view = iota
	viewDBSchema
	viewTable
	viewInfo
	viewLLMLogs
	viewError
)

type Model struct {
	width, height int
	view          view
	error         error
	dbSchema      string
	queryResults  []map[string]any
	viewport      viewport.Model
	table         table.Model
	server        server.Server
	llmLogsList   list.Model
	logs          []chatLog
}

type chatLog struct {
	Prompt   string
	Response string
	Error    error
	Time     time.Time
}

func New(width, height int) Model {
	t := table.New()
	t.SetSize(width-1, height)
	t.SetSelectionMode(table.SelectionCell | table.SelectionRow)
	t.SetTheme(styles.TableTheme())

	ls := list.New([]list.Item{}, width, height)
	ls.SetPlaceholder("No LLM logs found.")

	return Model{
		width:       width,
		height:      height,
		viewport:    viewport.New(width, height),
		table:       t,
		llmLogsList: ls,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	m.viewport.Width = width
	m.viewport.Height = height

	m.table.SetSize(width-1, height)

	m.llmLogsList.SetSize(width, height)

	switch m.view {
	case viewConnectionInfo, viewInfo, viewDBSchema:
		m.viewport.Height = height
		m.viewport.Width = width
	case viewTable:
		m.table.SetSize(width, height)
	}
}

func (m *Model) SetConnectionInfo(s server.Server) {
	m.server = s
	m.view = viewConnectionInfo
	m.setViewportContent()
}

func (m *Model) SetSchema(schema string) {
	m.dbSchema = strings.TrimSpace(schema)
}

func (m *Model) ShowConnectionInfo() {
	m.view = viewConnectionInfo
	m.setViewportContent()
}

func (m *Model) ShowDBSchema() {
	m.view = viewDBSchema
}

func (m *Model) ShowLLMLogs() {
	m.view = viewLLMLogs
}

func (m *Model) IsViewChangeRequired() bool {
	if m.view != viewConnectionInfo && len(m.queryResults) > 0 && m.view != viewTable {
		m.view = viewTable
		return true
	}

	if m.view == viewConnectionInfo {
		return false
	}

	if m.view != viewConnectionInfo {
		m.view = viewConnectionInfo
		return true
	}

	return false
}

func (m *Model) SetError(err error) {
	m.error = err
	m.view = viewError
}

func (m *Model) SetQueryResults(result db.QueryResult) error {
	m.queryResults = nil

	if result.Type() != db.QuerySelect {
		rows := result.Rows()
		rows.Close()
		affected := rows.CommandTag().RowsAffected()

		content := lipgloss.JoinVertical(
			lipgloss.Left,
			result.Query(),
			"\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(
				fmt.Sprintf("Query executed successfully. Affected rows: %d", affected),
			),
		)

		m.viewport.SetContent(padding.Render(content))
		m.view = viewInfo

		return nil
	}

	results, headers, err := db.ExtractResultsFromRows(result.Rows())
	m.queryResults = results

	if err != nil {
		m.error = err
		return err
	}

	if len(results) == 0 {
		m.table.SetHeaders([]string{})
		m.table.SetRows([][]string{})
		m.table.SetSelectedCell(0, 0)
		m.viewport.SetContent("No results found.")
		m.view = viewInfo
		return nil
	}

	rows, headers := m.buildDataTable(headers, results)

	m.table.SetHeaders(headers)
	m.table.SetRows(rows)
	m.table.SetSelectedCell(0, 0)
	m.view = viewTable

	return nil
}

func (m *Model) GetQueryResults() []map[string]any {
	return m.queryResults
}

func (m *Model) GetDatabaseSchema() string {
	return m.dbSchema
}

func (m *Model) SetLLMLogs(response llm.Response, query string) {
	newLog := chatLog{
		Prompt:   query,
		Response: response.Response,
		Time:     time.Now(),
	}

	m.logs = append([]chatLog{newLog}, m.logs...)
	m.view = viewLLMLogs
	m.llmLogsList.SetItems(processLogs(m.logs))
}

func (m *Model) SetLLMLogsError(err error, query string) {
	newLog := chatLog{
		Prompt: query,
		Error:  err,
		Time:   time.Now(),
	}

	m.logs = append([]chatLog{newLog}, m.logs...)
	m.view = viewLLMLogs
	m.llmLogsList.SetItems(processLogs(m.logs))
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearYankMsg:
		m.table.SetTheme(styles.TableTheme())

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.view == viewLLMLogs {
				log := m.logs[m.llmLogsList.GetIndex()]

				if log.Error != nil {
					return m, nil
				}

				return m, func() tea.Msg {
					return LLMResponseSelectedMsg{
						Response: strings.TrimSpace(log.Response),
					}
				}
			}

		case "y":
			if m.view == viewTable {
				return m.yankSelectedCell()
			}

		case "Y":
			if m.view == viewTable {
				return m.yankSelectedRow()
			}
		}
	}

	var cmds []tea.Cmd

	switch m.view {
	case viewConnectionInfo, viewDBSchema, viewInfo:
		m.setViewportContent()

		vp, cmd := m.viewport.Update(msg)
		m.viewport = vp
		cmds = append(cmds, cmd)

	case viewTable:
		t, cmd := m.table.Update(msg)
		m.table = t
		cmds = append(cmds, cmd)

	case viewLLMLogs:
		l, cmd := m.llmLogsList.Update(msg)
		m.llmLogsList = l
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.view {
	case viewTable:
		// TODO: Fix the height of the table in gotable library
		return lipgloss.NewStyle().Height(m.height).Render(m.table.View())

	case viewLLMLogs:
		return m.llmLogsList.View()

	case viewError:
		return m.renderError(m.width, m.height)

	default:
		return m.viewport.View()
	}
}

func (m *Model) renderError(width, height int) string {
	return styles.Error.
		Padding(0, 1).
		Height(height).
		Width(width).
		Render(m.error.Error())
}

func (m *Model) buildDataTable(headers []string, results []map[string]any) ([][]string, []string) {
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

func (m *Model) dispatchClearYankMsg() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return clearYankMsg{}
	})
}

func (m *Model) setViewportContent() {
	switch m.view {
	case viewConnectionInfo:
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Render(fmt.Sprintf("Connected to server: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Name))),
			lipgloss.NewStyle().Render(fmt.Sprintf("Database: %s", lipgloss.NewStyle().Bold(true).Render(m.server.Database))),
			lipgloss.NewStyle().Render(fmt.Sprintf("Host: %s", lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s:%d", m.server.Address, m.server.Port)))),
		)
		m.viewport.SetContent(padding.Render(content))

	case viewDBSchema:
		m.viewport.SetContent(padding.Render(m.dbSchema))
	}
}

func (m Model) yankSelectedCell() (tea.Model, tea.Cmd) {
	if cell, ok := m.table.GetSelectedCell(); ok {
		clipboard.Write(cell)

		defaultTheme := styles.TableTheme()
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

	return m, nil
}

func (m Model) yankSelectedRow() (tea.Model, tea.Cmd) {
	row := m.table.GetSelectedRow()

	data := m.queryResults[row]

	var jsonData []byte
	var err error

	jsonData, err = json.MarshalIndent(data, "", "  ")

	if err != nil {
		return m, func() tea.Msg {
			return clearYankMsg{}
		}
	}

	content := strings.TrimSpace(string(jsonData))

	clipboard.Write(content)

	defaultTheme := styles.TableTheme()
	selectedRow := defaultTheme.SelectedRow.
		Background(defaultTheme.SelectedRow.GetForeground()).
		Foreground(defaultTheme.SelectedRow.GetBackground())

	theme := table.Theme{
		Header:       defaultTheme.Header,
		Border:       defaultTheme.Border,
		Cell:         defaultTheme.Cell,
		SelectedRow:  selectedRow,
		SelectedCell: selectedRow,
	}

	m.table.SetTheme(theme)

	return m, m.dispatchClearYankMsg()
}

func processLogs(logs []chatLog) []list.Item {
	items := make([]list.Item, len(logs))

	for i, n := range logs {
		items[i] = list.Item{
			Title:       n.Prompt,
			Subtitle:    n.Time.Format("02/01/2006, 15:04:05"),
			Description: n.Response,
		}

		if n.Error != nil {
			items[i].Description = n.Error.Error()
			items[i].Styles = &list.ItemStyles{
				Title:       styles.Text,
				Subtitle:    styles.Subtext1,
				Description: styles.Error,
				SelectedBorder: lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(styles.Error.GetForeground()).
					Padding(0, 1),
			}
		}
	}

	return items
}
