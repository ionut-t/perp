package servers

import (
	"errors"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
)

type createServerMsg struct {
	server server.CreateServer
}

type updateServerMsg struct {
	server  server.Server
	updated server.CreateServer
}

type serverFormModel struct {
	serverForm   *huh.Form
	editedServer *server.Server
	servers      []server.Server
}

func newServerFormModel(servers []server.Server) serverFormModel {
	name := huh.NewInput().Title("Name").Key("name")
	name.Validate(func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("name cannot be empty")
		}

		for _, srv := range servers {
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

	serverForm := huh.NewForm(
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

	serverForm.WithTheme(styles.HuhThemeCatppuccin())
	serverForm.WithKeyMap(getKeymap())

	return serverFormModel{
		servers:    servers,
		serverForm: serverForm,
	}
}

func editServerFormModel(servers []server.Server, server *server.Server) serverFormModel {
	name := huh.NewInput().Title("Name").Key("name")
	name.Value(&server.Name)
	name.Validate(func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("name cannot be empty")
		}

		for _, srv := range servers {
			if server.ID != srv.ID && srv.Name == s {
				return errors.New("a server with this name already exists")
			}
		}

		return nil
	})

	address := huh.NewInput().Title("Address").Key("address").Validate(validateAddress)
	address.Value(&server.Address)

	portValue := strconv.Itoa(server.Port)
	port := huh.NewInput().Title("Port").Key("port").Validate(validatePort)
	port.Value(&portValue)

	username := huh.NewInput().Title("Username").Key("username").Validate(validateUsername)
	username.Value(&server.Username)

	password := huh.NewInput().Title("Password").Key("password").EchoMode(huh.EchoModePassword)
	password.Value(&server.Password)

	database := huh.NewInput().Title("Database").Key("database").Validate(validateDatabase)
	database.Value(&server.Database)

	shareDatabaseSchemaLLM := huh.NewConfirm().
		Title("Share Database Schema with LLM?").
		Key("shareDatabaseSchemaLLM").
		Affirmative("Yes").
		Negative("No").
		Value(&server.ShareDatabaseSchemaLLM)

	name.Focus()

	serverForm := huh.NewForm(
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

	serverForm.WithTheme(styles.HuhThemeCatppuccin())
	serverForm.WithKeyMap(getKeymap())

	return serverFormModel{
		servers:      servers,
		editedServer: server,
		serverForm:   serverForm,
	}
}

func (m serverFormModel) Init() tea.Cmd {
	return nil
}

func (m serverFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	serverForm, cmd := m.serverForm.Update(msg)
	m.serverForm = serverForm.(*huh.Form)
	cmds = append(cmds, cmd)

	if m.serverForm.State == huh.StateCompleted {
		value := server.CreateServer{
			Name:                   m.serverForm.GetString("name"),
			Address:                m.serverForm.GetString("address"),
			Port:                   m.serverForm.GetString("port"),
			Username:               m.serverForm.GetString("username"),
			Password:               m.serverForm.GetString("password"),
			Database:               m.serverForm.GetString("database"),
			ShareDatabaseSchemaLLM: m.serverForm.GetBool("shareDatabaseSchemaLLM"),
		}

		if m.editedServer != nil {
			return m, utils.Dispatch(updateServerMsg{
				server:  *m.editedServer,
				updated: value,
			})
		} else {
			return m, utils.Dispatch(createServerMsg{server: value})
		}
	}

	return m, tea.Batch(cmds...)
}

func (m serverFormModel) View() string {
	return styles.Primary.Render(m.serverForm.View())
}

func getKeymap() *huh.KeyMap {
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
