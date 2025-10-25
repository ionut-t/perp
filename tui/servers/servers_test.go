package servers

import (
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/ionut-t/perp/pkg/server"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupServers  []server.Server
		expectedView  view
		expectedCount int
	}{
		{
			name:          "new with no servers shows form",
			setupServers:  nil,
			expectedView:  viewForm,
			expectedCount: 0,
		},
		{
			name: "new with servers shows select",
			setupServers: []server.Server{
				{
					ID:        uuid.New(),
					Name:      "Test Server",
					Address:   "localhost",
					Port:      5432,
					Database:  "testdb",
					Username:  "user",
					Password:  "pass",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expectedView:  viewSelect,
			expectedCount: 1,
		},
		{
			name: "new with multiple servers",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
				{ID: uuid.New(), Name: "Server 2", CreatedAt: time.Now()},
				{ID: uuid.New(), Name: "Server 3", CreatedAt: time.Now()},
			},
			expectedView:  viewSelect,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			if tt.setupServers != nil {
				saveTestServers(t, tempDir, tt.setupServers)
			}

			m := New(tempDir)

			if m.view != tt.expectedView {
				t.Errorf("Expected view %v, got %v", tt.expectedView, m.view)
			}

			if len(m.servers) != tt.expectedCount {
				t.Errorf("Expected %d servers, got %d", tt.expectedCount, len(m.servers))
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupServers  []server.Server
		initialView   view
		msg           tea.Msg
		expectedView  view
		expectedCmd   bool
		validateModel func(t *testing.T, m Model, cmd tea.Cmd)
	}{
		// Window resize tests
		{
			name: "window resize updates dimensions",
			msg:  tea.WindowSizeMsg{Width: 100, Height: 50},
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				if m.width != 100 || m.height != 50 {
					t.Errorf("Expected dimensions 100x50, got %dx%d", m.width, m.height)
				}
			},
		},

		// Message handling tests
		{
			name: "newServerMsg switches to form",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          newServerMsg{},
			expectedView: viewForm,
			expectedCmd:  true,
		},
		{
			name: "press 'q' in select view quits",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView: viewSelect,
			msg:         tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expectedCmd: true,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				if cmd == nil {
					t.Error("Expected quit command")
				}
			},
		},
		{
			name: "edit server message switches to form",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          editServerMsg{Server: server.Server{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()}},
			expectedView: viewForm,
			expectedCmd:  true,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				// editedServer is now handled internally by serverForm
				if m.view != viewForm {
					t.Error("Expected to be in form view for editing")
				}
			},
		},
		{
			name: "press enter in select view returns selected server",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView: viewSelect,
			msg:         tea.KeyMsg{Type: tea.KeyEnter},
			expectedCmd: true,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				if cmd == nil {
					t.Error("Expected command to be returned")
				}
			},
		},

		// Key navigation tests - Form view
		{
			name:         "press esc in form view with servers returns to select",
			setupServers: []server.Server{{ID: uuid.New(), Name: "Server 1"}},
			initialView:  viewForm,
			msg:          tea.KeyMsg{Type: tea.KeyEsc},
			expectedView: viewSelect,
		},
		{
			name:         "press esc in form view without servers stays in form",
			setupServers: nil,
			initialView:  viewForm,
			msg:          tea.KeyMsg{Type: tea.KeyEsc},
			expectedView: viewForm,
		},

		// Ctrl+C tests
		{
			name:        "ctrl+c quits from any view",
			initialView: viewSelect,
			msg:         tea.KeyMsg{Type: tea.KeyCtrlC},
			expectedCmd: true,
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			if tt.setupServers != nil {
				saveTestServers(t, tempDir, tt.setupServers)
			}

			m := New(tempDir)
			if tt.initialView != 0 {
				m.view = tt.initialView
				if tt.initialView == viewForm {
					m.serverForm = newServerFormModel(m.servers)
				}
			}

			model, cmd := m.Update(tt.msg)
			updatedModel := model.(Model)

			if tt.expectedView != 0 && updatedModel.view != tt.expectedView {
				t.Errorf("Expected view %v, got %v", tt.expectedView, updatedModel.view)
			}

			if tt.expectedCmd && cmd == nil {
				t.Error("Expected command but got nil")
			}

			if tt.validateModel != nil {
				tt.validateModel(t, updatedModel, cmd)
			}
		})
	}
}

