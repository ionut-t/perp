package servers

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/constants"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/pkg/server"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(styles.Primary.GetForeground()).Bold(true)
)

type item struct {
	title  string
	server server.Server
}

func (i item) Title() string       { return i.title }
func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i := listItem.(item)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(i.Title()))
}

type newServerMsg struct{}

type editServerMsg struct {
	Server server.Server
}

type deleteServerMsg struct {
	Server server.Server
}

type serversListModel struct {
	width, height int
	list          list.Model
	showPassword  bool
}

func newServersListModel(servers []server.Server) serversListModel {
	items := make([]list.Item, len(servers))
	for i, srv := range servers {
		items[i] = item{
			title:  srv.Name,
			server: srv,
		}
	}

	l := list.New(items, itemDelegate{}, 50, 5)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.Title = "Select a server"
	l.Styles = styles.ListStyles()

	return serversListModel{
		list: l,
	}
}

func (m *serversListModel) setSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *serversListModel) setServers(servers []server.Server) {
	items := make([]list.Item, len(servers))
	for i, srv := range servers {
		items[i] = item{
			title:  srv.Name,
			server: srv,
		}
	}

	m.list.SetItems(items)
}

func (m serversListModel) Init() tea.Cmd {
	return nil
}

func (m serversListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.list.Items()) == 0 {
				return m, nil
			}

			if selected, ok := m.list.SelectedItem().(item); ok {
				return m, func() tea.Msg {
					return SelectedServerMsg{Server: selected.server}
				}
			}

		case "e":
			selected := m.list.SelectedItem().(item)
			return m, func() tea.Msg {
				return editServerMsg{Server: selected.server}
			}

		case "n":
			return m, func() tea.Msg {
				return newServerMsg{}
			}

		case "ctrl+d":
			selected := m.list.SelectedItem().(item)
			return m, func() tea.Msg {
				return deleteServerMsg{Server: selected.server}
			}

		case "p":
			m.showPassword = !m.showPassword
			return m, nil
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m serversListModel) View() string {
	logo := m.renderLogo()
	logoHeight := lipgloss.Height(logo)

	serverListContent := m.list.View()

	helpText := m.renderHelpText()
	helpHeight := lipgloss.Height(helpText)

	// Calculate available height for the list
	leftPanelHeight := m.height - logoHeight - helpHeight - 2

	leftPanelStyled := lipgloss.NewStyle().
		Width(m.width / 2).
		Height(leftPanelHeight).
		Render(serverListContent)

	// Right panel: Server info
	serverInfo := m.renderServerInfo()

	// Main content: left and right panels side by side
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanelStyled,
		serverInfo,
	)

	// Full view: logo, main content, help text
	return lipgloss.JoinVertical(
		lipgloss.Left,
		logo,
		mainContent,
		helpText,
	)
}

func (m *serversListModel) renderLogo() string {
	logo := constants.Logo

	logoW := lipgloss.Width(logo)

	version := lipgloss.Place(
		logoW,
		1,
		lipgloss.Center,
		lipgloss.Center,
		styles.Primary.Render(version.Version()),
	)

	return lipgloss.Place(
		m.width,
		lipgloss.Height(logo)+4,
		lipgloss.Center,
		lipgloss.Center,
		styles.Primary.Render(logo+version),
	)
}

func (m *serversListModel) renderHelpText() string {
	var sb strings.Builder
	sb.WriteString(styles.Subtext0.Render("Press n to create a new server") + "\n")
	sb.WriteString(styles.Subtext0.Render("Press e to edit the selected server") + "\n")
	sb.WriteString(styles.Subtext0.Render("Press ctrl+d to delete the selected server") + "\n")
	sb.WriteString(styles.Subtext0.Render("Press p to toggle password visibility") + "\n")

	return sb.String()
}

func (m *serversListModel) renderServerInfo() string {
	if len(m.list.Items()) == 0 {
		emptyMessage := lipgloss.NewStyle().
			Width(m.width/2).
			Height(m.height-lipgloss.Height(m.renderLogo())-lipgloss.Height(m.renderHelpText())-2).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(styles.Subtext0.GetForeground()).
			Padding(1, 2).
			Render(styles.Subtext0.Render("No servers available"))
		return emptyMessage
	}

	selected, ok := m.list.SelectedItem().(item)
	if !ok {
		noSelectionMessage := lipgloss.NewStyle().
			Width(m.width/2).
			Height(m.height-lipgloss.Height(m.renderLogo())-lipgloss.Height(m.renderHelpText())-2).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(styles.Subtext0.GetForeground()).
			Padding(1, 2).
			Render(styles.Subtext0.Render("No server selected."))
		return noSelectionMessage
	}

	srv := selected.server

	createdAt := srv.CreatedAt.Local().Format("02/01/2006 15:04:05")
	updatedAt := srv.UpdatedAt.Local().Format("02/01/2006 15:04:05")

	schemaShared := "No"
	if srv.ShareDatabaseSchemaLLM {
		schemaShared = "Yes"
	}

	var sb strings.Builder
	sb.WriteString("Name: " + srv.Name + "\n")
	sb.WriteString("Address: " + srv.Address + "\n")
	sb.WriteString("Port: " + strconv.Itoa(srv.Port) + "\n")
	sb.WriteString("Username: " + srv.Username + "\n")

	password := server.MaskedPassword
	connectionString := srv.MaskedString()
	if m.showPassword {
		password = srv.Password
		connectionString = srv.String()

		if password == "" {
			password = "N/A"
		}
	}

	sb.WriteString("Password: " + password + "\n")
	sb.WriteString("Database: " + srv.Database + "\n")

	sb.WriteString("Connection URI: " + connectionString + "\n")
	sb.WriteString("Share Database Schema with LLM: " + schemaShared + "\n")
	sb.WriteString("Created At: " + createdAt + "\n")
	sb.WriteString("Updated At: " + updatedAt + "\n")

	logoHeight := lipgloss.Height(m.renderLogo())
	helpHeight := lipgloss.Height(m.renderHelpText())

	serverInfo := lipgloss.NewStyle().
		Width(m.width/2).
		Height(m.height-logoHeight-helpHeight-2).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Subtext0.GetForeground()).
		Padding(1, 2).
		Render(sb.String())

	return serverInfo
}
