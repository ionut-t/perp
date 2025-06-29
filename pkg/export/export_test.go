package export

import (
	"reflect"
	"testing"
)

func TestGenerateUniqueName_NoConflict(t *testing.T) {
	names := []string{"foo", "bar"}
	result := generateUniqueName("baz.json", names)
	if result != "baz.json" {
		t.Errorf("expected 'baz.json', got '%s'", result)
	}
}

func TestGenerateUniqueName_OneConflict(t *testing.T) {
	names := []string{"foo", "bar", "baz"}
	result := generateUniqueName("baz.json", names)
	if result != "baz-1.json" {
		t.Errorf("expected 'baz-1.json', got '%s'", result)
	}
}

func TestGenerateUniqueName_MultipleConflicts(t *testing.T) {
	names := []string{"foo", "bar", "baz", "baz-1", "baz-2"}
	result := generateUniqueName("baz.json", names)
	if result != "baz-3.json" {
		t.Errorf("expected 'baz-3.json', got '%s'", result)
	}
}

func TestGenerateUniqueName_ConflictWithSimilarNames(t *testing.T) {
	names := []string{"baz", "baz-1", "baz-2", "baz-10"}
	result := generateUniqueName("baz.json", names)
	if result != "baz-3.json" {
		t.Errorf("expected 'baz-3.json', got '%s'", result)
	}
}

func TestGenerateUniqueName_EmptyNames(t *testing.T) {
	names := []string{}
	result := generateUniqueName("foo.json", names)
	if result != "foo.json" {
		t.Errorf("expected 'foo.json', got '%s'", result)
	}
}

func TestPrepareJSON(t *testing.T) {
	t.Parallel()

	sampleResults := []map[string]any{
		{"id": 1, "name": "John", "email": "john@example.com"},
		{"id": 2, "name": "Jane", "email": "jane@example.com"},
		{"id": 3, "name": "Bob", "email": "bob@example.com"},
	}

	tests := []struct {
		name         string
		queryResults []map[string]any
		rows         []int
		all          bool
		expectError  bool
		expectedData any
		description  string
	}{
		{
			name:         "export single row",
			queryResults: sampleResults,
			rows:         []int{2},
			all:          false,
			expectError:  false,
			expectedData: map[string]any{"id": 2, "name": "Jane", "email": "jane@example.com"},
			description:  "Should export a single row by index",
		},
		{
			name:         "export multiple rows",
			queryResults: sampleResults,
			rows:         []int{1, 3},
			all:          false,
			expectError:  false,
			expectedData: []map[string]any{
				{"id": 1, "name": "John", "email": "john@example.com"},
				{"id": 3, "name": "Bob", "email": "bob@example.com"},
			},
			description: "Should export multiple rows by indices",
		},
		{
			name:         "export all rows",
			queryResults: sampleResults,
			rows:         []int{},
			all:          true,
			expectError:  false,
			expectedData: []map[string]any{
				{"id": 1, "name": "John", "email": "john@example.com"},
				{"id": 2, "name": "Jane", "email": "jane@example.com"},
				{"id": 3, "name": "Bob", "email": "bob@example.com"},
			},
			description: "Should export all rows when all=true",
		},
		{
			name:         "export with invalid row index (too high)",
			queryResults: sampleResults,
			rows:         []int{5},
			all:          false,
			expectError:  false,
			expectedData: nil,
			description:  "Should handle invalid row indices gracefully",
		},
		{
			name:         "export with invalid row index (zero)",
			queryResults: sampleResults,
			rows:         []int{0},
			all:          false,
			expectError:  false,
			expectedData: nil,
			description:  "Should handle zero index gracefully",
		},
		{
			name:         "export with negative row index",
			queryResults: sampleResults,
			rows:         []int{-1},
			all:          false,
			expectError:  false,
			expectedData: nil,
			description:  "Should handle negative indices gracefully",
		},
		{
			name:         "export with mixed valid and invalid indices",
			queryResults: sampleResults,
			rows:         []int{1, 5, 2},
			all:          false,
			expectError:  false,
			expectedData: []map[string]any{
				{"id": 1, "name": "John", "email": "john@example.com"},
				{"id": 2, "name": "Jane", "email": "jane@example.com"},
			},
			description: "Should export only valid indices and skip invalid ones",
		},
		{
			name:         "export with nil query results",
			queryResults: nil,
			rows:         []int{1},
			all:          false,
			expectError:  true,
			expectedData: nil,
			description:  "Should return error when query results are nil",
		},
		{
			name:         "export with empty query results",
			queryResults: []map[string]any{},
			rows:         []int{1},
			all:          false,
			expectError:  false,
			expectedData: nil,
			description:  "Should handle empty query results",
		},
		{
			name:         "export all with empty query results",
			queryResults: []map[string]any{},
			rows:         []int{},
			all:          true,
			expectError:  false,
			expectedData: []map[string]any{},
			description:  "Should return empty slice when exporting all from empty results",
		},
		{
			name:         "export with empty rows slice",
			queryResults: sampleResults,
			rows:         []int{},
			all:          false,
			expectError:  false,
			expectedData: nil,
			description:  "Should return nil when no rows specified and all=false",
		},
		{
			name:         "export single row at boundary",
			queryResults: sampleResults,
			rows:         []int{3},
			all:          false,
			expectError:  false,
			expectedData: map[string]any{"id": 3, "name": "Bob", "email": "bob@example.com"},
			description:  "Should export row at the upper boundary correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareJSON(tt.queryResults, tt.rows, tt.all)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("PrepareJSON() expected error but got none: %s", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("PrepareJSON() unexpected error: %v (%s)", err, tt.description)
			}

			// For error cases, don't check the result
			if tt.expectError {
				return
			}

			// Check result
			if !reflect.DeepEqual(result, tt.expectedData) {
				t.Errorf("PrepareJSON() = %v, expected %v (%s)", result, tt.expectedData, tt.description)
			}
		})
	}
}

