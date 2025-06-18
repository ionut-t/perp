package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       CreateServer
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, srv *Server)
	}{
		{
			name: "create valid server",
			input: CreateServer{
				Name:                   "Test Server",
				Address:                "localhost",
				Port:                   "5432",
				Username:               "user",
				Password:               "pass",
				Database:               "testdb",
				ShareDatabaseSchemaLLM: true,
			},
			expectError: false,
			validate: func(t *testing.T, srv *Server) {
				if srv.Name != "Test Server" {
					t.Errorf("Expected name 'Test Server', got '%s'", srv.Name)
				}
				if srv.Port != 5432 {
					t.Errorf("Expected port 5432, got %d", srv.Port)
				}
				if srv.ID == uuid.Nil {
					t.Error("Expected valid UUID, got nil")
				}
				if srv.ShareDatabaseSchemaLLM != true {
					t.Error("Expected ShareDatabaseSchemaLLM to be true")
				}
			},
		},
		{
			name: "invalid port",
			input: CreateServer{
				Name:     "Test Server",
				Address:  "localhost",
				Port:     "invalid",
				Username: "user",
				Password: "pass",
				Database: "testdb",
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "empty port",
			input: CreateServer{
				Name:     "Test Server",
				Address:  "localhost",
				Port:     "",
				Username: "user",
				Password: "pass",
				Database: "testdb",
			},
			expectError: true,
			errorMsg:    "invalid port",
		},
		{
			name: "duplicate server name",
			input: CreateServer{
				Name:     "Duplicate Server",
				Address:  "localhost",
				Port:     "5432",
				Username: "user",
				Password: "pass",
				Database: "testdb",
			},
			expectError: false,
			validate: func(t *testing.T, srv *Server) {
				// First creation should succeed
				if srv == nil {
					t.Error("Expected server to be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			if tt.name == "duplicate server name" {
				first, err := New(tt.input, tempDir)
				if err != nil {
					t.Fatalf("Failed to create first server: %v", err)
				}

				second, err := New(tt.input, tempDir)
				if err == nil {
					t.Error("Expected error when creating duplicate server name")
				}
				if second != nil {
					t.Error("Expected nil server when creation fails")
				}
				if err != nil && !contains(err.Error(), "already exists") {
					t.Errorf("Expected 'already exists' error, got: %v", err)
				}

				if tt.validate != nil {
					tt.validate(t, first)
				}
				return
			}

			srv, err := New(tt.input, tempDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if err != nil && tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, srv)
			}

			servers, err := Load(tempDir)
			if err != nil {
				t.Fatalf("Failed to load servers: %v", err)
			}
			if len(servers) != 1 {
				t.Errorf("Expected 1 server, got %d", len(servers))
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupServers  []Server
		expectError   bool
		expectedCount int
		validateOrder func(t *testing.T, servers []Server)
	}{
		{
			name:          "load empty file",
			setupServers:  nil,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "load single server",
			setupServers: []Server{
				{
					ID:        uuid.New(),
					Name:      "Server 1",
					Address:   "localhost",
					Port:      5432,
					Username:  "user",
					Password:  "pass",
					Database:  "db1",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "load multiple servers - verify they maintain file order",
			setupServers: []Server{
				{
					ID:        uuid.New(),
					Name:      "New Server",
					Address:   "localhost",
					Port:      5433,
					Username:  "user",
					Password:  "pass",
					Database:  "db2",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        uuid.New(),
					Name:      "Old Server",
					Address:   "localhost",
					Port:      5432,
					Username:  "user",
					Password:  "pass",
					Database:  "db1",
					CreatedAt: time.Now().Add(-time.Hour),
					UpdatedAt: time.Now().Add(-time.Hour),
				},
			},
			expectError:   false,
			expectedCount: 2,
			validateOrder: func(t *testing.T, servers []Server) {
				if servers[0].Name != "New Server" {
					t.Errorf("Expected 'New Server' first, got '%s'", servers[0].Name)
				}
				if servers[1].Name != "Old Server" {
					t.Errorf("Expected 'Old Server' second, got '%s'", servers[1].Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			if tt.setupServers != nil {
				if err := saveServers(tempDir, tt.setupServers); err != nil {
					t.Fatalf("Failed to setup servers: %v", err)
				}
			}

			servers, err := Load(tempDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(servers) != tt.expectedCount {
				t.Errorf("Expected %d servers, got %d", tt.expectedCount, len(servers))
			}

			if tt.validateOrder != nil {
				tt.validateOrder(t, servers)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initial     Server
		update      CreateServer
		expectError bool
		validate    func(t *testing.T, srv *Server)
	}{
		{
			name: "update all fields",
			initial: Server{
				ID:                     uuid.New(),
				Name:                   "Original",
				Address:                "localhost",
				Port:                   5432,
				Username:               "user1",
				Password:               "pass1",
				Database:               "db1",
				ShareDatabaseSchemaLLM: false,
				CreatedAt:              time.Now().Add(-time.Hour),
				UpdatedAt:              time.Now().Add(-time.Hour),
			},
			update: CreateServer{
				Name:                   "Updated",
				Address:                "newhost",
				Port:                   "5433",
				Username:               "user2",
				Password:               "pass2",
				Database:               "db2",
				ShareDatabaseSchemaLLM: true,
			},
			expectError: false,
			validate: func(t *testing.T, srv *Server) {
				if srv.Name != "Updated" {
					t.Errorf("Expected name 'Updated', got '%s'", srv.Name)
				}
				if srv.Port != 5433 {
					t.Errorf("Expected port 5433, got %d", srv.Port)
				}
				if srv.ShareDatabaseSchemaLLM != true {
					t.Error("Expected ShareDatabaseSchemaLLM to be true")
				}
				if srv.UpdatedAt.Before(srv.CreatedAt) {
					t.Error("UpdatedAt should be after CreatedAt")
				}
			},
		},
		{
			name: "update with invalid port",
			initial: Server{
				ID:        uuid.New(),
				Name:      "Original",
				Address:   "localhost",
				Port:      5432,
				Username:  "user",
				Password:  "pass",
				Database:  "db",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			update: CreateServer{
				Name:     "Updated",
				Address:  "localhost",
				Port:     "not-a-number",
				Username: "user",
				Password: "pass",
				Database: "db",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			if err := saveServers(tempDir, []Server{tt.initial}); err != nil {
				t.Fatalf("Failed to save initial server: %v", err)
			}

			srv := tt.initial
			err := srv.Update(tt.update, tempDir)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, &srv)
			}

			servers, err := Load(tempDir)
			if err != nil {
				t.Fatalf("Failed to load servers: %v", err)
			}
			if len(servers) != 1 {
				t.Errorf("Expected 1 server, got %d", len(servers))
			}
			if servers[0].Name != srv.Name {
				t.Error("Update was not persisted")
			}
		})
	}
}

func TestEnableDatabaseSchemaLLM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initial      bool
		enable       bool
		expectChange bool
	}{
		{
			name:         "enable from false",
			initial:      false,
			enable:       true,
			expectChange: true,
		},
		{
			name:         "disable from true",
			initial:      true,
			enable:       false,
			expectChange: true,
		},
		{
			name:         "no change when already enabled",
			initial:      true,
			enable:       true,
			expectChange: false,
		},
		{
			name:         "no change when already disabled",
			initial:      false,
			enable:       false,
			expectChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			srv := Server{
				ID:                     uuid.New(),
				Name:                   "Test Server",
				Address:                "localhost",
				Port:                   5432,
				Username:               "user",
				Password:               "pass",
				Database:               "db",
				ShareDatabaseSchemaLLM: tt.initial,
				CreatedAt:              time.Now().Add(-time.Hour),
				UpdatedAt:              time.Now().Add(-time.Hour),
			}

			originalUpdatedAt := srv.UpdatedAt

			if err := saveServers(tempDir, []Server{srv}); err != nil {
				t.Fatalf("Failed to save server: %v", err)
			}

			err := srv.EnableDatabaseSchemaLLM(tt.enable, tempDir)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if srv.ShareDatabaseSchemaLLM != tt.enable {
				t.Errorf("Expected ShareDatabaseSchemaLLM to be %v, got %v", tt.enable, srv.ShareDatabaseSchemaLLM)
			}

			if tt.expectChange {
				if srv.UpdatedAt.Equal(originalUpdatedAt) {
					t.Error("Expected UpdatedAt to change")
				}
			} else {
				if !srv.UpdatedAt.Equal(originalUpdatedAt) {
					t.Error("Expected UpdatedAt to remain unchanged")
				}
			}

			servers, err := Load(tempDir)
			if err != nil {
				t.Fatalf("Failed to load servers: %v", err)
			}
			if len(servers) != 1 {
				t.Fatalf("Expected 1 server, got %d", len(servers))
			}
			if servers[0].ShareDatabaseSchemaLLM != tt.enable {
				t.Error("Change was not persisted")
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupServers   []Server
		deleteID       uuid.UUID
		expectedCount  int
		expectError    bool
		validateResult func(t *testing.T, remaining []Server)
	}{
		{
			name: "delete existing server",
			setupServers: []Server{
				{
					ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:      "Server 1",
					Address:   "localhost",
					Port:      5432,
					CreatedAt: time.Now(),
				},
				{
					ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					Name:      "Server 2",
					Address:   "localhost",
					Port:      5433,
					CreatedAt: time.Now(),
				},
			},
			deleteID:      uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			expectedCount: 1,
			expectError:   false,
			validateResult: func(t *testing.T, remaining []Server) {
				if remaining[0].Name != "Server 2" {
					t.Errorf("Wrong server remained, expected 'Server 2', got '%s'", remaining[0].Name)
				}
			},
		},
		{
			name: "delete non-existent server",
			setupServers: []Server{
				{
					ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
					Name:      "Server 1",
					Address:   "localhost",
					Port:      5432,
					CreatedAt: time.Now(),
				},
			},
			deleteID:      uuid.MustParse("99999999-9999-9999-9999-999999999999"),
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "delete from empty list",
			setupServers:  nil,
			deleteID:      uuid.New(),
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer os.RemoveAll(tempDir)

			if tt.setupServers != nil {
				if err := saveServers(tempDir, tt.setupServers); err != nil {
					t.Fatalf("Failed to setup servers: %v", err)
				}
			}

			remaining, err := Delete(tt.deleteID, tempDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(remaining) != tt.expectedCount {
				t.Errorf("Expected %d servers remaining, got %d", tt.expectedCount, len(remaining))
			}

			if tt.validateResult != nil {
				tt.validateResult(t, remaining)
			}

			// Verify deletion was persisted
			servers, err := Load(tempDir)
			if err != nil {
				t.Fatalf("Failed to load servers: %v", err)
			}
			if len(servers) != tt.expectedCount {
				t.Error("Deletion was not persisted")
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		server   Server
		expected string
	}{
		{
			name: "standard connection string",
			server: Server{
				Username: "user",
				Password: "pass",
				Address:  "localhost",
				Port:     5432,
				Database: "mydb",
			},
			expected: "postgres://user:pass@localhost:5432/mydb",
		},
		{
			name: "connection with special characters",
			server: Server{
				Username: "user@domain",
				Password: "p@ss!word",
				Address:  "db.example.com",
				Port:     5433,
				Database: "my-db",
			},
			expected: "postgres://user@domain:p@ss!word@db.example.com:5433/my-db",
		},
		{
			name: "connection with IP address",
			server: Server{
				Username: "admin",
				Password: "secret",
				Address:  "192.168.1.100",
				Port:     5432,
				Database: "production",
			},
			expected: "postgres://admin:secret@192.168.1.100:5432/production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.server.ConnectionString()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFilePermissions(t *testing.T) {
	t.Parallel()

	tempDir := setupTempDir(t)
	defer os.RemoveAll(tempDir)

	server := CreateServer{
		Name:     "Test Server",
		Address:  "localhost",
		Port:     "5432",
		Username: "user",
		Password: "pass",
		Database: "db",
	}

	_, err := New(server, tempDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	filePath := filepath.Join(tempDir, "servers.json")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0644 {
		t.Errorf("Expected file permissions 0644, got %v", mode.Perm())
	}
}

func TestJSONMarshaling(t *testing.T) {
	t.Parallel()

	tempDir := setupTempDir(t)
	defer os.RemoveAll(tempDir)

	server := CreateServer{
		Name:                   "Test Server",
		Address:                "localhost",
		Port:                   "5432",
		Username:               "user",
		Password:               "pass",
		Database:               "testdb",
		ShareDatabaseSchemaLLM: true,
	}

	created, err := New(server, tempDir)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	filePath := filepath.Join(tempDir, "servers.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var rawServers []map[string]any
	if err := json.Unmarshal(data, &rawServers); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(rawServers) != 1 {
		t.Fatalf("Expected 1 server in JSON, got %d", len(rawServers))
	}

	srv := rawServers[0]

	expectedFields := []string{
		"id", "name", "address", "port", "database",
		"username", "password", "createdAt", "updatedAt",
		"shareDatabaseSchemaLLM",
	}

	for _, field := range expectedFields {
		if _, ok := srv[field]; !ok {
			t.Errorf("Expected field '%s' not found in JSON", field)
		}
	}

	if srv["name"] != "Test Server" {
		t.Errorf("Expected name 'Test Server', got %v", srv["name"])
	}
	if srv["shareDatabaseSchemaLLM"] != true {
		t.Errorf("Expected shareDatabaseSchemaLLM to be true, got %v", srv["shareDatabaseSchemaLLM"])
	}

	if !contains(string(data), "  ") {
		t.Error("Expected JSON to be indented")
	}

	loaded, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load servers: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 loaded server, got %d", len(loaded))
	}

	if loaded[0].ID != created.ID {
		t.Error("Loaded server ID doesn't match created server ID")
	}
}

func TestConcurrentOperations(t *testing.T) {
	t.Parallel()

	tempDir := setupTempDir(t)
	defer os.RemoveAll(tempDir)

	for i := range 5 {
		server := CreateServer{
			Name:     string(rune('A' + i)),
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}
		_, err := New(server, tempDir)
		if err != nil {
			t.Fatalf("Failed to create initial server: %v", err)
		}
	}

	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_, err := Load(tempDir)
			if err != nil {
				t.Errorf("Concurrent load failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify final state
	servers, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load final state: %v", err)
	}

	if len(servers) != 5 {
		t.Errorf("Expected 5 servers, got %d", len(servers))
	}
}

// Helper functions

func setupTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "server_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

func saveServers(storage string, servers []Server) error {
	path := filepath.Join(storage, "servers.json")
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) != -1))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Benchmarks

func BenchmarkNew(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "server_bench")
	defer os.RemoveAll(tempDir)

	for i := 0; b.Loop(); i++ {
		server := CreateServer{
			Name:     "BenchServer" + string(rune(i)),
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}
		_, err := New(server, tempDir)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "server_bench")
	defer os.RemoveAll(tempDir)

	// Create some servers
	for i := range 100 {
		server := CreateServer{
			Name:     "Server" + string(rune(i)),
			Address:  "localhost",
			Port:     "5432",
			Username: "user",
			Password: "pass",
			Database: "db",
		}
		New(server, tempDir)
	}

	for b.Loop() {
		_, err := Load(tempDir)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}
