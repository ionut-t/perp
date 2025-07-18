package servers

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/perp/internal/constants"
	"github.com/ionut-t/perp/internal/version"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/ui/styles"
)

const (
	selectionListHeight    = 6
	selectionViewMinHeight = 12
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
	serverForm    *huh.Form
	selectForm    *huh.Form
	view          view
	editedServer  *server.Server
	width, height int
}

func New(storage string) Model {
	servers, err := server.Load(storage)

	if err != nil {
		servers = []server.Server{}
	}

	selectServer := huh.NewSelect[server.Server]()
	selectServer.Title("Select a server")
	options := make([]huh.Option[server.Server], len(servers))

	for i, srv := range servers {
		options[i] = huh.NewOption(srv.Name, srv)

	}
	selectServer.Options(options...)

	var currentView view
	if len(servers) == 0 {
		currentView = viewForm
	}

	m := Model{
		storage: storage,
		servers: servers,
		view:    currentView,
	}

	if currentView == viewForm {
		m.initialiseCreateForm()
	} else {
		m.initialiseSelectForm()
	}

	return m
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) Init() tea.Cmd {
	return cursor.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.view == viewForm && len(m.servers) > 0 {
				m.view = viewSelect
				m.serverForm = nil
			}

		case "n":
			if m.view == viewSelect {
				m.initialiseCreateForm()
				m.view = viewForm
				return m, cursor.Blink
			}

		case "e":
			if m.view == viewSelect && m.selectForm != nil {
				selected := m.selectForm.GetFocusedField().GetValue().(server.Server)
				m.editedServer = &selected
				m.initialiseUpdateForm()
				m.view = viewForm
				return m, cursor.Blink
			}

		case "ctrl+d":
			if m.view == viewSelect && m.selectForm != nil {
				selected := m.selectForm.GetFocusedField().GetValue().(server.Server)

				if servers, err := server.Delete(selected.ID, m.storage); err == nil {
					m.servers = servers

					if len(m.servers) == 0 {
						m.initialiseCreateForm()
						m.view = viewForm
						m.selectForm = nil
					} else {
						m.initialiseSelectForm()
						_, cmd := m.selectForm.Update(nil)

						return m, cmd
					}
				}
			}

		case "ctrl+c":
			return m, tea.Quit

		case "q":
			if m.view == viewSelect {
				return m, tea.Quit
			}

		case "enter":
			if m.selectForm != nil && m.view == viewSelect {
				if server, ok := m.selectForm.GetFocusedField().GetValue().(server.Server); ok {
					return m, func() tea.Msg {
						return SelectedServerMsg{Server: server}
					}

				}
			}
		}
	}

	if m.view == viewSelect && m.selectForm != nil {
		s, cmd := m.selectForm.Update(msg)
		m.selectForm = s.(*huh.Form)
		cmds = append(cmds, cmd)
	}

	if m.serverForm != nil && m.view == viewForm {
		form, cmd := m.serverForm.Update(msg)
		m.serverForm = form.(*huh.Form)
		cmds = append(cmds, cmd)

		if m.serverForm.State == huh.StateCompleted {
			if m.editedServer != nil {
				m.editServer()
				_, cmd = m.selectForm.Update(nil)

				return m, cmd
			} else {
				if cmd := m.createServer(); cmd != nil {
					return m, cmd
				}

				_, cmd = m.selectForm.Update(nil)

				return m, cmd
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.view == viewSelect {
		server := m.selectForm.GetFocusedField().GetValue().(server.Server)

		logo := m.renderLogo()
		logoHeight := lipgloss.Height(logo)

		if m.height-lipgloss.Height(logo) <= selectionViewMinHeight {
			logoHeight = 2
		}

		serverSelect := m.renderServerSelect(m.width/2, m.height-logoHeight)

		createdAt := server.CreatedAt.Local().Format("02/01/2006 15:04:05")
		updatedAt := server.UpdatedAt.Local().Format("02/01/2006 15:04:05")

		schemaShared := "No"
		if server.ShareDatabaseSchemaLLM {
			schemaShared = "Yes"
		}

		serverInfo := lipgloss.NewStyle().
			Width(m.width/2).
			Height(m.height-4-logoHeight).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("5")).
			Padding(0, 1).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					lipgloss.NewStyle().Render("Name: "+server.Name),
					lipgloss.NewStyle().Render("Address: "+server.Address),
					lipgloss.NewStyle().Render("Port: "+strconv.Itoa(server.Port)),
					lipgloss.NewStyle().Render("Username: "+server.Username),
					lipgloss.NewStyle().Render("Database: "+server.Database),
					lipgloss.NewStyle().Render("Share Database Schema with LLM: "+schemaShared),
					lipgloss.NewStyle().Render("Created At: "+createdAt),
					lipgloss.NewStyle().Render("Updated At: "+updatedAt),
				),
			)

		if m.width <= 80 {
			serverInfo = ""
			serverSelect = m.renderServerSelect(m.width-2, m.height-logoHeight)
		}

		if m.height-lipgloss.Height(logo) <= selectionViewMinHeight {
			return lipgloss.JoinHorizontal(
				lipgloss.Left,
				serverSelect,
				serverInfo,
			)
		}

		return lipgloss.JoinVertical(
			lipgloss.Left,
			logo,
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				serverSelect,
				serverInfo,
			),
		)
	}

	return m.serverForm.View()
}

