package export

import (
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

func TestPrepareCSV_UsesQueryColumnOrder(t *testing.T) {
	queryResults := []map[string]any{
		{"name": "foo", "id": 1, "created_at": "2026-01-01"},
		{"name": "bar", "id": 2, "created_at": "2026-01-02"},
	}
	columns := []string{"name", "id", "created_at"}

	data, err := PrepareCSV(queryResults, columns, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]string{
		{"name", "id", "created_at"},
		{"foo", "1", "2026-01-01"},
		{"bar", "2", "2026-01-02"},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestPrepareCSV_SelectedRowsUseQueryColumnOrder(t *testing.T) {
	queryResults := []map[string]any{
		{"name": "foo", "id": 1},
		{"name": "bar", "id": 2},
		{"name": "baz", "id": 3},
	}
	columns := []string{"name", "id"}

	data, err := PrepareCSV(queryResults, columns, nil, []int{1, 3}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]string{
		{"name", "id"},
		{"foo", "1"},
		{"baz", "3"},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestPrepareCSV_FallsBackToSortedKeysWithoutColumns(t *testing.T) {
	queryResults := []map[string]any{
		{"name": "foo", "id": 1, "created_at": "2026-01-01"},
	}

	data, err := PrepareCSV(queryResults, nil, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]string{
		{"created_at", "id", "name"},
		{"2026-01-01", "1", "foo"},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestPrepareCSV_FallsBackWhenColumnsDoNotCoverResults(t *testing.T) {
	queryResults := []map[string]any{
		{"name": "foo", "id": 1},
	}

	// Stale/partial columns must not silently drop the "id" column.
	data, err := PrepareCSV(queryResults, []string{"name"}, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]string{
		{"id", "name"},
		{"1", "foo"},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestPrepareCSV_NoResults(t *testing.T) {
	if _, err := PrepareCSV(nil, []string{"id"}, nil, nil, true); err == nil {
		t.Error("expected an error for empty query results")
	}
}

func TestPrepareCSV_NullValuesAreEmptyFields(t *testing.T) {
	var nilPtr *string

	queryResults := []map[string]any{
		{"id": 1, "name": nil, "notes": nilPtr, "tags": []string(nil)},
	}
	columns := []string{"id", "name", "notes", "tags"}

	data, err := PrepareCSV(queryResults, columns, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]string{
		{"id", "name", "notes", "tags"},
		{"1", "", "", ""},
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("expected %v, got %v", expected, data)
	}
}

func TestPrepareCSV_NullColumnStillIncludedInHeader(t *testing.T) {
	// A NULL value is still a present key, so the column must not be dropped
	// from the header (which would trigger the sorted-keys fallback).
	queryResults := []map[string]any{
		{"name": nil, "id": 1},
	}

	data, err := PrepareCSV(queryResults, []string{"name", "id"}, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(data[0], []string{"name", "id"}) {
		t.Errorf("expected header [name id], got %v", data[0])
	}
}

func TestPrepareCSV_FormatsPostgresTypes(t *testing.T) {
	numeric := pgtype.Numeric{}
	if err := numeric.Scan("1234.56"); err != nil {
		t.Fatalf("failed to build numeric: %v", err)
	}

	queryResults := []map[string]any{
		{
			"data":     []byte{1, 2, 3},
			"payload":  []byte(`{"a": 1}`),
			"tags":     []any{"a", "b"},
			"amount":   numeric,
			"id":       [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			"quantity": int32(42),
		},
	}
	columns := []string{"data", "payload", "tags", "amount", "id", "quantity"}
	columnTypes := map[string]uint32{
		"data":     pgtype.ByteaOID,
		"payload":  pgtype.JSONBOID,
		"tags":     pgtype.TextArrayOID,
		"amount":   pgtype.NumericOID,
		"id":       pgtype.UUIDOID,
		"quantity": pgtype.Int4OID,
	}

	data, err := PrepareCSV(queryResults, columns, columnTypes, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		`\x010203`,
		`{"a":1}`,
		"{a,b}",
		"1234.56",
		"01020304-0506-0708-090a-0b0c0d0e0f10",
		"42",
	}
	if !reflect.DeepEqual(data[1], expected) {
		t.Errorf("expected %v, got %v", expected, data[1])
	}
}

func TestPrepareCSV_FormatsFloatsAndTimestamps(t *testing.T) {
	queryResults := []map[string]any{
		{
			"price":      1.5,
			"ratio":      float32(0.25),
			"created_at": time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}
	columns := []string{"price", "ratio", "created_at"}
	columnTypes := map[string]uint32{
		"price":      pgtype.Float8OID,
		"ratio":      pgtype.Float4OID,
		"created_at": pgtype.TimestamptzOID,
	}

	data, err := PrepareCSV(queryResults, columns, columnTypes, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"1.5", "0.25", "2026-01-02T03:04:05Z"}
	if !reflect.DeepEqual(data[1], expected) {
		t.Errorf("expected %v, got %v", expected, data[1])
	}
}

func TestPrepareCSV_LeavesPreformattedValuesUntouched(t *testing.T) {
	// The psql command path formats its values up front and reports no column
	// types, so they must pass through unchanged.
	queryResults := []map[string]any{
		{"Name": "public", "Size": "8192 bytes", "Owner": "postgres"},
	}
	columns := []string{"Name", "Size", "Owner"}

	data, err := PrepareCSV(queryResults, columns, nil, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"public", "8192 bytes", "postgres"}
	if !reflect.DeepEqual(data[1], expected) {
		t.Errorf("expected %v, got %v", expected, data[1])
	}
}

func TestPrepareCSV_FormatsJSONArrayColumn(t *testing.T) {
	// A jsonb column holding a top-level array decodes to []any, which must
	// still be written as JSON rather than as Go's map/slice rendering.
	queryResults := []map[string]any{
		{
			"provider": "the_odds_api",
			"payload": []any{
				map[string]any{"home_team": "Ipswich Town", "price": 5.0},
				map[string]any{"home_team": "Liverpool", "price": 1.53},
			},
		},
	}
	columns := []string{"provider", "payload"}
	columnTypes := map[string]uint32{
		"provider": pgtype.TextOID,
		"payload":  pgtype.JSONBOID,
	}

	data, err := PrepareCSV(queryResults, columns, columnTypes, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"the_odds_api",
		`[{"home_team":"Ipswich Town","price":5},{"home_team":"Liverpool","price":1.53}]`,
	}
	if !reflect.DeepEqual(data[1], expected) {
		t.Errorf("expected %v, got %v", expected, data[1])
	}
}
