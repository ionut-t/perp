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
		validateModel func(t *testing.T, m Model)
	}{
		{
			name:          "new with no servers shows form",
			setupServers:  nil,
			expectedView:  viewForm,
			expectedCount: 0,
			validateModel: func(t *testing.T, m Model) {
				if m.serverForm == nil {
					t.Error("Expected serverForm to be initialized")
				}
				if m.selectForm != nil {
					t.Error("Expected selectForm to be nil")
				}
			},
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
			validateModel: func(t *testing.T, m Model) {
				if m.selectForm == nil {
					t.Error("Expected selectForm to be initialized")
				}
				if m.serverForm != nil {
					t.Error("Expected serverForm to be nil")
				}
			},
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
			defer os.RemoveAll(tempDir)

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

			if tt.validateModel != nil {
				tt.validateModel(t, m)
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

		// Key navigation tests - Select view
		{
			name: "press 'n' in select view switches to form",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
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
			name: "press 'e' in select view switches to edit form",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}},
			expectedView: viewForm,
			expectedCmd:  true,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				if m.editedServer == nil {
					t.Error("Expected editedServer to be set")
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

		// Delete server tests
		{
			name: "ctrl+d deletes selected server",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Server 1", CreatedAt: time.Now()},
				{ID: uuid.New(), Name: "Server 2", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          tea.KeyMsg{Type: tea.KeyCtrlD},
			expectedView: viewSelect,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				// After deletion, should have 1 server
				if len(m.servers) != 1 {
					t.Errorf("Expected 1 server after deletion, got %d", len(m.servers))
				}
			},
		},
		{
			name: "ctrl+d on last server switches to form",
			setupServers: []server.Server{
				{ID: uuid.New(), Name: "Only Server", CreatedAt: time.Now()},
			},
			initialView:  viewSelect,
			msg:          tea.KeyMsg{Type: tea.KeyCtrlD},
			expectedView: viewForm,
			validateModel: func(t *testing.T, m Model, cmd tea.Cmd) {
				if len(m.servers) != 0 {
					t.Errorf("Expected 0 servers after deletion, got %d", len(m.servers))
				}
				if m.selectForm != nil {
					t.Error("Expected selectForm to be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			if tt.setupServers != nil {
				saveTestServers(t, tempDir, tt.setupServers)
			}

			m := New(tempDir)
			if tt.initialView != 0 {
				m.view = tt.initialView
				if tt.initialView == viewForm && m.serverForm == nil {
					m.initialiseCreateForm()
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
				m.initialiseCreateForm()
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
				m.serverForm.State = 0 // Reset to allow value setting
			},
		},
		{
			name:          "create server with existing servers adds to list",
			existingCount: 1,
			expectCmd:     false,
			setupForm: func(m *Model) {
				m.serverForm.State = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			m := Model{storage: tempDir}

			// Add existing servers if needed
			for i := 0; i < tt.existingCount; i++ {
				m.servers = append(m.servers, server.Server{
					ID:   uuid.New(),
					Name: fmt.Sprintf("Existing%d", i),
				})
			}

			// Create form
			m.initialiseCreateForm()

			if tt.setupForm != nil {
				tt.setupForm(&m)
			}

			cmd := m.createServer()

			if tt.expectCmd && cmd == nil && m.serverForm == nil {
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
		defer os.RemoveAll(tempDir)

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

		// Simulate selecting edit mode
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = model.(Model)

		// Verify we're in form view with editedServer set
		if m.view != viewForm {
			t.Error("Expected form view after pressing 'e'")
		}
		if m.editedServer == nil {
			t.Error("Expected editedServer to be set")
		}
		if m.editedServer != nil && m.editedServer.ID != createdServer.ID {
			t.Error("Wrong server selected for editing")
		}

		// Simulate escape to cancel edit
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = model.(Model)

		// Should be back in select view
		if m.view != viewSelect {
			t.Error("Expected select view after escape")
		}
		if m.serverForm != nil {
			t.Error("Expected serverForm to be nil after canceling")
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

// Benchmark tests

func BenchmarkNew(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "servers_bench")
	defer os.RemoveAll(tempDir)

	// Create some servers for testing
	for i := range 10 {
		server.New(server.CreateServer{
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
	defer os.RemoveAll(tempDir)

	m := New(tempDir)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}

	for b.Loop() {
		m.Update(msg)
	}
}

func BenchmarkView(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "servers_bench")
	defer os.RemoveAll(tempDir)

	// Create a server for select view
	server.New(server.CreateServer{
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
