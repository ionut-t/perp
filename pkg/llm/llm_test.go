package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sql fenced block",
			input:    "Here is the query:\n```sql\nSELECT 1;\n```",
			expected: "SELECT 1;",
		},
		{
			name:     "plain fenced block without sql marker",
			input:    "Here is the query:\n```\nSELECT 1;\n```",
			expected: "SELECT 1;",
		},
		{
			name: "plain fence does not drop leading characters",
			// Previously advanced by len("```sql")=6 instead of len("```")=3,
			// causing the first 3 chars of the query to be silently dropped.
			input:    "```\nSEL\n```",
			expected: "SEL",
		},
		{
			name:     "sql fence preserves full query",
			input:    "```sql\nSELECT * FROM users;\n```",
			expected: "SELECT * FROM users;",
		},
		{
			name:     "no fence returns text as-is",
			input:    "SELECT 1;",
			expected: "SELECT 1;",
		},
		{
			name:     "unterminated fence returns rest of text",
			input:    "```sql\nSELECT 1;",
			expected: "SELECT 1;",
		},
		{
			name:     "leading and trailing whitespace is trimmed",
			input:    "```sql\n\n  SELECT 1;  \n\n```",
			expected: "SELECT 1;",
		},
		{
			name:     "sql marker is case insensitive",
			input:    "```SQL\nSELECT 1;\n```",
			expected: "SELECT 1;",
		},
		{
			name:  "sql fence takes priority over earlier plain fence",
			input: "```\nignored\n```\n\n```sql\nSELECT 1;\n```",
			// The function scans the whole text for "```sql" first, so the sql
			// block wins even though a plain fence appears earlier.
			expected: "SELECT 1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ExtractQuery(tt.input))
		})
	}
}

// TestExtractQueryPlainFenceNoDrop is a focused regression test for the bug
// where a plain ``` fence caused the first 3 characters of the query to be
// dropped because the advance was len("```sql")=6 instead of len("```")=3.
func TestExtractQueryPlainFenceNoDrop(t *testing.T) {
	t.Parallel()

	queries := []string{
		"SEL",
		"SELECT",
		"SELECT * FROM t",
		"ABC",
	}

	for _, q := range queries {
		input := "```\n" + q + "\n```"
		assert.Equal(t, q, ExtractQuery(input), "plain fence dropped chars for input %q", input)
	}
}
