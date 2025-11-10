package servers

import (
	"sort"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/pkg/server"
)

type SelectedServerMsg struct {
	Server server.Server
}

type view int

const (
	viewSelect view = iota
	viewForm
)

type Model struct {
	storage       string
	servers       []server.Server
	serverForm    serverFormModel
	serversList   serversListModel
	view          view
	width, height int
}

func New(storage string) Model {
	servers, err := server.Load(storage)
	if err != nil {
		servers = []server.Server{}
	}

	var currentView view
	var serverForm serverFormModel
	if len(servers) == 0 {
		currentView = viewForm
		serverForm = newServerFormModel(servers)
	}

	return Model{
		storage:     storage,
		servers:     servers,
		view:        currentView,
		serversList: newServersListModel(servers),
		serverForm:  serverForm,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.serversList.setSize(width, height)
}

func (m Model) Init() tea.Cmd {
	return cursor.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case newServerMsg:
		m.serverForm = newServerFormModel(m.servers)
		m.view = viewForm
		return m, cursor.Blink

	case editServerMsg:
		m.serverForm = editServerFormModel(m.servers, &msg.Server)
		m.view = viewForm
		return m, cursor.Blink

	case deleteServerMsg:
		if servers, err := server.Delete(msg.Server.ID, m.storage); err == nil {
			m.servers = servers
			m.serversList.setServers(m.servers)

			if len(m.servers) == 0 {
				m.serverForm = newServerFormModel(m.servers)
				m.view = viewForm
			}
		}

	case createServerMsg:
		return m, m.createServer(msg.server)

	case updateServerMsg:
		m.editServer(msg.server, msg.updated)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.view == viewForm && len(m.servers) > 0 {
				m.view = viewSelect
				return m, nil
			}

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	switch m.view {
	case viewSelect:
		serverList, cmd := m.serversList.Update(msg)
		m.serversList = serverList.(serversListModel)
		cmds = append(cmds, cmd)
	case viewForm:
		serverForm, cmd := m.serverForm.Update(msg)
		m.serverForm = serverForm.(serverFormModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.view {
	case viewSelect:
		return m.serversList.View()
	case viewForm:
		return m.serverForm.View()
	}

	return ""
}

func (m *Model) createServer(newServer server.CreateServer) tea.Cmd {
	srv, err := server.New(newServer, m.storage)

	if err != nil {
		m.view = viewForm
	} else {
		if len(m.servers) == 0 {
			return func() tea.Msg {
				return SelectedServerMsg{Server: *srv}
			}
		}

		m.servers = append(m.servers, *srv)
		sort.Slice(m.servers, func(i, j int) bool {
			return m.servers[i].CreatedAt.After(m.servers[j].CreatedAt)
		})
		m.view = viewSelect
		m.serversList.setServers(m.servers)
	}

	return nil
}

func (m *Model) editServer(server server.Server, updatedServer server.CreateServer) {
	m.view = viewSelect

	err := server.Update(updatedServer, m.storage)

	if err != nil {
		m.view = viewForm
	} else {
		for i, srv := range m.servers {
			if srv.ID == server.ID {
				m.servers[i] = server
				break
			}
		}

		sort.Slice(m.servers, func(i, j int) bool {
			return m.servers[i].CreatedAt.After(m.servers[j].CreatedAt)
		})

		m.view = viewSelect
		m.serversList.setServers(m.servers)
	}
}

func (m Model) CanTriggerLeaderKey() bool {
	return m.view == viewSelect
}
