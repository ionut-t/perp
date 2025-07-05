package db

import (
	"encoding/json"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jackc/pgx/pgtype"
)

func TestFormatValue(t *testing.T) {
	t.Parallel()

	uuidBytes := [16]byte{0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00}

	jsonBytes, _ := json.Marshal(map[string]string{"key": "value"})
	jsonMap := map[string]any{"foo": "bar", "bazz": "buzz"}

	testCases := []struct {
		name     string
		input    any
		oid      uint32
		expected any
	}{
		{"nil value", nil, 0, nil},
		{"string value", "hello", pgtype.TextOID, "hello"},
		{"integer value", 123, pgtype.Int4OID, "123"},
		{"float value", 123.45, pgtype.Float8OID, "123.450000"},

		{
			name:     "json value from bytes",
			input:    jsonBytes,
			oid:      pgtype.JSONOID,
			expected: "{\"key\":\"value\"}",
		},
		{
			name:     "jsonb value from map",
			input:    jsonMap,
			oid:      pgtype.JSONBOID,
			expected: "{\"bazz\":\"buzz\",\"foo\":\"bar\"}",
		},
		{
			name:     "uuid value",
			input:    uuidBytes,
			oid:      pgtype.UUIDOID,
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "bytea value",
			input:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
			oid:      pgtype.ByteaOID,
			expected: "\\xdeadbeef",
		},
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			oid:      pgtype.TextArrayOID,
			expected: "{a,b,c}",
		},
		{
			name:     "integer slice",
			input:    []int{1, 2, 3},
			oid:      pgtype.Int4ArrayOID,
			expected: "{1,2,3}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, FormatValue(tc.input, tc.oid))
		})
	}
}

func TestStripSQLComments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comments",
			input:    `SELECT * FROM users;`,
			expected: `SELECT * FROM users;`,
		},
		{
			name:     "simple single-line comment",
			input:    `SELECT * FROM users; -- get all users`,
			expected: `SELECT * FROM users; `,
		},
		{
			name:     "single-line comment with newline",
			input:    `SELECT * FROM users; -- get all users\nSELECT 1;`,
			expected: `SELECT * FROM users; `,
		},
		{
			name:     "simple multi-line comment",
			input:    `SELECT * /* get all users */ FROM users;`,
			expected: `SELECT *  FROM users;`,
		},
		{
			name:     "multi-line comment with newlines",
			input:    `SELECT * /*\n get all users \n*/ FROM users;`,
			expected: `SELECT *  FROM users;`,
		},
		{
			name:     "comment in string",
			input:    `SELECT '-- this is not a comment' FROM users;`,
			expected: `SELECT '-- this is not a comment' FROM users;`,
		},
		{
			name:     "string with escaped quote",
			input:    `SELECT 'it''s a string' FROM users;`,
			expected: `SELECT 'it''s a string' FROM users;`,
		},
		{
			name:     "dollar-quoted string",
			input:    `SELECT $tag$ -- not a comment $tag$ FROM users;`,
			expected: `SELECT $tag$ -- not a comment $tag$ FROM users;`,
		},
		{
			name:     "comment after dollar-quoted string",
			input:    `SELECT $tag$ not a comment $tag$; -- a real comment`,
			expected: `SELECT $tag$ not a comment $tag$; `,
		},
		{
			name: "complex query",
			input: `
-- Get all active users
SELECT
    u.id,
    u.name,
    p.profile_data ->> 'email' AS email -- Extract email from JSON
FROM
    users u
    JOIN profiles p ON u.id = p.user_id
WHERE
    u.active = TRUE; /* Only active users */
`,
			expected: `

SELECT
    u.id,
    u.name,
    p.profile_data ->> 'email' AS email 
FROM
    users u
    JOIN profiles p ON u.id = p.user_id
WHERE
    u.active = TRUE; 
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripSQLComments(tc.input)
			if result != tc.expected {
				t.Errorf("\n--- FAIL: %s ---\nInput:    %q\nGot:      %q\nExpected: %q", tc.name, tc.input, result, tc.expected)
			}
		})
	}
}
