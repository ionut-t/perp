package psql

import (
	"strings"
	"testing"
)

func TestValidatePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pattern     string
		expectError bool
	}{
		{
			name:        "valid simple pattern",
			pattern:     "users",
			expectError: false,
		},
		{
			name:        "valid pattern with wildcard",
			pattern:     "user*",
			expectError: false,
		},
		{
			name:        "valid pattern with question mark",
			pattern:     "user?",
			expectError: false,
		},
		{
			name:        "valid pattern with schema",
			pattern:     "public.users",
			expectError: false,
		},
		{
			name:        "valid pattern with underscore",
			pattern:     "user_data",
			expectError: false,
		},
		{
			name:        "valid pattern with dollar sign",
			pattern:     "user$data",
			expectError: false,
		},
		{
			name:        "valid pattern with hyphen",
			pattern:     "user-data",
			expectError: false,
		},
		{
			name:        "SQL injection attempt - single quote",
			pattern:     "'; DROP TABLE users; --",
			expectError: true,
		},
		{
			name:        "SQL injection attempt - semicolon",
			pattern:     "test; DELETE FROM users",
			expectError: true,
		},
		{
			name:        "SQL injection attempt - OR clause",
			pattern:     "' OR '1'='1",
			expectError: true,
		},
		{
			name:        "pattern with spaces",
			pattern:     "user data",
			expectError: true,
		},
		{
			name:        "pattern with parentheses",
			pattern:     "user(data)",
			expectError: true,
		},
		{
			name:        "pattern with brackets",
			pattern:     "user[0]",
			expectError: true,
		},
		{
			name:        "empty pattern is valid",
			pattern:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePattern(tt.pattern)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for pattern: %s", tt.pattern)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for pattern %s: %v", tt.pattern, err)
			}
		})
	}
}

func TestPatternToLike(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple pattern",
			input:    "users",
			expected: "users",
		},
		{
			name:     "wildcard asterisk",
			input:    "user*",
			expected: "user%",
		},
		{
			name:     "wildcard question mark",
			input:    "user?",
			expected: "user_",
		},
		{
			name:     "multiple wildcards",
			input:    "u*er?",
			expected: "u%er_",
		},
		{
			name:     "escape percent sign",
			input:    "user%",
			expected: "user\\%",
		},
		{
			name:     "escape underscore",
			input:    "user_data",
			expected: "user\\_data",
		},
		{
			name:     "escape backslash",
			input:    "user\\data",
			expected: "user\\\\data",
		},
		{
			name:     "escape single quote",
			input:    "user'data",
			expected: "user''data",
		},
		{
			name:     "complex pattern with schema",
			input:    "public.user*",
			expected: "public.user%",
		},
		{
			name:     "pattern with dollar sign",
			input:    "user$data",
			expected: "user$data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patternToLike(tt.input)

			if result != tt.expected {
				t.Errorf("patternToLike(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSchemaAndTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		expectedSchema string
		expectedTable  string
	}{
		{
			name:           "table only",
			input:          "users",
			expectedSchema: "",
			expectedTable:  "users",
		},
		{
			name:           "schema and table",
			input:          "public.users",
			expectedSchema: "public",
			expectedTable:  "users",
		},
		{
			name:           "schema with wildcard",
			input:          "public.user*",
			expectedSchema: "public",
			expectedTable:  "user*",
		},
		{
			name:           "multiple dots - only first is schema separator",
			input:          "my.schema.table",
			expectedSchema: "my",
			expectedTable:  "schema.table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table := parseSchemaAndTable(tt.input)

			if schema != tt.expectedSchema {
				t.Errorf("schema = %q, expected %q", schema, tt.expectedSchema)
			}

			if table != tt.expectedTable {
				t.Errorf("table = %q, expected %q", table, tt.expectedTable)
			}
		})
	}
}

func TestBuildPatternCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pattern   string
		schemaCol string
		nameCol   string
		expected  string
	}{
		{
			name:      "empty pattern",
			pattern:   "",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  "",
		},
		{
			name:      "table only pattern",
			pattern:   "users",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  " AND c.relname LIKE 'users' ESCAPE '\\'",
		},
		{
			name:      "table with wildcard",
			pattern:   "user*",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  " AND c.relname LIKE 'user%' ESCAPE '\\'",
		},
		{
			name:      "schema and table",
			pattern:   "public.users",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  " AND n.nspname LIKE 'public' ESCAPE '\\' AND c.relname LIKE 'users' ESCAPE '\\'",
		},
		{
			name:      "schema and table with wildcards",
			pattern:   "pub*.user?",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  " AND n.nspname LIKE 'pub%' ESCAPE '\\' AND c.relname LIKE 'user_' ESCAPE '\\'",
		},
		{
			name:      "pattern with underscore (should be escaped)",
			pattern:   "user_data",
			schemaCol: "n.nspname",
			nameCol:   "c.relname",
			expected:  " AND c.relname LIKE 'user\\_data' ESCAPE '\\'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPatternCondition(tt.pattern, tt.schemaCol, tt.nameCol)

			if result != tt.expected {
				t.Errorf("buildPatternCondition() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSQLInjectionPrevention(t *testing.T) {
	t.Parallel()

	// Test that various SQL injection attempts are blocked
	injectionAttempts := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"'; DELETE FROM users WHERE '1'='1",
		"admin'--",
		"' UNION SELECT * FROM passwords--",
		"1'; DROP TABLE users CASCADE; --",
		"anything' OR 'x'='x",
	}

	for _, attempt := range injectionAttempts {
		t.Run("injection_attempt_"+attempt, func(t *testing.T) {
			err := validatePattern(attempt)

			if err == nil {
				t.Errorf("SQL injection attempt was not blocked: %s", attempt)
			}

			if !strings.Contains(err.Error(), "invalid pattern") {
				t.Errorf("Expected 'invalid pattern' error, got: %v", err)
			}
		})
	}
}

func TestPatternEscaping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		pattern              string
		shouldContainLiteral string // What the pattern should match literally after escaping
	}{
		{
			name:                 "percent sign is escaped",
			pattern:              "100%",
			shouldContainLiteral: "\\%",
		},
		{
			name:                 "underscore is escaped",
			pattern:              "user_id",
			shouldContainLiteral: "\\_",
		},
		{
			name:                 "backslash is escaped",
			pattern:              "path\\to\\file",
			shouldContainLiteral: "\\\\",
		},
		{
			name:                 "single quote is escaped",
			pattern:              "O'Brien",
			shouldContainLiteral: "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patternToLike(tt.pattern)

			if !strings.Contains(result, tt.shouldContainLiteral) {
				t.Errorf("patternToLike(%q) = %q, expected to contain %q",
					tt.pattern, result, tt.shouldContainLiteral)
			}
		})
	}
}

func TestCommandTypeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdType  CommandType
		expected string
	}{
		{CmdListMaterializedViews, "list-materialized-views"},
		{CmdListExtensions, "list-extensions"},
		{CmdListPrivileges, "list-privileges"},
		{CmdConnInfo, "connection-info"},
		{CmdToggleExpanded, "toggle-expanded"},
		{CmdUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.cmdType.String()
			if result != tt.expected {
				t.Errorf("CommandType.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