func TestView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupModel   func() Model
		validateView func(t *testing.T, view string)
	}{
		{
			name: "view renders form when in form view",
			setupModel: func() Model {
				m := Model{view: viewForm}
				m.serverForm = newServerFormModel([]server.Server{})
				return m
			},
			validateView: func(t *testing.T, view string) {
				if view == "" {
					t.Error("Expected non-empty view")
				}
			},
		},
		{
			name: "view renders select when in select view",
			setupModel: func() Model {
				tempDir := setupTempDir(t)
				servers := []server.Server{
					{
						ID:                     uuid.New(),
						Name:                   "Test Server",
						Address:                "localhost",
						Port:                   5432,
						Database:               "testdb",
						Username:               "testuser",
						ShareDatabaseSchemaLLM: true,
						CreatedAt:              time.Now(),
						UpdatedAt:              time.Now(),
					},
				}
				saveTestServers(t, tempDir, servers)
				m := New(tempDir)
				m.width = 100
				m.height = 50
				return m
			},
			validateView: func(t *testing.T, view string) {
				if view == "" {
					t.Error("Expected non-empty view")
				}
				// Check for key instructions
				expectedTexts := []string{
					"Press n to create",
					"Press e to edit",
					"Press ctrl+d to delete",
				}
				for _, text := range expectedTexts {
					if !contains(view, text) {
						t.Errorf("Expected view to contain '%s'", text)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			view := m.View()
			tt.validateView(t, view)
		})
	}
}

func TestCreateServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		existingCount int
		expectCmd     bool
		setupForm     func(m *Model)
	}{
		{
			name:          "create server with no existing servers returns cmd",
			existingCount: 0,
			expectCmd:     true,
			setupForm: func(m *Model) {
				// serverForm is initialized, no need to reset state
			},
		},
		{
			name:          "create server with existing servers adds to list",
			existingCount: 1,
			expectCmd:     false,
			setupForm: func(m *Model) {
				// serverForm is initialized, no need to reset state
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			// Add existing servers if needed
			for i := 0; i < tt.existingCount; i++ {
				_, _ = server.New(server.CreateServer{
					Name:     fmt.Sprintf("Existing%d", i),
					Address:  "localhost",
					Port:     "5432",
					Username: "user",
					Password: "pass",
					Database: "db",
				}, tempDir)
			}

			// Initialize model with New() to ensure proper initialization
			m := New(tempDir)

			// Create form
			m.serverForm = newServerFormModel(m.servers)

			if tt.setupForm != nil {
				tt.setupForm(&m)
			}

			// Create a test server
			testServer := server.CreateServer{
				Name:     "Test Server",
				Address:  "localhost",
				Port:     "5432",
				Username: "user",
				Password: "pass",
				Database: "testdb",
			}
			cmd := m.createServer(testServer)

			if tt.expectCmd && cmd == nil {
				t.Error("Expected command but got nil")
			}
			if !tt.expectCmd && cmd != nil {
				t.Error("Expected no command but got one")
			}
		})
	}
}

func TestEditServerFlow(t *testing.T) {
	t.Parallel()

	t.Run("test edit server state transitions", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		// Create initial server
		createdServer, _ := server.New(server.CreateServer{
			Name:     "Test Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)

		// Set proper dimensions for the model
		m.SetSize(100, 50)

		// Verify we start in select view
		if m.view != viewSelect {
			t.Error("Expected select view initially")
		}

		// Manually trigger editServerMsg since key press 'e' goes through the list
		model, _ := m.Update(editServerMsg{Server: *createdServer})
		m = model.(Model)

		// Verify we're in form view
		if m.view != viewForm {
			t.Error("Expected form view after edit message")
		}

		// Simulate escape to cancel edit
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = model.(Model)

		// Should be back in select view
		if m.view != viewSelect {
			t.Error("Expected select view after escape")
		}
	})
}

func TestValidationFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validator func(string) error
		input     string
		expectErr bool
	}{
		// Address validation
		{"valid address", validateAddress, "localhost", false},
		{"empty address", validateAddress, "", true},
		{"address with spaces", validateAddress, "  ", true},

		// Port validation
		{"valid port", validatePort, "5432", false},
		{"empty port", validatePort, "", true},
		{"non-numeric port", validatePort, "abc", true},
		{"negative port", validatePort, "-1", false}, // Allowed by current implementation

		// Database validation
		{"valid database", validateDatabase, "mydb", false},
		{"empty database", validateDatabase, "", true},

		// Username validation
		{"valid username", validateUsername, "user", false},
		{"empty username", validateUsername, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.input)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Helper functions

func removeTempDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to remove temp dir %s: %v", dir, err)
	}
}

func setupTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "servers_ui_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

func saveTestServers(t *testing.T, storage string, servers []server.Server) {
	t.Helper()
	for _, srv := range servers {
		_, err := server.New(server.CreateServer{
			Name:                   srv.Name,
			Address:                srv.Address,
			Port:                   fmt.Sprintf("%d", srv.Port),
			Username:               srv.Username,
			Password:               srv.Password,
			Database:               srv.Database,
			ShareDatabaseSchemaLLM: srv.ShareDatabaseSchemaLLM,
		}, storage)
		if err != nil {
			// If creation fails due to duplicate, that's okay for our tests
			if !contains(err.Error(), "already exists") {
				t.Fatalf("Failed to save test server: %v", err)
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Integration tests - test full key press flows

func TestKeyPressIntegration(t *testing.T) {
	t.Parallel()

	t.Run("press 'n' key creates newServerMsg", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		// Create a server so we start in select view
		_, _ = server.New(server.CreateServer{
			Name:     "Test Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)

		// Send 'n' key to the model
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		m = model.(Model)

		// Should return a command that generates newServerMsg
		if cmd == nil {
			t.Fatal("Expected command from 'n' key press")
		}

		// Execute the command to get the message
		msg := cmd()
		if _, ok := msg.(newServerMsg); !ok {
			t.Errorf("Expected newServerMsg, got %T", msg)
		}

		// Process the message
		model, _ = m.Update(msg)
		m = model.(Model)

		// Should now be in form view
		if m.view != viewForm {
			t.Errorf("Expected form view after processing newServerMsg, got %v", m.view)
		}
	})

	t.Run("press 'e' key creates editServerMsg", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		createdServer, _ := server.New(server.CreateServer{
			Name:     "Test Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)

		// Send 'e' key
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = model.(Model)

		if cmd == nil {
			t.Fatal("Expected command from 'e' key press")
		}

		// Execute command
		msg := cmd()
		editMsg, ok := msg.(editServerMsg)
		if !ok {
			t.Fatalf("Expected editServerMsg, got %T", msg)
		}

		// Verify it's editing the correct server
		if editMsg.Server.ID != createdServer.ID {
			t.Error("editServerMsg contains wrong server ID")
		}

		// Process message
		model, _ = m.Update(msg)
		m = model.(Model)

		if m.view != viewForm {
			t.Error("Expected form view after processing editServerMsg")
		}
	})

	t.Run("press ctrl+d deletes server", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		// Create two servers
		_, _ = server.New(server.CreateServer{
			Name:     "Server 1",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		_, _ = server.New(server.CreateServer{
			Name:     "Server 2",
			Address:  "localhost",
			Port:     "5433",
			Username: "user",
			Password: "pass",
			Database: "db2",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)

		if len(m.servers) != 2 {
			t.Fatalf("Expected 2 servers, got %d", len(m.servers))
		}

		// Get the currently selected server (first in list)
		firstServerID := m.servers[0].ID

		// Send ctrl+d
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		m = model.(Model)

		if cmd == nil {
			t.Fatal("Expected command from ctrl+d")
		}

		// Execute command
		msg := cmd()
		deleteMsg, ok := msg.(deleteServerMsg)
		if !ok {
			t.Fatalf("Expected deleteServerMsg, got %T", msg)
		}

		// Verify the message contains the first server (which is selected by default)
		if deleteMsg.Server.ID != firstServerID {
			t.Errorf("deleteServerMsg contains wrong server ID. Expected %s, got %s",
				firstServerID, deleteMsg.Server.ID)
		}

		// Process message
		model, _ = m.Update(msg)
		m = model.(Model)

		// Should have 1 server left
		if len(m.servers) != 1 {
			t.Errorf("Expected 1 server after deletion, got %d", len(m.servers))
		}

		// Verify the deleted server is not in the list
		for _, srv := range m.servers {
			if srv.ID == firstServerID {
				t.Error("Deleted server still in list")
			}
		}
	})

	t.Run("deleting last server switches to form view", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		// Create only one server
		_, _ = server.New(server.CreateServer{
			Name:     "Only Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)

		if len(m.servers) != 1 {
			t.Fatalf("Expected 1 server, got %d", len(m.servers))
		}

		// Verify we start in select view
		if m.view != viewSelect {
			t.Fatal("Expected select view initially")
		}

		// Get the only server's ID
		serverID := m.servers[0].ID

		// Send ctrl+d to delete
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		m = model.(Model)

		if cmd == nil {
			t.Fatal("Expected command from ctrl+d")
		}

		// Execute command
		msg := cmd()
		deleteMsg, ok := msg.(deleteServerMsg)
		if !ok {
			t.Fatalf("Expected deleteServerMsg, got %T", msg)
		}

		// Verify correct server
		if deleteMsg.Server.ID != serverID {
			t.Error("deleteServerMsg contains wrong server ID")
		}

		// Process deletion message
		model, _ = m.Update(msg)
		m = model.(Model)

		// Should now have 0 servers
		if len(m.servers) != 0 {
			t.Errorf("Expected 0 servers after deletion, got %d", len(m.servers))
		}

		// Should have switched to form view
		if m.view != viewForm {
			t.Errorf("Expected form view after deleting last server, got %v", m.view)
		}
	})
}

func TestUILayout(t *testing.T) {
	t.Parallel()

	t.Run("view contains help text", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		_, _ = server.New(server.CreateServer{
			Name:     "Test Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)
		view := m.View()

		expectedHelpText := []string{
			"Press n to create a new server",
			"Press e to edit the selected server",
			"Press ctrl+d to delete the selected server",
		}

		for _, text := range expectedHelpText {
			if !contains(view, text) {
				t.Errorf("View missing help text: %s", text)
			}
		}
	})

	t.Run("view shows server info panel", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		_, _ = server.New(server.CreateServer{
			Name:                   "Test Server",
			Address:                "testhost",
			Port:                   "5432",
			Username:               "testuser",
			Password:               "pass",
			Database:               "testdb",
			ShareDatabaseSchemaLLM: true,
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)
		view := m.View()

		expectedInfo := []string{
			"Name: Test Server",
			"Address: testhost",
			"Port: 5432",
			"Username: testuser",
			"Database: testdb",
			"Share Database Schema with LLM: Yes",
		}

		for _, info := range expectedInfo {
			if !contains(view, info) {
				t.Errorf("View missing server info: %s", info)
			}
		}
	})

	t.Run("view shows Select a server title", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		_, _ = server.New(server.CreateServer{
			Name:     "Test",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)
		view := m.View()

		if !contains(view, "Select a server") {
			t.Error("View missing 'Select a server' title")
		}
	})

	t.Run("empty state shows appropriate message", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		// Create and immediately delete to get empty state
		srv, _ := server.New(server.CreateServer{
			Name:     "Temp",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		_, _ = server.Delete(srv.ID, tempDir)

		// Now create model with no servers, but force select view
		m := Model{
			storage:     tempDir,
			servers:     []server.Server{},
			serversList: newServersListModel([]server.Server{}),
			view:        viewSelect,
			width:       100,
			height:      50,
		}
		m.serversList.setSize(100, 50)

		view := m.View()

		if !contains(view, "No servers available") {
			t.Error("Empty state should show 'No servers available'")
		}
	})
}

func TestServerListNavigation(t *testing.T) {
	t.Parallel()

	t.Run("enter key on selected server returns SelectedServerMsg", func(t *testing.T) {
		tempDir := setupTempDir(t)
		defer removeTempDir(t, tempDir)

		srv, _ := server.New(server.CreateServer{
			Name:     "Test Server",
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)

		m := New(tempDir)
		m.SetSize(100, 50)

		// Press enter
		model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = model.(Model)

		if cmd == nil {
			t.Fatal("Expected command from enter key")
		}

		msg := cmd()
		selectedMsg, ok := msg.(SelectedServerMsg)
		if !ok {
			t.Fatalf("Expected SelectedServerMsg, got %T", msg)
		}

		if selectedMsg.Server.ID != srv.ID {
			t.Error("SelectedServerMsg contains wrong server")
		}
	})
}

// Benchmark tests

func BenchmarkNew(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "servers_bench")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create some servers for testing
	for i := range 10 {
		_, _ = server.New(server.CreateServer{
			Name:     "Server" + string(rune('A'+i)),
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}, tempDir)
	}

	for b.Loop() {
		New(tempDir)
	}
}

func BenchmarkUpdate(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "servers_bench")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	m := New(tempDir)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	for b.Loop() {
		m.Update(msg)
	}
}

func BenchmarkView(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "servers_bench")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a server for select view
	_, _ = server.New(server.CreateServer{
		Name:     "Bench Server",
		Address:  "localhost",
		Port:     "5432",
		Username: "user",
		Password: "pass",
		Database: "db",
	}, tempDir)

	m := New(tempDir)
	m.width = 100
	m.height = 50

	for b.Loop() {
		_ = m.View()
	}
}
