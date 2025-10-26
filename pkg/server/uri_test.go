package server

import (
	"testing"
)

func TestParseConnectionURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		uri         string
		expected    *ParsedURI
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid postgresql URI with all components",
			uri:  "postgresql://myuser:mypass@localhost:5432/mydb",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "myuser",
				Password: "mypass",
				Host:     "localhost",
				Port:     "5432",
				Database: "mydb",
			},
			expectError: false,
		},
		{
			name: "valid postgres URI (alternative protocol)",
			uri:  "postgres://testuser:testpass@dbhost:5433/testdb",
			expected: &ParsedURI{
				Protocol: "postgres",
				Username: "testuser",
				Password: "testpass",
				Host:     "dbhost",
				Port:     "5433",
				Database: "testdb",
			},
			expectError: false,
		},
		{
			name: "URI without password",
			uri:  "postgresql://user@localhost:5432/mydb",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "user",
				Password: "",
				Host:     "localhost",
				Port:     "5432",
				Database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without port (uses default for postgres)",
			uri:  "postgresql://user:pass@hostname/database",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "user",
				Password: "pass",
				Host:     "hostname",
				Port:     "5432", // default
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "URI with query parameters",
			uri:  "postgresql://user:pass@localhost:5432/mydb?sslmode=disable&connect_timeout=10",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "user",
				Password: "pass",
				Host:     "localhost",
				Port:     "5432",
				Database: "mydb", // query params stripped
			},
			expectError: false,
		},
		{
			name: "URI with special characters in password",
			uri:  "postgresql://user:p@ss%21word@localhost:5432/mydb",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "user",
				Password: "p@ss!word", // URL decoded
				Host:     "localhost",
				Port:     "5432",
				Database: "mydb",
			},
			expectError: false,
		},
		{
			name:        "empty URI",
			uri:         "",
			expected:    nil,
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "URI with only whitespace",
			uri:         "   ",
			expected:    nil,
			expectError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name: "postgres alternative scheme",
			uri:  "postgres://user:pass@localhost:5433/mydb",
			expected: &ParsedURI{
				Protocol: "postgres",
				Username: "user",
				Password: "pass",
				Host:     "localhost",
				Port:     "5433",
				Database: "mydb",
			},
			expectError: false,
		},
		{
			name: "postgresql with IPv6 host",
			uri:  "postgresql://admin:secret@[::1]:5432/production",
			expected: &ParsedURI{
				Protocol: "postgresql",
				Username: "admin",
				Password: "secret",
				Host:     "::1",
				Port:     "5432",
				Database: "production",
			},
			expectError: false,
		},
		{
			name:        "URI without protocol",
			uri:         "user:pass@localhost:5432/mydb",
			expected:    nil,
			expectError: true,
			errorMsg:    "invalid URI format", // url.Parse treats "user" as scheme but missing host
		},
		{
			name:        "URI without username",
			uri:         "postgresql://:pass@localhost:5432/mydb",
			expected:    nil,
			expectError: true,
			errorMsg:    "missing username",
		},
		{
			name:        "URI without host",
			uri:         "postgresql://user:pass@:5432/mydb",
			expected:    nil,
			expectError: true,
			errorMsg:    "missing host",
		},
		{
			name:        "URI without database",
			uri:         "postgresql://user:pass@localhost:5432",
			expected:    nil,
			expectError: true,
			errorMsg:    "missing database",
		},
		{
			name:        "URI without database (with trailing slash)",
			uri:         "postgresql://user:pass@localhost:5432/",
			expected:    nil,
			expectError: true,
			errorMsg:    "missing database",
		},
		{
			name:        "malformed URI",
			uri:         "this is not a uri",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseConnectionURI(tt.uri)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Result is nil but no error was returned")
			}

			// Compare all fields
			if result.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol: expected %s, got %s", tt.expected.Protocol, result.Protocol)
			}
			if result.Username != tt.expected.Username {
				t.Errorf("Username: expected %s, got %s", tt.expected.Username, result.Username)
			}
			if result.Password != tt.expected.Password {
				t.Errorf("Password: expected %s, got %s", tt.expected.Password, result.Password)
			}
			if result.Host != tt.expected.Host {
				t.Errorf("Host: expected %s, got %s", tt.expected.Host, result.Host)
			}
			if result.Port != tt.expected.Port {
				t.Errorf("Port: expected %s, got %s", tt.expected.Port, result.Port)
			}
			if result.Database != tt.expected.Database {
				t.Errorf("Database: expected %s, got %s", tt.expected.Database, result.Database)
			}
		})
	}
}

func TestParsedURIToCreateServer(t *testing.T) {
	t.Parallel()

	parsed := &ParsedURI{
		Protocol: "postgresql",
		Username: "testuser",
		Password: "testpass",
		Host:     "testhost",
		Port:     "5432",
		Database: "testdb",
	}

	result := parsed.ToCreateServer("MyServer", true)

	if result.Name != "MyServer" {
		t.Errorf("Name: expected MyServer, got %s", result.Name)
	}
	if result.Address != "testhost" {
		t.Errorf("Address: expected testhost, got %s", result.Address)
	}
	if result.Port != "5432" {
		t.Errorf("Port: expected 5432, got %s", result.Port)
	}
	if result.Username != "testuser" {
		t.Errorf("Username: expected testuser, got %s", result.Username)
	}
	if result.Password != "testpass" {
		t.Errorf("Password: expected testpass, got %s", result.Password)
	}
	if result.Database != "testdb" {
		t.Errorf("Database: expected testdb, got %s", result.Database)
	}
	if !result.ShareDatabaseSchemaLLM {
		t.Error("ShareDatabaseSchemaLLM: expected true, got false")
	}
}

func TestParsedURIString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		parsed   *ParsedURI
		expected string
	}{
		{
			name: "URI with password",
			parsed: &ParsedURI{
				Protocol: "postgresql",
				Username: "user",
				Password: "secret",
				Host:     "localhost",
				Port:     "5432",
				Database: "mydb",
			},
			expected: "postgresql://user:****@localhost:5432/mydb",
		},
		{
			name: "URI without password",
			parsed: &ParsedURI{
				Protocol: "postgres",
				Username: "appuser",
				Password: "",
				Host:     "dbserver",
				Port:     "5432",
				Database: "production",
			},
			expected: "postgres://appuser:@dbserver:5432/production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.parsed.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