func (m *Model) initialiseCreateForm() {
	name := huh.NewInput().Title("Name").Key("name")
	name.Validate(func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("name cannot be empty")
		}

		for _, srv := range m.servers {
			if srv.Name == s {
				return errors.New("a server with this name already exists")
			}
		}

		return nil
	})

	address := huh.NewInput().Title("Address").Key("address").Validate(validateAddress)
	port := huh.NewInput().Title("Port").Key("port").Validate(validatePort)
	username := huh.NewInput().Title("Username").Key("username").Validate(validateUsername)
	password := huh.NewInput().Title("Password").Key("password").EchoMode(huh.EchoModePassword)
	database := huh.NewInput().Title("Database").Key("database").Validate(validateDatabase)
	shareDatabaseSchemaLLM := huh.NewConfirm().
		Title("Share Database Schema with LLM?").
		Key("shareDatabaseSchemaLLM").
		Affirmative("Yes").
		Negative("No")

	name.Focus()

	m.serverForm = huh.NewForm(
		huh.NewGroup(
			name,
			address,
			port,
			username,
			password,
			database,
			shareDatabaseSchemaLLM,
		),
	)
	m.serverForm.WithTheme(styles.ThemeCatppuccin())
	m.serverForm.WithKeyMap(m.getKeymap())
}

func (m *Model) initialiseUpdateForm() {
	name := huh.NewInput().Title("Name").Key("name")
	name.Value(&m.editedServer.Name)
	name.Validate(func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("name cannot be empty")
		}

		for _, srv := range m.servers {
			if srv.ID != m.editedServer.ID && srv.Name == s {
				return errors.New("a server with this name already exists")
			}
		}

		return nil
	})

	address := huh.NewInput().Title("Address").Key("address").Validate(validateAddress)
	address.Value(&m.editedServer.Address)

	portValue := strconv.Itoa(m.editedServer.Port)
	port := huh.NewInput().Title("Port").Key("port").Validate(validatePort)
	port.Value(&portValue)

	username := huh.NewInput().Title("Username").Key("username").Validate(validateUsername)
	username.Value(&m.editedServer.Username)

	password := huh.NewInput().Title("Password").Key("password").EchoMode(huh.EchoModePassword)
	password.Value(&m.editedServer.Password)

	database := huh.NewInput().Title("Database").Key("database").Validate(validateDatabase)
	database.Value(&m.editedServer.Database)

	shareDatabaseSchemaLLM := huh.NewConfirm().
		Title("Share Database Schema with LLM?").
		Key("shareDatabaseSchemaLLM").
		Affirmative("Yes").
		Negative("No").
		Value(&m.editedServer.ShareDatabaseSchemaLLM)

	name.Focus()

	m.serverForm = huh.NewForm(
		huh.NewGroup(
			name,
			address,
			port,
			username,
			password,
			database,
			shareDatabaseSchemaLLM,
		),
	)

	m.serverForm.WithTheme(styles.ThemeCatppuccin())
	m.serverForm.WithKeyMap(m.getKeymap())
}

