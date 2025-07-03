package db

import (
	"testing"
)

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
