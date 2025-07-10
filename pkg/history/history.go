package history

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	historyFileName = ".history"
)

type Entry struct {
	Query string
	Time  time.Time
}

// Thread-safe history manager
type manager struct {
	mu      sync.RWMutex
	storage string
}

// Global manager instance with sync.Once for initialization
var (
	globalManager *manager
	managerOnce   sync.Once
)

// getManager returns a singleton manager instance for the given storage path
func getManager(storage string) *manager {
	managerOnce.Do(func() {
		globalManager = &manager{storage: storage}
	})
	return globalManager
}

// Add adds a new query to the history and returns the updated history logs.
func Add(query string, storage string, maxEntries, maxAgeInDays int) ([]Entry, error) {
	if maxEntries <= 0 {
		maxEntries = 1000
	}

	if maxAgeInDays <= 0 {
		maxAgeInDays = 90
	}

	manager := getManager(storage)
	manager.mu.Lock()
	defer manager.mu.Unlock()

	path := filepath.Join(storage, historyFileName)

	history, err := readHistoryLogs(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return getUniqueSortedHistory(history), nil
	}

	// Add new entry
	newLog := Entry{
		Query: query,
		Time:  time.Now(),
	}
	history = append(history, newLog)

	// Clean up old entries before writing
	history = cleanupHistory(history, maxEntries, time.Duration(maxAgeInDays)*time.Hour*24)

	// Write updated history
	if err := writeHistoryLogs(path, history); err != nil {
		return nil, err
	}

	return getUniqueSortedHistory(history), nil
}

// Get retrieves the history logs from the storage.
func Get(storage string) ([]Entry, error) {
	manager := getManager(storage)
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	path := filepath.Join(storage, historyFileName)

	history, err := readHistoryLogs(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}

	return getUniqueSortedHistory(history), nil
}

// writeHistoryLogs performs atomic writes to prevent corruption during concurrent access.
func writeHistoryLogs(path string, history []Entry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tempPath := path + ".tmp"

	var buf bytes.Buffer
	for i, log := range history {
		if i > 0 {
			buf.WriteString("\n")
		}

		buf.WriteString("---\n")
		buf.WriteString(log.Time.Format(time.RFC3339))
		buf.WriteString("\n")
		buf.WriteString(log.Query)
		buf.WriteString("\n---")
	}

	if err := os.WriteFile(tempPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write temporary history file: %w", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to replace history file: %w", err)
	}

	return nil
}

// readHistoryLogs reads the history logs from the specified path.
func readHistoryLogs(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var history []Entry
	entries := bytes.SplitSeq(data, []byte("\n---\n"))

	for entry := range entries {
		entry = bytes.TrimSpace(entry)
		if len(entry) == 0 {
			continue
		}

		// Remove leading/trailing --- if present
		entry = bytes.TrimPrefix(entry, []byte("---\n"))
		entry = bytes.TrimSuffix(entry, []byte("\n---"))

		// Find the timestamp line
		lines := bytes.SplitN(entry, []byte("\n"), 2)
		if len(lines) < 2 {
			continue
		}

		// Parse timestamp
		timeStr := string(bytes.TrimSpace(lines[0]))
		parsedTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}

		// Extract query content
		queryContent := bytes.TrimSpace(lines[1])

		query := string(queryContent)
		if query != "" {
			history = append(history, Entry{
				Query: query,
				Time:  parsedTime,
			})
		}
	}

	return history, nil
}

func getUniqueSortedHistory(history []Entry) []Entry {
	slices.SortFunc(history, func(a, b Entry) int {
		return b.Time.Compare(a.Time)
	})

	uniqueHistory := make([]Entry, 0, len(history))
	seen := make(map[string]bool)

	for _, log := range history {
		query := strings.TrimSpace(log.Query)
		if !seen[query] {
			seen[query] = true
			uniqueHistory = append(uniqueHistory, log)
		}
	}

	return uniqueHistory
}

// cleanupHistory removes old entries and keeps only the most recent ones.
func cleanupHistory(history []Entry, maxEntries int, maxAge time.Duration) []Entry {
	now := time.Now()
	cutoffTime := now.Add(-maxAge)

	// First, remove entries older than the cutoff time
	filtered := make([]Entry, 0, len(history))
	for _, log := range history {
		if log.Time.After(cutoffTime) {
			filtered = append(filtered, log)
		}
	}

	// Sort by time (newest first)
	slices.SortFunc(filtered, func(a, b Entry) int {
		return b.Time.Compare(a.Time)
	})

	// Keep only the most recent entries if we exceed the max count
	if len(filtered) > maxEntries {
		filtered = filtered[:maxEntries]
	}

	return filtered
}
