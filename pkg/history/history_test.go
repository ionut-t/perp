package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		existingLogs   []HistoryLog
		expectedCount  int
		expectError    bool
		validateResult func(t *testing.T, logs []HistoryLog)
	}{
		{
			name:          "add new query to empty history",
			query:         "SELECT * FROM users",
			existingLogs:  nil,
			expectedCount: 1,
			expectError:   false,
			validateResult: func(t *testing.T, logs []HistoryLog) {
				if logs[0].Query != "SELECT * FROM users" {
					t.Errorf("Expected query 'SELECT * FROM users', got '%s'", logs[0].Query)
				}
			},
		},
		{
			name:  "add query to existing history",
			query: "INSERT INTO users VALUES (1, 'John')",
			existingLogs: []HistoryLog{
				{Query: "SELECT * FROM users", Time: time.Now().Add(-time.Hour)},
			},
			expectedCount: 2,
			expectError:   false,
			validateResult: func(t *testing.T, logs []HistoryLog) {
				// Should be sorted newest first
				if logs[0].Query != "INSERT INTO users VALUES (1, 'John')" {
					t.Errorf("Expected newest query first, got '%s'", logs[0].Query)
				}
			},
		},
		{
			name:          "add empty query",
			query:         "",
			existingLogs:  nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "add whitespace-only query",
			query:         "   \n\t   ",
			existingLogs:  nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:  "add duplicate query",
			query: "SELECT * FROM users",
			existingLogs: []HistoryLog{
				{Query: "SELECT * FROM users", Time: time.Now().Add(-time.Hour)},
			},
			expectedCount: 1,
			expectError:   false,
			validateResult: func(t *testing.T, logs []HistoryLog) {
				// Should keep the newer entry
				if time.Since(logs[0].Time) > time.Minute {
					t.Error("Expected the newer duplicate to be kept")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Fatalf("Failed to remove temp dir: %v", err)
				}
			}()

			// Setup existing history if provided
			if tt.existingLogs != nil {
				err := writeHistoryLogs(filepath.Join(tempDir, historyFileName), tt.existingLogs)
				if err != nil {
					t.Fatalf("Failed to setup existing history: %v", err)
				}
			}

			logs, err := Add(tt.query, tempDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(logs) != tt.expectedCount {
				t.Errorf("Expected %d logs, got %d", tt.expectedCount, len(logs))
			}

			if tt.validateResult != nil {
				tt.validateResult(t, logs)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name          string
		setupLogs     []HistoryLog
		expectedCount int
		expectError   bool
	}{
		{
			name:          "get from empty history",
			setupLogs:     nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "get from existing history",
			setupLogs: []HistoryLog{
				{Query: "SELECT 1", Time: time.Now().Add(-time.Hour)},
				{Query: "SELECT 2", Time: time.Now().Add(-2 * time.Hour)},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "get sorted history (newest first)",
			setupLogs: []HistoryLog{
				{Query: "Old query", Time: time.Now().Add(-2 * time.Hour)},
				{Query: "New query", Time: time.Now().Add(-time.Hour)},
			},
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			// Setup history file if needed
			if tt.setupLogs != nil {
				err := writeHistoryLogs(filepath.Join(tempDir, historyFileName), tt.setupLogs)
				if err != nil {
					t.Fatalf("Failed to setup history: %v", err)
				}
			}

			logs, err := Get(tempDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(logs) != tt.expectedCount {
				t.Errorf("Expected %d logs, got %d", tt.expectedCount, len(logs))
			}

			// Verify sorting (newest first)
			for i := 1; i < len(logs); i++ {
				if logs[i-1].Time.Before(logs[i].Time) {
					t.Error("History logs are not sorted correctly (newest first)")
				}
			}
		})
	}
}

func TestReadHistoryLogs(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		expectedLogs []HistoryLog
		expectError  bool
	}{
		{
			name:        "read valid history file",
			fileContent: "---\n2024-01-01T12:00:00Z\nSELECT * FROM users\n---\n\n---\n2024-01-01T13:00:00Z\nINSERT INTO users VALUES (1)\n---",
			expectedLogs: []HistoryLog{
				{Query: "SELECT * FROM users", Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
				{Query: "INSERT INTO users VALUES (1)", Time: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)},
			},
			expectError: false,
		},
		{
			name:         "read empty file",
			fileContent:  "",
			expectedLogs: nil,
			expectError:  false,
		},
		{
			name:         "read file with invalid timestamp",
			fileContent:  "---\ninvalid-timestamp\nSELECT 1\n---",
			expectedLogs: nil,
			expectError:  false,
		},
		{
			name:         "read file with malformed entry",
			fileContent:  "---\n2024-01-01T12:00:00Z\n---",
			expectedLogs: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			historyPath := filepath.Join(tempDir, historyFileName)
			if tt.fileContent != "" {
				err := os.WriteFile(historyPath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			logs, err := readHistoryLogs(historyPath)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil && !os.IsNotExist(err) {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(logs) != len(tt.expectedLogs) {
				t.Errorf("Expected %d logs, got %d", len(tt.expectedLogs), len(logs))
			}

			for i, expectedLog := range tt.expectedLogs {
				if i >= len(logs) {
					break
				}
				if logs[i].Query != expectedLog.Query {
					t.Errorf("Log %d: expected query '%s', got '%s'", i, expectedLog.Query, logs[i].Query)
				}
				if !logs[i].Time.Equal(expectedLog.Time) {
					t.Errorf("Log %d: expected time %v, got %v", i, expectedLog.Time, logs[i].Time)
				}
			}
		})
	}
}

func TestWriteHistoryLogs(t *testing.T) {
	tempDir := setupTempDir(t)
	defer removeTempDir(t, tempDir)

	historyPath := filepath.Join(tempDir, historyFileName)
	logs := []HistoryLog{
		{Query: "SELECT 1", Time: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
		{Query: "SELECT 2", Time: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)},
	}

	err := writeHistoryLogs(historyPath, logs)
	if err != nil {
		t.Fatalf("Failed to write history logs: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("History file was not created")
	}

	// Read back and verify content
	readLogs, err := readHistoryLogs(historyPath)
	if err != nil {
		t.Fatalf("Failed to read back history logs: %v", err)
	}

	if len(readLogs) != len(logs) {
		t.Errorf("Expected %d logs after read, got %d", len(logs), len(readLogs))
	}

	for i, log := range logs {
		if i >= len(readLogs) {
			break
		}
		if readLogs[i].Query != log.Query {
			t.Errorf("Log %d: expected query '%s', got '%s'", i, log.Query, readLogs[i].Query)
		}
		if !readLogs[i].Time.Equal(log.Time) {
			t.Errorf("Log %d: expected time %v, got %v", i, log.Time, readLogs[i].Time)
		}
	}
}

func TestGetUniqueSortedHistory(t *testing.T) {
	now := time.Now()
	logs := []HistoryLog{
		{Query: "SELECT 1", Time: now.Add(-3 * time.Hour)},
		{Query: "SELECT 2", Time: now.Add(-time.Hour)},
		{Query: "SELECT 1", Time: now.Add(-2 * time.Hour)}, // Duplicate
		{Query: "SELECT 3", Time: now},
	}

	result := getUniqueSortedHistory(logs)

	// Should have 3 unique queries
	if len(result) != 3 {
		t.Errorf("Expected 3 unique logs, got %d", len(result))
	}

	// Should be sorted newest first
	for i := 1; i < len(result); i++ {
		if result[i-1].Time.Before(result[i].Time) {
			t.Error("History is not sorted correctly (newest first)")
		}
	}

	// Should keep the newer duplicate
	for _, log := range result {
		if log.Query == "SELECT 1" {
			expectedTime := now.Add(-2 * time.Hour)
			if !log.Time.Equal(expectedTime) {
				t.Error("Should keep the newer duplicate entry")
			}
		}
	}
}

func TestCleanupHistory(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		inputLogs     []HistoryLog
		expectedCount int
		description   string
	}{
		{
			name: "cleanup old entries",
			inputLogs: []HistoryLog{
				{Query: "Recent", Time: now.Add(-time.Hour)},
				{Query: "Old", Time: now.Add(-100 * 24 * time.Hour)}, // Older than 90 days
				{Query: "Recent2", Time: now.Add(-2 * time.Hour)},
			},
			expectedCount: 2,
			description:   "Should remove entries older than 90 days",
		},
		{
			name:          "limit max entries",
			inputLogs:     generateManyLogs(1200, now), // More than maxHistoryEntries (1000)
			expectedCount: maxHistoryEntries,
			description:   "Should limit to maxHistoryEntries",
		},
		{
			name: "no cleanup needed",
			inputLogs: []HistoryLog{
				{Query: "Recent1", Time: now.Add(-time.Hour)},
				{Query: "Recent2", Time: now.Add(-2 * time.Hour)},
			},
			expectedCount: 2,
			description:   "Should not remove recent entries within limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanupHistory(tt.inputLogs)

			if len(result) != tt.expectedCount {
				t.Errorf("%s: expected %d entries, got %d", tt.description, tt.expectedCount, len(result))
			}

			// Verify all remaining entries are within age limit
			cutoffTime := now.Add(-maxHistoryAge)
			for _, log := range result {
				if log.Time.Before(cutoffTime) {
					t.Errorf("Found entry older than cutoff time: %v", log.Time)
				}
			}

			// Verify sorting (newest first)
			for i := 1; i < len(result); i++ {
				if result[i-1].Time.Before(result[i].Time) {
					t.Error("Cleanup result is not sorted correctly")
				}
			}
		})
	}
}

func TestFilePermissions(t *testing.T) {
	tempDir := setupTempDir(t)
	defer removeTempDir(t, tempDir)

	// Test directory creation
	subDir := filepath.Join(tempDir, "subdir")
	historyPath := filepath.Join(subDir, historyFileName)

	logs := []HistoryLog{
		{Query: "SELECT 1", Time: time.Now()},
	}

	err := writeHistoryLogs(historyPath, logs)
	if err != nil {
		t.Fatalf("Failed to write to subdirectory: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Subdirectory was not created")
	}
}

// Table-driven tests for edge cases

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		action      func(tempDir string) error
		expectError bool
		description string
	}{
		{
			name: "invalid storage path",
			action: func(tempDir string) error {
				_, err := Add("SELECT 1", "/invalid/path/that/does/not/exist")
				return err
			},
			expectError: true,
			description: "Should handle invalid storage paths",
		},
		{
			name: "very long query",
			action: func(tempDir string) error {
				longQuery := strings.Repeat("SELECT ", 10000) + "1"
				_, err := Add(longQuery, tempDir)
				return err
			},
			expectError: false,
			description: "Should handle very long queries",
		},
		{
			name: "query with special characters",
			action: func(tempDir string) error {
				specialQuery := "SELECT 'test\\nwith\\ttabs\\rand\\\"quotes\\'';"
				_, err := Add(specialQuery, tempDir)
				return err
			},
			expectError: false,
			description: "Should handle queries with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := setupTempDir(t)
			defer removeTempDir(t, tempDir)

			err := tt.action(tempDir)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := setupTempDir(t)
	defer removeTempDir(t, tempDir)

	// Test concurrent adds
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			query := fmt.Sprintf("SELECT %d", id)
			_, err := Add(query, tempDir)
			if err != nil {
				t.Errorf("Concurrent add failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify final state
	logs, err := Get(tempDir)
	if err != nil {
		t.Fatalf("Failed to get final logs: %v", err)
	}

	if len(logs) != 10 {
		t.Errorf("Expected 10 unique logs after concurrent access, got %d", len(logs))
	}
}

// Helper functions

func removeTempDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("Failed to remove temp dir %s: %v", dir, err)
	}
}

func setupTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "history_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

func generateManyLogs(count int, baseTime time.Time) []HistoryLog {
	logs := make([]HistoryLog, count)
	for i := range count {
		logs[i] = HistoryLog{
			Query: fmt.Sprintf("SELECT %d", i),
			Time:  baseTime.Add(-time.Duration(i) * time.Minute),
		}
	}
	return logs
}

// Benchmark tests

func BenchmarkAdd(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "history_bench")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	for i := 0; b.Loop(); i++ {
		query := fmt.Sprintf("SELECT %d FROM table", i)
		_, err := Add(query, tempDir)
		if err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "history_bench")
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			b.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Setup some history
	for i := range 100 {
		query := fmt.Sprintf("SELECT %d FROM table", i)
		_, _ = Add(query, tempDir)
	}

	for b.Loop() {
		_, err := Get(tempDir)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}

func BenchmarkCleanupHistory(b *testing.B) {
	now := time.Now()
	logs := generateManyLogs(2000, now)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanupHistory(logs)
	}
}