func TestPrepareJSONEdgeCases(t *testing.T) {
	t.Parallel()
	// Test with single element query results
	singleResult := []map[string]any{
		{"id": 1, "name": "Single"},
	}

	tests := []struct {
		name         string
		queryResults []map[string]any
		rows         []int
		all          bool
		expectedType string
		description  string
	}{
		{
			name:         "single result export single row",
			queryResults: singleResult,
			rows:         []int{1},
			all:          false,
			expectedType: "map[string]interface {}",
			description:  "Should return single map for single row export",
		},
		{
			name:         "single result export multiple rows",
			queryResults: singleResult,
			rows:         []int{1, 2},
			all:          false,
			expectedType: "[]map[string]interface {}",
			description:  "Should return slice even when only one valid row",
		},
		{
			name:         "all export returns slice type",
			queryResults: singleResult,
			rows:         []int{},
			all:          true,
			expectedType: "[]map[string]interface {}",
			description:  "Should always return slice when all=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PrepareJSON(tt.queryResults, tt.rows, tt.all)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			resultType := reflect.TypeOf(result).String()
			if resultType != tt.expectedType {
				t.Errorf("Expected type %s, got %s (%s)", tt.expectedType, resultType, tt.description)
			}
		})
	}
}

func BenchmarkPrepareJSONSingle(b *testing.B) {
	sampleResults := []map[string]any{
		{"id": 1, "name": "John"},
		{"id": 2, "name": "Jane"},
		{"id": 3, "name": "Bob"},
	}

	for b.Loop() {
		_, _ = PrepareJSON(sampleResults, []int{2}, false)
	}
}

func BenchmarkPrepareJSONMultiple(b *testing.B) {
	sampleResults := make([]map[string]any, 1000)
	for i := range sampleResults {
		sampleResults[i] = map[string]any{
			"id":   i + 1,
			"name": "User" + string(rune(i)),
		}
	}

	rows := []int{1, 100, 200, 300, 400, 500}

	b.ResetTimer()
	for b.Loop() {
		_, _ = PrepareJSON(sampleResults, rows, false)
	}
}

func BenchmarkPrepareJSONAll(b *testing.B) {
	sampleResults := make([]map[string]any, 100)
	for i := range sampleResults {
		sampleResults[i] = map[string]any{
			"id":   i + 1,
			"name": "User" + string(rune(i)),
		}
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = PrepareJSON(sampleResults, []int{}, true)
	}
}