func (m *Model) initialiseSelectForm() {
	selectServer := huh.NewSelect[server.Server]()
	selectServer.WithTheme(styles.ThemeCatppuccin())
	selectServer.Title("Select a server")
	selectServer.Key("select")
	options := make([]huh.Option[server.Server], len(m.servers))
	selectServer.Height(selectionListHeight)

	for i, srv := range m.servers {
		options[i] = huh.NewOption(srv.Name, srv)
	}

	selectServer.Options(options...)
	selectServer.Focus()

	m.selectForm = huh.NewForm(
		huh.NewGroup(selectServer),
	)

}

func (m *Model) createServer() tea.Cmd {
	name := m.serverForm.GetString("name")
	address := m.serverForm.GetString("address")
	port := m.serverForm.GetString("port")
	username := m.serverForm.GetString("username")
	password := m.serverForm.GetString("password")
	database := m.serverForm.GetString("database")
	shareDatabaseSchemaLLM := m.serverForm.GetBool("shareDatabaseSchemaLLM")

	newServer := server.CreateServer{
		Name:                   name,
		Address:                address,
		Port:                   port,
		Username:               username,
		Password:               password,
		Database:               database,
		ShareDatabaseSchemaLLM: shareDatabaseSchemaLLM,
	}

	srv, err := server.New(newServer, m.storage)

	if err != nil {
		m.serverForm.State = huh.StateNormal
		m.view = viewForm
	} else {
		if len(m.servers) == 0 {
			return func() tea.Msg {
				return SelectedServerMsg{Server: *srv}
			}
		}

		m.servers = append(m.servers, *srv)
		slices.SortFunc(m.servers, func(a, b server.Server) int {
			return -1 * a.CreatedAt.Compare(b.CreatedAt)
		})
		m.view = viewSelect
		m.serverForm = nil
		m.initialiseSelectForm()
	}

	return nil
}

func (m *Model) editServer() {
	name := m.serverForm.GetString("name")
	address := m.serverForm.GetString("address")
	port := m.serverForm.GetString("port")
	username := m.serverForm.GetString("username")
	password := m.serverForm.GetString("password")
	database := m.serverForm.GetString("database")
	shareDatabaseSchemaLLM := m.serverForm.GetBool("shareDatabaseSchemaLLM")

	m.view = viewSelect
	m.serverForm = nil

	err := m.editedServer.Update(server.CreateServer{
		Name:                   name,
		Address:                address,
		Port:                   port,
		Username:               username,
		Password:               password,
		Database:               database,
		ShareDatabaseSchemaLLM: shareDatabaseSchemaLLM,
	}, m.storage)

	if err != nil {
		m.serverForm.State = huh.StateNormal
		m.view = viewForm
	} else {
		for i, srv := range m.servers {
			if srv.ID == m.editedServer.ID {
				m.servers[i] = *m.editedServer
				break
			}
		}

		slices.SortFunc(m.servers, func(a, b server.Server) int {
			return -1 * a.CreatedAt.Compare(b.CreatedAt)
		})

		m.editedServer = nil
		m.view = viewSelect
		m.initialiseSelectForm()
	}
}

func (m Model) getKeymap() *huh.KeyMap {
	keymap := huh.NewDefaultKeyMap()
	keymap.Confirm.Accept.Unbind()
	keymap.Confirm.Reject.Unbind()

	return keymap
}

func validateAddress(address string) error {
	if strings.TrimSpace(address) == "" {
		return errors.New("address cannot be empty")
	}
	return nil
}

func validatePort(port string) error {
	if strings.TrimSpace(port) == "" {
		return errors.New("port cannot be empty")
	}

	if _, err := strconv.Atoi(port); err != nil {
		return errors.New("port must be a valid integer")
	}

	return nil
}

func validateDatabase(database string) error {
	if strings.TrimSpace(database) == "" {
		return errors.New("database cannot be empty")
	}
	return nil
}

func validateUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username cannot be empty")
	}
	return nil
}

func (m *Model) renderServerSelect(width, height int) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.selectForm.View(),
				"\n",
				styles.Overlay0.Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						"Press n to create a new server",
						"Press e to edit the selected server",
						"Press ctrl+d to delete the selected server",
					),
				),
			),
		)
}

func (m *Model) renderLogo() string {
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
