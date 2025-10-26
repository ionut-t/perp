package psql

import (
	"testing"
)

func TestParseNewCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectedCmd CommandType
		expectError bool
	}{
		// Materialized views
		{
			name:        "parse \\dm",
			input:       "\\dm",
			expectedCmd: CmdListMaterializedViews,
			expectError: false,
		},
		{
			name:        "parse \\dm+",
			input:       "\\dm+",
			expectedCmd: CmdListMaterializedViews,
			expectError: false,
		},

		// Extensions
		{
			name:        "parse \\dx",
			input:       "\\dx",
			expectedCmd: CmdListExtensions,
			expectError: false,
		},
		{
			name:        "parse \\dx+",
			input:       "\\dx+",
			expectedCmd: CmdListExtensions,
			expectError: false,
		},

		// Privileges
		{
			name:        "parse \\dp",
			input:       "\\dp",
			expectedCmd: CmdListPrivileges,
			expectError: false,
		},
		{
			name:        "parse \\z (alternative)",
			input:       "\\z",
			expectedCmd: CmdListPrivileges,
			expectError: false,
		},

		// Connection info
		{
			name:        "parse \\conninfo",
			input:       "\\conninfo",
			expectedCmd: CmdConnInfo,
			expectError: false,
		},

		// Toggle expanded
		{
			name:        "parse \\x",
			input:       "\\x",
			expectedCmd: CmdToggleExpanded,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cmd.Type != tt.expectedCmd {
				t.Errorf("Expected command type %v, got %v", tt.expectedCmd, cmd.Type)
			}

			if cmd.Raw != tt.input {
				t.Errorf("Expected raw input %q, got %q", tt.input, cmd.Raw)
			}
		})
	}
}

func TestIsExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected bool
	}{
		{
			name:     "\\dm without plus",
			raw:      "\\dm",
			expected: false,
		},
		{
			name:     "\\dm with plus",
			raw:      "\\dm+",
			expected: true,
		},
		{
			name:     "\\dx without plus",
			raw:      "\\dx",
			expected: false,
		},
		{
			name:     "\\dx with plus",
			raw:      "\\dx+",
			expected: true,
		},
		{
			name:     "command with semicolon",
			raw:      "\\dm+;",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &Command{Raw: tt.raw}
			result := cmd.IsExtended()

			if result != tt.expected {
				t.Errorf("IsExtended() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCommandDescriptions(t *testing.T) {
	t.Parallel()

	// Verify that all new commands have descriptions
	requiredDescriptions := map[string]bool{
		PSQL_ListMaterializedViews:     false,
		PSQL_ListMaterializedViewsPlus: false,
		PSQL_ListExtensions:            false,
		PSQL_ListExtensionsPlus:        false,
		PSQL_ListPrivileges:            false,
		PSQL_ListPrivilegesAlt:         false,
		PSQL_ConnInfo:                  false,
		PSQL_ToggleExpanded:            false,
	}

	for _, desc := range CommandDescriptions {
		if _, exists := requiredDescriptions[desc.Command]; exists {
			requiredDescriptions[desc.Command] = true
		}
	}

	for cmd, found := range requiredDescriptions {
		if !found {
			t.Errorf("Missing description for command: %s", cmd)
		}
	}
}
