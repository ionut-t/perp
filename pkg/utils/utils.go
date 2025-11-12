package utils

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ClearMsg struct{}

// ParseTableNames is a helper function that extracts and deduplicates table names from a raw input string.
func ParseTableNames(input string) []string {
	var tables []string
	seen := make(map[string]bool)

	// Split the input string by common delimiters like comma, space, tab, or newline.
	for _, table := range strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}) {
		trimmedTable := strings.TrimSpace(table)
		if trimmedTable != "" && !seen[trimmedTable] {
			tables = append(tables, trimmedTable)
			seen[trimmedTable] = true
		}
	}
	return tables
}

// ClearAfter returns a command that triggers a notification clear after a specified duration.
func ClearAfter(duration time.Duration) tea.Cmd {
	return tea.Tick(
		duration,
		func(t time.Time) tea.Msg {
			return ClearMsg{}
		},
	)
}

func Dispatch(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

func Duration(duration time.Duration) string {
	switch {
	case duration < time.Millisecond:
		return fmt.Sprintf("%dµs", duration.Microseconds())
	case duration < time.Second:
		return fmt.Sprintf("%dms", duration.Milliseconds())
	default:
		return fmt.Sprintf("%.3fs", duration.Seconds())
	}
}

// GenerateUniqueName generates a unique name by appending a counter suffix if the name already exists.
// It filters existing names by extension and performs case-insensitive comparison.
// The oldName parameter allows excluding a specific name from conflict checks (useful when renaming).
func GenerateUniqueName(existingNames []string, name, oldName string) string {
	ext := filepath.Ext(name)
	baseName := strings.TrimSuffix(name, ext)

	// Build a list of existing base names with the same extension, excluding oldName
	existingBaseNames := make(map[string]bool)
	for _, existing := range existingNames {
		if filepath.Ext(existing) == ext && existing != oldName {
			existingBaseNames[strings.ToLower(strings.TrimSuffix(existing, ext))] = true
		}
	}

	// Check if the base name conflicts and generate a unique name
	uniqueName := baseName
	counter := 1
	for existingBaseNames[strings.ToLower(uniqueName)] {
		uniqueName = baseName + "-" + strconv.Itoa(counter)
		counter++
	}

	return uniqueName + ext
}
